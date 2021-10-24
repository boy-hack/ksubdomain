package core

import (
	"github.com/google/gopacket/layers"
	"net"
)

// 本地状态表
type StatusTable struct {
	Domain      string // 查询域名
	Dns         string // 查询dns
	Time        int64  // 发送时间
	Retry       int    // 重试次数
	DomainLevel int    // 域名层级
}

// 接收结果数据结构
type RecvResult struct {
	Subdomain string
	Answers   []layers.DNSResourceRecord
}

type EthTable struct {
	SrcIp  net.IP
	Device string
	SrcMac net.HardwareAddr
	DstMac net.HardwareAddr
}
