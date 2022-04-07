一个简单的调用例子
注意: 不要启动多个ksubdomain，ksubdomain启动一个就可以发挥最大作用。

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
可以看到调用很简单，就是填写`options`参数，然后调用runner启动就好了，重要的是options填什么。
options的参数结构
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
1. ksubdomain底层接口只是一个dns验证器，如果要通过一级域名枚举，需要把全部的域名都放入`Domain`字段中，可以看enum参数是怎么写的 `cmd/ksubdomain/enum.go`
2. Write参数是一个outputter.Output接口，用途是如何处理DNS返回的接口，ksubdomain已经内置了三种接口在 `runner/outputter/output`中，主要作用是把数据存入内存、数据写入文件、数据打印到屏幕，可以自己实现这个接口，实现自定义的操作。
3. ProcessBar参数是一个processbar.ProcessBar接口，主要用途是将程序内`成功个数`、`发送个数`、`队列数`、`接收数`、`失败数`、`耗时`传递给用户，实现这个参数可以时时获取这些。
4. EtherInfo是*device.EtherTable类型，用来获取网卡的信息，一般用函数`options.GetDeviceConfig()`即可自动获取网卡配置。

