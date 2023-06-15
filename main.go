package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/gopacket"
	layers "github.com/google/gopacket/layers"
)

func main() {
	// 创建一个新的 DNS 服务器
	dnsserver := &DNSServer{
		hostfile: "/etc/hostsdns", // 指定 Hosts 文件路径
		port:     53,              // 指定服务端口
	}

	// 初始化 DNS 服务器
	err := dnsserver.init()
	if err != nil {
		log.Fatal("Failed to initialize DNS server:", err)
	}

	// 监听 Hosts 文件变动并重新加载
	go func() {
		for range time.Tick(time.Second) {
			// 检查 Hosts 文件是否发生变动
			modified, err := dnsserver.isHostsFileModified()
			if err != nil {
				log.Println("Failed to check Hosts file modification:", err)
				continue
			}

			if modified {
				// Hosts 文件发生变动，重新加载
				err := dnsserver.loadHostsFile()
				if err != nil {
					log.Println("Failed to reload Hosts file:", err)
					continue
				}

				log.Println("Hosts file reloaded")
			}
		}
	}()

	// 启动 DNS 服务器
	err = dnsserver.start()
	if err != nil {
		log.Fatal("Failed to start DNS server:", err)
	}
}

type DNSRecord struct {
	ip     string
	iptype layers.DNSType
}

func NewDNSRecord(ip string, dnsType layers.DNSType) DNSRecord {
	return DNSRecord{
		ip:     ip,
		iptype: dnsType,
	}
}

type DNSServer struct {
	hostfile         string               // Hosts 文件路径
	hosts            map[string]DNSRecord // 域名与 IP 地址的映射
	lastModifiedTime time.Time            // 文件最终修改时间
	lock             sync.RWMutex         // 读写锁，用于保护 Hosts 映射的并发访问
	port             int                  // 服务启用的udp端口
}

func (dnsserver *DNSServer) init() error {
	err := dnsserver.loadHostsFile()
	if err != nil {
		return err
	}

	return nil
}

// loadHostsFile 加载 Hosts 文件并解析域名与 IP 地址的映射
func (dnsserver *DNSServer) loadHostsFile() error {
	dnsserver.lock.Lock()
	defer dnsserver.lock.Unlock()

	hosts, err := readHostsFile(dnsserver.hostfile)
	if err != nil {
		return err
	}

	fi, err := os.Stat(dnsserver.hostfile)
	if err != nil {
		return err
	}

	dnsserver.hosts = hosts
	dnsserver.lastModifiedTime = fi.ModTime()

	return nil
}

// isHostsFileModified 检查 Hosts 文件是否发生变动
func (dnsserver *DNSServer) isHostsFileModified() (bool, error) {
	fi, err := os.Stat(dnsserver.hostfile)
	if err != nil {
		return false, err
	}

	// 比较文件的修改时间
	return fi.ModTime().After(dnsserver.lastModifiedTime), nil
}

func (dnsserver *DNSServer) start() error {

	server := net.UDPAddr{
		Port: dnsserver.port,
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
		dnsserver.serverDns(u, clientAddr, tcp)
	}

}

// handleDNSQuery 处理 DNS 查询请求
func (dnsserver *DNSServer) serverDns(u *net.UDPConn, clientAddr net.Addr, request *layers.DNS) {
	replyMess := request
	// 解析域名对应的 IP 地址
	dnsRecord, ok := dnsserver.hosts[string(request.Questions[0].Name)]
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
		Type:  dnsRecord.iptype,
		IP:    net.ParseIP(dnsRecord.ip),
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

// readHostsFile 读取 Hosts 文件并解析域名与 IP 地址的映射
func readHostsFile(filename string) (map[string]DNSRecord, error) {
	hosts := make(map[string]DNSRecord)

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 忽略空行和以 # 开头的注释行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			// 不符合域名与 IP 地址对的格式，忽略该行并打印警告日志
			log.Println("Invalid hosts file entry:", line)
			continue
		}

		ip := fields[0]
		for i := 1; i < len(fields); i++ {
			hostname := fields[i]
			// 判断 IP 地址类型，并存储到相应的 map 中
			ipAddr := net.ParseIP(ip)
			switch {
			case ipAddr.To4() != nil:
				hosts[hostname+"."] = NewDNSRecord(ip, layers.DNSTypeA)
				hosts[hostname] = NewDNSRecord(ip, layers.DNSTypeA)
			case ipAddr.To16() != nil:
				hosts[hostname+"."] = NewDNSRecord(ip, layers.DNSTypeAAAA)
				hosts[hostname] = NewDNSRecord(ip, layers.DNSTypeAAAA)
			default:
				// 无法确定类型的IP地址，打印警告日志
				log.Println("Unknown IP address type:", ip)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return hosts, nil
}
