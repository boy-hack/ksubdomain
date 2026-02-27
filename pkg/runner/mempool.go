package runner

import (
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// MemoryPool implements a memory object pool.
// Optimization point 5: object pool reuse strategy optimization.
// Purpose: reuse frequently allocated objects (DNS layers, serialize buffers, slices).
// Benefit: reduces memory allocation frequency and GC pressure, lowers latency.
// Key: objects must be reset before being returned to the pool to avoid data contamination.
type MemoryPool struct {
	dnsPool      sync.Pool // DNS query/response layer object pool
	bufPool      sync.Pool // gopacket serialize buffer pool
	questionPool sync.Pool // DNS question slice pool
	answerPool   sync.Pool // DNS answer slice pool
}

// GlobalMemPool is the global memory pool instance
var GlobalMemPool = NewMemoryPool()

// NewMemoryPool creates a new memory pool
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

// GetDNS retrieves a DNS object from the pool.
// Note: objects obtained from the pool may contain stale data; all fields must be reset.
func (p *MemoryPool) GetDNS() *layers.DNS {
	dns := p.dnsPool.Get().(*layers.DNS)
	// Reset slice length (retain underlying array capacity)
	dns.Questions = dns.Questions[:0]
	dns.Answers = dns.Answers[:0]
	// Nil out infrequently used fields to avoid memory leaks
	dns.Authorities = nil
	dns.Additionals = nil
	// Reset all flags to default values
	dns.ID = 0
	dns.QR = false      // Query message
	dns.OpCode = 0      // Standard query
	dns.AA = false      // Not authoritative
	dns.TC = false      // Not truncated
	dns.RD = true       // Recursion desired
	dns.RA = false      // Recursion not available
	dns.Z = 0           // Reserved bits
	dns.ResponseCode = 0 // No error
	dns.QDCount = 0
	dns.ANCount = 0
	dns.NSCount = 0
	dns.ARCount = 0
	return dns
}

// PutDNS returns a DNS object to the pool
func (p *MemoryPool) PutDNS(dns *layers.DNS) {
	if dns != nil {
		p.dnsPool.Put(dns)
	}
}

// GetBuffer retrieves a serialize buffer from the pool
func (p *MemoryPool) GetBuffer() gopacket.SerializeBuffer {
	buf := p.bufPool.Get().(gopacket.SerializeBuffer)
	buf.Clear()
	return buf
}

// PutBuffer returns a serialize buffer to the pool
func (p *MemoryPool) PutBuffer(buf gopacket.SerializeBuffer) {
	if buf != nil {
		p.bufPool.Put(buf)
	}
}

// GetDNSQuestions retrieves a DNS question slice from the pool
func (p *MemoryPool) GetDNSQuestions() []layers.DNSQuestion {
	questions := p.questionPool.Get().([]layers.DNSQuestion)
	return questions[:0]
}

// PutDNSQuestions returns a DNS question slice to the pool
func (p *MemoryPool) PutDNSQuestions(questions []layers.DNSQuestion) {
	if questions != nil {
		p.questionPool.Put(questions)
	}
}

// GetDNSAnswers retrieves a DNS answer slice from the pool
func (p *MemoryPool) GetDNSAnswers() []layers.DNSResourceRecord {
	answers := p.answerPool.Get().([]layers.DNSResourceRecord)
	return answers[:0]
}

// PutDNSAnswers returns a DNS answer slice to the pool
func (p *MemoryPool) PutDNSAnswers(answers []layers.DNSResourceRecord) {
	if answers != nil {
		p.answerPool.Put(answers)
	}
}
