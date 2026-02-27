package runner

import (
	"context"
	"math"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/boy-hack/ksubdomain/v2/internal/utils"
	"github.com/boy-hack/ksubdomain/v2/pkg/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/options"
	"github.com/boy-hack/ksubdomain/v2/pkg/device"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/processbar"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/statusdb"
	"github.com/google/gopacket/pcap"
	"github.com/phayes/freeport"
	"go.uber.org/ratelimit"
)

// Runner represents the runtime structure for subdomain scanning
type Runner struct {
	statusDB        *statusdb.StatusDb // Status database
	options         *options.Options   // Configuration options
	rateLimiter     ratelimit.Limiter  // Rate limiter
	pcapHandle      *pcap.Handle       // Network capture handle
	successCount    uint64             // Success count
	sendCount       uint64             // Send count
	receiveCount    uint64             // Receive count
	failedCount     uint64             // Failed count
	domainChan      chan string        // Domain sending channel
	resultChan      chan result.Result // Result receiving channel
	listenPort      int                // Listening port
	dnsID           uint16             // DNS request ID
	maxRetryCount   int                // Maximum retry count
	timeoutSeconds  int64              // Timeout in seconds
	initialLoadDone chan struct{}      // Initial load done signal
	predictLoadDone chan struct{}      // Predict load done signal
	startTime       time.Time          // Start time
	stopSignal      chan struct{}      // Stop signal
}

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

// New creates a new Runner instance
func New(opt *options.Options) (*Runner, error) {
	var err error
	version := pcap.Version()
	r := new(Runner)
	gologger.Infof(version)
	r.options = opt
	r.statusDB = statusdb.CreateMemoryDB()

	// Log DNS server information
	gologger.Infof("Default DNS servers: %s\n", utils.SliceToString(opt.Resolvers))
	if len(opt.SpecialResolvers) > 0 {
		var keys []string
		for k := range opt.SpecialResolvers {
			keys = append(keys, k)
		}
		gologger.Infof("Special DNS servers: %s\n", utils.SliceToString(keys))
	}

	// Initialize network device
	r.pcapHandle, err = device.PcapInit(opt.EtherInfo.Device)
	if err != nil {
		return nil, err
	}

	// Set rate limit
	cpuLimit := float64(runtime.NumCPU() * 10000)
	rateLimit := int(math.Min(cpuLimit, float64(opt.Rate)))

	// Mac platform optimization: BPF buffer has stricter limits.
	// Recommended rate < 50000 pps to avoid buffer overflow.
	if runtime.GOOS == "darwin" && rateLimit > 50000 {
		gologger.Warningf("Mac platform detected: current rate %d pps may cause buffer issues\n", rateLimit)
		gologger.Warningf("Suggestion: use -b flag to limit bandwidth (e.g., -b 5m) or lower the rate\n")
		gologger.Warningf("Note: Mac BPF buffer has been optimized to 2MB, but rate < 50000 pps is still recommended\n")
	}

	r.rateLimiter = ratelimit.New(rateLimit)
	gologger.Infof("Rate limit: %d pps\n", rateLimit)

	// Initialize channels
	r.domainChan = make(chan string, 50000)
	r.resultChan = make(chan result.Result, 5000)
	r.stopSignal = make(chan struct{})

	// Get a free port
	freePort, err := freeport.GetFreePort()
	if err != nil {
		return nil, err
	}
	r.listenPort = freePort
	gologger.Infof("Listening port: %d\n", freePort)

	// Set other parameters
	r.dnsID = 0x2021 // ksubdomain's birthday
	r.maxRetryCount = opt.Retry
	r.timeoutSeconds = int64(opt.TimeOut)
	r.initialLoadDone = make(chan struct{})
	r.predictLoadDone = make(chan struct{})
	r.startTime = time.Now()
	return r, nil
}

// selectDNSServer intelligently selects a DNS server based on the domain name
func (r *Runner) selectDNSServer(domain string) string {
	dnsServers := r.options.Resolvers
	specialDNSServers := r.options.SpecialResolvers

	// Select a specific DNS server based on domain suffix
	if len(specialDNSServers) > 0 {
		for suffix, servers := range specialDNSServers {
			if strings.HasSuffix(domain, suffix) {
				dnsServers = servers
				break
			}
		}
	}

	// Randomly select a DNS server
	idx := getRandomIndex() % len(dnsServers)
	return dnsServers[idx]
}

// getRandomIndex returns a random index
func getRandomIndex() int {
	return int(rand.Int31())
}

// updateStatusBar updates the progress bar status
func (r *Runner) updateStatusBar() {
	if r.options.ProcessBar != nil {
		queueLength := r.statusDB.Length()
		elapsedSeconds := int(time.Since(r.startTime).Seconds())
		data := &processbar.ProcessData{
			SuccessIndex: r.successCount,
			SendIndex:    r.sendCount,
			QueueLength:  queueLength,
			RecvIndex:    r.receiveCount,
			FaildIndex:   r.failedCount,
			Elapsed:      elapsedSeconds,
		}
		r.options.ProcessBar.WriteData(data)
	}
}

// loadDomainsFromSource loads domains from the source channel
func (r *Runner) loadDomainsFromSource(wg *sync.WaitGroup) {
	defer wg.Done()
	// Load domains from the domain source
	for domain := range r.options.Domain {
		r.domainChan <- domain
	}
	// Notify that the initial load is complete
	r.initialLoadDone <- struct{}{}
}

// monitorProgress monitors the scan progress
func (r *Runner) monitorProgress(ctx context.Context, cancelFunc context.CancelFunc, wg *sync.WaitGroup) {
	var initialLoadCompleted bool = false
	var initialLoadPredict bool = false
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	defer wg.Done()
	for {
		select {
		case <-ticker.C:
			// Update status bar
			r.updateStatusBar()
			// Check if scan is complete
			if initialLoadCompleted && initialLoadPredict {
				queueLength := r.statusDB.Length()
				if queueLength <= 0 {
					gologger.Printf("\n")
					gologger.Infof("Scan completed")
					cancelFunc() // Use the passed cancelFunc
					return
				}
			}
		case <-r.initialLoadDone:
			// Start retry mechanism after initial load is complete
			go r.retry(ctx)
			initialLoadCompleted = true
		case <-r.predictLoadDone:
			initialLoadPredict = true
		case <-ctx.Done():
			return
		}
	}
}

// processPredictedDomains handles predicted domain names
func (r *Runner) processPredictedDomains(ctx context.Context, wg *sync.WaitGroup, predictChan chan string) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case domain := <-predictChan:
			r.domainChan <- domain
		}
	}
}

// RunEnumeration starts the subdomain enumeration process
func (r *Runner) RunEnumeration(ctx context.Context) {
	// Create a cancellable context
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	// Create wait group; need to wait for 5 goroutines (recv, send, monitor, result, load)
	wg := &sync.WaitGroup{}
	wg.Add(5)

	// Start receive processing
	go r.recvChanel(ctx, wg)

	// Start send processing (under waitgroup management)
	go r.sendCycleWithContext(ctx, wg)

	// Monitor progress
	go r.monitorProgress(ctx, cancelFunc, wg)

	// Create predicted domain channel
	predictChan := make(chan string, 1000)
	if r.options.Predict {
		wg.Add(1)
		// Start predicted domain processing
		go r.processPredictedDomains(ctx, wg, predictChan)
	} else {
		r.predictLoadDone <- struct{}{}
	}

	// Start result processing (under waitgroup management)
	go r.handleResultWithContext(ctx, wg, predictChan)

	// Load domains from source
	go r.loadDomainsFromSource(wg)

	// Wait for all goroutines to complete
	wg.Wait()

	// Close all channels
	close(predictChan)
	// Safely close channels
	close(r.resultChan)
	close(r.domainChan)
}

// Close closes the Runner and releases resources
func (r *Runner) Close() {
	// Close network capture handle
	if r.pcapHandle != nil {
		r.pcapHandle.Close()
	}

	// Close status database
	if r.statusDB != nil {
		r.statusDB.Close()
	}

	// Close all output handlers
	for _, out := range r.options.Writer {
		err := out.Close()
		if err != nil {
			gologger.Errorf("Failed to close output handler: %v", err)
		}
	}

	// Close progress bar
	if r.options.ProcessBar != nil {
		r.options.ProcessBar.Close()
	}
}
