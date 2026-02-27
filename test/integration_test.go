// +build integration

package test

import (
	"context"
	"testing"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/options"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter"
	output2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter/output"
	processbar2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/processbar"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
	"github.com/stretchr/testify/assert"
)

// TestBasicVerification is a basic verification test
func TestBasicVerification(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Known existing domains
	domains := []string{
		"www.baidu.com",
		"www.google.com",
		"dns.google",
	}

	domainChan := make(chan string, len(domains))
	for _, domain := range domains {
		domainChan <- domain
	}
	close(domainChan)

	// Collect results
	results := &testOutputter{results: make([]result.Result, 0)}

	opt := &options.Options{
		Rate:      1000,
		Domain:    domainChan,
		Resolvers: options.GetResolvers(nil),
		Silent:    true,
		TimeOut:   10,
		Retry:     3,
		Method:    options.VerifyType,
		Writer:    []outputter.Output{results},
		ProcessBar: &processbar2.FakeProcess{},
		EtherInfo: options.GetDeviceConfig(options.GetResolvers(nil)),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	r, err := runner.New(opt)
	assert.NoError(t, err)

	r.RunEnumeration(ctx)
	r.Close()

	// Verify results
	assert.Greater(t, len(results.results), 0, "Should find at least one domain")

	for _, res := range results.results {
		assert.NotEmpty(t, res.Subdomain, "Domain should not be empty")
		assert.Greater(t, len(res.Answers), 0, "Should have at least one answer")
		t.Logf("Found: %s => %v", res.Subdomain, res.Answers)
	}
}

// TestCNAMEParsing tests CNAME record parsing correctness
func TestCNAMEParsing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Domains known to have CNAME records
	domains := []string{
		"www.github.com", // usually has CNAME
		"www.baidu.com",  // may have CNAME
	}

	domainChan := make(chan string, len(domains))
	for _, domain := range domains {
		domainChan <- domain
	}
	close(domainChan)

	results := &testOutputter{results: make([]result.Result, 0)}

	opt := &options.Options{
		Rate:      1000,
		Domain:    domainChan,
		Resolvers: options.GetResolvers(nil),
		Silent:    true,
		TimeOut:   10,
		Retry:     3,
		Method:    options.VerifyType,
		Writer:    []outputter.Output{results},
		ProcessBar: &processbar2.FakeProcess{},
		EtherInfo: options.GetDeviceConfig(options.GetResolvers(nil)),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	r, err := runner.New(opt)
	assert.NoError(t, err)

	r.RunEnumeration(ctx)
	r.Close()

	// Check CNAME record format
	for _, res := range results.results {
		for _, answer := range res.Answers {
			// Should not have incorrect string concatenation like "comcom"
			assert.NotContains(t, answer, "comcom", "Should not have incorrect string concatenation")
			assert.NotContains(t, answer, "\x00", "Should not contain null characters")

			t.Logf("%s => %s", res.Subdomain, answer)
		}
	}
}

// TestHighSpeed tests high-speed scanning
func TestHighSpeed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Generate 100 test domains
	domains := make([]string, 100)
	for i := 0; i < 100; i++ {
		if i%2 == 0 {
			domains[i] = "www.baidu.com" // exists
		} else {
			domains[i] = "nonexistent12345.baidu.com" // does not exist
		}
	}

	domainChan := make(chan string, len(domains))
	for _, domain := range domains {
		domainChan <- domain
	}
	close(domainChan)

	results := &testOutputter{results: make([]result.Result, 0)}

	opt := &options.Options{
		Rate:      10000, // high speed
		Domain:    domainChan,
		Resolvers: options.GetResolvers(nil),
		Silent:    true,
		TimeOut:   6,
		Retry:     3,
		Method:    options.VerifyType,
		Writer:    []outputter.Output{results},
		ProcessBar: &processbar2.FakeProcess{},
		EtherInfo: options.GetDeviceConfig(options.GetResolvers(nil)),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	r, err := runner.New(opt)
	assert.NoError(t, err)

	r.RunEnumeration(ctx)
	r.Close()

	// Should find approximately 50 (existing domains)
	assert.Greater(t, len(results.results), 40, "High-speed mode should find most existing domains")
	assert.Less(t, len(results.results), 60, "Should not have too many false positives")

	t.Logf("High-speed scan results: found %d/%d domains", len(results.results), len(domains))
}

// TestRetryMechanism tests the retry mechanism
func TestRetryMechanism(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	domains := []string{"www.example.com"}

	domainChan := make(chan string, len(domains))
	for _, domain := range domains {
		domainChan <- domain
	}
	close(domainChan)

	results := &testOutputter{results: make([]result.Result, 0)}

	// Test different retry counts
	retryCounts := []int{1, 3, 5}

	for _, retryCount := range retryCounts {
		opt := &options.Options{
			Rate:      1000,
			Domain:    domainChan,
			Resolvers: options.GetResolvers(nil),
			Silent:    true,
			TimeOut:   3,
			Retry:     retryCount,
			Method:    options.VerifyType,
			Writer:    []outputter.Output{results},
			ProcessBar: &processbar2.FakeProcess{},
			EtherInfo: options.GetDeviceConfig(options.GetResolvers(nil)),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		r, err := runner.New(opt)
		assert.NoError(t, err)

		startTime := time.Now()
		r.RunEnumeration(ctx)
		elapsed := time.Since(startTime)
		r.Close()

		cancel()

		t.Logf("Retry count %d: elapsed %v, results %d",
			retryCount, elapsed, len(results.results))
	}
}

// TestWildcardDetection tests wildcard DNS detection
func TestWildcardDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Test a known wildcard domain
	// Note: requires an actual wildcard domain
	domain := "baidu.com" // example

	isWild, ips := runner.IsWildCard(domain)

	if isWild {
		t.Logf("Wildcard detected: %s, IPs: %v", domain, ips)
		assert.Greater(t, len(ips), 0, "Wildcard should return an IP list")
	} else {
		t.Logf("No wildcard detected: %s", domain)
	}
}

// testOutputter is a test output handler
type testOutputter struct {
	results []result.Result
	mu      sync.Mutex
}

func (t *testOutputter) WriteDomainResult(r result.Result) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.results = append(t.results, r)
	return nil
}

func (t *testOutputter) Close() error {
	return nil
}
