package main

import (
	"bufio"
	"context"
	"os"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/core/options"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter"
	output2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter/output"
	processbar2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/processbar"
	"github.com/urfave/cli/v2"
)

var commonFlags = []cli.Flag{
	// Target domain(s)
	&cli.StringSliceFlag{
		Name:    "domain",
		Aliases: []string{"d"},
		Usage:   "Target domain(s) to scan",
	},
	
	// Network bandwidth limit
	// Internationalization: bandwidth (recommended) replaces band (kept for compatibility)
	&cli.StringFlag{
		Name:     "bandwidth",
		Aliases:  []string{"band", "b"},
		Usage:    "Network bandwidth limit (e.g., 5m=5Mbps, 10m=10Mbps, 100m=100Mbps) [Recommended: use --bandwidth]",
		Required: false,
		Value:    "3m",
	},
	
	// DNS resolvers
	&cli.StringSliceFlag{
		Name:     "resolvers",
		Aliases:  []string{"r"},
		Usage:    "DNS resolver servers (e.g., 8.8.8.8, 1.1.1.1), uses built-in resolvers by default",
		Required: false,
	},
	
	// Output file
	&cli.StringFlag{
		Name:     "output",
		Aliases:  []string{"o"},
		Usage:    "Output file path",
		Required: false,
		Value:    "",
	},
	
	// Output format
	// Internationalization: format (recommended) replaces output-type (kept for compatibility)
	&cli.StringFlag{
		Name:     "format",
		Aliases:  []string{"output-type", "oy", "f"},
		Usage:    "Output format: txt (default), json, csv, jsonl [Recommended: use --format or -f]",
		Required: false,
		Value:    "txt",
	},
	
	// Silent mode
	&cli.BoolFlag{
		Name:  "silent",
		Usage: "Silent mode: only output domain names to screen",
		Value: false,
	},	
	// Colorized output
	&cli.BoolFlag{
		Name:    "color",
		Aliases: []string{"c"},
		Usage:   "Enable colorized output (beautified mode)",
		Value:   false,
	},
	
	// Beautified output
	&cli.BoolFlag{
		Name:  "beautify",
		Usage: "Enable beautified output with colors and summary statistics",
		Value: false,
	},
	&cli.BoolFlag{
		Name:    "only-domain",
		Aliases: []string{"od"},
		Usage:   "只输出域名,不显示IP (修复 Issue #67)",
		Value:   false,
	},
	&cli.IntFlag{
		Name:  "retry",
		Usage: "Retry count for failed queries (-1 for infinite retries)",
		Value: 3,
	},
	
	// Timeout
	&cli.IntFlag{
		Name:  "timeout",
		Usage: "Timeout in seconds for each DNS query",
		Value: 6,
	},
	
	// Read from stdin
	&cli.BoolFlag{
		Name:  "stdin",
		Usage: "Read domains from standard input (pipe)",
		Value: false,
	},
	
	// Suppress screen output
	// Internationalization: quiet (recommended) replaces not-print (kept for compatibility)
	&cli.BoolFlag{
		Name:    "quiet",
		Aliases: []string{"not-print", "np", "q", "no-output"},
		Usage:   "Suppress screen output (save to file only) [Recommended: use --quiet or -q]",
		Value:   false,
	},
	
	// Network interface
	// Internationalization: interface (recommended) replaces eth (kept for compatibility)
	// 支持多次指定（--eth eth0 --eth eth1）以开启多网卡并发发包
	&cli.StringSliceFlag{
		Name:    "interface",
		Aliases: []string{"eth", "e", "i"},
		Usage:   "Network interface name(s); can be repeated for multi-NIC (e.g. --eth eth0 --eth eth1) [Recommended: use --interface]",
	},
	
	// Wildcard filter
	// Internationalization: wildcard-filter (recommended) replaces wild-filter-mode
	&cli.StringFlag{
		Name:    "wildcard-filter",
		Aliases: []string{"wild-filter-mode", "wf"},
		Usage:   "Wildcard DNS filtering mode: none (default), basic, advanced [Recommended: use --wildcard-filter]",
		Value:   "none",
	},
	
	// Prediction mode
	&cli.BoolFlag{
		Name:     "predict",
		Usage:    "Enable AI-powered subdomain prediction",
		Required: false,
	},
}

var verifyCommand = &cli.Command{
	Name:    string(options.VerifyType),
	Aliases: []string{"v"},
	Usage:   "Verification mode: verify domain list for DNS resolution",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:     "filename",
			Aliases:  []string{"f"},
			Usage:    "Domain list file path (one domain per line)",
			Required: false,
			Value:    "",
		},
	}, commonFlags...),
	Action: func(c *cli.Context) error {
		if c.NumFlags() == 0 {
			cli.ShowCommandHelpAndExit(c, "verify", 0)
		}
		var domains []string
		processBar := &processbar2.ScreenProcess{Silent: c.Bool("silent")}
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
		if c.Bool("quiet") {
			processBar = nil
		}
		
		var screenWriter outputter.Output
		var err error
		
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
				// JSONL (JSON Lines) 格式: 每行一个 JSON,便于流式处理
				p, err := output2.NewJSONLOutput(outputFile)
				if err != nil {
					gologger.Fatalf(err.Error())
				}
				writer = append(writer, p)
			default:
				gologger.Fatalf("输出类型错误:%s 暂不支持 (支持: txt, json, csv, jsonl)", outputType)
			}
		}
		resolver := options.GetResolvers(c.StringSlice("resolvers"))
		
		// Support both old (band) and new (bandwidth) parameter names
		bandwidthValue := c.String("bandwidth")
		if bandwidthValue == "" || bandwidthValue == "3m" {
			// Fallback to old parameter for compatibility
			bandwidthValue = c.String("band")
		}

		// 收集网卡列表（支持重复 --eth 多网卡）
		ethNames := c.StringSlice("interface")
		etherInfos := options.GetDeviceConfigs(ethNames, resolver)

		opt := &options.Options{
			Rate:               options.Band2Rate(bandwidthValue),
			Domain:             render,
			Resolvers:          resolver,
			Silent:             c.Bool("silent"),
			TimeOut:            c.Int("timeout"),
			Retry:              c.Int("retry"),
			Method:             options.VerifyType,
			Writer:             writer,
			ProcessBar:         processBar,
			EtherInfo:          etherInfos[0], // 向后兼容
			EtherInfos:         etherInfos,
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
		// Exit 1 when nothing resolved — lets shell pipelines use && correctly.
		if r.SuccessCount() == 0 {
			os.Exit(1)
		}
		return nil
	},
}
