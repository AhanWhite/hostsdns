package server

import (
	"net"

	"github.com/google/gopacket/layers"
)

type Interface interface {

	// InitServer 初始化服务
	InitServer() error

	// Start 启动服务
	Start() error

	// LoadHostsFile 加载 Hosts 文件并解析域名与 IP 地址的映射
	LoadHostsFile() error

	// IsHostsFileModified 判断解析文件是否被修改
	IsHostsFileModified() (bool, error)

	// handleDNSQuery 处理 DNS 查询请求
	handleDNSQuery(u *net.UDPConn, clientAddr net.Addr, request *layers.DNS)
}
