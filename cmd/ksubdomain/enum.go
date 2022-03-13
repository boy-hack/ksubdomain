package main

import (
	"github.com/boy-hack/ksubdomain/core"
	"github.com/boy-hack/ksubdomain/core/gologger"
	"github.com/boy-hack/ksubdomain/core/options"
	"github.com/boy-hack/ksubdomain/runner"
	"github.com/urfave/cli/v2"
)

var enumCommand = &cli.Command{
	Name:    runner.EnumType,
	Aliases: []string{"e"},
	Usage:   "枚举域名",
	Flags: append(commonFlags, []cli.Flag{
		&cli.StringFlag{
			Name:     "domain",
			Aliases:  []string{"d"},
			Usage:    "爆破的域名",
			Required: false,
			Value:    "",
		},
		&cli.StringFlag{
			Name:     "domainList",
			Aliases:  []string{"dl"},
			Usage:    "从文件中指定域名",
			Required: false,
			Value:    "",
		},
		&cli.StringFlag{
			Name:     "filename",
			Aliases:  []string{"f"},
			Usage:    "字典路径",
			Required: false,
			Value:    "",
		},
		&cli.BoolFlag{
			Name:  "skip-wild",
			Usage: "跳过泛解析域名",
			Value: false,
		},
		&cli.IntFlag{
			Name:    "level",
			Aliases: []string{"l"},
			Usage:   "枚举几级域名，默认为2，二级域名",
			Value:   2,
		},
		&cli.StringFlag{
			Name:    "level-dict",
			Aliases: []string{"ld"},
			Usage:   "枚举多级域名的字典文件，当level大于2时候使用，不填则会默认",
			Value:   "",
		},
	}...),
	Action: func(c *cli.Context) error {
		if c.NumFlags() == 0 {
			cli.ShowCommandHelpAndExit(c, "enum", 0)
		}
		var domains []string
		// handle domain
		if c.String("domain") != "" {
			domains = append(domains, c.String("domain"))
		}
		if c.String("domainList") != "" {
			dl, err := core.LinesInFile(c.String("domainList"))
			if err != nil {
				gologger.Fatalf("读取domain文件失败:%s\n", err.Error())
			}
			domains = append(dl, domains...)
		}
		levelDict := c.String("level-dict")
		var levelDomains []string
		if levelDict != "" {
			dl, err := core.LinesInFile(levelDict)
			if err != nil {
				gologger.Fatalf("读取domain文件失败:%s,请检查--level-dict参数\n", err.Error())
			}
			levelDomains = dl
		} else if c.Int("level") > 2 {
			levelDomains = core.GetDefaultSubNextData()
		}

		opt := &options.Options{
			Rate:         options.Band2Rate(c.String("band")),
			Domain:       domains,
			FileName:     c.String("filename"),
			Resolvers:    options.GetResolvers(c.String("resolvers")),
			Output:       c.String("output"),
			Silent:       c.Bool("silent"),
			Stdin:        c.Bool("stdin"),
			SkipWildCard: c.Bool("skip-wild"),
			TimeOut:      c.Int("timeout"),
			Retry:        c.Int("retry"),
			Method:       runner.EnumType,
			OnlyDomain:   c.Bool("only-domain"),
			NotPrint:     c.Bool("not-print"),
			Level:        c.Int("level"),
			LevelDomains: levelDomains,
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
