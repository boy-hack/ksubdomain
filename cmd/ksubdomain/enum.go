package main

import (
	"bufio"
	"context"
	"math/rand"
	"os"

	core2 "github.com/boy-hack/ksubdomain/pkg/core"
	"github.com/boy-hack/ksubdomain/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/pkg/core/ns"
	"github.com/boy-hack/ksubdomain/pkg/core/options"
	"github.com/boy-hack/ksubdomain/pkg/runner"
	"github.com/boy-hack/ksubdomain/pkg/runner/outputter"
	output2 "github.com/boy-hack/ksubdomain/pkg/runner/outputter/output"
	processbar2 "github.com/boy-hack/ksubdomain/pkg/runner/processbar"
	"github.com/urfave/cli/v2"
)

var enumCommand = &cli.Command{
	Name:    string(options.EnumType),
	Aliases: []string{"e"},
	Usage:   "枚举域名",
	Flags: append(commonFlags, []cli.Flag{
		&cli.StringFlag{
			Name:     "filename",
			Aliases:  []string{"f"},
			Usage:    "字典路径",
			Required: false,
			Value:    "",
		},
		&cli.BoolFlag{
			Name:  "ns",
			Usage: "读取域名ns记录并加入到ns解析器中",
			Value: false,
		},
	}...),
	Action: func(c *cli.Context) error {
		if c.NumFlags() == 0 {
			cli.ShowCommandHelpAndExit(c, "enum", 0)
		}
		var domains []string
		var processBar processbar2.ProcessBar = &processbar2.ScreenProcess{}
		var err error

		// handle domain
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

		wildIPS := make([]string, 0)
		if c.String("wild-filter-mode") != "none" {
			for _, sub := range domains {
				ok, ips := runner.IsWildCard(sub)
				if ok {
					wildIPS = append(wildIPS, ips...)
					gologger.Infof("发现泛解析域名:%s", sub)
				}
			}
		}

		render := make(chan string)
		go func() {
			defer close(render)
			filename := c.String("filename")
			if filename == "" {
				subdomainDict := core2.GetDefaultSubdomainData()
				for _, domain := range domains {
					for _, sub := range subdomainDict {
						dd := sub + "." + domain
						render <- dd
					}
				}
			} else {
				f2, err := os.Open(filename)
				if err != nil {
					gologger.Fatalf("打开文件:%s 出现错误:%s", c.String("filename"), err.Error())
				}
				defer f2.Close()
				iofile := bufio.NewScanner(f2)
				iofile.Split(bufio.ScanLines)
				for iofile.Scan() {
					sub := iofile.Text()
					for _, domain := range domains {
						render <- sub + "." + domain
					}
				}
			}
		}()
		// 取域名的dns,加入到resolver中
		specialDns := make(map[string][]string)
		defaultResolver := options.GetResolvers(c.StringSlice("resolvers"))
		if c.Bool("ns") {
			for _, domain := range domains {
				nsServers, ips, err := ns.LookupNS(domain, defaultResolver[rand.Intn(len(defaultResolver))])
				if err != nil {
					continue
				}
				specialDns[domain] = ips
				gologger.Infof("%s ns:%v", domain, nsServers)
			}

		}
		if c.Bool("not-print") {
			processBar = nil
		}

		// 输出到屏幕
		if c.Bool("not-print") {
			processBar = nil
		}

		screenWriter, err := output2.NewScreenOutput(c.Bool("silent"))
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
			Resolvers:          defaultResolver,
			Silent:             c.Bool("silent"),
			TimeOut:            c.Int("timeout"),
			Retry:              c.Int("retry"),
			Method:             options.VerifyType,
			Writer:             writer,
			ProcessBar:         processBar,
			SpecialResolvers:   specialDns,
			WildcardFilterMode: c.String("wild-filter-mode"),
			WildIps:            wildIPS,
			Predict:            c.Bool("predict"),
		}
		opt.Check()
		opt.EtherInfo = options.GetDeviceConfig(c.String("eth"))
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
