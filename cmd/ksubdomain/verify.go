package main

import (
	"bufio"
	"context"
	"github.com/boy-hack/ksubdomain/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/pkg/core/options"
	"github.com/boy-hack/ksubdomain/pkg/runner"
	"github.com/boy-hack/ksubdomain/pkg/runner/outputter"
	output2 "github.com/boy-hack/ksubdomain/pkg/runner/outputter/output"
	processbar2 "github.com/boy-hack/ksubdomain/pkg/runner/processbar"
	"github.com/urfave/cli/v2"
	"os"
)

var commonFlags = []cli.Flag{
	&cli.StringSliceFlag{
		Name:    "domain",
		Aliases: []string{"d"},
		Usage:   "域名",
	},
	&cli.StringFlag{
		Name:     "band",
		Aliases:  []string{"b"},
		Usage:    "宽带的下行速度，可以5M,5K,5G",
		Required: false,
		Value:    "3m",
	},
	&cli.StringSliceFlag{
		Name:     "resolvers",
		Aliases:  []string{"r"},
		Usage:    "dns服务器，默认会使用内置dns",
		Required: false,
	},
	&cli.StringFlag{
		Name:     "output",
		Aliases:  []string{"o"},
		Usage:    "输出文件名",
		Required: false,
		Value:    "",
	},
	&cli.StringFlag{
		Name:     "output-type",
		Aliases:  []string{"oy"},
		Usage:    "输出文件类型: json, txt, csv",
		Required: false,
		Value:    "txt",
	},
	&cli.BoolFlag{
		Name:  "silent",
		Usage: "使用后屏幕将仅输出域名",
		Value: false,
	},
	&cli.IntFlag{
		Name:  "retry",
		Usage: "重试次数,当为-1时将一直重试",
		Value: 3,
	},
	&cli.IntFlag{
		Name:  "timeout",
		Usage: "超时时间",
		Value: 6,
	},
	&cli.BoolFlag{
		Name:  "stdin",
		Usage: "接受stdin输入",
		Value: false,
	},
	&cli.BoolFlag{
		Name:    "not-print",
		Aliases: []string{"np"},
		Usage:   "不打印域名结果",
		Value:   false,
	},
	&cli.StringFlag{
		Name:    "eth",
		Aliases: []string{"e"},
		Usage:   "指定网卡名称",
	},
	&cli.StringFlag{
		Name:  "wild-filter-mode",
		Usage: "泛解析过滤模式[从最终结果过滤泛解析域名]: basic(基础), advanced(高级), none(不过滤)",
		Value: "none",
	},
	&cli.BoolFlag{
		Name:     "predict",
		Usage:    "启用预测域名模式",
		Required: false,
	},
}

var verifyCommand = &cli.Command{
	Name:    string(options.VerifyType),
	Aliases: []string{"v"},
	Usage:   "验证模式",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:     "filename",
			Aliases:  []string{"f"},
			Usage:    "验证域名的文件路径",
			Required: false,
			Value:    "",
		},
	}, commonFlags...),
	Action: func(c *cli.Context) error {
		if c.NumFlags() == 0 {
			cli.ShowCommandHelpAndExit(c, "verify", 0)
		}
		var domains []string
		var processBar processbar2.ProcessBar = &processbar2.ScreenProcess{}
		if c.StringSlice("domain") != nil {
			domains = append(domains, c.StringSlice("domain")...)
		}
		if c.Bool("stdin") {
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				domains = append(domains, scanner.Text())
			}
		}
		render := make(chan string)
		// 读取文件
		go func() {
			for _, line := range domains {
				render <- line
			}
			if c.String("filename") != "" {
				f2, err := os.Open(c.String("filename"))
				if err != nil {
					gologger.Fatalf("打开文件:%s 出现错误:%s", c.String("filename"), err.Error())
				}
				defer f2.Close()
				iofile := bufio.NewScanner(f2)
				iofile.Split(bufio.ScanLines)
				for iofile.Scan() {
					render <- iofile.Text()
				}
			}
			close(render)
		}()

		// 输出到屏幕
		if c.Bool("not-print") {
			processBar = nil
		}
		screenWriter, err := output2.NewScreenOutput()
		if err != nil {
			gologger.Fatalf(err.Error())
		}
		var writer []outputter.Output
		if !c.Bool("not-print") {
			writer = append(writer, screenWriter)
		}
		if c.String("output") != "" {
			outputFile := c.String("output")
			outputType := c.String("output-type")
			wildFilterMode := c.String("wild-filter-mode")
			switch outputType {
			case "txt":
				p, err := output2.NewPlainOutput(outputFile, wildFilterMode)
				if err != nil {
					gologger.Fatalf(err.Error())
				}
				writer = append(writer, p)
			case "json":
				p := output2.NewJsonOutput(outputFile, wildFilterMode)
				writer = append(writer, p)
			case "csv":
				p := output2.NewCsvOutput(outputFile, wildFilterMode)
				writer = append(writer, p)
			default:
				gologger.Fatalf("输出类型错误:%s 暂不支持", outputType)
			}
		}

		opt := &options.Options{
			Rate:               options.Band2Rate(c.String("band")),
			Domain:             render,
			Resolvers:          options.GetResolvers(c.StringSlice("resolvers")),
			Silent:             c.Bool("silent"),
			TimeOut:            c.Int("timeout"),
			Retry:              c.Int("retry"),
			Method:             options.VerifyType,
			Writer:             writer,
			ProcessBar:         processBar,
			EtherInfo:          options.GetDeviceConfig(c.String("eth")),
			WildcardFilterMode: c.String("wild-filter-mode"),
			Predict:            c.Bool("predict"),
		}
		opt.Check()
		ctx := context.Background()
		r, err := runner.New(opt)
		if err != nil {
			gologger.Fatalf("%s\n", err.Error())
			return nil
		}
		r.RunEnumeration(ctx)
		r.Close()
		return nil
	},
}
