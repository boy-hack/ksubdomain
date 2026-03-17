// Package sdk provides a simple Go SDK for ksubdomain
//
// # Basic usage (blocking, collect all results):
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
//	for _, r := range results {
//	    fmt.Printf("%s => %v\n", r.Domain, r.Records)
//	}
//
// # Stream usage (callback-based, real-time):
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	err := scanner.EnumStream(ctx, "example.com", func(r sdk.Result) {
//	    fmt.Printf("%s => %v\n", r.Domain, r.Records)
//	})
package sdk

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/options"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter"
	processbar2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/processbar"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
)

// Config holds scanner configuration.
// Timeout is no longer configurable: the scanner uses a dynamic RTT-based
// timeout with a hardcoded upper bound of 10 s.
type Config struct {
	// Bandwidth limit (e.g., "5m", "10m", "100m")
	Bandwidth string

	// Retry count for timed-out queries (-1 for infinite)
	Retry int

	// DNS resolvers; nil means built-in defaults
	Resolvers []string

	// Network adapter name; empty means auto-detect
	Device string

	// Dictionary file path (for Enum mode); empty means built-in list
	Dictionary string

	// Enable AI-powered subdomain prediction
	Predict bool

	// Wildcard filter mode: "none" (default), "basic", "advanced"
	WildcardFilter string

	// Silent disables the progress bar
	Silent bool

	// ExtraWriters allows callers to inject custom output sinks.
	// Each writer's WriteDomainResult is called for every resolved result,
	// in addition to the SDK's internal collection/stream logic.
	// Writers must implement outputter.Output (WriteDomainResult + Close).
	// Close() on each writer is called after the scan completes.
	ExtraWriters []outputter.Output
}

// DefaultConfig is a ready-to-use Config with sensible defaults.
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

// Result represents a single resolved subdomain.
type Result struct {
	Domain  string   // Resolved subdomain
	Type    string   // DNS record type (A, CNAME, NS, PTR, …)
	Records []string // Record values
}

// Scanner performs subdomain enumeration and verification.
type Scanner struct {
	config *Config
}

// NewScanner creates a Scanner.  If config is nil, DefaultConfig is used.
func NewScanner(config *Config) *Scanner {
	if config == nil {
		config = DefaultConfig
	}
	if config.Bandwidth == "" {
		config.Bandwidth = "5m"
	}
	if config.Retry == 0 {
		config.Retry = 3
	}
	return &Scanner{config: config}
}

// ---------------------------------------------------------------------------
// Blocking API
// ---------------------------------------------------------------------------

// Enum enumerates subdomains for domain, returning all results when done.
func (s *Scanner) Enum(domain string) ([]Result, error) {
	return s.EnumWithContext(context.Background(), domain)
}

// EnumWithContext is like Enum but respects ctx for cancellation.
func (s *Scanner) EnumWithContext(ctx context.Context, domain string) ([]Result, error) {
	dictChan, err := s.buildDictChan(domain)
	if err != nil {
		return nil, err
	}
	return s.scanCollect(ctx, dictChan, options.EnumType)
}

// Verify verifies each domain in domains, returning those that resolve.
func (s *Scanner) Verify(domains []string) ([]Result, error) {
	return s.VerifyWithContext(context.Background(), domains)
}

// VerifyWithContext is like Verify but respects ctx.
func (s *Scanner) VerifyWithContext(ctx context.Context, domains []string) ([]Result, error) {
	ch := make(chan string, len(domains))
	for _, d := range domains {
		ch <- d
	}
	close(ch)
	return s.scanCollect(ctx, ch, options.VerifyType)
}

// ---------------------------------------------------------------------------
// Stream API
// ---------------------------------------------------------------------------

// EnumStream enumerates subdomains for domain and calls callback for each
// result as it arrives.  It blocks until scanning is complete or ctx is
// cancelled.
//
// callback is called from multiple goroutines; implementations must be
// goroutine-safe (e.g., protect shared state with a mutex).
//
// Example:
//
//	err := scanner.EnumStream(ctx, "example.com", func(r sdk.Result) {
//	    fmt.Println(r.Domain)
//	})
func (s *Scanner) EnumStream(ctx context.Context, domain string, callback func(Result)) error {
	dictChan, err := s.buildDictChan(domain)
	if err != nil {
		return err
	}
	return s.scanStream(ctx, dictChan, options.EnumType, callback)
}

// VerifyStream verifies domains and calls callback for each resolved result.
// It blocks until scanning is complete or ctx is cancelled.
func (s *Scanner) VerifyStream(ctx context.Context, domains []string, callback func(Result)) error {
	ch := make(chan string, len(domains))
	for _, d := range domains {
		ch <- d
	}
	close(ch)
	return s.scanStream(ctx, ch, options.VerifyType, callback)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// buildDictChan returns a channel that emits fully-qualified domain names
// to enumerate under domain.
func (s *Scanner) buildDictChan(domain string) (chan string, error) {
	if s.config.Dictionary != "" {
		// TODO: load from file
		return nil, fmt.Errorf("dictionary file loading not yet implemented in SDK")
	}
	ch := make(chan string, 1000)
	go func() {
		defer close(ch)
		// Built-in minimal list; full implementation uses core.GetDefaultSubdomainData()
		subdomains := []string{"www", "mail", "ftp", "blog", "api", "dev", "test"}
		for _, sub := range subdomains {
			ch <- sub + "." + domain
		}
	}()
	return ch, nil
}

// buildOptions constructs runner.Options from the scanner config.
// primaryWriter is the SDK-internal writer (resultCollector or streamCollector).
// Any ExtraWriters from config are appended after it.
func (s *Scanner) buildOptions(domainChan chan string, method string, primaryWriter outputter.Output) *options.Options {
	resolvers := options.GetResolvers(s.config.Resolvers)

	writers := make([]outputter.Output, 0, 1+len(s.config.ExtraWriters))
	writers = append(writers, primaryWriter)
	writers = append(writers, s.config.ExtraWriters...)

	opt := &options.Options{
		Rate:               options.Band2Rate(s.config.Bandwidth),
		Domain:             domainChan,
		Resolvers:          resolvers,
		Silent:             s.config.Silent,
		Retry:              s.config.Retry,
		Method:             method,
		Writer:             writers,
		ProcessBar:         &processbar2.FakeProcess{},
		EtherInfo:          options.GetDeviceConfig(resolvers),
		WildcardFilterMode: s.config.WildcardFilter,
		Predict:            s.config.Predict,
	}
	if s.config.Device != "" {
		opt.EtherInfo.Device = s.config.Device
	}
	opt.Check()
	return opt
}

// scanCollect runs the scan and returns collected results.
func (s *Scanner) scanCollect(ctx context.Context, domainChan chan string, method string) ([]Result, error) {
	collector := &resultCollector{results: make([]Result, 0)}
	opt := s.buildOptions(domainChan, method, collector)
	r, err := runner.New(opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}
	r.RunEnumeration(ctx)
	r.Close()
	return collector.results, nil
}

// scanStream runs the scan, calling callback for each result in real-time.
func (s *Scanner) scanStream(ctx context.Context, domainChan chan string, method string, callback func(Result)) error {
	streamer := &streamCollector{callback: callback}
	opt := s.buildOptions(domainChan, method, streamer)
	r, err := runner.New(opt)
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}
	r.RunEnumeration(ctx)
	r.Close()
	return nil
}

// ---------------------------------------------------------------------------
// outputter.Output implementations
// ---------------------------------------------------------------------------

// resultCollector accumulates results for the blocking API.
type resultCollector struct {
	results []Result
	mu      sync.Mutex
}

func (rc *resultCollector) WriteDomainResult(r result.Result) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.results = append(rc.results, parseResult(r))
	return nil
}

func (rc *resultCollector) Close() error { return nil }

// streamCollector forwards each result to a user-supplied callback.
// WriteDomainResult may be called from multiple goroutines; the callback
// itself is invoked under no lock — callers are responsible for thread safety.
type streamCollector struct {
	callback func(Result)
}

func (sc *streamCollector) WriteDomainResult(r result.Result) error {
	sc.callback(parseResult(r))
	return nil
}

func (sc *streamCollector) Close() error { return nil }

// parseResult converts an internal result.Result to the public Result type.
func parseResult(r result.Result) Result {
	recordType := "A"
	records := make([]string, 0, len(r.Answers))

	for _, answer := range r.Answers {
		switch {
		case strings.HasPrefix(answer, "CNAME "):
			recordType = "CNAME"
			records = append(records, answer[6:])
		case strings.HasPrefix(answer, "NS "):
			recordType = "NS"
			records = append(records, answer[3:])
		case strings.HasPrefix(answer, "PTR "):
			recordType = "PTR"
			records = append(records, answer[4:])
		default:
			records = append(records, answer)
		}
	}
	if len(records) == 0 {
		records = r.Answers
	}
	return Result{
		Domain:  r.Subdomain,
		Type:    recordType,
		Records: records,
	}
}
