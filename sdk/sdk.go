// Package sdk provides a simple Go SDK for ksubdomain
// 
// Example:
//
//	scanner := sdk.NewScanner(&sdk.Config{
//	    Bandwidth: "5m",
//	    Retry:     3,
//	})
//	
//	results, err := scanner.Enum("example.com")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	
//	for _, result := range results {
//	    fmt.Printf("%s => %s\n", result.Domain, result.IP)
//	}
package sdk

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/options"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter"
	processbar2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/processbar"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
)

// Config Scanner configuration
type Config struct {
	// Bandwidth downstream speed (e.g., "5m", "10m", "100m")
	Bandwidth string

	// Retry count (-1 for infinite)
	Retry int

	// DNS resolvers (nil for default)
	Resolvers []string

	// Network adapter name (empty for auto-detect)
	Device string

	// Dictionary file path (for enum mode)
	Dictionary string

	// Enable prediction mode
	Predict bool

	// Wildcard filter mode: "none", "basic", "advanced"
	WildcardFilter string

	// Silent mode (no progress bar)
	Silent bool
}

// DefaultConfig returns default configuration
var DefaultConfig = &Config{
	Bandwidth:      "5m",
	Retry:          3,
	Resolvers:      nil,
	Device:         "",
	Dictionary:     "",
	Predict:        false,
	WildcardFilter: "none",
	Silent:         false,
}

// Result scan result
type Result struct {
	Domain  string   // Subdomain
	Type    string   // Record type (A, CNAME, NS, etc.)
	Records []string // Record values
}

// Scanner subdomain scanner
type Scanner struct {
	config *Config
}

// NewScanner creates a new scanner with given config
func NewScanner(config *Config) *Scanner {
	if config == nil {
		config = DefaultConfig
	}

	// Apply defaults
	if config.Bandwidth == "" {
		config.Bandwidth = "5m"
	}
	if config.Retry == 0 {
		config.Retry = 3
	}

	return &Scanner{
		config: config,
	}
}

// Enum enumerates subdomains for given domain
func (s *Scanner) Enum(domain string) ([]Result, error) {
	return s.EnumWithContext(context.Background(), domain)
}

// EnumWithContext enumerates subdomains with context support
func (s *Scanner) EnumWithContext(ctx context.Context, domain string) ([]Result, error) {
	// Load dictionary
	var dictChan chan string
	if s.config.Dictionary != "" {
		// TODO: Load from file
		return nil, fmt.Errorf("dictionary file not yet implemented in SDK")
	} else {
		// Use built-in default dictionary
		dictChan = make(chan string, 1000)
		go func() {
			defer close(dictChan)
			// Load built-in subdomain list
			// This will be implemented using core.GetDefaultSubdomainData()
			subdomains := []string{"www", "mail", "ftp", "blog", "api", "dev", "test"}
			for _, sub := range subdomains {
				dictChan <- sub + "." + domain
			}
		}()
	}

	return s.scan(ctx, dictChan, options.EnumType)
}

// Verify verifies a list of domains
func (s *Scanner) Verify(domains []string) ([]Result, error) {
	return s.VerifyWithContext(context.Background(), domains)
}

// VerifyWithContext verifies domains with context support
func (s *Scanner) VerifyWithContext(ctx context.Context, domains []string) ([]Result, error) {
	domainChan := make(chan string, len(domains))
	for _, domain := range domains {
		domainChan <- domain
	}
	close(domainChan)

	return s.scan(ctx, domainChan, options.VerifyType)
}

// scan internal scan implementation
func (s *Scanner) scan(ctx context.Context, domainChan chan string, method string) ([]Result, error) {
	// Collect results
	collector := &resultCollector{
		results: make([]Result, 0),
	}

	// Get resolvers
	resolvers := options.GetResolvers(s.config.Resolvers)

	// Build options
	opt := &options.Options{
		Rate:               options.Band2Rate(s.config.Bandwidth),
		Domain:             domainChan,
		Resolvers:          resolvers,
		Silent:             s.config.Silent,
		Retry:              s.config.Retry,
		Method:             method,
		Writer:             []outputter.Output{collector},
		ProcessBar:         &processbar2.FakeProcess{},
		EtherInfo:          options.GetDeviceConfig(resolvers),
		WildcardFilterMode: s.config.WildcardFilter,
		Predict:            s.config.Predict,
	}

	// Override device if specified
	if s.config.Device != "" {
		opt.EtherInfo.Device = s.config.Device
	}

	opt.Check()

	// Create runner
	r, err := runner.New(opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	// Run enumeration
	r.RunEnumeration(ctx)
	r.Close()

	return collector.results, nil
}

// resultCollector collects scan results
type resultCollector struct {
	results []Result
	mu      sync.Mutex
}

func (rc *resultCollector) WriteDomainResult(r result.Result) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Parse record type
	recordType := "A"
	records := make([]string, 0, len(r.Answers))

	for _, answer := range r.Answers {
		if strings.HasPrefix(answer, "CNAME ") {
			recordType = "CNAME"
			records = append(records, answer[6:])
		} else if strings.HasPrefix(answer, "NS ") {
			recordType = "NS"
			records = append(records, answer[3:])
		} else if strings.HasPrefix(answer, "PTR ") {
			recordType = "PTR"
			records = append(records, answer[4:])
		} else {
			records = append(records, answer)
		}
	}

	if len(records) == 0 {
		records = r.Answers
	}

	rc.results = append(rc.results, Result{
		Domain:  r.Subdomain,
		Type:    recordType,
		Records: records,
	})

	return nil
}

func (rc *resultCollector) Close() error {
	return nil
}
