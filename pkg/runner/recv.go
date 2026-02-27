package runner

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// parseDNSName parses the DNS domain name format.
// DNS name format: length prefix + label + ... + terminator
// Example: \x03www\x06google\x03com\x00 represents www.google.com
// Fix Issue #70: correctly parses CNAME/NS/PTR records to avoid concatenation errors like "comcom"
func parseDNSName(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}

	var result []byte
	i := 0

	for i < len(raw) {
		// Read label length
		length := int(raw[i])

		// 0x00 means end of name
		if length == 0 {
			break
		}

		// 0xC0 prefix means compression pointer (RFC 1035)
		// Compressed format: top 2 bits are 11, lower 14 bits are offset
		if length >= 0xC0 {
			// Compression pointer; not handled here (only present in full DNS packets)
			break
		}

		// Add dot separator (except for the first label)
		if len(result) > 0 {
			result = append(result, '.')
		}

		i++

		// Guard against out-of-bounds
		if i+length > len(raw) {
			break
		}

		// Append label content
		result = append(result, raw[i:i+length]...)
		i += length
	}

	return string(result)
}

// dnsRecord2String converts a DNS resource record to a string.
// Fix Issue #70: use parseDNSName to correctly parse domain name format.
func dnsRecord2String(rr layers.DNSResourceRecord) (string, error) {
	if rr.Class == layers.DNSClassIN {
		switch rr.Type {
		case layers.DNSTypeA, layers.DNSTypeAAAA:
			if rr.IP != nil {
				return rr.IP.String(), nil
			}
		case layers.DNSTypeNS:
			if rr.NS != nil {
				// Fix: use parseDNSName to parse NS record
				ns := parseDNSName(rr.NS)
				if ns != "" {
					return "NS " + ns, nil
				}
			}
		case layers.DNSTypeCNAME:
			if rr.CNAME != nil {
				// Fix: use parseDNSName to parse CNAME record
				cname := parseDNSName(rr.CNAME)
				if cname != "" {
					return "CNAME " + cname, nil
				}
			}
		case layers.DNSTypePTR:
			if rr.PTR != nil {
				// Fix: use parseDNSName to parse PTR record
				ptr := parseDNSName(rr.PTR)
				if ptr != "" {
					return "PTR " + ptr, nil
				}
			}
		case layers.DNSTypeTXT:
			if rr.TXT != nil {
				// TXT records are plain text, no parsing needed
				return "TXT " + string(rr.TXT), nil
			}
		}
	}
	return "", errors.New("dns record error")
}

// Pre-allocated decoder object pool to avoid frequent creation
var decoderPool = sync.Pool{
	New: func() interface{} {
		var eth layers.Ethernet
		var ipv4 layers.IPv4
		var ipv6 layers.IPv6
		var udp layers.UDP
		var dns layers.DNS
		parser := gopacket.NewDecodingLayerParser(
			layers.LayerTypeEthernet, &eth, &ipv4, &ipv6, &udp, &dns)

		return &decodingContext{
			parser:  parser,
			eth:     &eth,
			ipv4:    &ipv4,
			ipv6:    &ipv6,
			udp:     &udp,
			dns:     &dns,
			decoded: make([]gopacket.LayerType, 0, 5),
		}
	},
}

// decodingContext holds the decoding context
type decodingContext struct {
	parser  *gopacket.DecodingLayerParser
	eth     *layers.Ethernet
	ipv4    *layers.IPv4
	ipv6    *layers.IPv6
	udp     *layers.UDP
	dns     *layers.DNS
	decoded []gopacket.LayerType
}

// processPacket parses a DNS response packet and handles it
func (r *Runner) processPacket(data []byte, dnsChanel chan<- layers.DNS) {
	// Get decoder from pool
	dc := decoderPool.Get().(*decodingContext)
	defer decoderPool.Put(dc)

	// Clear the decoded layer types slice
	dc.decoded = dc.decoded[:0]

	// Parse the packet
	err := dc.parser.DecodeLayers(data, &dc.decoded)
	if err != nil {
		return
	}

	// Check if it is a DNS response
	if !dc.dns.QR {
		return
	}

	// Verify DNS ID matches
	if dc.dns.ID != r.dnsID {
		return
	}

	// Ensure there is at least one question
	if len(dc.dns.Questions) == 0 {
		return
	}

	// Record number of received packets
	atomic.AddUint64(&r.receiveCount, 1)

	// Send DNS response to the processing channel
	select {
	case dnsChanel <- *dc.dns:
	}
}

// recvChanel implements the functionality to receive DNS responses
func (r *Runner) recvChanel(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	var (
		snapshotLen = 65536
		timeout     = 5 * time.Second
		err         error
	)
	inactive, err := pcap.NewInactiveHandle(r.options.EtherInfo.Device)
	if err != nil {
		gologger.Errorf("Failed to create network capture handle: %v", err)
		return
	}
	err = inactive.SetSnapLen(snapshotLen)
	if err != nil {
		gologger.Errorf("Failed to set snapshot length: %v", err)
		return
	}
	defer inactive.CleanUp()

	if err = inactive.SetTimeout(timeout); err != nil {
		gologger.Errorf("Failed to set timeout: %v", err)
		return
	}

	err = inactive.SetImmediateMode(true)
	if err != nil {
		gologger.Errorf("Failed to set immediate mode: %v", err)
		return
	}

	handle, err := inactive.Activate()
	if err != nil {
		gologger.Errorf("Failed to activate network capture: %v", err)
		return
	}
	defer handle.Close()

	err = handle.SetBPFFilter(fmt.Sprintf("udp and src port 53 and dst port %d", r.listenPort))
	if err != nil {
		gologger.Errorf("Failed to set BPF filter: %v", err)
		return
	}

	// Create DNS response processing channel with adequate buffer size
	dnsChanel := make(chan layers.DNS, 10000)

	// Use multiple goroutines to process DNS responses for higher concurrency
	processorCount := runtime.NumCPU() * 2
	var processorWg sync.WaitGroup
	processorWg.Add(processorCount)

	// Start multiple processing goroutines
	for i := 0; i < processorCount; i++ {
		go func() {
			defer processorWg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case dns, ok := <-dnsChanel:
					if !ok {
						return
					}

					subdomain := string(dns.Questions[0].Name)
					r.statusDB.Del(subdomain)
					if dns.ANCount > 0 {
						atomic.AddUint64(&r.successCount, 1)
						var answers []string
						for _, v := range dns.Answers {
							answer, err := dnsRecord2String(v)
							if err != nil {
								continue
							}
							answers = append(answers, answer)
						}
						r.resultChan <- result.Result{
							Subdomain: subdomain,
							Answers:   answers,
						}
					}
				}
			}
		}()
	}

	// Use a goroutine to read network packets
	packetChan := make(chan []byte, 10000)

	// Start packet receiving goroutine
	go func() {
		for {
			data, _, err := handle.ReadPacketData()
			if err != nil {
				if errors.Is(err, pcap.NextErrorTimeoutExpired) {
					continue
				}
				return
			}

			select {
			case <-ctx.Done():
				return
			case packetChan <- data:
				// Packet sent to processing channel
			}
		}
	}()

	// Start multiple packet parsing goroutines
	parserCount := runtime.NumCPU() * 2
	var parserWg sync.WaitGroup
	parserWg.Add(parserCount)

	for i := 0; i < parserCount; i++ {
		go func() {
			defer parserWg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case data, ok := <-packetChan:
					if !ok {
						return
					}
					r.processPacket(data, dnsChanel)
				}
			}
		}()
	}

	// Wait for context to be done
	<-ctx.Done()

	// Close channels
	close(packetChan)
	close(dnsChanel)

	// Wait for all processing and parsing goroutines to finish
	parserWg.Wait()
	processorWg.Wait()
}
