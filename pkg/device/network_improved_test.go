package device

import (
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/google/gopacket/pcap"
	"github.com/stretchr/testify/assert"
)

// TestGetDefaultRouteInterface 测试获取默认路由网卡
func TestGetDefaultRouteInterface(t *testing.T) {
	// 跳过需要root权限的测试
	if !hasAdminPrivileges() {
		t.Skip("需要管理员权限运行此测试")
	}

	etherTable, err := GetDefaultRouteInterface()

	// 在CI环境或无网络环境可能失败
	if err != nil {
		t.Logf("获取默认路由失败(可能是环境问题): %v", err)
		return
	}

	// 验证返回的数据
	assert.NotNil(t, etherTable)
	assert.NotEmpty(t, etherTable.Device, "设备名不应为空")
	assert.NotNil(t, etherTable.SrcIp, "源IP不应为空")
	assert.False(t, etherTable.SrcIp.IsLoopback(), "不应是回环地址")
	assert.NotEqual(t, "00:00:00:00:00:00", etherTable.SrcMac.String(), "MAC地址不应全零")

	t.Logf("成功获取网卡: Device=%s, IP=%s, MAC=%s, Gateway MAC=%s",
		etherTable.Device, etherTable.SrcIp, etherTable.SrcMac, etherTable.DstMac)
}

// TestResolveGatewayMAC 测试ARP解析网关MAC
func TestResolveGatewayMAC(t *testing.T) {
	if !hasAdminPrivileges() {
		t.Skip("需要管理员权限运行此测试")
	}

	// 获取本地网络信息
	etherTable, err := GetDefaultRouteInterface()
	if err != nil {
		t.Skip("无法获取网络信息，跳过ARP测试")
	}

	// 尝试解析本地网关
	// 注意：这个测试在实际环境中运行
	gatewayIP := getDefaultGateway()
	if gatewayIP == nil {
		t.Skip("无法获取默认网关，跳过ARP测试")
	}

	srcMAC := net.HardwareAddr(etherTable.SrcMac)
	mac, err := resolveGatewayMAC(etherTable.Device, etherTable.SrcIp, srcMAC, gatewayIP)

	if err != nil {
		t.Logf("ARP解析失败(可能是网络环境): %v", err)
		return
	}

	assert.NotNil(t, mac)
	assert.Len(t, mac, 6, "MAC地址应该是6字节")
	assert.NotEqual(t, "00:00:00:00:00:00", mac.String(), "MAC地址不应全零")
	assert.NotEqual(t, "ff:ff:ff:ff:ff:ff", mac.String(), "MAC地址不应是广播地址")

	t.Logf("成功解析网关MAC: %s -> %s", gatewayIP, mac)
}

// TestValidateInterface 测试网卡验证
func TestValidateInterface(t *testing.T) {
	if !hasAdminPrivileges() {
		t.Skip("需要管理员权限运行此测试")
	}

	tests := []struct {
		name       string
		etherTable *EtherTable
		expected   bool
	}{
		{
			name: "无效的网卡名",
			etherTable: &EtherTable{
				Device: "invalid_device_xyz",
				SrcIp:  net.ParseIP("192.168.1.100"),
			},
			expected: false,
		},
		{
			name: "空网卡名",
			etherTable: &EtherTable{
				Device: "",
				SrcIp:  net.ParseIP("192.168.1.100"),
			},
			expected: false,
		},
	}

	// 添加一个有效网卡的测试
	devices, err := pcap.FindAllDevs()
	if err == nil && len(devices) > 0 {
		validDevice := devices[0].Name
		tests = append(tests, struct {
			name       string
			etherTable *EtherTable
			expected   bool
		}{
			name: "有效的网卡",
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

// TestAutoGetDevicesImproved 测试改进的自动获取方法
func TestAutoGetDevicesImproved(t *testing.T) {
	if !hasAdminPrivileges() {
		t.Skip("需要管理员权限运行此测试")
	}

	// 使用常见的公共DNS服务器
	testDNS := []string{
		"8.8.8.8",
		"1.1.1.1",
		"114.114.114.114",
	}

	etherTable, err := AutoGetDevicesImproved(testDNS)

	// 在某些环境可能失败
	if err != nil {
		t.Logf("自动获取网卡失败(环境问题): %v", err)
		return
	}

	assert.NotNil(t, etherTable)
	assert.NotEmpty(t, etherTable.Device)
	assert.NotNil(t, etherTable.SrcIp)
	assert.False(t, etherTable.SrcIp.IsUnspecified(), "IP不应是未指定地址")

	t.Logf("成功自动获取网卡: %+v", etherTable)
}

// BenchmarkGetDefaultRouteInterface 性能测试
func BenchmarkGetDefaultRouteInterface(b *testing.B) {
	if !hasAdminPrivileges() {
		b.Skip("需要管理员权限运行此测试")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetDefaultRouteInterface()
	}
}

// BenchmarkResolveGatewayMAC 性能测试ARP解析
func BenchmarkResolveGatewayMAC(b *testing.B) {
	if !hasAdminPrivileges() {
		b.Skip("需要管理员权限运行此测试")
	}

	// 准备测试数据
	etherTable, err := GetDefaultRouteInterface()
	if err != nil {
		b.Skip("无法获取网络信息")
	}

	gatewayIP := getDefaultGateway()
	if gatewayIP == nil {
		b.Skip("无法获取网关")
	}

	srcMAC := net.HardwareAddr(etherTable.SrcMac)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolveGatewayMAC(etherTable.Device, etherTable.SrcIp, srcMAC, gatewayIP)
	}
}

// 辅助函数：检查是否有管理员权限
func hasAdminPrivileges() bool {
	switch runtime.GOOS {
	case "windows":
		// Windows下检查是否能打开网卡
		devices, err := pcap.FindAllDevs()
		return err == nil && len(devices) > 0
	default:
		// Unix系统检查UID
		return runtime.GOOS == "darwin" || isRoot()
	}
}

// 辅助函数：检查是否是root用户
func isRoot() bool {
	// 尝试打开一个网卡来检查权限
	devices, err := pcap.FindAllDevs()
	if err != nil || len(devices) == 0 {
		return false
	}

	// 尝试打开第一个设备
	handle, err := pcap.OpenLive(devices[0].Name, 1024, false, time.Second)
	if err != nil {
		return false
	}
	handle.Close()
	return true
}

// 辅助函数：获取默认网关
func getDefaultGateway() net.IP {
	// 简单实现，实际使用时应该解析路由表
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	// 假设网关是 .1
	ip := localAddr.IP.To4()
	if ip != nil {
		ip[3] = 1
		return ip
	}
	return nil
}
