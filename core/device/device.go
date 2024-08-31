package device

import (
	"context"
	"errors"
	"fmt"
	"github.com/boy-hack/ksubdomain/core"
	"github.com/boy-hack/ksubdomain/core/gologger"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"net"
	"time"
)

func AutoGetDevices() *EtherTable {
	domain := core.RandomStr(4) + ".w8ay.fun"
	devices, err := pcap.FindAllDevs()
	if err != nil {
		gologger.Fatalf("获取网络设备失败:%s\n", err.Error())
	}
	deviceData := make(map[string]net.IP)
	signal := make(chan *EtherTable)
	for _, d := range devices {
		for _, address := range d.Addresses {
			ip := address.IP
			if len(ip) == 0 {
				continue
			}
			if ip.To4() != nil {
				deviceData[d.Name] = ip
			}
		}
	}
	ctx := context.Background()
	// 在初始上下文的基础上创建一个有取消功能的上下文
	ctx, cancel := context.WithCancel(ctx)
	for drviceName, _ := range deviceData {
		go func(drviceName string, domain string, ctx context.Context) {
			var (
				snapshot_len int32         = 1024
				promiscuous  bool          = false
				timeout      time.Duration = -1 * time.Second
				handle       *pcap.Handle
			)
			var err error
			handle, err = pcap.OpenLive(
				drviceName,
				snapshot_len,
				promiscuous,
				timeout,
			)
			if err != nil {
				panic("pcap打开失败:" + err.Error())
				return
			}
			defer handle.Close()
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
					var data []byte
					var decoded []gopacket.LayerType
					data, _, err = handle.ReadPacketData()
					if err != nil {
						if errors.Is(pcap.NextErrorTimeoutExpired, err) {
							continue
						}
						return
					}
					err = parser.DecodeLayers(data, &decoded)
					if err != nil {
						continue
					}
					if !dns.QR {
						continue
					}
					for _, v := range dns.Questions {
						fmt.Println(v)
						if string(v.Name) == domain {
							etherTable := EtherTable{
								SrcIp:  ipv4.DstIP,
								Device: drviceName,
								SrcMac: SelfMac(eth.DstMAC),
								DstMac: SelfMac(eth.SrcMAC),
							}
							signal <- &etherTable
							return
						}
					}
				}
			}
		}(drviceName, domain, ctx)
	}
	index := 0
	for {
		select {
		case c := <-signal:
			cancel()
			return c
		default:
			net.LookupHost(domain)
			fmt.Printf(".")
			time.Sleep(time.Second * 1)
			index += 1
			if index > 60 {
				gologger.Errorf("获取网络设备失败:%s\n", err.Error())
				cancel()
				return nil
			}
		}
	}
}
func GetIpv4Devices() (keys []string, data map[string]net.IP) {
	devices, err := pcap.FindAllDevs()
	data = make(map[string]net.IP)
	if err != nil {
		gologger.Fatalf("获取网络设备失败:%s\n", err.Error())
	}
	for _, d := range devices {
		for _, address := range d.Addresses {
			ip := address.IP
			if ip.To4() != nil && !ip.IsLoopback() {
				gologger.Printf("  [%d] Name: %s\n", len(keys), d.Name)
				gologger.Printf("  Description: %s\n", d.Description)
				gologger.Printf("  Devices addresses: %s\n", d.Description)
				gologger.Printf("  IP address: %s\n", ip)
				gologger.Printf("  Subnet mask: %s\n\n", address.Netmask.String())
				data[d.Name] = ip
				keys = append(keys, d.Name)
			}
		}
	}
	return
}
func PcapInit(devicename string) (*pcap.Handle, error) {
	var (
		snapshot_len int32 = 1024
		//promiscuous  bool  = false
		err     error
		timeout time.Duration = -1 * time.Second
	)
	handle, err := pcap.OpenLive(devicename, snapshot_len, false, timeout)
	if err != nil {
		gologger.Fatalf("pcap初始化失败:%s\n", err.Error())
		return nil, err
	}
	return handle, nil
}
