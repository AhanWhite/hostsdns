package config

import (
	"bufio"
	"log"
	"net"
	"os"
	"strings"

	"github.com/google/gopacket/layers"

	dnsrecord "github.com/ahanwhite/hostsdns/pkg/record"
)

// readHostsFile 读取 Hosts 文件并解析域名与 IP 地址的映射
func ReadHostsFile(filename string) (map[string]dnsrecord.DNSRecord, error) {
	hosts := make(map[string]dnsrecord.DNSRecord)

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
				hosts[hostname+"."] = dnsrecord.NewDNSRecord(ip, layers.DNSTypeA)
				hosts[hostname] = dnsrecord.NewDNSRecord(ip, layers.DNSTypeA)
			case ipAddr.To16() != nil:
				hosts[hostname+"."] = dnsrecord.NewDNSRecord(ip, layers.DNSTypeAAAA)
				hosts[hostname] = dnsrecord.NewDNSRecord(ip, layers.DNSTypeAAAA)
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
