package main

import (
	"bufio"
	"context"
	"math/rand"
	"os"

	core2 "github.com/boy-hack/ksubdomain/v2/pkg/core"
	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/core/ns"
	"github.com/boy-hack/ksubdomain/v2/pkg/core/options"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter"
	output2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter/output"
	processbar2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/processbar"
	"github.com/urfave/cli/v2"
)

var enumCommand = &cli.Command{
	Name:    string(options.EnumType),
	Aliases: []string{"e"},
	Usage:   "Enumeration mode: brute-force subdomains using dictionary",
	Flags: append(commonFlags, []cli.Flag{
		&cli.StringFlag{
			Name:     "filename",
			Aliases:  []string{"f"},
			Usage:    "Subdomain dictionary file path",
			Required: false,
			Value:    "",
		},
		// Use NS records
		// Internationalization: use-ns-records (recommended) replaces ns
		&cli.BoolFlag{
			Name:    "use-ns-records",
			Aliases: []string{"ns"},
			Usage:   "Query and use domain's NS records as DNS resolvers [Recommended: use --use-ns-records]",
			Value:   false,
		},
		&cli.StringFlag{
			Name:    "domain-list",
			Aliases: []string{"ds"},
			Usage:   "Domain list file for batch enumeration",
			Value:   "",
		},
	}...),
	Action: func(c *cli.Context) error {
		if c.NumFlags() == 0 {
			cli.ShowCommandHelpAndExit(c, "enum", 0)
		}
		var domains []string
		processBar := &processbar2.ScreenProcess{Silent: c.Bool("silent")}

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
		if c.String("domain-list") != "" {
			filename := c.String("domain-list")
			f, err := os.Open(filename)
			if err != nil {
				gologger.Fatalf("打开文件:%s 出现错误:%s", filename, err.Error())
			}
			defer f.Close()
			scanner := bufio.NewScanner(f)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				domain := scanner.Text()
				domains = append(domains, domain)
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
		// Support both old (ns) and new (use-ns-records) parameter names
		useNS := c.Bool("use-ns-records")
		if !useNS {
			useNS = c.Bool("ns")
		}

		if useNS {
			for _, domain := range domains {
				nsServers, ips, err := ns.LookupNS(domain, defaultResolver[rand.Intn(len(defaultResolver))])
				if err != nil {
					continue
				}
				specialDns[domain] = ips
				gologger.Infof("%s ns:%v", domain, nsServers)
			}

		}
		if c.Bool("quiet") {
			processBar = nil
		}

		// 输出到屏幕
		if c.Bool("quiet") {
			processBar = nil
		}
		var screenWriter outputter.Output

		// 美化输出模式
		if c.Bool("beautify") || c.Bool("color") {
			useColor := c.Bool("color") || c.Bool("beautify")
			onlyDomain := c.Bool("only-domain")
			screenWriter, err = output2.NewBeautifiedOutput(c.Bool("silent"), useColor, onlyDomain)
		} else {
			screenWriter, err = output2.NewScreenOutput(c.Bool("silent"))
		}
		if err != nil {
			gologger.Fatalf(err.Error())
		}
		var writer []outputter.Output
		if !c.Bool("quiet") {
			writer = append(writer, screenWriter)
		}
		if c.String("output") != "" {
			outputFile := c.String("output")

			// Support both old and new parameter names
			outputType := c.String("format")
			if outputType == "" || outputType == "txt" {
				outputType = c.String("output-type")
			}

			wildFilterMode := c.String("wildcard-filter")
			if wildFilterMode == "" || wildFilterMode == "none" {
				wildFilterMode = c.String("wild-filter-mode")
			}
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
			case "jsonl":
				// JSONL (JSON Lines) format: One JSON per line for streaming
				p, err := output2.NewJSONLOutput(outputFile)
				if err != nil {
					gologger.Fatalf(err.Error())
				}
				writer = append(writer, p)
			default:
				gologger.Fatalf("输出类型错误:%s 暂不支持 (支持: txt, json, csv, jsonl)", outputType)
			}
		}
		// Support both old (band) and new (bandwidth) parameter names
		bandwidthValue := c.String("bandwidth")
		if bandwidthValue == "" || bandwidthValue == "3m" {
			bandwidthValue = c.String("band")
		}

		// 收集网卡列表（支持重复 --eth 多网卡）
		ethNames := c.StringSlice("interface")
		etherInfos := options.GetDeviceConfigs(ethNames, defaultResolver)

		opt := &options.Options{
			Rate:               options.Band2Rate(bandwidthValue),
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
			EtherInfo:          etherInfos[0], // 向后兼容
			EtherInfos:         etherInfos,
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
		// Exit 1 when nothing resolved — lets shell pipelines use && correctly.
		if r.SuccessCount() == 0 {
			os.Exit(1)
		}
		return nil
	},
}
