package runner

import (
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// MemoryPool 实现内存对象池
// 优化点5: 对象池复用策略优化
// 用途: 复用频繁分配的对象(DNS层、序列化缓冲区、切片)
// 收益: 减少内存分配次数和GC压力,降低延迟
// 关键: 归还前必须重置对象状态,避免数据污染
type MemoryPool struct {
	dnsPool      sync.Pool // DNS查询/响应层对象池
	bufPool      sync.Pool // gopacket序列化缓冲区池
	questionPool sync.Pool // DNS问题切片池
	answerPool   sync.Pool // DNS应答切片池
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
// 注意: 从池中获取的对象可能包含旧数据,必须重置所有字段
func (p *MemoryPool) GetDNS() *layers.DNS {
	dns := p.dnsPool.Get().(*layers.DNS)
	// 重置切片长度(保留底层数组容量)
	dns.Questions = dns.Questions[:0]
	dns.Answers = dns.Answers[:0]
	// nil 掉不常用字段,避免内存泄漏
	dns.Authorities = nil
	dns.Additionals = nil
	// 重置所有标志位为默认值
	dns.ID = 0
	dns.QR = false      // 查询报文
	dns.OpCode = 0      // 标准查询
	dns.AA = false      // 非权威应答
	dns.TC = false      // 未截断
	dns.RD = true       // 期望递归
	dns.RA = false      // 递归不可用
	dns.Z = 0           // 保留位
	dns.ResponseCode = 0 // 无错误
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
