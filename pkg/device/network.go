package device

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"

	"github.com/boy-hack/ksubdomain/v2/pkg/gologger"
	"github.com/boy-hack/ksubdomain/v2/internal/utils"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// GetAllIPv4Devices returns all available IPv4 network interfaces
func GetAllIPv4Devices() ([]string, map[string]net.IP) {
	devices, err := pcap.FindAllDevs()
	deviceNames := []string{}
	deviceMap := make(map[string]net.IP)

	if err != nil {
		gologger.Fatalf("Failed to get network devices: %s\n", err.Error())
		return deviceNames, deviceMap
	}

	for _, d := range devices {
		for _, address := range d.Addresses {
			ip := address.IP
			// Keep only IPv4 non-loopback addresses
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
	// Valid DNS list
	var validDNS []string

	// 1. First check user-provided DNS servers
	if len(userDNS) > 0 {
		for _, dns := range userDNS {

			if ValidDNS(dns) {
				validDNS = append(validDNS, dns)
			} else {
				gologger.Warningf("User-provided DNS server is invalid: %s\n", dns)
			}
		}
	}

	// 2. If all user DNS are invalid, try system DNS
	if len(validDNS) == 0 {
		gologger.Infof("Trying to get system DNS servers...\n")
		systemDNS, err := utils.GetSystemDefaultDNS()
		if err == nil && len(systemDNS) > 0 {
			for _, dns := range systemDNS {
				if ValidDNS(dns) {
					validDNS = append(validDNS, dns)
				} else {
					gologger.Debugf("System DNS server is invalid: %s\n", dns)
				}
			}
		} else {
			gologger.Warningf("Failed to get system DNS: %v\n", err)
		}
	}

	if len(validDNS) == 0 {
		return nil, fmt.Errorf("no valid DNS found, cannot proceed with testing")
	}

	gologger.Infof("Using the following DNS servers for testing: %v\n", validDNS)
	return AutoGetDevicesWithDNS(validDNS), nil
}

// AutoGetDevicesWithDNS automatically detects the external network interface using the specified DNS servers.
// If the provided DNS servers are invalid, it falls back to the system DNS.
func AutoGetDevicesWithDNS(validDNS []string) *EtherTable {
	// Get all IPv4 interfaces
	deviceNames, _ := GetAllIPv4Devices()
	if len(deviceNames) == 0 {
		gologger.Fatalf("No available IPv4 network interfaces found\n")
		return nil
	}

	// Create a random domain name for testing
	domain := utils.RandomStr(6) + ".baidu.com"
	signal := make(chan *EtherTable)

	// Start context to control all goroutines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test all interfaces
	activeDeviceCount := 0
	for _, deviceName := range deviceNames {
		gologger.Infof("Testing connectivity of interface %s...\n", deviceName)
		go testDeviceConnectivity(ctx, deviceName, domain, signal)
		activeDeviceCount++
	}
	// Wait for test results or timeout
	return waitForDeviceTest(signal, domain, validDNS, 30)
}

// testDeviceConnectivity tests the connectivity of a network interface
func testDeviceConnectivity(ctx context.Context, deviceName string, domain string, signal chan<- *EtherTable) {
	var (
		snapshot_len int32         = 2048                   // Increased capture size
		promiscuous  bool          = true                   // Enable promiscuous mode
		timeout      time.Duration = 500 * time.Millisecond // Increased timeout
	)

	handle, err := pcap.OpenLive(deviceName, snapshot_len, promiscuous, timeout)
	if err != nil {
		gologger.Debugf("Cannot open interface %s: %s\n", deviceName, err.Error())
		return
	}
	defer handle.Close()

	// Add BPF filter to capture only DNS response packets
	err = handle.SetBPFFilter("udp port 53")
	if err != nil {
		gologger.Debugf("Failed to set filter on %s: %s\n", deviceName, err.Error())
		// Continue trying without returning
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
				continue // Don't return immediately, keep trying
			}

			var decoded []gopacket.LayerType
			err = parser.DecodeLayers(data, &decoded)
			if err != nil {
				continue
			}

			// Check if the DNS layer was decoded
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

			// Process DNS responses only
			if !dns.QR {
				continue
			}

			// Check if it matches our test domain
			for _, q := range dns.Questions {
				questionName := string(q.Name)
				gologger.Debugf("Received DNS response on %s, domain: %s\n", deviceName, questionName)
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

// waitForDeviceTest waits for device test results
func waitForDeviceTest(signal <-chan *EtherTable, domain string, dnsServers []string, timeout int) *EtherTable {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	count := 0
	// Round-robin over DNS server list
	dnsIndex := 0

	for {
		select {
		case result := <-signal:
			gologger.Infof("Successfully detected external interface: %s\n", result.Device)
			return result
		case <-ticker.C:
			// Try a DNS query every second, rotating through DNS servers
			currentDNS := dnsServers[dnsIndex]
			dnsIndex = (dnsIndex + 1) % len(dnsServers)

			go func(server string) {
				ip, err := LookUpIP(domain, server)
				if err != nil {
					gologger.Debugf("DNS query failed (%s): %s\n", server, err.Error())
				} else if ip != nil {
					gologger.Debugf("DNS query succeeded (%s): %s -> %s\n", server, domain, ip.String())
				}
			}(currentDNS)

			fmt.Print(".")
			count++

			if count >= timeout {
				gologger.Fatalf("Timed out detecting network device, please specify the interface manually\n")
				return nil
			}
		}
	}
}

// LookUpIP queries a domain name using the specified DNS server and returns the IP address
func LookUpIP(fqdn, serverAddr string) (net.IP, error) {
	var m dns.Msg
	client := dns.Client{}
	client.Timeout = time.Second
	m.SetQuestion(dns.Fqdn(fqdn), dns.TypeA)
	r, _, err := client.Exchange(&m, serverAddr+":53")

	if err != nil {
		return nil, err
	}

	// Check for a response
	if r == nil || len(r.Answer) == 0 {
		return nil, fmt.Errorf("no DNS reply")
	}

	// Try to get an A record
	for _, ans := range r.Answer {
		if a, ok := ans.(*dns.A); ok {
			return a.A, nil
		}
	}

	return nil, fmt.Errorf("no A record")
}
