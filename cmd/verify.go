package main

import (
	"github.com/urfave/cli/v2"
	"ksubdomain/core/gologger"
	"ksubdomain/core/options"
	"ksubdomain/runner"
)

var verifyCommand = &cli.Command{
	Name:    "verify",
	Aliases: []string{"v"},
	Usage:   "验证模式",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "band",
			Aliases:  []string{"b"},
			Usage:    "宽带的下行速度，可以5M,5K,5G",
			Required: false,
			Value:    "1m",
		},
		&cli.StringFlag{
			Name:     "filename",
			Aliases:  []string{"f"},
			Usage:    "验证域名文件路径",
			Required: false,
			Value:    "",
		},
		&cli.StringFlag{
			Name:     "resolvers",
			Aliases:  []string{"r"},
			Usage:    "dns服务器地址",
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
		&cli.BoolFlag{
			Name:  "skip-wild",
			Usage: "跳过泛解析域名",
			Value: false,
		},
		&cli.IntFlag{
			Name:  "retry",
			Usage: "重试次数",
			Value: 3,
		},
		&cli.IntFlag{
			Name:  "timeout",
			Usage: "超时时间",
			Value: 6,
		},
		&cli.BoolFlag{
			Name:  "stdin",
			Usage: "使用stdin输入",
			Value: false,
		},
	},
	Action: func(c *cli.Context) error {
		if c.NumFlags() == 0 {
			cli.ShowCommandHelpAndExit(c, "verify", 0)
		}
		opt := &options.Options{
			Rate:         options.Band2Rate(c.String("band")),
			Domain:       nil,
			FileName:     c.String("filename"),
			Resolvers:    options.GetResolvers(c.String("resolvers")),
			Output:       c.String("output"),
			Silent:       c.Bool("silent"),
			Stdin:        c.Bool("stdin"),
			SkipWildCard: c.Bool("skip-wild"),
			TimeOut:      c.Int("timeout"),
			Retry:        c.Int("retry"),
			Method:       "verify",
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
