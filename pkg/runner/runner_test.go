package runner

import (
	"context"
	"testing"

	"github.com/boy-hack/ksubdomain/v2/pkg/core"
	"github.com/boy-hack/ksubdomain/v2/pkg/core/options"
	"github.com/boy-hack/ksubdomain/v2/pkg/device"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter/output"
	processbar2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/processbar"
	"github.com/stretchr/testify/assert"
)

func TestV(t *testing.T) {
	for i := 0; i < 2; i++ {
		domainChanel := make(chan string)
		eth := options.GetDeviceConfig([]string{"114.114.114.114"})
		domains := []string{"stu.baidu.com", "www.baidu.com"}
		go func() {
			for _, d := range domains {
				domainChanel <- d
			}
			close(domainChanel)
		}()
		w, _ := output.NewScreenOutput(true)
		opt := &options.Options{
			Rate:      options.Band2Rate("1m"),
			Domain:    domainChanel,
			Resolvers: options.GetResolvers(nil),
			Silent:    true,
			TimeOut:   5,
			Retry:     1,
			Method:    options.VerifyType,
			Writer: []outputter.Output{
				w,
			},
			EtherInfo: eth,
		}
		opt.Check()
		r, err := New(opt)
		assert.NoError(t, err)
		ctx := context.Background()
		r.RunEnumeration(ctx)
		r.Close()
	}
}
func TestVerify(t *testing.T) {
	process := processbar2.FakeScreenProcess{}
	screenPrinter, _ := output.NewScreenOutputNoWidth(false)
	domains := []string{"stu.baidu.com", "haokan.baidu.com"}
	domainChanel := make(chan string)
	eth, err := device.AutoGetDevices(nil)
	assert.NoError(t, err)

	go func() {
		for _, d := range domains {
			domainChanel <- d
		}
		close(domainChanel)
	}()
	opt := &options.Options{
		Rate:      options.Band2Rate("1m"),
		Domain:    domainChanel,
		Resolvers: options.GetResolvers(nil),
		Silent:    false,
		TimeOut:   5,
		Retry:     1,
		Method:    options.VerifyType,
		Writer: []outputter.Output{
			screenPrinter,
		},
		ProcessBar: &process,
		EtherInfo:  eth,
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
	eth, err := device.AutoGetDevices(nil)
	assert.NoError(t, err)
	opt := &options.Options{
		Rate:      options.Band2Rate("1m"),
		Domain:    domainChanel,
		Resolvers: options.GetResolvers(nil),
		Silent:    false,
		TimeOut:   5,
		Retry:     1,
		Method:    options.EnumType,
		Writer: []outputter.Output{
			screenPrinter,
		},
		ProcessBar: &process,
		EtherInfo:  eth,
	}
	opt.Check()
	r, err := New(opt)
	assert.NoError(t, err)
	ctx := context.Background()
	r.RunEnumeration(ctx)
	r.Close()
}

func TestPredict(t *testing.T) {
	process := processbar2.ScreenProcess{}
	screenPrinter, _ := output.NewScreenOutputNoWidth(false)
	domains := []string{"stu.baidu.com"}
	domainChanel := make(chan string)
	eth, err := device.AutoGetDevices([]string{"1.1.1.1"})
	if err != nil {
		t.Fatalf(err.Error())
	}

	go func() {
		for _, d := range domains {
			domainChanel <- d
		}
		close(domainChanel)
	}()
	opt := &options.Options{
		Rate:      options.Band2Rate("1m"),
		Domain:    domainChanel,
		Resolvers: options.GetResolvers(nil),
		Silent:    false,
		TimeOut:   5,
		Retry:     1,
		Method:    options.VerifyType,
		Writer: []outputter.Output{
			screenPrinter,
		},
		ProcessBar: &process,
		EtherInfo:  eth,
		Predict:    true,
	}
	opt.Check()
	r, err := New(opt)
	assert.NoError(t, err)
	ctx := context.Background()
	r.RunEnumeration(ctx)
	r.Close()
}
