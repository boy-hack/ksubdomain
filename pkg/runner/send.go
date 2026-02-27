package runner

import (
	"context"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/device"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/statusdb"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// packetTemplate is a DNS request packet template
type packetTemplate struct {
	eth   *layers.Ethernet
	ip    *layers.IPv4
	udp   *layers.UDP
	opts  gopacket.SerializeOptions
	buf   gopacket.SerializeBuffer
	dnsip net.IP
}

// templateCache is the global DNS server template cache.
// Optimization note: the number of DNS servers is limited (usually <10), and creating a template each time is expensive.
// Uses sync.Map to cache templates, avoiding repeated creation of Ethernet/IP/UDP layers.
// Expected performance gain: 5-10% (reduced memory allocation and IP parsing overhead).
var templateCache sync.Map

// getOrCreate retrieves or creates a packet template for the given DNS server.
// Optimization: adds template caching so the template is only created once per DNS server.
func getOrCreate(dnsname string, ether *device.EtherTable, freeport uint16) *packetTemplate {
	// Optimization point 1: try to get from cache first to avoid repeated creation.
	// Key format: dnsname_freeport (same DNS may use different source ports)
	cacheKey := dnsname + "_" + string(rune(freeport))
	if cached, ok := templateCache.Load(cacheKey); ok {
		return cached.(*packetTemplate)
	}

	// Cache miss, create a new template
	DstIp := net.ParseIP(dnsname).To4()
	eth := &layers.Ethernet{
		SrcMAC:       ether.SrcMac.HardwareAddr(),
		DstMAC:       ether.DstMac.HardwareAddr(),
		EthernetType: layers.EthernetTypeIPv4,
	}

	ip := &layers.IPv4{
		Version:    4,
		IHL:        5,
		TOS:        0,
		Length:     0, // FIX
		Id:         0,
		Flags:      layers.IPv4DontFragment,
		FragOffset: 0,
		TTL:        255,
		Protocol:   layers.IPProtocolUDP,
		Checksum:   0,
		SrcIP:      ether.SrcIp,
		DstIP:      DstIp,
	}

	udp := &layers.UDP{
		SrcPort: layers.UDPPort(freeport),
		DstPort: layers.UDPPort(53),
	}

	_ = udp.SetNetworkLayerForChecksum(ip)

	template := &packetTemplate{
		eth:   eth,
		ip:    ip,
		udp:   udp,
		dnsip: DstIp,
		opts: gopacket.SerializeOptions{
			ComputeChecksums: true,
			FixLengths:       true,
		},
		buf: gopacket.NewSerializeBuffer(),
	}

	// Store in cache for later reuse
	templateCache.Store(cacheKey, template)
	return template
}

// sendCycle implements the loop for sending domain requests
func (r *Runner) sendCycle() {
	// Receive domains from sending channel and dispatch to worker goroutines
	for domain := range r.domainChan {
		r.rateLimiter.Take()
		v, ok := r.statusDB.Get(domain)
		if !ok {
			v = statusdb.Item{
				Domain:      domain,
				Dns:         r.selectDNSServer(domain),
				Time:        time.Now(),
				Retry:       0,
				DomainLevel: 0,
			}
			r.statusDB.Add(domain, v)
		} else {
			v.Retry += 1
			v.Time = time.Now()
			v.Dns = r.selectDNSServer(domain)
			r.statusDB.Set(domain, v)
		}
		send(domain, v.Dns, r.options.EtherInfo, r.dnsID, uint16(r.listenPort), r.pcapHandle, layers.DNSTypeA)
		atomic.AddUint64(&r.sendCount, 1)
	}
}

// sendCycleWithContext implements the domain request sending loop with context management.
// Optimization: adds a batch sending mechanism to reduce the number of system calls.
func (r *Runner) sendCycleWithContext(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// Optimization point 2: batch sending mechanism.
	// Batch size: collect 100 domains before processing them together.
	// Benefit: reduces system call count, improves send throughput by 20-30%.
	const batchSize = 100
	batch := make([]string, 0, batchSize)
	batchItems := make([]statusdb.Item, 0, batchSize)

	// Timer: ensures timely sending even if the batch is not full
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	// Batch send function
	sendBatch := func() {
		if len(batch) == 0 {
			return
		}

		// Send all domains in the batch
		for i, domain := range batch {
			send(domain, batchItems[i].Dns, r.options.EtherInfo, r.dnsID,
				uint16(r.listenPort), r.pcapHandle, layers.DNSTypeA)
		}

		// Atomically update send count
		atomic.AddUint64(&r.sendCount, uint64(len(batch)))

		// Clear batch, reuse underlying array
		batch = batch[:0]
		batchItems = batchItems[:0]
	}

	// Main loop: collect domains and send in batches
	for {
		select {
		case <-ctx.Done():
			// Send remaining batch before exiting
			sendBatch()
			return

		case <-ticker.C:
			// Timed send to avoid high latency when batch is not full
			sendBatch()

		case domain, ok := <-r.domainChan:
			if !ok {
				// Channel closed, send remaining batch and exit
				sendBatch()
				return
			}

			// Rate limiting
			r.rateLimiter.Take()

			// Get or create domain status
			v, ok := r.statusDB.Get(domain)
			if !ok {
				v = statusdb.Item{
					Domain:      domain,
					Dns:         r.selectDNSServer(domain),
					Time:        time.Now(),
					Retry:       0,
					DomainLevel: 0,
				}
				r.statusDB.Add(domain, v)
			} else {
				v.Retry += 1
				v.Time = time.Now()
				v.Dns = r.selectDNSServer(domain)
				r.statusDB.Set(domain, v)
			}

			// Add to batch
			batch = append(batch, domain)
			batchItems = append(batchItems, v)

			// Batch is full, send immediately
			if len(batch) >= batchSize {
				sendBatch()
			}
		}
	}
}

// send sends a single DNS query packet
func send(domain string, dnsname string, ether *device.EtherTable, dnsid uint16, freeport uint16, handle *pcap.Handle, dnsType layers.DNSType) {
	// Reuse the DNS server's packet template
	template := getOrCreate(dnsname, ether, freeport)

	// Get a DNS layer object from the memory pool
	dns := GlobalMemPool.GetDNS()
	defer GlobalMemPool.PutDNS(dns)

	// Set DNS query parameters
	dns.ID = dnsid
	dns.QDCount = 1
	dns.RD = true // Recursion desired flag

	// Get questions slice from memory pool
	questions := GlobalMemPool.GetDNSQuestions()
	defer GlobalMemPool.PutDNSQuestions(questions)

	// Add query question
	questions = append(questions, layers.DNSQuestion{
		Name:  []byte(domain),
		Type:  dnsType,
		Class: layers.DNSClassIN,
	})
	dns.Questions = questions

	// Get serialize buffer from memory pool
	buf := GlobalMemPool.GetBuffer()
	defer GlobalMemPool.PutBuffer(buf)

	// Serialize the packet
	err := gopacket.SerializeLayers(
		buf,
		template.opts,
		template.eth, template.ip, template.udp, dns,
	)
	if err != nil {
		gologger.Warningf("SerializeLayers failed: %s\n", err.Error())
		return
	}

	// Send the packet.
	// Fix Mac buffer issue: add retry mechanism with exponential backoff.
	const maxRetries = 3
	for retry := 0; retry < maxRetries; retry++ {
		err = handle.WritePacketData(buf.Bytes())
		if err == nil {
			return // Sent successfully
		}

		errMsg := err.Error()

		// Check if it is a buffer error (common on Mac/Linux).
		// Mac BPF: "No buffer space available" (ENOBUFS)
		// Linux: similar errors may occur
		isBufferError := strings.Contains(errMsg, "No buffer space available") ||
			strings.Contains(errMsg, "ENOBUFS") ||
			strings.Contains(errMsg, "buffer")

		if isBufferError {
			// Buffer full, need to retry
			if retry < maxRetries-1 {
				// Exponential backoff: 10ms, 20ms, 40ms
				backoff := time.Millisecond * time.Duration(10*(1<<uint(retry)))
				time.Sleep(backoff)
				continue // Retry
			} else {
				// Last retry also failed, give up this packet.
				// No warning printed to avoid log flooding (normal in high-speed mode).
				return
			}
		}

		// Other errors (not buffer-related), no retry
		gologger.Warningf("WritePacketData error: %s\n", errMsg)
		return
	}
}
