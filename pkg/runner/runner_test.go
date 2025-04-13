package runner

import (
	"context"
	"testing"

	"github.com/boy-hack/ksubdomain/pkg/core"
	"github.com/boy-hack/ksubdomain/pkg/core/options"
	"github.com/boy-hack/ksubdomain/pkg/device"
	"github.com/boy-hack/ksubdomain/pkg/runner/outputter"
	"github.com/boy-hack/ksubdomain/pkg/runner/outputter/output"
	processbar2 "github.com/boy-hack/ksubdomain/pkg/runner/processbar"
	"github.com/stretchr/testify/assert"
)

func TestVerify(t *testing.T) {
	process := processbar2.FakeScreenProcess{}
	screenPrinter, _ := output.NewScreenOutputNoWidth(false)
	domains := []string{"stu.baidu.com", "haokan.baidu.com"}
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
		Resolvers:   options.GetResolvers(nil),
		Silent:      false,
		TimeOut:     5,
		Retry:       1,
		Method:      options.VerifyType,
		Writer: []outputter.Output{
			screenPrinter,
		},
		ProcessBar: &process,
		EtherInfo:  device.AutoGetDevices(),
	}
	opt.Check()
	r, err := New(opt)
	assert.NoError(t, err)
	ctx := context.Background()
	r.RunEnumeration(ctx)
	r.Close()
}

func TestEnum(t *testing.T) {
	process := processbar2.ScreenProcess{}
	screenPrinter, _ := output.NewScreenOutputNoWidth(false)
	domains := core.GetDefaultSubdomainData()
	domainChanel := make(chan string)
	go func() {
		for _, d := range domains {
			domainChanel <- d + ".baidu.com"
		}
		close(domainChanel)
	}()
	opt := &options.Options{
		Rate:        options.Band2Rate("1m"),
		Domain:      domainChanel,
		DomainTotal: len(domains),
		Resolvers:   options.GetResolvers(nil),
		Silent:      false,
		TimeOut:     5,
		Retry:       1,
		Method:      options.EnumType,
		Writer: []outputter.Output{
			screenPrinter,
		},
		ProcessBar: &process,
		EtherInfo:  device.AutoGetDevices(),
	}
	opt.Check()
	r, err := New(opt)
	assert.NoError(t, err)
	ctx := context.Background()
	r.RunEnumeration(ctx)
	r.Close()
}

func TestManyRunner(t *testing.T) {
	for i := 0; i < 5; i++ {
		//TestRunner(t)
	}
}
