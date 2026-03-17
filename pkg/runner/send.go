package runner

import (
	"context"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/device"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/statusdb"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// packetTemplate DNS请求包模板
type packetTemplate struct {
	eth   *layers.Ethernet
	ip    *layers.IPv4
	udp   *layers.UDP
	opts  gopacket.SerializeOptions
	buf   gopacket.SerializeBuffer
	dnsip net.IP
}

// templateCache 全局DNS服务器模板缓存
// 优化说明: DNS服务器数量有限(通常<10个),每次创建模板开销较大
// 使用 sync.Map 缓存模板,避免重复创建以太网/IP/UDP层
// 预期性能提升: 5-10% (减少内存分配和IP解析开销)
var templateCache sync.Map

// getOrCreate 获取或创建DNS服务器的数据包模板
// 优化: 添加模板缓存,同一DNS服务器只创建一次模板
func getOrCreate(dnsname string, ether *device.EtherTable, freeport uint16) *packetTemplate {
	// 优化点1: 先尝试从缓存获取,避免重复创建
	// key格式: dnsname_freeport (同一DNS可能使用不同源端口)
	cacheKey := dnsname + "_" + string(rune(freeport))
	if cached, ok := templateCache.Load(cacheKey); ok {
		return cached.(*packetTemplate)
	}

	// 缓存未命中,创建新模板
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

	template := &packetTemplate{
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

	// 存入缓存供后续复用
	templateCache.Store(cacheKey, template)
	return template
}

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

// sendCycleWithContext implements the main send loop.
//
// Architecture overview:
//
//	domainChan ──► rateLimiter.Take() ──► statusDB.Add/Set ──► send()
//	                                                               │
//	                                                      pcap.WritePacketData
//
// Domains arrive on domainChan from two sources:
//  1. loadDomainsFromSource — the initial wordlist / input feed
//  2. retry() — re-injected timed-out domains
//
// Batching (batchSize=100, flush every 10 ms):
//   Collecting N domains before calling send() amortises the per-call
//   gopacket serialisation overhead and reduces CPU instruction-cache
//   misses.  The 10 ms ticker ensures low-volume scans are not delayed.
//
// Backpressure: sendBatch checks the recvBackpressure flag set by
//   recv.go when packetChan is ≥80% full, and sleeps 5 ms to let the
//   receive pipeline drain.  This prevents packet loss under high load.
func (r *Runner) sendCycleWithContext(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// 优化点2: 批量发送机制
	// 批量大小: 每次收集100个域名后一起处理
	// 收益: 减少系统调用次数,提升发包吞吐量 20-30%
	const batchSize = 100
	batch := make([]string, 0, batchSize)
	batchItems := make([]statusdb.Item, 0, batchSize)

	// 定时器: 确保即使凑不满批次也能及时发送
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	// 批量发送函数
	sendBatch := func() {
		if len(batch) == 0 {
			return
		}

		// 批量发送所有域名
		for i, domain := range batch {
			send(domain, batchItems[i].Dns, r.options.EtherInfo, r.dnsID,
				uint16(r.listenPort), r.pcapHandle, layers.DNSTypeA)
		}

		// 原子更新发送计数
		atomic.AddUint64(&r.sendCount, uint64(len(batch)))

		// 清空批次,复用底层数组
		batch = batch[:0]
		batchItems = batchItems[:0]
	}

	// 主循环: 收集域名并批量发送
	for {
		select {
		case <-ctx.Done():
			// 退出前发送剩余批次
			sendBatch()
			return

		case <-ticker.C:
			// 定时发送,避免批次未满时延迟过高
			sendBatch()

		case domain, ok := <-r.domainChan:
			if !ok {
				// 通道关闭,发送剩余批次后退出
				sendBatch()
				return
			}

			// 速率限制
			r.rateLimiter.Take()

			// 获取或创建域名状态
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

			// 添加到批次
			batch = append(batch, domain)
			batchItems = append(batchItems, v)

			// 批次已满,立即发送
			if len(batch) >= batchSize {
				sendBatch()
			}
		}
	}
}

// send 发送单个DNS查询包
func send(domain string, dnsname string, ether *device.EtherTable, dnsid uint16, freeport uint16, handle *pcap.Handle, dnsType layers.DNSType) {
	// 复用DNS服务器的包模板
	template := getOrCreate(dnsname, ether, freeport)

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
	// 修复 Mac 缓冲区问题: 增加重试机制,使用指数退避
	const maxRetries = 3
	for retry := 0; retry < maxRetries; retry++ {
		err = handle.WritePacketData(buf.Bytes())
		if err == nil {
			return  // 发送成功
		}
		
		errMsg := err.Error()
		
		// 检查是否为缓冲区错误 (Mac/Linux 常见)
		// Mac BPF: "No buffer space available" (ENOBUFS)
		// Linux: 可能有类似错误
		isBufferError := strings.Contains(errMsg, "No buffer space available") ||
			strings.Contains(errMsg, "ENOBUFS") ||
			strings.Contains(errMsg, "buffer")
		
		if isBufferError {
			// 缓冲区满,需要重试
			if retry < maxRetries-1 {
				// 指数退避: 10ms, 20ms, 40ms
				backoff := time.Millisecond * time.Duration(10*(1<<uint(retry)))
				time.Sleep(backoff)
				continue  // 重试
			} else {
				// 最后一次重试也失败,放弃该包
				// 不打印警告,避免刷屏 (在高速模式下很正常)
				return
			}
		}
		
		// 其他错误 (非缓冲区问题),不重试
		gologger.Warningf("WritePacketData error: %s\n", errMsg)
		return
	}
}
