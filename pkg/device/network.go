package device

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"

	"github.com/boy-hack/ksubdomain/pkg/core"
	"github.com/boy-hack/ksubdomain/pkg/core/gologger"
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

// 通过网卡名称获取对应的EtherTable
func GetDevicesByName(deviceName string) (*EtherTable, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, fmt.Errorf("获取网络设备失败: %s", err.Error())
	}

	var foundDevice *pcap.Interface
	for _, d := range devices {
		if d.Name == deviceName {
			foundDevice = &d
			break
		}
	}

	if foundDevice == nil {
		return nil, fmt.Errorf("未找到指定网卡: %s", deviceName)
	}

	// 创建随机域名用于测试
	domain := core.RandomStr(6) + ".w8ay.fun"
	signal := make(chan *EtherTable)
	go testDeviceConnectivity(context.Background(), deviceName, domain, signal)
	return waitForDeviceTest(signal, domain, 60), nil
}

// 优化后的自动获取外网发包网卡功能
func AutoGetDevices() *EtherTable {
	// 获取所有IPv4网卡
	deviceNames, _ := GetAllIPv4Devices()
	if len(deviceNames) == 0 {
		gologger.Fatalf("未发现可用的IPv4网卡\n")
		return nil
	}

	// 创建随机域名用于测试
	domain := core.RandomStr(6) + ".w8ay.fun"
	signal := make(chan *EtherTable)

	// 启动上下文，用于控制所有goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 测试所有网卡
	for _, deviceName := range deviceNames {
		gologger.Infof("正在测试网卡 %s 的连通性...\n", deviceName)
		go testDeviceConnectivity(ctx, deviceName, domain, signal)
	}

	// 等待测试结果或超时
	return waitForDeviceTest(signal, domain, 30)
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
func waitForDeviceTest(signal <-chan *EtherTable, domain string, timeout int) *EtherTable {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	count := 0
	for {
		select {
		case result := <-signal:
			gologger.Infof("成功获取到外网网卡: %s\n", result.Device)
			return result
		case <-ticker.C:
			// 每秒尝试一次DNS查询
			go func() {
				err := LookUpIP(domain, "1.1.1.1")
				if err != nil {
					gologger.Debugf("DNS查询失败: %s\n", err.Error())
				}
			}()
			fmt.Print(".")
			count++

			if count >= timeout {
				gologger.Fatalf("获取网络设备超时，请尝试手动指定网卡\n")
				return nil
			}
		}
	}
}

func LookUpIP(fqdn, serverAddr string) error {
	var m dns.Msg
	client := dns.Client{}
	client.Timeout = time.Second
	m.SetQuestion(dns.Fqdn(fqdn), dns.TypeNS)
	_, _, err := client.Exchange(&m, serverAddr+":53")
	return err
}
