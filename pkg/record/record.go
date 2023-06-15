package record

import "github.com/google/gopacket/layers"

type DNSRecord struct {
	IP     string
	IPType layers.DNSType
}

func NewDNSRecord(ip string, dnsType layers.DNSType) DNSRecord {
	return DNSRecord{
		IP:     ip,
		IPType: dnsType,
	}
}
