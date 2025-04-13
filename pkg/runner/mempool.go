package runner

import (
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// MemoryPool 实现内存对象池
// 用于复用频繁分配的对象，减少GC压力
type MemoryPool struct {
	dnsPool      sync.Pool
	bufPool      sync.Pool
	questionPool sync.Pool
	answerPool   sync.Pool
}

// 全局内存池实例
var GlobalMemPool = NewMemoryPool()

// NewMemoryPool 创建一个新的内存池
func NewMemoryPool() *MemoryPool {
	return &MemoryPool{
		dnsPool: sync.Pool{
			New: func() interface{} {
				return &layers.DNS{
					Questions: make([]layers.DNSQuestion, 0, 1),
					Answers:   make([]layers.DNSResourceRecord, 0, 4),
				}
			},
		},
		bufPool: sync.Pool{
			New: func() interface{} {
				return gopacket.NewSerializeBuffer()
			},
		},
		questionPool: sync.Pool{
			New: func() interface{} {
				return make([]layers.DNSQuestion, 0, 1)
			},
		},
		answerPool: sync.Pool{
			New: func() interface{} {
				return make([]layers.DNSResourceRecord, 0, 4)
			},
		},
	}
}

// GetDNS 获取一个DNS对象
func (p *MemoryPool) GetDNS() *layers.DNS {
	dns := p.dnsPool.Get().(*layers.DNS)
	dns.Questions = dns.Questions[:0]
	dns.Answers = dns.Answers[:0]
	dns.Authorities = nil
	dns.Additionals = nil
	dns.ID = 0
	dns.QR = false
	dns.OpCode = 0
	dns.AA = false
	dns.TC = false
	dns.RD = true
	dns.RA = false
	dns.Z = 0
	dns.ResponseCode = 0
	dns.QDCount = 0
	dns.ANCount = 0
	dns.NSCount = 0
	dns.ARCount = 0
	return dns
}

// PutDNS 回收一个DNS对象
func (p *MemoryPool) PutDNS(dns *layers.DNS) {
	if dns != nil {
		p.dnsPool.Put(dns)
	}
}

// GetBuffer 获取一个序列化缓冲区
func (p *MemoryPool) GetBuffer() gopacket.SerializeBuffer {
	buf := p.bufPool.Get().(gopacket.SerializeBuffer)
	buf.Clear()
	return buf
}

// PutBuffer 回收一个序列化缓冲区
func (p *MemoryPool) PutBuffer(buf gopacket.SerializeBuffer) {
	if buf != nil {
		p.bufPool.Put(buf)
	}
}

// GetDNSQuestions 获取DNS问题切片
func (p *MemoryPool) GetDNSQuestions() []layers.DNSQuestion {
	questions := p.questionPool.Get().([]layers.DNSQuestion)
	return questions[:0]
}

// PutDNSQuestions 回收DNS问题切片
func (p *MemoryPool) PutDNSQuestions(questions []layers.DNSQuestion) {
	if questions != nil {
		p.questionPool.Put(questions)
	}
}

// GetDNSAnswers 获取DNS应答切片
func (p *MemoryPool) GetDNSAnswers() []layers.DNSResourceRecord {
	answers := p.answerPool.Get().([]layers.DNSResourceRecord)
	return answers[:0]
}

// PutDNSAnswers 回收DNS应答切片
func (p *MemoryPool) PutDNSAnswers(answers []layers.DNSResourceRecord) {
	if answers != nil {
		p.answerPool.Put(answers)
	}
}
