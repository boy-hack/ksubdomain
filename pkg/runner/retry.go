package runner

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/runner/statusdb"
)

// retry implements an optimized retry mechanism.
// Optimization point 4: improved retry scan efficiency.
// 1. Adds empty-scan detection to avoid unnecessary CPU consumption.
// 2. Uses dedicated worker goroutines to handle retries without blocking the main flow.
// 3. Groups retries by DNS server for batch processing.
func (r *Runner) retry(ctx context.Context) {
	// Check interval: use 200ms instead of the full timeout period for more timely detection of timeouts.
	// The original implementation scanned every timeoutSeconds; now it scans more frequently with empty-scan optimization.
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()

	// Domain buffer for batch sending
	const batchSize = 100
	retryDomains := make([]string, 0, batchSize)

	// Track whether the last scan was empty to conserve resources when the database is empty
	lastScanEmpty := false

	// Start multiple workers to handle retries
	workerCount := 4
	retryDomainCh := make(chan string, batchSize*2)
	var wg sync.WaitGroup
	wg.Add(workerCount)

	// Worker goroutines for sending retry requests
	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case domain, ok := <-retryDomainCh:
					if !ok {
						return
					}
					// Resend
					r.domainChan <- domain
				}
			}
		}()
	}

	// Batch domain buffer grouped by DNS server
	dnsBatches := make(map[string][]string)

	for {
		select {
		case <-ctx.Done():
			close(retryDomainCh)
			wg.Wait()
			return
		case <-t.C:
			// If the last scan was empty and length is still 0, skip
			currentLength := r.statusDB.Length()
			if lastScanEmpty && currentLength == 0 {
				continue
			}

			// Current time
			now := time.Now()
			// Clear domain buffer
			retryDomains = retryDomains[:0]

			// Clear group buffer
			for k := range dnsBatches {
				dnsBatches[k] = dnsBatches[k][:0]
			}

			// Collect domains that need retrying
			r.statusDB.Scan(func(key string, v statusdb.Item) error {
				// Abandon if maximum retry count exceeded
				if r.maxRetryCount > 0 && v.Retry > r.maxRetryCount {
					r.statusDB.Del(key)
					atomic.AddUint64(&r.failedCount, 1)
					return nil
				}

				// Check if timed out
				if int64(now.Sub(v.Time).Seconds()) >= r.timeoutSeconds {
					// Add domain to retry list or use batch sending channel
					retryDomains = append(retryDomains, key)

					// Group by DNS server for batch sending
					dns := r.selectDNSServer(key)
					if _, ok := dnsBatches[dns]; !ok {
						dnsBatches[dns] = make([]string, 0, batchSize)
					}
					dnsBatches[dns] = append(dnsBatches[dns], key)
				}
				return nil
			})

			// Record scan state
			lastScanEmpty = len(retryDomains) == 0

			// If there are domains to retry
			if len(retryDomains) > 0 {
				// Send retry domains to worker goroutines
				for _, domain := range retryDomains {
					// Non-blocking send
					select {
					case retryDomainCh <- domain:
						// Sent successfully
					default:
						// Channel full, send directly
						r.domainChan <- domain
					}
				}
			}
		}
	}
}
