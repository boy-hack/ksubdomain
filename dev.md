一个简单的调用例子

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
	screenPrinter, _ := output.NewScreenOutput()

	domains := []string{"www.hacking8.com", "x.hacking8.com"}
	opt := &options.Options{
		Rate:        options.Band2Rate("1m"),
		Domain:      strings.NewReader(strings.Join(domains, "\n")),
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
需要填写`options`参数
```go
type Options struct {
	Rate        int64              // 每秒发包速率
	Domain      io.Reader          // 域名输入
	DomainTotal int                // 扫描域名总数
	Resolvers   []string           // dns resolvers
	Silent      bool               // 安静模式
	TimeOut     int                // 超时时间 单位(秒)
	Retry       int                // 最大重试次数
	Method      string             // verify模式 enum模式 test模式
	DnsType     string             // dns类型 a ns aaaa
	Writer      []outputter.Output // 输出结构
	ProcessBar  processbar.ProcessBar
	EtherInfo   *device.EtherTable // 网卡信息
}
```
