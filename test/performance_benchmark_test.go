// +build performance

package test

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/options"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter"
	output2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter/output"
	processbar2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/processbar"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
)

// Benchmark100kDomains is a performance benchmark for 100k domains.
// Reference test from README:
// - Test environment: 4-core CPU, 5M bandwidth
// - Dictionary size: 100k domains
// - Target: ~30 seconds to complete
// - Success rate: > 95%
func Benchmark100kDomains(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping performance benchmark (use -tags=performance to run)")
	}

	// Create 100k domain dictionary
	dictFile := createBenchmarkDict(b, 100000)
	defer os.Remove(dictFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runBenchmark(b, dictFile, 100000)
	}
}

// Benchmark10kDomains is a quick test for 10k domains
func Benchmark10kDomains(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping performance benchmark")
	}

	dictFile := createBenchmarkDict(b, 10000)
	defer os.Remove(dictFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runBenchmark(b, dictFile, 10000)
	}
}

// Benchmark1kDomains is a basic test for 1k domains
func Benchmark1kDomains(b *testing.B) {
	dictFile := createBenchmarkDict(b, 1000)
	defer os.Remove(dictFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runBenchmark(b, dictFile, 1000)
	}
}

// createBenchmarkDict creates a test dictionary file
func createBenchmarkDict(b *testing.B, count int) string {
	b.Helper()

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("ksubdomain_bench_%d.txt", count))
	f, err := os.Create(tmpFile)
	if err != nil {
		b.Fatalf("Failed to create dictionary file: %v", err)
	}
	defer f.Close()

	writer := bufio.NewWriter(f)

	// Generate domain list.
	// Use some realistic domain patterns to improve test authenticity.
	baseDomains := []string{
		"example.com",
		"test.com",
		"demo.org",
		"sample.net",
	}

	prefixes := []string{
		"www", "mail", "ftp", "blog", "shop", "admin",
		"api", "dev", "test", "staging", "prod", "app",
		"web", "mobile", "cdn", "static", "img", "media",
	}

	for i := 0; i < count; i++ {
		var domain string

		if i < len(prefixes)*len(baseDomains) {
			// Use common prefixes
			prefix := prefixes[i%len(prefixes)]
			base := baseDomains[i/len(prefixes)%len(baseDomains)]
			domain = fmt.Sprintf("%s.%s", prefix, base)
		} else {
			// Generate random subdomains
			base := baseDomains[i%len(baseDomains)]
			domain = fmt.Sprintf("subdomain%d.%s", i, base)
		}

		_, err := writer.WriteString(domain + "\n")
		if err != nil {
			b.Fatalf("Failed to write dictionary: %v", err)
		}
	}

	err = writer.Flush()
	if err != nil {
		b.Fatalf("Failed to flush dictionary: %v", err)
	}

	b.Logf("Created dictionary: %s (%d domains)", tmpFile, count)
	return tmpFile
}

// runBenchmark runs a performance test
func runBenchmark(b *testing.B, dictFile string, expectedCount int) {
	b.Helper()

	// Open dictionary file
	file, err := os.Open(dictFile)
	if err != nil {
		b.Fatalf("Failed to open dictionary: %v", err)
	}
	defer file.Close()

	// Read all domains into a channel
	domainChan := make(chan string, 10000)
	go func() {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			domainChan <- scanner.Text()
		}
		close(domainChan)
	}()

	// Collect results
	results := &perfOutputter{
		results:      make([]result.Result, 0, expectedCount),
		startTime:    time.Now(),
		totalDomains: expectedCount,
	}

	// Configure scan parameters (reference README test config)
	opt := &options.Options{
		Rate:       options.Band2Rate("5m"), // 5M bandwidth
		Domain:     domainChan,
		Resolvers:  options.GetResolvers(nil),
		Silent:     true,
		TimeOut:    6,
		Retry:      3,
		Method:     options.VerifyType,
		Writer:     []outputter.Output{results},
		ProcessBar: &processbar2.FakeProcess{},
		EtherInfo:  options.GetDeviceConfig(options.GetResolvers(nil)),
	}

	// Create context (5-minute timeout, sufficient for 100k domains)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Record start time
	startTime := time.Now()

	// Run scan
	r, err := runner.New(opt)
	if err != nil {
		b.Fatalf("Failed to create runner: %v", err)
	}

	r.RunEnumeration(ctx)
	r.Close()

	// Calculate performance metrics
	elapsed := time.Since(startTime)
	successCount := len(results.results)
	successRate := float64(successCount) / float64(expectedCount) * 100
	domainsPerSecond := float64(expectedCount) / elapsed.Seconds()

	// Report performance metrics
	b.ReportMetric(elapsed.Seconds(), "total_seconds")
	b.ReportMetric(float64(successCount), "success_count")
	b.ReportMetric(successRate, "success_rate_%")
	b.ReportMetric(domainsPerSecond, "domains/sec")

	// Log output
	b.Logf("Performance test results:")
	b.Logf("  - Dictionary size: %d domains", expectedCount)
	b.Logf("  - Total elapsed:   %v", elapsed)
	b.Logf("  - Success count:   %d", successCount)
	b.Logf("  - Success rate:    %.2f%%", successRate)
	b.Logf("  - Speed:           %.0f domains/s", domainsPerSecond)

	// Performance baseline check (reference README: 100k domains ~30 seconds)
	if expectedCount == 100000 {
		// 100k domains should complete within 60 seconds (with tolerance)
		if elapsed.Seconds() > 60 {
			b.Logf("⚠️  Performance warning: 100k domains took %.1f seconds (target < 60s)", elapsed.Seconds())
		} else if elapsed.Seconds() <= 30 {
			b.Logf("✅ Excellent performance: 100k domains completed in %.1f seconds (meets README standard)", elapsed.Seconds())
		} else {
			b.Logf("✓  Good performance: 100k domains completed in %.1f seconds", elapsed.Seconds())
		}
	}
}

// perfOutputter is the performance test output handler
type perfOutputter struct {
	results      []result.Result
	mu           sync.Mutex
	startTime    time.Time
	totalDomains int
	lastReport   time.Time
}

func (p *perfOutputter) WriteDomainResult(r result.Result) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.results = append(p.results, r)

	// Report progress every 1000 results
	if len(p.results)%1000 == 0 {
		elapsed := time.Since(p.startTime)
		rate := float64(len(p.results)) / elapsed.Seconds()
		progress := float64(len(p.results)) / float64(p.totalDomains) * 100

		// Avoid frequent output
		if time.Since(p.lastReport) > time.Second {
			fmt.Printf("\rProgress: %d/%d (%.1f%%), Speed: %.0f domains/s, Elapsed: %v",
				len(p.results), p.totalDomains, progress, rate, elapsed.Round(time.Second))
			p.lastReport = time.Now()
		}
	}

	return nil
}

func (p *perfOutputter) Close() error {
	elapsed := time.Since(p.startTime)
	fmt.Printf("\nFinal result: %d/%d, Elapsed: %v\n",
		len(p.results), p.totalDomains, elapsed.Round(time.Millisecond))
	return nil
}
