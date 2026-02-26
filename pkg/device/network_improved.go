package device

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// GetDefaultRouteInterface 获取默认路由的网卡设备
// 这是最可靠的方法，因为默认路由的网卡通常就是外网通信的网卡
func GetDefaultRouteInterface() (*EtherTable, error) {
	var defaultInterface string
	var gatewayIP net.IP

	switch runtime.GOOS {
	case "windows":
		// Windows: 使用 route print 获取默认路由
		cmd := exec.Command("route", "print", "0.0.0.0")
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("执行route命令失败: %v", err)
		}

		// 解析输出获取默认网关和接口
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "0.0.0.0") && strings.Contains(line, "0.0.0.0") {
				fields := strings.Fields(line)
				if len(fields) >= 5 {
					gatewayIP = net.ParseIP(fields[2])
					// 获取接口IP
					localIP := net.ParseIP(fields[3])
					if localIP != nil {
						// 查找对应的网卡
						interfaces, _ := pcap.FindAllDevs()
						for _, iface := range interfaces {
							for _, addr := range iface.Addresses {
								if addr.IP.Equal(localIP) {
									defaultInterface = iface.Name
									break
								}
							}
							if defaultInterface != "" {
								break
							}
						}
					}
					break
				}
			}
		}

	case "linux":
		// Linux: 使用 ip route 获取默认路由
		cmd := exec.Command("ip", "route", "show", "default")
		output, err := cmd.Output()
		if err != nil {
			// 尝试使用 route 命令
			cmd = exec.Command("route", "-n")
			output, err = cmd.Output()
			if err != nil {
				return nil, fmt.Errorf("获取路由信息失败: %v", err)
			}
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "default") || strings.HasPrefix(line, "0.0.0.0") {
				fields := strings.Fields(line)
				if len(fields) >= 5 {
					// ip route 格式: default via 192.168.1.1 dev eth0
					if fields[0] == "default" && len(fields) >= 5 {
						gatewayIP = net.ParseIP(fields[2])
						defaultInterface = fields[4]
					} else if fields[0] == "0.0.0.0" {
						// route -n 格式
						gatewayIP = net.ParseIP(fields[1])
						defaultInterface = fields[len(fields)-1]
					}
					break
				}
			}
		}

	case "darwin":
		// macOS: 使用 route get 获取默认路由
		cmd := exec.Command("route", "get", "default")
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("获取路由信息失败: %v", err)
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "gateway:") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					gatewayIP = net.ParseIP(strings.TrimSpace(parts[1]))
				}
			} else if strings.HasPrefix(line, "interface:") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					defaultInterface = strings.TrimSpace(parts[1])
				}
			}
		}
	}

	if defaultInterface == "" || gatewayIP == nil {
		return nil, fmt.Errorf("无法获取默认路由信息")
	}

	gologger.Infof("找到默认路由网卡: %s, 网关: %s\n", defaultInterface, gatewayIP.String())

	// 获取网卡的IP和MAC地址
	etherTable, err := getInterfaceDetails(defaultInterface, gatewayIP)
	if err != nil {
		return nil, err
	}

	return etherTable, nil
}

// getInterfaceDetails 获取网卡详细信息，包括通过ARP获取网关MAC
func getInterfaceDetails(deviceName string, gatewayIP net.IP) (*EtherTable, error) {
	// 获取网卡信息
	interfaces, err := pcap.FindAllDevs()
	if err != nil {
		return nil, fmt.Errorf("获取网卡列表失败: %v", err)
	}

	var srcIP net.IP
	var srcMAC net.HardwareAddr

	// 查找指定网卡的IP和MAC
	for _, iface := range interfaces {
		if iface.Name == deviceName {
			// 获取IP地址
			for _, addr := range iface.Addresses {
				if addr.IP.To4() != nil && !addr.IP.IsLoopback() {
					srcIP = addr.IP
					break
				}
			}
			break
		}
	}

	if srcIP == nil {
		return nil, fmt.Errorf("无法获取网卡 %s 的IP地址", deviceName)
	}

	// 获取网卡MAC地址
	iface, err := net.InterfaceByName(deviceName)
	if err == nil && iface.HardwareAddr != nil {
		srcMAC = iface.HardwareAddr
	} else {
		// 如果标准方法失败，尝试从系统获取
		srcMAC, _ = getMACAddress(deviceName)
	}

	if srcMAC == nil {
		// 使用默认MAC
		srcMAC = net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		gologger.Warningf("无法获取网卡MAC地址，使用默认值\n")
	}

	// 通过ARP获取网关MAC地址
	gatewayMAC, err := resolveGatewayMAC(deviceName, srcIP, srcMAC, gatewayIP)
	if err != nil {
		gologger.Warningf("ARP解析网关MAC失败: %v，将使用广播地址\n", err)
		gatewayMAC = net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	}

	etherTable := &EtherTable{
		SrcIp:  srcIP,
		Device: deviceName,
		SrcMac: SelfMac(srcMAC),
		DstMac: SelfMac(gatewayMAC),
	}

	gologger.Infof("网卡配置: IP=%s, MAC=%s, Gateway MAC=%s\n",
		srcIP.String(), srcMAC.String(), gatewayMAC.String())

	return etherTable, nil
}

// resolveGatewayMAC 通过ARP请求获取网关的MAC地址
func resolveGatewayMAC(deviceName string, srcIP net.IP, srcMAC net.HardwareAddr, gatewayIP net.IP) (net.HardwareAddr, error) {
	// 打开网卡进行ARP操作
	handle, err := pcap.OpenLive(deviceName, 2048, true, time.Second)
	if err != nil {
		return nil, fmt.Errorf("打开网卡失败: %v", err)
	}
	defer handle.Close()

	// 设置过滤器只接收ARP回复
	err = handle.SetBPFFilter(fmt.Sprintf("arp and arp[6:2] = 2 and src host %s", gatewayIP.String()))
	if err != nil {
		gologger.Debugf("设置BPF过滤器失败: %v\n", err)
	}

	// 构建ARP请求包
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, // 广播
		EthernetType: layers.EthernetTypeARP,
	}

	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   srcMAC,
		SourceProtAddress: srcIP.To4(),
		DstHwAddress:      net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		DstProtAddress:    gatewayIP.To4(),
	}

	// 序列化数据包
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err = gopacket.SerializeLayers(buffer, opts, eth, arp)
	if err != nil {
		return nil, fmt.Errorf("构建ARP包失败: %v", err)
	}

	// 发送ARP请求
	outgoingPacket := buffer.Bytes()
	err = handle.WritePacketData(outgoingPacket)
	if err != nil {
		return nil, fmt.Errorf("发送ARP请求失败: %v", err)
	}

	// 等待ARP回复
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("ARP响应超时")
		case packet := <-packetSource.Packets():
			if packet == nil {
				continue
			}

			// 解析ARP层
			if arpLayer := packet.Layer(layers.LayerTypeARP); arpLayer != nil {
				arpReply, ok := arpLayer.(*layers.ARP)
				if ok && arpReply.Operation == layers.ARPReply {
					// 检查是否是我们请求的网关IP的回复
					if net.IP(arpReply.SourceProtAddress).Equal(gatewayIP) {
						return net.HardwareAddr(arpReply.SourceHwAddress), nil
					}
				}
			}
		}
	}
}

// getMACAddress 获取网卡MAC地址的辅助函数
func getMACAddress(deviceName string) (net.HardwareAddr, error) {
	// 尝试通过系统命令获取MAC地址
	switch runtime.GOOS {
	case "windows":
		// Windows: 使用 getmac 命令
		cmd := exec.Command("getmac", "/v")
		output, err := cmd.Output()
		if err != nil {
			return nil, err
		}
		// 解析输出找到对应网卡的MAC
		// 这里需要更复杂的解析逻辑
		_ = output

	case "linux", "darwin":
		// Linux/macOS: 使用 ifconfig
		cmd := exec.Command("ifconfig", deviceName)
		output, err := cmd.Output()
		if err != nil {
			return nil, err
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "ether") || strings.Contains(line, "HWaddr") {
				fields := strings.Fields(line)
				for i, field := range fields {
					if field == "ether" || field == "HWaddr" {
						if i+1 < len(fields) {
							mac, err := net.ParseMAC(fields[i+1])
							if err == nil {
								return mac, nil
							}
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("无法获取MAC地址")
}

// AutoGetDevicesImproved 改进的自动获取网卡方法
// 优先使用路由表和ARP，失败时再回退到DNS探测
func AutoGetDevicesImproved(userDNS []string) (*EtherTable, error) {
	gologger.Infof("尝试通过默认路由获取网卡信息...\n")

	// 方法1: 通过默认路由获取
	etherTable, err := GetDefaultRouteInterface()
	if err == nil {
		// 验证网卡是否可用
		if validateInterface(etherTable) {
			gologger.Infof("成功通过默认路由获取网卡信息\n")
			return etherTable, nil
		}
	}

	gologger.Warningf("默认路由方法失败: %v，尝试DNS探测方法\n", err)

	// 方法2: 回退到原始的DNS探测方法
	return AutoGetDevices(userDNS)
}

// validateInterface 验证网卡是否可用
func validateInterface(etherTable *EtherTable) bool {
	// 尝试打开网卡
	handle, err := pcap.OpenLive(etherTable.Device, 1024, false, time.Second)
	if err != nil {
		return false
	}
	defer handle.Close()

	// 检查是否能设置BPF过滤器
	err = handle.SetBPFFilter("udp")
	return err == nil
}
