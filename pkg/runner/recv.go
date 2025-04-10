package runner

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/boy-hack/ksubdomain/pkg/runner/result"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// dnsRecord2String 将DNS记录转换为字符串
func dnsRecord2String(rr layers.DNSResourceRecord) (string, error) {
	if rr.Class == layers.DNSClassIN {
		switch rr.Type {
		case layers.DNSTypeA, layers.DNSTypeAAAA:
			if rr.IP != nil {
				return rr.IP.String(), nil
			}
		case layers.DNSTypeNS:
			if rr.NS != nil {
				return "NS " + string(rr.NS), nil
			}
		case layers.DNSTypeCNAME:
			if rr.CNAME != nil {
				return "CNAME " + string(rr.CNAME), nil
			}
		case layers.DNSTypePTR:
			if rr.PTR != nil {
				return "PTR " + string(rr.PTR), nil
			}
		case layers.DNSTypeTXT:
			if rr.TXT != nil {
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
	if dc.dns.ID != r.dnsid {
		return
	}

	// 确认有查询问题
	if len(dc.dns.Questions) == 0 {
		return
	}

	// 记录接收包数量
	atomic.AddUint64(&r.recvIndex, 1)

	// 向处理通道发送DNS响应
	select {
	case dnsChanel <- *dc.dns:
	default:
		// 通道满了，直接处理
		dns := *dc.dns
		subdomain := string(dns.Questions[0].Name)
		r.hm.Del(subdomain)
		if dns.ANCount > 0 {
			atomic.AddUint64(&r.successIndex, 1)
			var answers []string
			for _, v := range dns.Answers {
				answer, err := dnsRecord2String(v)
				if err != nil {
					continue
				}
				answers = append(answers, answer)
			}
			r.recver <- result.Result{
				Subdomain: subdomain,
				Answers:   answers,
			}
		}
	}
}

// recvChanel 实现接收DNS响应的功能
func (r *Runner) recvChanel(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()
	var (
		snapshotLen = 65536
		timeout     = 5 * time.Second
		err         error
	)
	inactive, err := pcap.NewInactiveHandle(r.options.EtherInfo.Device)
	if err != nil {
		return err
	}
	err = inactive.SetSnapLen(snapshotLen)
	if err != nil {
		return err
	}
	defer inactive.CleanUp()
	if err = inactive.SetTimeout(timeout); err != nil {
		return err
	}
	err = inactive.SetImmediateMode(true)
	if err != nil {
		return err
	}
	handle, err := inactive.Activate()
	if err != nil {
		return err
	}
	defer handle.Close()

	err = handle.SetBPFFilter(fmt.Sprintf("udp and src port 53 and dst port %d", r.freeport))
	if err != nil {
		return errors.New(fmt.Sprintf("SetBPFFilter Faild:%s", err.Error()))
	}

	// 创建DNS响应处理通道，缓冲大小适当增加
	dnsChanel := make(chan layers.DNS, 10000)

	// 使用多个协程处理DNS响应，提高并发效率
	processorCount := runtime.NumCPU()
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
					r.hm.Del(subdomain)
					if dns.ANCount > 0 {
						atomic.AddUint64(&r.successIndex, 1)
						var answers []string
						for _, v := range dns.Answers {
							answer, err := dnsRecord2String(v)
							if err != nil {
								continue
							}
							answers = append(answers, answer)
						}
						r.recver <- result.Result{
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
				if errors.Is(pcap.NextErrorTimeoutExpired, err) {
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
	parserCount := runtime.NumCPU()
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

	return nil
}
