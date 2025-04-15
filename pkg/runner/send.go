package runner

import (
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/boy-hack/ksubdomain/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/pkg/device"
	"github.com/boy-hack/ksubdomain/pkg/runner/statusdb"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// packetTemplateCache 缓存DNS服务器的包模板
type packetTemplateCache struct {
	mu      sync.RWMutex
	entries map[string]*packetTemplate
}

// packetTemplate DNS请求包模板
type packetTemplate struct {
	eth   *layers.Ethernet
	ip    *layers.IPv4
	udp   *layers.UDP
	opts  gopacket.SerializeOptions
	buf   gopacket.SerializeBuffer
	dnsip net.IP
}

func newPacketTemplateCache() *packetTemplateCache {
	return &packetTemplateCache{
		entries: make(map[string]*packetTemplate),
	}
}

func (c *packetTemplateCache) getOrCreate(dnsname string, ether *device.EtherTable, freeport uint16) *packetTemplate {
	c.mu.RLock()
	template, exists := c.entries[dnsname]
	c.mu.RUnlock()

	if exists {
		return template
	}

	// 创建新模板
	DstIp := net.ParseIP(dnsname).To4()
	eth := &layers.Ethernet{
		SrcMAC:       ether.SrcMac.HardwareAddr(),
		DstMAC:       ether.DstMac.HardwareAddr(),
		EthernetType: layers.EthernetTypeIPv4,
	}

	ip := &layers.IPv4{
		Version:    4,
		IHL:        5,
		TOS:        0,
		Length:     0, // FIX
		Id:         0,
		Flags:      layers.IPv4DontFragment,
		FragOffset: 0,
		TTL:        255,
		Protocol:   layers.IPProtocolUDP,
		Checksum:   0,
		SrcIP:      ether.SrcIp,
		DstIP:      DstIp,
	}

	udp := &layers.UDP{
		SrcPort: layers.UDPPort(freeport),
		DstPort: layers.UDPPort(53),
	}

	_ = udp.SetNetworkLayerForChecksum(ip)

	template = &packetTemplate{
		eth:   eth,
		ip:    ip,
		udp:   udp,
		dnsip: DstIp,
		opts: gopacket.SerializeOptions{
			ComputeChecksums: true,
			FixLengths:       true,
		},
		buf: gopacket.NewSerializeBuffer(),
	}

	c.mu.Lock()
	c.entries[dnsname] = template
	c.mu.Unlock()

	return template
}

// 发送包的缓存
var templateCache = newPacketTemplateCache()

// sendCycle 实现发送域名请求的循环
func (r *Runner) sendCycle() {
	// 从发送通道接收域名，分发给工作协程
	for domain := range r.domainChan {
		r.rateLimiter.Take()
		v, ok := r.statusDB.Get(domain)
		if !ok {
			v = statusdb.Item{
				Domain:      domain,
				Dns:         r.selectDNSServer(domain),
				Time:        time.Now(),
				Retry:       0,
				DomainLevel: 0,
			}
			r.statusDB.Add(domain, v)
		} else {
			v.Retry += 1
			v.Time = time.Now()
			v.Dns = r.selectDNSServer(domain)
			r.statusDB.Set(domain, v)
		}
		send(domain, v.Dns, r.options.EtherInfo, r.dnsID, uint16(r.listenPort), r.pcapHandle, layers.DNSTypeA)
		atomic.AddUint64(&r.sendCount, 1)
	}
}

// send 发送单个DNS查询包
func send(domain string, dnsname string, ether *device.EtherTable, dnsid uint16, freeport uint16, handle *pcap.Handle, dnsType layers.DNSType) {
	// 复用DNS服务器的包模板
	template := templateCache.getOrCreate(dnsname, ether, freeport)

	// 从内存池获取DNS层对象
	dns := GlobalMemPool.GetDNS()
	defer GlobalMemPool.PutDNS(dns)

	// 设置DNS查询参数
	dns.ID = dnsid
	dns.QDCount = 1
	dns.RD = true // 递归查询标识

	// 从内存池获取questions切片
	questions := GlobalMemPool.GetDNSQuestions()
	defer GlobalMemPool.PutDNSQuestions(questions)

	// 添加查询问题
	questions = append(questions, layers.DNSQuestion{
		Name:  []byte(domain),
		Type:  dnsType,
		Class: layers.DNSClassIN,
	})
	dns.Questions = questions

	// 从内存池获取序列化缓冲区
	buf := GlobalMemPool.GetBuffer()
	defer GlobalMemPool.PutBuffer(buf)

	// 序列化数据包
	err := gopacket.SerializeLayers(
		buf,
		template.opts,
		template.eth, template.ip, template.udp, dns,
	)
	if err != nil {
		gologger.Warningf("SerializeLayers faild:%s\n", err.Error())
		return
	}

	// 发送数据包
	err = handle.WritePacketData(buf.Bytes())
	if err == nil {
		return
	}
	// 如果是缓冲区错误，等待一段时间后重试
	if strings.Contains(err.Error(), "No buffer space available") {
		time.Sleep(time.Millisecond * 10)
		return
	}
	gologger.Warningf("WritePacketDate error:%s\n", err.Error())
}
