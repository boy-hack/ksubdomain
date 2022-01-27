package core

import (
	"github.com/google/gopacket/layers"
)

// 接收结果数据结构
type RecvResult struct {
	Subdomain string
	Answers   []layers.DNSResourceRecord
}
