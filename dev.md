[Outdated — pending rewrite]

A simple usage example.
Note: Do not start multiple instances of ksubdomain. A single instance is enough to achieve maximum performance.

```go
package main

import (
	"context"
	"github.com/boy-hack/ksubdomain/core/gologger"
	"github.com/boy-hack/ksubdomain/core/options"
	"github.com/boy-hack/ksubdomain/runner"
	"github.com/boy-hack/ksubdomain/runner/outputter"
	"github.com/boy-hack/ksubdomain/runner/outputter/output"
	"github.com/boy-hack/ksubdomain/runner/processbar"
	"strings"
)

func main() {
	process := processbar.ScreenProcess{}
	screenPrinter, _ := output.NewScreenOutput(false)

	domains := []string{"www.hacking8.com", "x.hacking8.com"}
	domainChanel := make(chan string)
	go func() {
		for _, d := range domains {
			domainChanel <- d
		}
		close(domainChanel)
	}()
	opt := &options.Options{
		Rate:        options.Band2Rate("1m"),
		Domain:      domainChanel,
		DomainTotal: 2,
		Resolvers:   options.GetResolvers(""),
		Silent:      false,
		TimeOut:     10,
		Retry:       3,
		Method:      runner.VerifyType,
		DnsType:     "a",
		Writer: []outputter.Output{
			screenPrinter,
		},
		ProcessBar: &process,
		EtherInfo:  options.GetDeviceConfig(),
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

As you can see, the usage is straightforward: fill in the `options` struct and call the runner to start. The key is knowing what to put in each option field.

Options struct definition:
```go
type Options struct {
	Rate        int64              // Packet send rate per second
	Domain      io.Reader          // Domain input
	DomainTotal int                // Total number of domains to scan
	Resolvers   []string           // DNS resolvers
	Silent      bool               // Silent mode
	TimeOut     int                // Timeout in seconds
	Retry       int                // Maximum retry count
	Method      string             // verify mode / enum mode / test mode
	DnsType     string             // DNS record type: a, ns, aaaa
	Writer      []outputter.Output // Output handlers
	ProcessBar  processbar.ProcessBar
	EtherInfo   *device.EtherTable // Network adapter info
}
```

1. The underlying interface of ksubdomain is just a DNS validator. If you want to enumerate subdomains from a root domain, you need to put all the full domain names into the `Domain` field. See how the enum command does it in `cmd/ksubdomain/enum.go`.
2. The `Writer` field is an `outputter.Output` interface that defines how DNS responses are handled. ksubdomain ships with three built-in implementations in `runner/outputter/output`: store data in memory, write data to a file, and print data to the screen. You can implement this interface yourself for custom behavior.
3. The `ProcessBar` field is a `processbar.ProcessBar` interface. Its purpose is to expose internal statistics—success count, sent count, queue length, received count, failed count, and elapsed time—to the caller in real time.
4. `EtherInfo` is of type `*device.EtherTable` and is used to obtain network adapter information. You can usually just call `options.GetDeviceConfig()` to auto-detect the adapter configuration.
