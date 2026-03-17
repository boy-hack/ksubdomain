# SDK API Reference

Import path: `github.com/boy-hack/ksubdomain/v2/sdk`

---

## Types

### Config

```go
type Config struct {
    Bandwidth      string             // bandwidth cap, e.g. "5m" (default "5m")
    Retry          int                // per-domain retry count, -1 = infinite (default 3)
    Resolvers      []string           // DNS resolver IPs; nil = built-in defaults
    Device         string             // network interface; "" = auto-detect
    Dictionary     string             // wordlist file for Enum; "" = built-in list
    Predict        bool               // enable AI subdomain prediction
    WildcardFilter string             // "none" | "basic" | "advanced" (default "none")
    Silent         bool               // suppress progress bar
    ExtraWriters   []outputter.Output // additional output sinks (see Custom sinks)
}
```

> **Note**: Timeout is not configurable. The scanner uses a dynamic RTT-based
> timeout with a hardcoded upper bound of 10 s and lower bound of 1 s
> (TCP RFC 6298 EWMA, α=0.125, β=0.25).

### DefaultConfig

```go
var DefaultConfig = &Config{
    Bandwidth:      "5m",
    Retry:          3,
    WildcardFilter: "none",
}
```

### Result

```go
type Result struct {
    Domain  string   // resolved subdomain
    Type    string   // "A", "CNAME", "NS", "PTR", "TXT", "AAAA"
    Records []string // record values (IPs, target names, text, …)
}
```

---

## Functions

### NewScanner

```go
func NewScanner(config *Config) *Scanner
```

Creates a new Scanner.  If `config` is nil, `DefaultConfig` is used.
Applies defaults for zero-value fields (`Bandwidth`, `Retry`).

---

## Scanner methods

### Enum

```go
func (s *Scanner) Enum(domain string) ([]Result, error)
```

Enumerates subdomains of `domain` using the configured wordlist.
Blocks until the scan completes and returns all results.

### EnumWithContext

```go
func (s *Scanner) EnumWithContext(ctx context.Context, domain string) ([]Result, error)
```

Like `Enum`, but honours `ctx` for cancellation.

### EnumStream

```go
func (s *Scanner) EnumStream(ctx context.Context, domain string, callback func(Result)) error
```

Enumerates subdomains and calls `callback` for **each result as it arrives**,
without waiting for the scan to complete.  Blocks until done or `ctx` is
cancelled.

`callback` may be called from multiple goroutines concurrently.
Implementations must be goroutine-safe.

```go
var mu sync.Mutex
var results []sdk.Result

err := scanner.EnumStream(ctx, "example.com", func(r sdk.Result) {
    mu.Lock()
    results = append(results, r)
    mu.Unlock()
    fmt.Println(r.Domain)
})
```

### Verify

```go
func (s *Scanner) Verify(domains []string) ([]Result, error)
```

Verifies each domain in `domains`, returning those that resolve.
Blocks until complete.

### VerifyWithContext

```go
func (s *Scanner) VerifyWithContext(ctx context.Context, domains []string) ([]Result, error)
```

Like `Verify`, but honours `ctx`.

### VerifyStream

```go
func (s *Scanner) VerifyStream(ctx context.Context, domains []string, callback func(Result)) error
```

Verifies domains and calls `callback` for each resolved result in real-time.
Blocks until done or `ctx` is cancelled.

---

## Error handling

Sentinel errors are exported from the `sdk` package.  Use `errors.Is`:

```go
_, err := scanner.Enum("example.com")
switch {
case errors.Is(err, sdk.ErrPermissionDenied):
    log.Fatal("run with sudo or grant CAP_NET_RAW")
case errors.Is(err, sdk.ErrDeviceNotFound):
    log.Fatal("wrong interface name — check --eth flag or Config.Device")
case errors.Is(err, sdk.ErrDeviceNotActive):
    log.Fatal("interface is down — run: sudo ip link set <iface> up")
case err != nil:
    log.Fatal(err)
}
```

| Sentinel | Meaning |
|---|---|
| `ErrPermissionDenied` | Process lacks `CAP_NET_RAW` / not running as root |
| `ErrDeviceNotFound` | Network interface name does not exist |
| `ErrDeviceNotActive` | Interface exists but is not up |
| `ErrPcapInit` | libpcap/npcap initialisation failed (other reason) |
| `ErrDomainChanNil` | Internal: domain channel was nil (should not occur via SDK) |

---

## Custom output sinks

Implement `outputter.Output` from `github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter`:

```go
type Output interface {
    WriteDomainResult(result.Result) error
    Close() error
}
```

`result.Result`:

```go
// github.com/boy-hack/ksubdomain/v2/pkg/runner/result
type Result struct {
    Subdomain string   // e.g. "www.example.com"
    Answers   []string // raw answer strings, e.g. "CNAME foo.example.com", "1.2.3.4"
}
```

Example — write resolved domains to a channel for external processing:

```go
type chanWriter struct {
    ch chan<- string
}

func (w *chanWriter) WriteDomainResult(r result.Result) error {
    w.ch <- r.Subdomain
    return nil
}
func (w *chanWriter) Close() error { return nil }

// Usage:
ch := make(chan string, 1000)
scanner := sdk.NewScanner(&sdk.Config{
    ExtraWriters: []outputter.Output{&chanWriter{ch: ch}},
})
go scanner.Enum("example.com")
for domain := range ch {
    fmt.Println(domain)
}
```

---

## Complete example

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"

    "github.com/boy-hack/ksubdomain/v2/sdk"
)

func main() {
    scanner := sdk.NewScanner(&sdk.Config{
        Bandwidth:      "10m",
        Retry:          3,
        WildcardFilter: "basic",
        Silent:         true,
    })

    ctx, cancel := context.WithTimeout(context.Background(), 5*60*1e9) // 5 min
    defer cancel()

    err := scanner.EnumStream(ctx, "example.com", func(r sdk.Result) {
        fmt.Printf("[%s] %s => %v\n", r.Type, r.Domain, r.Records)
    })
    if err != nil {
        if errors.Is(err, sdk.ErrPermissionDenied) {
            log.Fatal("need sudo")
        }
        log.Fatal(err)
    }
}
```
