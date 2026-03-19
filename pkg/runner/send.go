package runner

import (
	"context"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
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

// getOrCreate 获取或创建 DNS 服务器的数据包模板（绑定到单张网卡的 templateCache）。
// key 格式：dnsname_freeport
func (iface *netInterface) getOrCreate(dnsname string) *packetTemplate {
	freeport := uint16(iface.listenPort)
	cacheKey := dnsname + "_" + string(rune(freeport))
	if cached, ok := iface.templateCache.Load(cacheKey); ok {
		return cached.(*packetTemplate)
	}

	ether := iface.etherInfo
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
		Length:     0,
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

	iface.templateCache.Store(cacheKey, template)
	return template
}

// sendCycleForIface 为单张网卡运行发送循环（A1 方案）。
// 多个 goroutine 共享同一个 domainChan，竞争消费实现自然负载均衡。
func (r *Runner) sendCycleForIface(ctx context.Context, wg *sync.WaitGroup, iface *netInterface) {
	defer wg.Done()

	const batchSize = 100
	batch := make([]string, 0, batchSize)
	batchItems := make([]statusdb.Item, 0, batchSize)

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	sendBatch := func() {
		if len(batch) == 0 {
			return
		}
		for i, domain := range batch {
			send(domain, batchItems[i].Dns, iface, r.dnsID, layers.DNSTypeA)
		}
		atomic.AddUint64(&r.sendCount, uint64(len(batch)))
		batch = batch[:0]
		batchItems = batchItems[:0]
	}

	for {
		select {
		case <-ctx.Done():
			sendBatch()
			return

		case <-ticker.C:
			sendBatch()

		case domain, ok := <-r.domainChan:
			if !ok {
				sendBatch()
				return
			}

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

			batch = append(batch, domain)
			batchItems = append(batchItems, v)

			if len(batch) >= batchSize {
				sendBatch()
			}
		}
	}
}

// send 发送单个DNS查询包（绑定到指定网卡上下文）
func send(domain string, dnsname string, iface *netInterface, dnsid uint16, dnsType layers.DNSType) {
	template := iface.getOrCreate(dnsname)

	dns := GlobalMemPool.GetDNS()
	defer GlobalMemPool.PutDNS(dns)

	dns.ID = dnsid
	dns.QDCount = 1
	dns.RD = true

	questions := GlobalMemPool.GetDNSQuestions()
	defer GlobalMemPool.PutDNSQuestions(questions)

	questions = append(questions, layers.DNSQuestion{
		Name:  []byte(domain),
		Type:  dnsType,
		Class: layers.DNSClassIN,
	})
	dns.Questions = questions

	buf := GlobalMemPool.GetBuffer()
	defer GlobalMemPool.PutBuffer(buf)

	err := gopacket.SerializeLayers(
		buf,
		template.opts,
		template.eth, template.ip, template.udp, dns,
	)
	if err != nil {
		gologger.Warningf("SerializeLayers faild:%s\n", err.Error())
		return
	}

	handle := iface.pcapHandle
	const maxRetries = 3
	for retry := 0; retry < maxRetries; retry++ {
		err = handle.WritePacketData(buf.Bytes())
		if err == nil {
			return
		}

		errMsg := err.Error()

		isBufferError := strings.Contains(errMsg, "No buffer space available") ||
			strings.Contains(errMsg, "ENOBUFS") ||
			strings.Contains(errMsg, "buffer")

		if isBufferError {
			if retry < maxRetries-1 {
				backoff := time.Millisecond * time.Duration(10*(1<<uint(retry)))
				time.Sleep(backoff)
				continue
			}
			return
		}

		gologger.Warningf("WritePacketData error: %s\n", errMsg)
		return
	}
}

// 保留旧 pcap.Handle 签名的引用，避免 recv.go 里的 testspeed 等直接用 handle 的地方编译失败。
// 实际发送路径已全部走 send(domain, dnsname, iface, ...)。
var _ = (*pcap.Handle)(nil)
