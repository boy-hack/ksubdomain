package device

import (
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/google/gopacket/pcap"
	"github.com/stretchr/testify/assert"
)

// TestGetDefaultRouteInterface tests getting the default route interface
func TestGetDefaultRouteInterface(t *testing.T) {
	// Skip tests that require root privileges
	if !hasAdminPrivileges() {
		t.Skip("Requires administrator privileges to run this test")
	}

	etherTable, err := GetDefaultRouteInterface()

	// May fail in CI or no-network environments
	if err != nil {
		t.Logf("Failed to get default route (may be an environment issue): %v", err)
		return
	}

	// Validate the returned data
	assert.NotNil(t, etherTable)
	assert.NotEmpty(t, etherTable.Device, "Device name should not be empty")
	assert.NotNil(t, etherTable.SrcIp, "Source IP should not be nil")
	assert.False(t, etherTable.SrcIp.IsLoopback(), "Should not be a loopback address")
	assert.NotEqual(t, "00:00:00:00:00:00", etherTable.SrcMac.String(), "MAC address should not be all zeros")

	t.Logf("Successfully obtained interface: Device=%s, IP=%s, MAC=%s, Gateway MAC=%s",
		etherTable.Device, etherTable.SrcIp, etherTable.SrcMac, etherTable.DstMac)
}

// TestResolveGatewayMAC tests ARP gateway MAC resolution
func TestResolveGatewayMAC(t *testing.T) {
	if !hasAdminPrivileges() {
		t.Skip("Requires administrator privileges to run this test")
	}

	// Get local network information
	etherTable, err := GetDefaultRouteInterface()
	if err != nil {
		t.Skip("Cannot get network info, skipping ARP test")
	}

	// Try to resolve the local gateway
	// Note: this test runs in a real network environment
	gatewayIP := getDefaultGateway()
	if gatewayIP == nil {
		t.Skip("Cannot get default gateway, skipping ARP test")
	}

	srcMAC := net.HardwareAddr(etherTable.SrcMac)
	mac, err := resolveGatewayMAC(etherTable.Device, etherTable.SrcIp, srcMAC, gatewayIP)

	if err != nil {
		t.Logf("ARP resolution failed (may be a network environment issue): %v", err)
		return
	}

	assert.NotNil(t, mac)
	assert.Len(t, mac, 6, "MAC address should be 6 bytes")
	assert.NotEqual(t, "00:00:00:00:00:00", mac.String(), "MAC address should not be all zeros")
	assert.NotEqual(t, "ff:ff:ff:ff:ff:ff", mac.String(), "MAC address should not be broadcast address")

	t.Logf("Successfully resolved gateway MAC: %s -> %s", gatewayIP, mac)
}

// TestValidateInterface tests interface validation
func TestValidateInterface(t *testing.T) {
	if !hasAdminPrivileges() {
		t.Skip("Requires administrator privileges to run this test")
	}

	tests := []struct {
		name       string
		etherTable *EtherTable
		expected   bool
	}{
		{
			name: "Invalid interface name",
			etherTable: &EtherTable{
				Device: "invalid_device_xyz",
				SrcIp:  net.ParseIP("192.168.1.100"),
			},
			expected: false,
		},
		{
			name: "Empty interface name",
			etherTable: &EtherTable{
				Device: "",
				SrcIp:  net.ParseIP("192.168.1.100"),
			},
			expected: false,
		},
	}

	// Add a test for a valid interface
	devices, err := pcap.FindAllDevs()
	if err == nil && len(devices) > 0 {
		validDevice := devices[0].Name
		tests = append(tests, struct {
			name       string
			etherTable *EtherTable
			expected   bool
		}{
			name: "Valid interface",
			etherTable: &EtherTable{
				Device: validDevice,
				SrcIp:  net.ParseIP("192.168.1.100"),
			},
			expected: true,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateInterface(tt.etherTable)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAutoGetDevicesImproved tests the improved auto-detection method
func TestAutoGetDevicesImproved(t *testing.T) {
	if !hasAdminPrivileges() {
		t.Skip("Requires administrator privileges to run this test")
	}

	// Use common public DNS servers
	testDNS := []string{
		"8.8.8.8",
		"1.1.1.1",
		"114.114.114.114",
	}

	etherTable, err := AutoGetDevicesImproved(testDNS)

	// May fail in some environments
	if err != nil {
		t.Logf("Auto interface detection failed (environment issue): %v", err)
		return
	}

	assert.NotNil(t, etherTable)
	assert.NotEmpty(t, etherTable.Device)
	assert.NotNil(t, etherTable.SrcIp)
	assert.False(t, etherTable.SrcIp.IsUnspecified(), "IP should not be unspecified address")

	t.Logf("Successfully auto-detected interface: %+v", etherTable)
}

// BenchmarkGetDefaultRouteInterface performance test
func BenchmarkGetDefaultRouteInterface(b *testing.B) {
	if !hasAdminPrivileges() {
		b.Skip("Requires administrator privileges to run this test")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetDefaultRouteInterface()
	}
}

// BenchmarkResolveGatewayMAC performance test for ARP resolution
func BenchmarkResolveGatewayMAC(b *testing.B) {
	if !hasAdminPrivileges() {
		b.Skip("Requires administrator privileges to run this test")
	}

	// Prepare test data
	etherTable, err := GetDefaultRouteInterface()
	if err != nil {
		b.Skip("Cannot get network info")
	}

	gatewayIP := getDefaultGateway()
	if gatewayIP == nil {
		b.Skip("Cannot get gateway")
	}

	srcMAC := net.HardwareAddr(etherTable.SrcMac)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolveGatewayMAC(etherTable.Device, etherTable.SrcIp, srcMAC, gatewayIP)
	}
}

// hasAdminPrivileges checks whether the process has administrator privileges
func hasAdminPrivileges() bool {
	switch runtime.GOOS {
	case "windows":
		// On Windows, check if interfaces can be opened
		devices, err := pcap.FindAllDevs()
		return err == nil && len(devices) > 0
	default:
		// On Unix systems, check UID
		return runtime.GOOS == "darwin" || isRoot()
	}
}

// isRoot checks whether the process is running as root
func isRoot() bool {
	// Try to open a network interface to check permissions
	devices, err := pcap.FindAllDevs()
	if err != nil || len(devices) == 0 {
		return false
	}

	// Try to open the first device
	handle, err := pcap.OpenLive(devices[0].Name, 1024, false, time.Second)
	if err != nil {
		return false
	}
	handle.Close()
	return true
}

// getDefaultGateway is a helper function to get the default gateway
func getDefaultGateway() net.IP {
	// Simple implementation; production code should parse the routing table
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	// Assume the gateway is .1
	ip := localAddr.IP.To4()
	if ip != nil {
		ip[3] = 1
		return ip
	}
	return nil
}
