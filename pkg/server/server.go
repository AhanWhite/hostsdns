package server

import (
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"github.com/ahanwhite/hostsdns/pkg/config"
	dnsrecord "github.com/ahanwhite/hostsdns/pkg/record"
)

func NewDnsServer(hostfile string, port int) Interface {
	return &DNSServer{
		Hostfile: hostfile, // 指定 Hosts 文件路径
		Port:     port,     // 指定服务端口
	}
}

type DNSServer struct {
	Hostfile         string                         // Hosts 文件路径
	hosts            map[string]dnsrecord.DNSRecord // 域名与 IP 地址的映射
	lastModifiedTime time.Time                      // 文件最终修改时间
	lock             sync.RWMutex                   // 读写锁，用于保护 Hosts 映射的并发访问
	Port             int                            // 服务启用的udp端口
}

func (dnsserver *DNSServer) InitServer() error {
	err := dnsserver.LoadHostsFile()
	if err != nil {
		return err
	}

	return nil
}

func (dnsserver *DNSServer) LoadHostsFile() error {
	dnsserver.lock.Lock()
	defer dnsserver.lock.Unlock()

	hosts, err := config.ReadHostsFile(dnsserver.Hostfile)
	if err != nil {
		return err
	}

	fi, err := os.Stat(dnsserver.Hostfile)
	if err != nil {
		return err
	}

	dnsserver.hosts = hosts
	dnsserver.lastModifiedTime = fi.ModTime()

	return nil
}

// IsHostsFileModified 检查 Hosts 文件是否发生变动
func (dnsserver *DNSServer) IsHostsFileModified() (bool, error) {
	fi, err := os.Stat(dnsserver.Hostfile)
	if err != nil {
		return false, err
	}

	// 比较文件的修改时间
	return fi.ModTime().After(dnsserver.lastModifiedTime), nil
}

func (dnsserver *DNSServer) Start() error {

	server := net.UDPAddr{
		Port: dnsserver.Port,
		IP:   net.ParseIP("0.0.0.0"),
	}

	u, _ := net.ListenUDP("udp", &server)

	log.Printf("DNSServer started.")

	for {
		tmp := make([]byte, 1024)
		_, addr, _ := u.ReadFrom(tmp)
		clientAddr := addr
		packet := gopacket.NewPacket(tmp, layers.LayerTypeDNS, gopacket.Default)
		dnsPacket := packet.Layer(layers.LayerTypeDNS)
		tcp, _ := dnsPacket.(*layers.DNS)
		dnsserver.handleDNSQuery(u, clientAddr, tcp)
	}

}

// handleDNSQuery 处理 DNS 查询请求
func (dnsserver *DNSServer) handleDNSQuery(u *net.UDPConn, clientAddr net.Addr, request *layers.DNS) {
	replyMess := request
	// 解析域名对应的 IP 地址
	record, ok := dnsserver.hosts[string(request.Questions[0].Name)]
	if !ok {
		// 创建错误的 DNS 应答
		errorAnswer := layers.DNSResourceRecord{
			Type:  layers.DNSTypeA,
			Class: layers.DNSClassIN,
			TTL:   0,
		}
		// 设置响应码为 NXDomain
		replyMess.ResponseCode = layers.DNSResponseCodeNXDomain
		replyMess.Answers = []layers.DNSResourceRecord{errorAnswer}

		dnsserver.response(u, clientAddr, replyMess)
		return
	}

	// 创建 DNS 应答
	dnsAnswer := layers.DNSResourceRecord{
		Type:  record.IPType,
		IP:    net.ParseIP(record.IP),
		Name:  request.Questions[0].Name,
		Class: layers.DNSClassIN,
	}

	log.Printf("dnsAnswer.Name: %s, ip: %v\n", dnsAnswer.Name, dnsAnswer.IP)

	// 构建 DNS 应答消息
	replyMess.QR = true
	replyMess.ANCount = 1
	replyMess.OpCode = layers.DNSOpCodeNotify
	replyMess.AA = true
	replyMess.Answers = append(replyMess.Answers, dnsAnswer)
	replyMess.ResponseCode = layers.DNSResponseCodeNoErr

	dnsserver.response(u, clientAddr, replyMess)

}

// response
func (dnsserver *DNSServer) response(u *net.UDPConn, clientAddr net.Addr, replyMess *layers.DNS) {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	if err := replyMess.SerializeTo(buf, opts); err != nil {
		panic(err)
	}
	if _, err := u.WriteTo(buf.Bytes(), clientAddr); err != nil {
		panic(err)
	}
}
