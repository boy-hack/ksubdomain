package main

import (
	"github.com/boy-hack/ksubdomain/core/gologger"
	"github.com/boy-hack/ksubdomain/core/options"
	"github.com/boy-hack/ksubdomain/runner"
	"github.com/urfave/cli/v2"
)

var commonFlags = []cli.Flag{
	&cli.StringFlag{
		Name:     "band",
		Aliases:  []string{"b"},
		Usage:    "宽带的下行速度，可以5M,5K,5G",
		Required: false,
		Value:    "2m",
	},
	&cli.StringFlag{
		Name:     "resolvers",
		Aliases:  []string{"r"},
		Usage:    "dns服务器文件路径，一行一个dns地址",
		Required: false,
		Value:    "",
	},
	&cli.StringFlag{
		Name:     "output",
		Aliases:  []string{"o"},
		Usage:    "输出文件名",
		Required: false,
		Value:    "",
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
		Name:    "only-domain",
		Aliases: []string{"od"},
		Usage:   "只打印域名，不显示ip",
		Value:   false,
	},
	&cli.BoolFlag{
		Name:    "not-print",
		Aliases: []string{"np"},
		Usage:   "不打印域名结果",
		Value:   false,
	},
	&cli.IntFlag{
		Name:  "dns-type",
		Usage: "dns类型 1为a记录 2为ns记录 5为cname记录 16为txt",
		Value: 1,
	},
}

var verifyCommand = &cli.Command{
	Name:    runner.VerifyType,
	Aliases: []string{"v"},
	Usage:   "验证模式",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:     "filename",
			Aliases:  []string{"f"},
			Usage:    "验证域名文件路径",
			Required: false,
			Value:    "",
		},
	}, commonFlags...),
	Action: func(c *cli.Context) error {
		if c.NumFlags() == 0 {
			cli.ShowCommandHelpAndExit(c, "verify", 0)
		}
		var domains []string
		if c.String("domain") != "" {
			domains = append(domains, c.String("domain"))
		}
		opt := &options.Options{
			Rate:         options.Band2Rate(c.String("band")),
			Domain:       domains,
			FileName:     c.String("filename"),
			Resolvers:    options.GetResolvers(c.String("resolvers")),
			Output:       c.String("output"),
			Silent:       c.Bool("silent"),
			Stdin:        c.Bool("stdin"),
			SkipWildCard: false,
			TimeOut:      c.Int("timeout"),
			Retry:        c.Int("retry"),
			Method:       runner.VerifyType,
			OnlyDomain:   c.Bool("only-domain"),
			NotPrint:     c.Bool("not-print"),
			DnsType:      c.Int("dns-type"),
		}
		opt.Check()

		r, err := runner.New(opt)
		if err != nil {
			gologger.Fatalf("%s\n", err.Error())
			return nil
		}
		r.RunEnumeration()
		r.Close()
		return nil
	},
}
