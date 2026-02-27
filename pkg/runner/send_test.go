package runner

import (
	"sync"
	"testing"

	"github.com/boy-hack/ksubdomain/v2/pkg/device"
	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"
)

// TestGetOrCreate_TemplateCache 测试模板缓存功能
func TestGetOrCreate_TemplateCache(t *testing.T) {
	ether := &device.EtherTable{
		SrcIp:  []byte{192, 168, 1, 100},
		SrcMac: device.SelfMac{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMac: device.SelfMac{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
	}

	dnsServer := "8.8.8.8"
	port := uint16(12345)

	// 首次调用 - 创建模板
	template1 := getOrCreate(dnsServer, ether, port)
	assert.NotNil(t, template1)
	assert.Equal(t, dnsServer, template1.dnsip.String())

	// 第二次调用 - 应该从缓存获取
	template2 := getOrCreate(dnsServer, ether, port)
	assert.NotNil(t, template2)

	// Should be the same object (same pointer)
	assert.Equal(t, template1, template2, "Should return the cached template")
}

// TestGetOrCreate_DifferentServers 测试不同 DNS 服务器使用不同模板
func TestGetOrCreate_DifferentServers(t *testing.T) {
	ether := &device.EtherTable{
		SrcIp:  []byte{192, 168, 1, 100},
		SrcMac: device.SelfMac{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMac: device.SelfMac{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
	}

	port := uint16(12345)

	template1 := getOrCreate("8.8.8.8", ether, port)
	template2 := getOrCreate("1.1.1.1", ether, port)

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

	dnsServer := "8.8.8.8"

	template1 := getOrCreate(dnsServer, ether, 12345)
	template2 := getOrCreate(dnsServer, ether, 54321)

	// 不同端口应该有不同的模板
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

	dnsServer := "8.8.8.8"
	port := uint16(12345)

	concurrency := 100
	var wg sync.WaitGroup
	wg.Add(concurrency)

	templates := make([]*packetTemplate, concurrency)

	// 并发获取模板
	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			templates[idx] = getOrCreate(dnsServer, ether, port)
		}(i)
	}

	wg.Wait()

	// All concurrent calls should return the same template
	firstTemplate := templates[0]
	for i := 1; i < concurrency; i++ {
		assert.Equal(t, firstTemplate, templates[i],
			"Concurrent calls should return the same cached template")
	}
}

// TestTemplateCache_MultipleServers 测试多服务器缓存
func TestTemplateCache_MultipleServers(t *testing.T) {
	ether := &device.EtherTable{
		SrcIp:  []byte{192, 168, 1, 100},
		SrcMac: device.SelfMac{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMac: device.SelfMac{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
	}

	port := uint16(53)

		// Common public DNS servers
		dnsServers := []string{
			"8.8.8.8",       // Google
			"8.8.4.4",       // Google
			"1.1.1.1",       // Cloudflare
			"1.0.0.1",       // Cloudflare
			"114.114.114.114", // 114 DNS
			"223.5.5.5",     // AliDNS
		}

	templates := make(map[string]*packetTemplate)

	// 为每个 DNS 服务器创建模板
	for _, dns := range dnsServers {
		templates[dns] = getOrCreate(dns, ether, port)
	}

	// Verify each server has a unique template
	for i, dns1 := range dnsServers {
		for j, dns2 := range dnsServers {
			if i == j {
				// Same server, fetching again should return from cache
				cached := getOrCreate(dns1, ether, port)
				assert.Equal(t, templates[dns1], cached)
			} else {
				// Different servers should have different templates
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

	// 预热缓存
	_ = getOrCreate("8.8.8.8", ether, 53)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getOrCreate("8.8.8.8", ether, 53)
	}
}

// BenchmarkGetOrCreate_CacheMiss 基准测试缓存未命中性能
func BenchmarkGetOrCreate_CacheMiss(b *testing.B) {
	ether := &device.EtherTable{
		SrcIp:  []byte{192, 168, 1, 100},
		SrcMac: device.SelfMac{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMac: device.SelfMac{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Each time use a different DNS server to trigger cache miss
		dns := "8.8.8." + string(rune(i%255))
		_ = getOrCreate(dns, ether, 53)
	}
}

// BenchmarkGetOrCreate_Concurrent 并发基准测试
func BenchmarkGetOrCreate_Concurrent(b *testing.B) {
	ether := &device.EtherTable{
		SrcIp:  []byte{192, 168, 1, 100},
		SrcMac: device.SelfMac{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMac: device.SelfMac{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = getOrCreate("8.8.8.8", ether, 53)
		}
	})
}
