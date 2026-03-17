# Developer Guide

> **This file supersedes the old (outdated) dev.md.**  
> Last updated: 2026-03 — reflects current architecture (v2, `chan string` domain input, sharded-lock statusDB, typed output interface).

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                           CLI / SDK                             │
│  cmd/ksubdomain/{enum,verify}   sdk/sdk.go                      │
└───────────────┬─────────────────────────────────────────────────┘
                │  options.Options
                ▼
┌──────────────────────────────────────────────────────────────┐
│                        pkg/runner                            │
│                                                              │
│  loadDomainsFromSource ──► domainChan (buf 50 000)           │
│                                │                             │
│  sendCycleWithContext ◄────────┘  ──► pcap WritePacketData   │
│       │ recvBackpressure flag                                │
│       ▼                                                      │
│  statusDB (sharded 64-bucket sync.Map)                       │
│       │                                                      │
│  retry() ─── every 200 ms ─── effectiveTimeoutSeconds()     │
│       │       (RTT EWMA, upper bound 10 s)                   │
│       ▼                                                      │
│  recvChanel ──► dnsChanel ──► handleResult ──► resultChan   │
│                                     │                        │
│                               outputter.Output               │
│                           (screen / file / SDK collector)    │
└──────────────────────────────────────────────────────────────┘
```

### Key goroutines (RunEnumeration)

| Goroutine | File | Role |
|---|---|---|
| `loadDomainsFromSource` | runner.go | Feed `domainChan` from `Options.Domain` |
| `sendCycleWithContext` | send.go | Consume `domainChan`, rate-limit, pcap send |
| `recvChanel` | recv.go | Capture DNS replies, parse, update statusDB |
| `retry` | retry.go | Scan statusDB every 200 ms, re-send timed-out domains |
| `monitorProgress` | runner.go | Update progress bar, detect completion |
| `handleResultWithContext` | result.go | Fan results to `outputter.Output` writers |

---

## Module path

```
github.com/boy-hack/ksubdomain/v2
```

---

## Quick-start (SDK — recommended)

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
        Bandwidth: "5m",
        Retry:     3,
    })

    // --- Blocking API ---
    results, err := scanner.Enum("example.com")
    if err != nil {
        if errors.Is(err, sdk.ErrPermissionDenied) {
            log.Fatal("run with sudo / grant CAP_NET_RAW")
        }
        log.Fatal(err)
    }
    for _, r := range results {
        fmt.Printf("%s [%s] %v\n", r.Domain, r.Type, r.Records)
    }

    // --- Stream API (real-time callback) ---
    ctx := context.Background()
    err = scanner.EnumStream(ctx, "example.com", func(r sdk.Result) {
        fmt.Printf("%s => %v\n", r.Domain, r.Records)
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### Config fields

| Field | Type | Default | Description |
|---|---|---|---|
| `Bandwidth` | `string` | `"5m"` | Network bandwidth limit (e.g., `"5m"`, `"100m"`) |
| `Retry` | `int` | `3` | Max retries per domain (-1 = infinite) |
| `Resolvers` | `[]string` | built-in | DNS resolver IPs |
| `Device` | `string` | auto | Network interface name |
| `Dictionary` | `string` | built-in | Subdomain wordlist file (enum mode) |
| `Predict` | `bool` | `false` | AI subdomain prediction |
| `WildcardFilter` | `string` | `"none"` | `"none"` / `"basic"` / `"advanced"` |
| `Silent` | `bool` | `false` | Suppress progress bar |
| `ExtraWriters` | `[]outputter.Output` | nil | Custom output sinks (see below) |

> **Timeout is not configurable.** The scanner uses a dynamic RTT-based
> timeout (TCP RFC 6298 EWMA, α=0.125, β=0.25) with an internal upper
> bound of 10 s and lower bound of 1 s.

---

## Advanced: runner.Options (low-level)

Use `options.Options` directly when you need full control:

```go
package main

import (
    "context"

    "github.com/boy-hack/ksubdomain/v2/pkg/core/options"
    "github.com/boy-hack/ksubdomain/v2/pkg/runner"
    "github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter"
    "github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter/output"
    processbar2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/processbar"
    "github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
)

func main() {
    screenWriter, _ := output.NewScreenOutput(false)

    domains := []string{"www.example.com", "api.example.com"}
    domainChan := make(chan string, len(domains))
    for _, d := range domains {
        domainChan <- d
    }
    close(domainChan)

    resolver := options.GetResolvers(nil)
    opt := &options.Options{
        Rate:       options.Band2Rate("3m"), // ≈ 37 500 pps
        Domain:     domainChan,
        Resolvers:  resolver,
        Silent:     false,
        Retry:      3,
        Method:     options.VerifyType,     // or options.EnumType
        Writer:     []outputter.Output{screenWriter},
        ProcessBar: &processbar2.ScreenProcess{},
        EtherInfo:  options.GetDeviceConfig(resolver),
    }
    opt.Check()

    r, err := runner.New(opt)
    if err != nil {
        gologger.Fatalf(err.Error())
    }
    ctx := context.Background()
    r.RunEnumeration(ctx)
    r.Close()
}
```

### Options fields

| Field | Type | Description |
|---|---|---|
| `Rate` | `int64` | Packets per second (use `Band2Rate("Nm")` to convert from bandwidth) |
| `Domain` | `chan string` | Input channel; close it after sending all domains |
| `Resolvers` | `[]string` | DNS resolver IPs |
| `Silent` | `bool` | Suppress log output |
| `Retry` | `int` | Max retries (-1 = infinite) |
| `Method` | `OptionMethod` | `VerifyType` or `EnumType` |
| `Writer` | `[]outputter.Output` | Output sinks; all receive every result |
| `ProcessBar` | `ProcessBar` | Progress bar implementation |
| `EtherInfo` | `*device.EtherTable` | Network interface config |
| `SpecialResolvers` | `map[string][]string` | Per-suffix DNS overrides |
| `WildcardFilterMode` | `string` | `"none"` / `"basic"` / `"advanced"` |
| `WildIps` | `[]string` | Known wildcard IPs to filter |
| `Predict` | `bool` | Enable AI subdomain prediction |

---

## Custom output sink

Implement `outputter.Output`:

```go
type outputter.Output interface {
    WriteDomainResult(result.Result) error
    Close() error
}
```

`result.Result`:

```go
type Result struct {
    Subdomain string
    Answers   []string // format: "CNAME foo.example.com", "1.2.3.4", "NS ns1…"
}
```

Inject via `Options.Writer` (runner API) or `Config.ExtraWriters` (SDK).

---

## Error handling

Named sentinel errors live in `pkg/core/errors` and are re-exported by the SDK:

```go
var (
    ErrPermissionDenied  // sudo required
    ErrDeviceNotFound    // interface name wrong
    ErrDeviceNotActive   // interface is down
    ErrPcapInit          // libpcap/npcap other failure
    ErrDomainChanNil     // forgot to set Options.Domain
)
```

Use `errors.Is` for type-safe checks:

```go
if errors.Is(err, sdk.ErrPermissionDenied) { ... }
```

---

## Backpressure

The sender automatically throttles when the receive pipeline is congested:

- **High-water mark**: `packetChan` ≥ 80 % full → sender sleeps 5 ms per batch
- **Low-water mark**: `packetChan` ≤ 50 % full → sender resumes normal speed

No manual configuration needed.

---

## Building

```bash
# All platforms
./build.sh

# Single binary (current platform)
go build -o ksubdomain ./cmd/ksubdomain

# With version injection
go build -ldflags "-X github.com/boy-hack/ksubdomain/v2/pkg/core/conf.Version=v2.x.y" \
    -o ksubdomain ./cmd/ksubdomain
```

---

## Testing

```bash
# Unit + integration (requires root/pcap)
sudo go test ./...

# Runner tests only
sudo go test ./pkg/runner/... -v

# SDK smoke test
sudo go test ./sdk/... -v
```

---

## Notes

- **One instance only.** Running multiple ksubdomain processes on the same
  interface at the same time will cause packet collisions. One instance
  already saturates available bandwidth.
- **Root / CAP_NET_RAW required.** Raw packet capture needs elevated
  privileges on Linux and macOS.
- **macOS BPF buffer.** Keep rate ≤ 50 000 pps on macOS to avoid
  `ENOBUFS` errors. Use `-b 5m` or lower.
- **WSL2.** Use `--interface eth0`; the default gateway detection may pick
  the wrong interface.
