package runner

import (
	"sync"
	"testing"

	"github.com/boy-hack/ksubdomain/v2/pkg/device"
	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"
)

// newTestIface 构建一个用于单元测试的 netInterface（无真实 pcap 句柄）
func newTestIface(ether *device.EtherTable, port int) *netInterface {
	return &netInterface{
		etherInfo:  ether,
		pcapHandle: nil,
		listenPort: port,
	}
}

// TestGetOrCreate_TemplateCache 测试模板缓存功能
func TestGetOrCreate_TemplateCache(t *testing.T) {
	ether := &device.EtherTable{
		SrcIp:  []byte{192, 168, 1, 100},
		SrcMac: device.SelfMac{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMac: device.SelfMac{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
	}

	dnsServer := "8.8.8.8"
	iface := newTestIface(ether, 12345)

	// 首次调用 - 创建模板
	template1 := iface.getOrCreate(dnsServer)
	assert.NotNil(t, template1)
	assert.Equal(t, dnsServer, template1.dnsip.String())

	// 第二次调用 - 应该从缓存获取
	template2 := iface.getOrCreate(dnsServer)
	assert.NotNil(t, template2)

	// 应该是同一个对象 (指针相同)
	assert.Equal(t, template1, template2, "应该返回缓存的模板")
}

// TestGetOrCreate_DifferentServers 测试不同 DNS 服务器使用不同模板
func TestGetOrCreate_DifferentServers(t *testing.T) {
	ether := &device.EtherTable{
		SrcIp:  []byte{192, 168, 1, 100},
		SrcMac: device.SelfMac{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMac: device.SelfMac{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
	}

	iface := newTestIface(ether, 12345)

	template1 := iface.getOrCreate("8.8.8.8")
	template2 := iface.getOrCreate("1.1.1.1")

	// 不同的 DNS 服务器应该有不同的模板
	assert.NotEqual(t, template1, template2)
	assert.Equal(t, "8.8.8.8", template1.dnsip.String())
	assert.Equal(t, "1.1.1.1", template2.dnsip.String())
}

// TestGetOrCreate_DifferentPorts 测试不同端口使用不同模板
func TestGetOrCreate_DifferentPorts(t *testing.T) {
	ether := &device.EtherTable{
		SrcIp:  []byte{192, 168, 1, 100},
		SrcMac: device.SelfMac{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMac: device.SelfMac{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
	}

	iface1 := newTestIface(ether, 12345)
	iface2 := newTestIface(ether, 54321)

	template1 := iface1.getOrCreate("8.8.8.8")
	template2 := iface2.getOrCreate("8.8.8.8")

	// 不同端口（不同 iface）应该有不同的模板
	assert.NotEqual(t, template1, template2)
	assert.Equal(t, layers.UDPPort(12345), template1.udp.SrcPort)
	assert.Equal(t, layers.UDPPort(54321), template2.udp.SrcPort)
}

// TestGetOrCreate_Concurrent 测试并发访问模板缓存
func TestGetOrCreate_Concurrent(t *testing.T) {
	ether := &device.EtherTable{
		SrcIp:  []byte{192, 168, 1, 100},
		SrcMac: device.SelfMac{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMac: device.SelfMac{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
	}

	iface := newTestIface(ether, 12345)

	concurrency := 100
	var wg sync.WaitGroup
	wg.Add(concurrency)

	templates := make([]*packetTemplate, concurrency)

	// 并发获取模板
	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			templates[idx] = iface.getOrCreate("8.8.8.8")
		}(i)
	}

	wg.Wait()

	// 所有并发调用应该返回同一个模板
	firstTemplate := templates[0]
	for i := 1; i < concurrency; i++ {
		assert.Equal(t, firstTemplate, templates[i],
			"并发调用应该返回相同的缓存模板")
	}
}

// TestTemplateCache_MultipleServers 测试多服务器缓存
func TestTemplateCache_MultipleServers(t *testing.T) {
	ether := &device.EtherTable{
		SrcIp:  []byte{192, 168, 1, 100},
		SrcMac: device.SelfMac{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMac: device.SelfMac{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
	}

	iface := newTestIface(ether, 53)

	// 常见的公共 DNS 服务器
	dnsServers := []string{
		"8.8.8.8",         // Google
		"8.8.4.4",         // Google
		"1.1.1.1",         // Cloudflare
		"1.0.0.1",         // Cloudflare
		"114.114.114.114", // 114 DNS
		"223.5.5.5",       // 阿里 DNS
	}

	templates := make(map[string]*packetTemplate)

	// 为每个 DNS 服务器创建模板
	for _, dns := range dnsServers {
		templates[dns] = iface.getOrCreate(dns)
	}

	// 验证每个服务器都有唯一的模板
	for i, dns1 := range dnsServers {
		for j, dns2 := range dnsServers {
			if i == j {
				// 同一个服务器,再次获取应该是缓存
				cached := iface.getOrCreate(dns1)
				assert.Equal(t, templates[dns1], cached)
			} else {
				// 不同服务器,模板应该不同
				assert.NotEqual(t, templates[dns1], templates[dns2])
			}
		}
	}
}

// BenchmarkGetOrCreate_CacheHit 基准测试缓存命中性能
func BenchmarkGetOrCreate_CacheHit(b *testing.B) {
	ether := &device.EtherTable{
		SrcIp:  []byte{192, 168, 1, 100},
		SrcMac: device.SelfMac{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMac: device.SelfMac{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
	}

	iface := newTestIface(ether, 53)
	// 预热缓存
	_ = iface.getOrCreate("8.8.8.8")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = iface.getOrCreate("8.8.8.8")
	}
}

// BenchmarkGetOrCreate_CacheMiss 基准测试缓存未命中性能
func BenchmarkGetOrCreate_CacheMiss(b *testing.B) {
	ether := &device.EtherTable{
		SrcIp:  []byte{192, 168, 1, 100},
		SrcMac: device.SelfMac{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMac: device.SelfMac{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
	}

	iface := newTestIface(ether, 53)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 每次使用不同的DNS服务器,触发缓存未命中
		dns := "8.8.8." + string(rune(i%255))
		_ = iface.getOrCreate(dns)
	}
}

// BenchmarkGetOrCreate_Concurrent 并发基准测试
func BenchmarkGetOrCreate_Concurrent(b *testing.B) {
	ether := &device.EtherTable{
		SrcIp:  []byte{192, 168, 1, 100},
		SrcMac: device.SelfMac{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMac: device.SelfMac{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
	}

	iface := newTestIface(ether, 53)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = iface.getOrCreate("8.8.8.8")
		}
	})
}
