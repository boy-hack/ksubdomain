package device

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"

	"github.com/boy-hack/ksubdomain/v2/pkg/core"
	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/utils"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// 获取所有IPv4网卡信息
func GetAllIPv4Devices() ([]string, map[string]net.IP) {
	devices, err := pcap.FindAllDevs()
	deviceNames := []string{}
	deviceMap := make(map[string]net.IP)

	if err != nil {
		gologger.Fatalf("获取网络设备失败: %s\n", err.Error())
		return deviceNames, deviceMap
	}

	for _, d := range devices {
		for _, address := range d.Addresses {
			ip := address.IP
			// 只保留IPv4且非回环地址
			if ip.To4() != nil {
				deviceMap[d.Name] = ip
				deviceNames = append(deviceNames, d.Name)
			}
		}
	}

	return deviceNames, deviceMap
}

func ValidDNS(dns string) bool {
	if dns == "" {
		return false
	}
	_, err := LookUpIP("www.baidu.com", dns)
	if err != nil {
		return false
	}
	return true
}

func AutoGetDevices(userDNS []string) (*EtherTable, error) {
	// 有效DNS列表
	var validDNS []string

	// 1. 首先检测用户提供的DNS
	if len(userDNS) > 0 {
		for _, dns := range userDNS {

			if ValidDNS(dns) {
				validDNS = append(validDNS, dns)
			} else {
				gologger.Warningf("用户提供的DNS服务器无效: %s\n", dns)
			}
		}
	}

	// 2. 如果用户DNS都无效，尝试系统DNS
	if len(validDNS) == 0 {
		gologger.Infof("尝试获取系统DNS服务器...\n")
		systemDNS, err := utils.GetSystemDefaultDNS()
		if err == nil && len(systemDNS) > 0 {
			for _, dns := range systemDNS {
				if ValidDNS(dns) {
					validDNS = append(validDNS, dns)
				} else {
					gologger.Debugf("系统DNS服务器无效: %s\n", dns)
				}
			}
		} else {
			gologger.Warningf("获取系统DNS失败: %v\n", err)
		}
	}

	if len(validDNS) == 0 {
		return nil, fmt.Errorf("没有找到有效DNS，无法进行测试")
	}

	gologger.Infof("使用以下DNS服务器进行测试: %v\n", validDNS)
	return AutoGetDevicesWithDNS(validDNS), nil
}

// AutoGetDevicesWithDNS 使用指定DNS自动获取外网发包网卡
// 如果传入的DNS无效，则尝试使用系统DNS
func AutoGetDevicesWithDNS(validDNS []string) *EtherTable {
	// 获取所有IPv4网卡
	deviceNames, _ := GetAllIPv4Devices()
	if len(deviceNames) == 0 {
		gologger.Fatalf("未发现可用的IPv4网卡\n")
		return nil
	}

	// 创建随机域名用于测试
	domain := core.RandomStr(6) + ".baidu.com"
	signal := make(chan *EtherTable)

	// 启动上下文，用于控制所有goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 测试所有网卡
	activeDeviceCount := 0
	for _, deviceName := range deviceNames {
		gologger.Infof("正在测试网卡 %s 的连通性...\n", deviceName)
		go testDeviceConnectivity(ctx, deviceName, domain, signal)
		activeDeviceCount++
	}
	// 等待测试结果或超时
	return waitForDeviceTest(signal, domain, validDNS, 30)
}

// 测试网卡连通性
func testDeviceConnectivity(ctx context.Context, deviceName string, domain string, signal chan<- *EtherTable) {
	var (
		snapshot_len int32         = 2048                   // 增加抓包大小
		promiscuous  bool          = true                   // 启用混杂模式
		timeout      time.Duration = 500 * time.Millisecond // 增加超时时间
	)

	handle, err := pcap.OpenLive(deviceName, snapshot_len, promiscuous, timeout)
	if err != nil {
		gologger.Debugf("无法打开网卡 %s: %s\n", deviceName, err.Error())
		return
	}
	defer handle.Close()

	// 添加BPF过滤器，只捕获DNS响应包
	err = handle.SetBPFFilter("udp port 53")
	if err != nil {
		gologger.Debugf("设置过滤器失败 %s: %s\n", deviceName, err.Error())
		// 继续尝试，不直接返回
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			var udp layers.UDP
			var dns layers.DNS
			var eth layers.Ethernet
			var ipv4 layers.IPv4

			parser := gopacket.NewDecodingLayerParser(
				layers.LayerTypeEthernet, &eth, &ipv4, &udp, &dns)

			data, _, err := handle.ReadPacketData()
			if err != nil {
				if errors.Is(err, pcap.NextErrorTimeoutExpired) {
					continue
				}
				continue // 不要立即返回，继续尝试
			}

			var decoded []gopacket.LayerType
			err = parser.DecodeLayers(data, &decoded)
			if err != nil {
				continue
			}

			// 检查是否解析到DNS层
			dnsFound := false
			for _, layerType := range decoded {
				if layerType == layers.LayerTypeDNS {
					dnsFound = true
					break
				}
			}
			if !dnsFound {
				continue
			}

			// 只处理DNS响应
			if !dns.QR {
				continue
			}

			// 检查是否匹配我们的测试域名
			for _, q := range dns.Questions {
				questionName := string(q.Name)
				gologger.Debugf("收到DNS响应 %s，域名: %s\n", deviceName, questionName)
				if questionName == domain || questionName == domain+"." {
					etherTable := EtherTable{
						SrcIp:  ipv4.DstIP,
						Device: deviceName,
						SrcMac: SelfMac(eth.DstMAC),
						DstMac: SelfMac(eth.SrcMAC),
					}
					signal <- &etherTable
					return
				}
			}
		}
	}
}

// 等待设备测试结果
func waitForDeviceTest(signal <-chan *EtherTable, domain string, dnsServers []string, timeout int) *EtherTable {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	count := 0
	// 轮询使用DNS服务器列表
	dnsIndex := 0

	for {
		select {
		case result := <-signal:
			gologger.Infof("成功获取到外网网卡: %s\n", result.Device)
			return result
		case <-ticker.C:
			// 每秒尝试一次DNS查询，轮换使用不同的DNS服务器
			currentDNS := dnsServers[dnsIndex]
			dnsIndex = (dnsIndex + 1) % len(dnsServers)

			go func(server string) {
				ip, err := LookUpIP(domain, server)
				if err != nil {
					gologger.Debugf("DNS查询失败(%s): %s\n", server, err.Error())
				} else if ip != nil {
					gologger.Debugf("DNS查询成功(%s): %s -> %s\n", server, domain, ip.String())
				}
			}(currentDNS)

			fmt.Print(".")
			count++

			if count >= timeout {
				gologger.Fatalf("获取网络设备超时，请尝试手动指定网卡\n")
				return nil
			}
		}
	}
}

// LookUpIP 使用指定DNS服务器查询域名并返回IP地址
func LookUpIP(fqdn, serverAddr string) (net.IP, error) {
	var m dns.Msg
	client := dns.Client{}
	client.Timeout = time.Second
	m.SetQuestion(dns.Fqdn(fqdn), dns.TypeA)
	r, _, err := client.Exchange(&m, serverAddr+":53")

	if err != nil {
		return nil, err
	}

	// 检查是否有响应
	if r == nil || len(r.Answer) == 0 {
		return nil, fmt.Errorf("无DNS回复")
	}

	// 尝试获取A记录
	for _, ans := range r.Answer {
		if a, ok := ans.(*dns.A); ok {
			return a.A, nil
		}
	}

	return nil, fmt.Errorf("无A记录")
}
