package runner

import (
	"testing"

	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"
)

// TestParseDNSName 测试 DNS 域名格式解析
// 修复 Issue #70 的核心函数
func TestParseDNSName(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name: "标准域名 - www.google.com",
			input: []byte{
				3, 'w', 'w', 'w',
				6, 'g', 'o', 'o', 'g', 'l', 'e',
				3, 'c', 'o', 'm',
				0, // 结束符
			},
			expected: "www.google.com",
		},
		{
			name: "二级域名 - baidu.com",
			input: []byte{
				5, 'b', 'a', 'i', 'd', 'u',
				3, 'c', 'o', 'm',
				0,
			},
			expected: "baidu.com",
		},
		{
			name: "三级域名 - mail.qq.com",
			input: []byte{
				4, 'm', 'a', 'i', 'l',
				2, 'q', 'q',
				3, 'c', 'o', 'm',
				0,
			},
			expected: "mail.qq.com",
		},
		{
			name:     "空输入",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "仅结束符",
			input:    []byte{0},
			expected: "",
		},
		{
			name: "无结束符域名",
			input: []byte{
				3, 'w', 'w', 'w',
				6, 'g', 'o', 'o', 'g', 'l', 'e',
				3, 'c', 'o', 'm',
			},
			expected: "www.google.com",
		},
		{
			name: "长域名",
			input: []byte{
				10, 's', 'u', 'b', 'd', 'o', 'm', 'a', 'i', 'n', '1',
				10, 's', 'u', 'b', 'd', 'o', 'm', 'a', 'i', 'n', '2',
				7, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
				3, 'c', 'o', 'm',
				0,
			},
			expected: "subdomain1.subdomain2.example.com",
		},
		{
			name: "压缩指针 (0xC0) - 应该停止",
			input: []byte{
				3, 'w', 'w', 'w',
				0xC0, 0x12, // 压缩指针
			},
			expected: "www",
		},
		{
			name: "异常长度 - 超出范围",
			input: []byte{
				100, 'a', 'b', 'c', // 长度100但数据不足
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDNSName(tt.input)
			assert.Equal(t, tt.expected, result, "DNS 域名解析结果不匹配")
		})
	}
}

// TestDNSRecord2String_CNAME 测试 CNAME 记录转换
func TestDNSRecord2String_CNAME(t *testing.T) {
	tests := []struct {
		name     string
		rr       layers.DNSResourceRecord
		expected string
		hasError bool
	}{
		{
			name: "标准 CNAME",
			rr: layers.DNSResourceRecord{
				Type:  layers.DNSTypeCNAME,
				Class: layers.DNSClassIN,
				CNAME: []byte{
					3, 'w', 'w', 'w',
					6, 'g', 'o', 'o', 'g', 'l', 'e',
					3, 'c', 'o', 'm',
					0,
				},
			},
			expected: "CNAME www.google.com",
			hasError: false,
		},
		{
			name: "CNAME - cdn.example.com",
			rr: layers.DNSResourceRecord{
				Type:  layers.DNSTypeCNAME,
				Class: layers.DNSClassIN,
				CNAME: []byte{
					3, 'c', 'd', 'n',
					7, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
					3, 'c', 'o', 'm',
					0,
				},
			},
			expected: "CNAME cdn.example.com",
			hasError: false,
		},
		{
			name: "空 CNAME",
			rr: layers.DNSResourceRecord{
				Type:  layers.DNSTypeCNAME,
				Class: layers.DNSClassIN,
				CNAME: nil,
			},
			expected: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := dnsRecord2String(tt.rr)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestDNSRecord2String_NS 测试 NS 记录转换
func TestDNSRecord2String_NS(t *testing.T) {
	tests := []struct {
		name     string
		rr       layers.DNSResourceRecord
		expected string
	}{
		{
			name: "标准 NS",
			rr: layers.DNSResourceRecord{
				Type:  layers.DNSTypeNS,
				Class: layers.DNSClassIN,
				NS: []byte{
					3, 'n', 's', '1',
					7, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
					3, 'c', 'o', 'm',
					0,
				},
			},
			expected: "NS ns1.example.com",
		},
		{
			name: "NS - dns.google.com",
			rr: layers.DNSResourceRecord{
				Type:  layers.DNSTypeNS,
				Class: layers.DNSClassIN,
				NS: []byte{
					3, 'd', 'n', 's',
					6, 'g', 'o', 'o', 'g', 'l', 'e',
					3, 'c', 'o', 'm',
					0,
				},
			},
			expected: "NS dns.google.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := dnsRecord2String(tt.rr)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDNSRecord2String_A 测试 A 记录转换
func TestDNSRecord2String_A(t *testing.T) {
	tests := []struct {
		name     string
		rr       layers.DNSResourceRecord
		expected string
	}{
		{
			name: "标准 A 记录",
			rr: layers.DNSResourceRecord{
				Type:  layers.DNSTypeA,
				Class: layers.DNSClassIN,
				IP:    []byte{192, 168, 1, 1},
			},
			expected: "192.168.1.1",
		},
		{
			name: "公网 IP",
			rr: layers.DNSResourceRecord{
				Type:  layers.DNSTypeA,
				Class: layers.DNSClassIN,
				IP:    []byte{8, 8, 8, 8},
			},
			expected: "8.8.8.8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := dnsRecord2String(tt.rr)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDNSRecord2String_PTR 测试 PTR 记录转换
func TestDNSRecord2String_PTR(t *testing.T) {
	rr := layers.DNSResourceRecord{
		Type:  layers.DNSTypePTR,
		Class: layers.DNSClassIN,
		PTR: []byte{
			7, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
			3, 'c', 'o', 'm',
			0,
		},
	}

	result, err := dnsRecord2String(rr)
	assert.NoError(t, err)
	assert.Equal(t, "PTR example.com", result)
}

// BenchmarkParseDNSName 基准测试 DNS 域名解析性能
func BenchmarkParseDNSName(b *testing.B) {
	input := []byte{
		3, 'w', 'w', 'w',
		6, 'g', 'o', 'o', 'g', 'l', 'e',
		3, 'c', 'o', 'm',
		0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseDNSName(input)
	}
}

// BenchmarkDNSRecord2String 基准测试 DNS 记录转换性能
func BenchmarkDNSRecord2String(b *testing.B) {
	rr := layers.DNSResourceRecord{
		Type:  layers.DNSTypeCNAME,
		Class: layers.DNSClassIN,
		CNAME: []byte{
			3, 'w', 'w', 'w',
			6, 'g', 'o', 'o', 'g', 'l', 'e',
			3, 'c', 'o', 'm',
			0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dnsRecord2String(rr)
	}
}
