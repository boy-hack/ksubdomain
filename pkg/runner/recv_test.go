package runner

import (
	"testing"

	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"
)

// TestParseDNSName tests DNS domain name format parsing.
// Core function for fixing Issue #70.
func TestParseDNSName(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name: "Standard domain - www.google.com",
			input: []byte{
				3, 'w', 'w', 'w',
				6, 'g', 'o', 'o', 'g', 'l', 'e',
				3, 'c', 'o', 'm',
				0, // terminator
			},
			expected: "www.google.com",
		},
		{
			name: "Second-level domain - baidu.com",
			input: []byte{
				5, 'b', 'a', 'i', 'd', 'u',
				3, 'c', 'o', 'm',
				0,
			},
			expected: "baidu.com",
		},
		{
			name: "Third-level domain - mail.qq.com",
			input: []byte{
				4, 'm', 'a', 'i', 'l',
				2, 'q', 'q',
				3, 'c', 'o', 'm',
				0,
			},
			expected: "mail.qq.com",
		},
		{
			name:     "Empty input",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "Terminator only",
			input:    []byte{0},
			expected: "",
		},
		{
			name: "Domain without terminator",
			input: []byte{
				3, 'w', 'w', 'w',
				6, 'g', 'o', 'o', 'g', 'l', 'e',
				3, 'c', 'o', 'm',
			},
			expected: "www.google.com",
		},
		{
			name: "Long domain",
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
			name: "Compression pointer (0xC0) - should stop",
			input: []byte{
				3, 'w', 'w', 'w',
				0xC0, 0x12, // compression pointer
			},
			expected: "www",
		},
		{
			name: "Abnormal length - out of range",
			input: []byte{
				100, 'a', 'b', 'c', // length 100 but insufficient data
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDNSName(tt.input)
			assert.Equal(t, tt.expected, result, "DNS domain name parsing result mismatch")
		})
	}
}

// TestDNSRecord2String_CNAME tests CNAME record conversion
func TestDNSRecord2String_CNAME(t *testing.T) {
	tests := []struct {
		name     string
		rr       layers.DNSResourceRecord
		expected string
		hasError bool
	}{
		{
			name: "Standard CNAME",
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
			name: "Empty CNAME",
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

// TestDNSRecord2String_NS tests NS record conversion
func TestDNSRecord2String_NS(t *testing.T) {
	tests := []struct {
		name     string
		rr       layers.DNSResourceRecord
		expected string
	}{
		{
			name: "Standard NS",
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

// TestDNSRecord2String_A tests A record conversion
func TestDNSRecord2String_A(t *testing.T) {
	tests := []struct {
		name     string
		rr       layers.DNSResourceRecord
		expected string
	}{
		{
			name: "Standard A record",
			rr: layers.DNSResourceRecord{
				Type:  layers.DNSTypeA,
				Class: layers.DNSClassIN,
				IP:    []byte{192, 168, 1, 1},
			},
			expected: "192.168.1.1",
		},
		{
			name: "Public IP",
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

// TestDNSRecord2String_PTR tests PTR record conversion
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

// BenchmarkParseDNSName benchmarks DNS domain name parsing performance
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

// BenchmarkDNSRecord2String benchmarks DNS record conversion performance
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
