package runner

import (
	"context"
	"github.com/boy-hack/ksubdomain/core/dns"
	"github.com/boy-hack/ksubdomain/core/gologger"
	"github.com/boy-hack/ksubdomain/core/options"
	"github.com/boy-hack/ksubdomain/runner/outputter"
	"github.com/boy-hack/ksubdomain/runner/outputter/output"
	"github.com/boy-hack/ksubdomain/runner/processbar"
	"strings"
	"testing"
)

func TestRunner(t *testing.T) {
	process := processbar.ScreenProcess{}
	screenPrinter, _ := output.NewScreenOutput(false)
	domains := []string{"stu.baidu.com", "haokan.baidu.com"}
	_, ns, err := dns.LookupNS("baidu.com", "1.1.1.1")
	if err != nil {
		t.Fatal(err)
	}
	opt := &options.Options{
		Rate:        options.Band2Rate("1m"),
		Domain:      strings.NewReader(strings.Join(domains, "\n")),
		DomainTotal: 2,
		Resolvers:   options.GetResolvers(""),
		Silent:      false,
		TimeOut:     10,
		Retry:       3,
		Method:      VerifyType,
		DnsType:     "a",
		Writer: []outputter.Output{
			screenPrinter,
		},
		ProcessBar: &process,
		EtherInfo:  options.GetDeviceConfig(),
		SpecialResolvers: map[string][]string{
			"baidu.com": ns,
		},
	}
	opt.Check()
	r, err := New(opt)
	if err != nil {
		gologger.Fatalf(err.Error())
	}
	ctx := context.Background()
	r.RunEnumeration(ctx)
	r.Close()

}
