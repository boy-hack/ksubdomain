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

// GetDefaultRouteInterface retrieves the network interface for the default route.
// This is the most reliable method because the default route interface is usually the one used for external communication.
func GetDefaultRouteInterface() (*EtherTable, error) {
	var defaultInterface string
	var gatewayIP net.IP

	switch runtime.GOOS {
	case "windows":
		// Windows: use 'route print' to get the default route
		cmd := exec.Command("route", "print", "0.0.0.0")
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to execute route command: %v", err)
		}

		// Parse output to get the default gateway and interface
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "0.0.0.0") && strings.Contains(line, "0.0.0.0") {
				fields := strings.Fields(line)
				if len(fields) >= 5 {
					gatewayIP = net.ParseIP(fields[2])
					// Get interface IP
					localIP := net.ParseIP(fields[3])
					if localIP != nil {
						// Find the corresponding interface
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
		// Linux: use 'ip route' to get the default route
		cmd := exec.Command("ip", "route", "show", "default")
		output, err := cmd.Output()
		if err != nil {
			// Try with the 'route' command
			cmd = exec.Command("route", "-n")
			output, err = cmd.Output()
			if err != nil {
				return nil, fmt.Errorf("failed to get routing information: %v", err)
			}
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "default") || strings.HasPrefix(line, "0.0.0.0") {
				fields := strings.Fields(line)
				if len(fields) >= 5 {
					// ip route format: default via 192.168.1.1 dev eth0
					if fields[0] == "default" && len(fields) >= 5 {
						gatewayIP = net.ParseIP(fields[2])
						defaultInterface = fields[4]
					} else if fields[0] == "0.0.0.0" {
						// route -n format
						gatewayIP = net.ParseIP(fields[1])
						defaultInterface = fields[len(fields)-1]
					}
					break
				}
			}
		}

	case "darwin":
		// macOS: use 'route get default' to get the default route
		cmd := exec.Command("route", "get", "default")
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to get routing information: %v", err)
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
		return nil, fmt.Errorf("unable to obtain default route information")
	}

	gologger.Infof("Found default route interface: %s, gateway: %s\n", defaultInterface, gatewayIP.String())

	// Get IP and MAC address of the interface
	etherTable, err := getInterfaceDetails(defaultInterface, gatewayIP)
	if err != nil {
		return nil, err
	}

	return etherTable, nil
}

// getInterfaceDetails retrieves detailed interface information, including the gateway MAC via ARP
func getInterfaceDetails(deviceName string, gatewayIP net.IP) (*EtherTable, error) {
	// Get interface information
	interfaces, err := pcap.FindAllDevs()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface list: %v", err)
	}

	var srcIP net.IP
	var srcMAC net.HardwareAddr

	// Find IP and MAC of the specified interface
	for _, iface := range interfaces {
		if iface.Name == deviceName {
			// Get IP address
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
		return nil, fmt.Errorf("unable to get IP address for interface %s", deviceName)
	}

	// Get interface MAC address
	iface, err := net.InterfaceByName(deviceName)
	if err == nil && iface.HardwareAddr != nil {
		srcMAC = iface.HardwareAddr
	} else {
		// If standard method fails, try getting from system
		srcMAC, _ = getMACAddress(deviceName)
	}

	if srcMAC == nil {
		// Use default MAC
		srcMAC = net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		gologger.Warningf("Unable to get interface MAC address, using default value\n")
	}

	// Get gateway MAC address via ARP
	gatewayMAC, err := resolveGatewayMAC(deviceName, srcIP, srcMAC, gatewayIP)
	if err != nil {
		gologger.Warningf("ARP resolution of gateway MAC failed: %v, will use broadcast address\n", err)
		gatewayMAC = net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	}

	etherTable := &EtherTable{
		SrcIp:  srcIP,
		Device: deviceName,
		SrcMac: SelfMac(srcMAC),
		DstMac: SelfMac(gatewayMAC),
	}

	gologger.Infof("Interface config: IP=%s, MAC=%s, Gateway MAC=%s\n",
		srcIP.String(), srcMAC.String(), gatewayMAC.String())

	return etherTable, nil
}

// resolveGatewayMAC resolves the gateway's MAC address via ARP request
func resolveGatewayMAC(deviceName string, srcIP net.IP, srcMAC net.HardwareAddr, gatewayIP net.IP) (net.HardwareAddr, error) {
	// Open the interface for ARP operations
	handle, err := pcap.OpenLive(deviceName, 2048, true, time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to open interface: %v", err)
	}
	defer handle.Close()

	// Set filter to receive only ARP replies
	err = handle.SetBPFFilter(fmt.Sprintf("arp and arp[6:2] = 2 and src host %s", gatewayIP.String()))
	if err != nil {
		gologger.Debugf("Failed to set BPF filter: %v\n", err)
	}

	// Build ARP request packet
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, // broadcast
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

	// Serialize the packet
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err = gopacket.SerializeLayers(buffer, opts, eth, arp)
	if err != nil {
		return nil, fmt.Errorf("failed to build ARP packet: %v", err)
	}

	// Send ARP request
	outgoingPacket := buffer.Bytes()
	err = handle.WritePacketData(outgoingPacket)
	if err != nil {
		return nil, fmt.Errorf("failed to send ARP request: %v", err)
	}

	// Wait for ARP reply
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("ARP response timed out")
		case packet := <-packetSource.Packets():
			if packet == nil {
				continue
			}

			// Parse ARP layer
			if arpLayer := packet.Layer(layers.LayerTypeARP); arpLayer != nil {
				arpReply, ok := arpLayer.(*layers.ARP)
				if ok && arpReply.Operation == layers.ARPReply {
					// Check if this is a reply for the gateway IP we requested
					if net.IP(arpReply.SourceProtAddress).Equal(gatewayIP) {
						return net.HardwareAddr(arpReply.SourceHwAddress), nil
					}
				}
			}
		}
	}
}

// getMACAddress is a helper function to get the MAC address of a network interface
func getMACAddress(deviceName string) (net.HardwareAddr, error) {
	// Try to get MAC address via system command
	switch runtime.GOOS {
	case "windows":
		// Windows: use 'getmac' command
		cmd := exec.Command("getmac", "/v")
		output, err := cmd.Output()
		if err != nil {
			return nil, err
		}
		// Parse output to find the MAC for the interface
		// More complex parsing logic would be needed here
		_ = output

	case "linux", "darwin":
		// Linux/macOS: use ifconfig
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

	return nil, fmt.Errorf("unable to get MAC address")
}

// AutoGetDevicesImproved is an improved method for automatically detecting the network interface.
// It prefers using the routing table and ARP, falling back to DNS probing on failure.
func AutoGetDevicesImproved(userDNS []string) (*EtherTable, error) {
	gologger.Infof("Trying to get interface info via default route...\n")

	// Method 1: Get via default route
	etherTable, err := GetDefaultRouteInterface()
	if err == nil {
		// Validate the interface is usable
		if validateInterface(etherTable) {
			gologger.Infof("Successfully obtained interface info via default route\n")
			return etherTable, nil
		}
	}

	gologger.Warningf("Default route method failed: %v, falling back to DNS probing\n", err)

	// Method 2: Fall back to original DNS probing method
	return AutoGetDevices(userDNS)
}

// validateInterface checks whether a network interface is usable
func validateInterface(etherTable *EtherTable) bool {
	// Try to open the interface
	handle, err := pcap.OpenLive(etherTable.Device, 1024, false, time.Second)
	if err != nil {
		return false
	}
	defer handle.Close()

	// Check if BPF filter can be set
	err = handle.SetBPFFilter("udp")
	return err == nil
}
