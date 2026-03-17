package runner

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// parseDNSName 解析 DNS 域名格式
// DNS 域名格式: 长度前缀 + 标签 + ... + 结束符
// 例如: \x03www\x06google\x03com\x00 表示 www.google.com
// 修复 Issue #70: 正确解析 CNAME/NS/PTR 等记录,避免出现 "comcom" 等拼接错误
func parseDNSName(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	
	var result []byte
	i := 0
	
	for i < len(raw) {
		// 读取标签长度
		length := int(raw[i])
		
		// 0x00 表示域名结束
		if length == 0 {
			break
		}
		
		// 0xC0 开头表示压缩指针 (RFC 1035)
		// 压缩格式: 前2位为11,后14位为偏移量
		if length >= 0xC0 {
			// 压缩指针,暂不处理(通常在完整DNS包中才有)
			break
		}
		
		// 添加点分隔符 (第一个标签除外)
		if len(result) > 0 {
			result = append(result, '.')
		}
		
		i++
		
		// 防止越界
		if i+length > len(raw) {
			break
		}
		
		// 添加标签内容
		result = append(result, raw[i:i+length]...)
		i += length
	}
	
	return string(result)
}

// dnsRecord2String 将DNS记录转换为字符串
// 修复 Issue #70: 使用 parseDNSName 正确解析域名格式
func dnsRecord2String(rr layers.DNSResourceRecord) (string, error) {
	if rr.Class == layers.DNSClassIN {
		switch rr.Type {
		case layers.DNSTypeA, layers.DNSTypeAAAA:
			if rr.IP != nil {
				return rr.IP.String(), nil
			}
		case layers.DNSTypeNS:
			if rr.NS != nil {
				// 修复: 使用 parseDNSName 解析 NS 记录
				ns := parseDNSName(rr.NS)
				if ns != "" {
					return "NS " + ns, nil
				}
			}
		case layers.DNSTypeCNAME:
			if rr.CNAME != nil {
				// 修复: 使用 parseDNSName 解析 CNAME 记录
				cname := parseDNSName(rr.CNAME)
				if cname != "" {
					return "CNAME " + cname, nil
				}
			}
		case layers.DNSTypePTR:
			if rr.PTR != nil {
				// 修复: 使用 parseDNSName 解析 PTR 记录
				ptr := parseDNSName(rr.PTR)
				if ptr != "" {
					return "PTR " + ptr, nil
				}
			}
		case layers.DNSTypeTXT:
			if rr.TXT != nil {
				// TXT 记录是纯文本,不需要解析
				return "TXT " + string(rr.TXT), nil
			}
		}
	}
	return "", errors.New("dns record error")
}

// 预分配解码器对象池，避免频繁创建
var decoderPool = sync.Pool{
	New: func() interface{} {
		var eth layers.Ethernet
		var ipv4 layers.IPv4
		var ipv6 layers.IPv6
		var udp layers.UDP
		var dns layers.DNS
		parser := gopacket.NewDecodingLayerParser(
			layers.LayerTypeEthernet, &eth, &ipv4, &ipv6, &udp, &dns)

		return &decodingContext{
			parser:  parser,
			eth:     &eth,
			ipv4:    &ipv4,
			ipv6:    &ipv6,
			udp:     &udp,
			dns:     &dns,
			decoded: make([]gopacket.LayerType, 0, 5),
		}
	},
}

// decodingContext 解码上下文
type decodingContext struct {
	parser  *gopacket.DecodingLayerParser
	eth     *layers.Ethernet
	ipv4    *layers.IPv4
	ipv6    *layers.IPv6
	udp     *layers.UDP
	dns     *layers.DNS
	decoded []gopacket.LayerType
}

// 解析DNS响应包并处理
func (r *Runner) processPacket(data []byte, dnsChanel chan<- layers.DNS) {
	// 从对象池获取解码器
	dc := decoderPool.Get().(*decodingContext)
	defer decoderPool.Put(dc)

	// 清空解码层类型切片
	dc.decoded = dc.decoded[:0]

	// 解析数据包
	err := dc.parser.DecodeLayers(data, &dc.decoded)
	if err != nil {
		return
	}

	// 检查是否为DNS响应
	if !dc.dns.QR {
		return
	}

	// 确认DNS ID匹配
	if dc.dns.ID != r.dnsID {
		return
	}

	// 确认有查询问题
	if len(dc.dns.Questions) == 0 {
		return
	}

	// 记录接收包数量
	atomic.AddUint64(&r.receiveCount, 1)

	// 向处理通道发送DNS响应
	select {
	case dnsChanel <- *dc.dns:
	}
}

// recvChanel 实现接收DNS响应的功能
func (r *Runner) recvChanel(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	var (
		snapshotLen = 65536
		timeout     = 5 * time.Second
		err         error
	)
	inactive, err := pcap.NewInactiveHandle(r.options.EtherInfo.Device)
	if err != nil {
		gologger.Errorf("创建网络捕获句柄失败: %v", err)
		return
	}
	err = inactive.SetSnapLen(snapshotLen)
	if err != nil {
		gologger.Errorf("设置抓包长度失败: %v", err)
		return
	}
	defer inactive.CleanUp()

	if err = inactive.SetTimeout(timeout); err != nil {
		gologger.Errorf("设置超时失败: %v", err)
		return
	}

	err = inactive.SetImmediateMode(true)
	if err != nil {
		gologger.Errorf("设置即时模式失败: %v", err)
		return
	}

	handle, err := inactive.Activate()
	if err != nil {
		gologger.Errorf("激活网络捕获失败: %v", err)
		return
	}
	defer handle.Close()

	err = handle.SetBPFFilter(fmt.Sprintf("udp and src port 53 and dst port %d", r.listenPort))
	if err != nil {
		gologger.Errorf("设置BPF过滤器失败: %v", err)
		return
	}

	// 创建DNS响应处理通道，缓冲大小适当增加
	dnsChanel := make(chan layers.DNS, 10000)

	// 使用多个协程处理DNS响应，提高并发效率
	processorCount := runtime.NumCPU() * 2
	var processorWg sync.WaitGroup
	processorWg.Add(processorCount)

	// 启动多个处理协程
	for i := 0; i < processorCount; i++ {
		go func() {
			defer processorWg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case dns, ok := <-dnsChanel:
					if !ok {
						return
					}

					subdomain := string(dns.Questions[0].Name)

					// 计算RTT并上报给动态超时追踪器
					if item, ok := r.statusDB.Get(subdomain); ok {
						rttSec := time.Since(item.Time).Seconds()
						if rttSec > 0 {
							r.rttTracker.recordSample(rttSec)
						}
					}

					r.statusDB.Del(subdomain)
					if dns.ANCount > 0 {
						atomic.AddUint64(&r.successCount, 1)
						var answers []string
						for _, v := range dns.Answers {
							answer, err := dnsRecord2String(v)
							if err != nil {
								continue
							}
							answers = append(answers, answer)
						}
						r.resultChan <- result.Result{
							Subdomain: subdomain,
							Answers:   answers,
						}
					}
				}
			}
		}()
	}

	// 使用多个接收协程读取网络数据包
	packetChan := make(chan []byte, 10000)

	// 启动数据包接收协程
	go func() {
		for {
			data, _, err := handle.ReadPacketData()
			if err != nil {
				if errors.Is(err, pcap.NextErrorTimeoutExpired) {
					continue
				}
				return
			}

			select {
			case <-ctx.Done():
				return
			case packetChan <- data:
				// 数据包已发送到处理通道
			}
		}
	}()

	// 启动多个数据包解析协程
	parserCount := runtime.NumCPU() * 2
	var parserWg sync.WaitGroup
	parserWg.Add(parserCount)

	for i := 0; i < parserCount; i++ {
		go func() {
			defer parserWg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case data, ok := <-packetChan:
					if !ok {
						return
					}
					r.processPacket(data, dnsChanel)
				}
			}
		}()
	}

	// 等待上下文结束
	<-ctx.Done()

	// 关闭通道
	close(packetChan)
	close(dnsChanel)

	// 等待所有处理和解析协程结束
	parserWg.Wait()
	processorWg.Wait()
}
