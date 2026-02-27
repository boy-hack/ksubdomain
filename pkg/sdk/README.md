# KSubdomain Go SDK

Simple and powerful Go SDK for integrating ksubdomain into your applications.

## 📦 Installation

```bash
go get github.com/boy-hack/ksubdomain/v2/sdk
```

## 🚀 Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "log"

    "github.com/boy-hack/ksubdomain/v2/sdk"
)

func main() {
    // Create scanner with default config
    scanner := sdk.NewScanner(sdk.DefaultConfig)

    // Enumerate subdomains
    results, err := scanner.Enum("example.com")
    if err != nil {
        log.Fatal(err)
    }

    // Process results
    for _, result := range results {
        fmt.Printf("%s => %v\n", result.Domain, result.Records)
    }
}
```

### Custom Configuration

```go
scanner := sdk.NewScanner(&sdk.Config{
    Bandwidth:      "10m",        // 10M bandwidth
    Retry:          5,            // Retry 5 times
    Timeout:        10,           // 10 seconds timeout
    Resolvers:      []string{"8.8.8.8", "1.1.1.1"},
    Predict:        true,         // Enable prediction
    WildcardFilter: "advanced",   // Advanced wildcard filtering
    Silent:         true,         // Silent mode
})

results, err := scanner.Enum("example.com")
```

### Verify Mode

```go
domains := []string{
    "www.example.com",
    "mail.example.com",
    "api.example.com",
}

results, err := scanner.Verify(domains)
if err != nil {
    log.Fatal(err)
}

for _, result := range results {
    fmt.Printf("✓ %s is alive\n", result.Domain)
}
```

### With Context (Timeout/Cancellation)

```go
import "context"

// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results, err := scanner.EnumWithContext(ctx, "example.com")

// With cancellation
ctx, cancel := context.WithCancel(context.Background())

go func() {
    time.Sleep(10 * time.Second)
    cancel() // Stop scanning after 10 seconds
}()

results, err := scanner.EnumWithContext(ctx, "example.com")
```

## 📚 API Reference

### Config

Configuration for the scanner.

```go
type Config struct {
    Bandwidth      string   // Bandwidth (e.g., "5m", "10m", "100m")
    Retry          int      // Retry count (-1 for infinite)
    Timeout        int      // Timeout in seconds
    Resolvers      []string // DNS resolvers (nil for default)
    Device         string   // Network adapter (empty for auto-detect)
    Dictionary     string   // Dictionary file path
    Predict        bool     // Enable prediction mode
    WildcardFilter string   // Wildcard filter: "none", "basic", "advanced"
    Silent         bool     // Silent mode (no progress output)
}
```

**DefaultConfig:**
```go
var DefaultConfig = &Config{
    Bandwidth:      "5m",
    Retry:          3,
    Timeout:        6,
    Resolvers:      nil,
    Device:         "",
    Dictionary:     "",
    Predict:        false,
    WildcardFilter: "none",
    Silent:         false,
}
```

### Scanner

Main scanner interface.

#### NewScanner

```go
func NewScanner(config *Config) *Scanner
```

Creates a new scanner with given configuration. If `config` is nil, uses `DefaultConfig`.

#### Enum

```go
func (s *Scanner) Enum(domain string) ([]Result, error)
```

Enumerates subdomains for the given domain.

#### EnumWithContext

```go
func (s *Scanner) EnumWithContext(ctx context.Context, domain string) ([]Result, error)
```

Enumerates subdomains with context support (timeout, cancellation).

#### Verify

```go
func (s *Scanner) Verify(domains []string) ([]Result, error)
```

Verifies a list of domains.

#### VerifyWithContext

```go
func (s *Scanner) VerifyWithContext(ctx context.Context, domains []string) ([]Result, error)
```

Verifies domains with context support.

### Result

Scan result structure.

```go
type Result struct {
    Domain  string   // Subdomain
    Type    string   // Record type (A, CNAME, NS, PTR, etc.)
    Records []string // Record values
}
```

## 📖 Examples

### Example 1: Simple Enumeration

```go
package main

import (
    "fmt"
    "log"
    "github.com/boy-hack/ksubdomain/v2/sdk"
)

func main() {
    scanner := sdk.NewScanner(nil) // Use default config
    
    results, err := scanner.Enum("example.com")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d subdomains:\n", len(results))
    for _, r := range results {
        fmt.Printf("  %s (%s)\n", r.Domain, r.Type)
    }
}
```

### Example 2: Batch Verification

```go
package main

import (
    "bufio"
    "fmt"
    "log"
    "os"
    
    "github.com/boy-hack/ksubdomain/v2/sdk"
)

func main() {
    // Read domains from file
    file, _ := os.Open("domains.txt")
    defer file.Close()
    
    var domains []string
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        domains = append(domains, scanner.Text())
    }
    
    // Verify
    ksubScanner := sdk.NewScanner(&sdk.Config{
        Bandwidth: "10m",
        Retry:     5,
    })
    
    results, err := ksubScanner.Verify(domains)
    if err != nil {
        log.Fatal(err)
    }
    
    // Save results
    outFile, _ := os.Create("alive.txt")
    defer outFile.Close()
    
    for _, r := range results {
        fmt.Fprintf(outFile, "%s => %s\n", r.Domain, r.Records[0])
    }
}
```

### Example 3: High-Speed Enumeration

```go
package main

import (
    "fmt"
    "log"
    "github.com/boy-hack/ksubdomain/v2/sdk"
)

func main() {
    // High-speed configuration
    scanner := sdk.NewScanner(&sdk.Config{
        Bandwidth:      "20m",       // High bandwidth
        Retry:          1,           // Fast mode: fewer retries
        Timeout:        3,           // Short timeout
        Predict:        true,        // Enable prediction
        WildcardFilter: "advanced",  // Advanced filtering
        Silent:         true,        // No progress output
    })
    
    results, err := scanner.Enum("example.com")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("High-speed scan found %d subdomains\n", len(results))
}
```

### Example 4: With Context and Timeout

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/boy-hack/ksubdomain/v2/sdk"
)

func main() {
    scanner := sdk.NewScanner(sdk.DefaultConfig)
    
    // Set 30 seconds timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    results, err := scanner.EnumWithContext(ctx, "example.com")
    if err != nil {
        if err == context.DeadlineExceeded {
            fmt.Println("Scan timeout, partial results:")
        } else {
            log.Fatal(err)
        }
    }
    
    for _, r := range results {
        fmt.Printf("%s => %v\n", r.Domain, r.Records)
    }
}
```

### Example 5: Integration with Other Tools

```go
package main

import (
    "fmt"
    "os/exec"
    "strings"
    
    "github.com/boy-hack/ksubdomain/v2/sdk"
)

func main() {
    scanner := sdk.NewScanner(nil)
    
    // 1. Enum subdomains
    results, _ := scanner.Enum("example.com")
    
    // 2. Extract domain names
    var domains []string
    for _, r := range results {
        domains = append(domains, r.Domain)
    }
    
    // 3. Pipe to httpx for HTTP probing
    cmd := exec.Command("httpx", "-silent")
    cmd.Stdin = strings.NewReader(strings.Join(domains, "\n"))
    
    output, err := cmd.Output()
    if err == nil {
        fmt.Printf("Live HTTP services:\n%s", output)
    }
}
```

## 🎯 Use Cases

### Web Application Scanning

```go
// Discover all subdomains, then scan for vulnerabilities
results, _ := scanner.Enum("target.com")
for _, r := range results {
    // Run nuclei, sqlmap, etc. on each subdomain
    runVulnScan(r.Domain)
}
```

### Asset Discovery

```go
// Monitor subdomain changes
oldResults := loadPreviousResults()
newResults, _ := scanner.Enum("company.com")

for _, r := range newResults {
    if !contains(oldResults, r.Domain) {
        alert(fmt.Sprintf("New subdomain found: %s", r.Domain))
    }
}
```

### Automated Reconnaissance

```go
// Periodic scanning with cron
func scanTask() {
    scanner := sdk.NewScanner(sdk.DefaultConfig)
    results, _ := scanner.Enum("target.com")
    
    saveToDatabase(results)
    generateReport(results)
    sendNotification(results)
}
```

## 🔧 Advanced Usage

### Custom DNS Resolvers

```go
scanner := sdk.NewScanner(&sdk.Config{
    Resolvers: []string{
        "8.8.8.8",
        "8.8.4.4",
        "1.1.1.1",
        "1.0.0.1",
    },
})
```

### Specify Network Adapter

```go
scanner := sdk.NewScanner(&sdk.Config{
    Device: "eth0", // or "en0" on macOS
})
```

### Enable Prediction Mode

```go
scanner := sdk.NewScanner(&sdk.Config{
    Predict: true, // AI-powered subdomain prediction
})
```

## 🐛 Error Handling

```go
results, err := scanner.Enum("example.com")
if err != nil {
    switch {
    case strings.Contains(err.Error(), "permission denied"):
        log.Fatal("Need root permission. Run with sudo.")
    
    case strings.Contains(err.Error(), "device not found"):
        log.Fatal("Network adapter not found. Try --device eth0")
    
    case strings.Contains(err.Error(), "network"):
        log.Fatal("Network error. Check your connection.")
    
    default:
        log.Fatal(err)
    }
}
```

## 📝 Requirements

- Go 1.23+
- libpcap (automatically handled in most cases)
- Root/Administrator permission (for network adapter access)

## 🔗 Links

- [GitHub Repository](https://github.com/boy-hack/ksubdomain)
- [Documentation](https://github.com/boy-hack/ksubdomain/tree/main/docs)
- [Issues](https://github.com/boy-hack/ksubdomain/issues)

## 📄 License

MIT License. See [LICENSE](../LICENSE) for details.

---

**Happy Scanning! 🚀**
