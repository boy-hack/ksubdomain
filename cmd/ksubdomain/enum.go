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
	"github.com/google/gopacket/layers" // Import for layers.DNSType
	"strings"                           // Import for strings.Split and strings.ToUpper
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
		&cli.StringFlag{
			Name:    "domain-list",
			Aliases: []string{"ds"},
			Usage:   "指定域名列表文件",
			Value:   "",
		},
		&cli.StringFlag{
			Name:    "qtype",
			Aliases: []string{"qt"},
			Usage:   "Specify DNS query types (comma-separated: A,AAAA,CNAME,MX,TXT,NS)",
			Value:   "A",
		},
	}...),
	Action: func(c *cli.Context) error {
		if c.NumFlags() == 0 {
			cli.ShowCommandHelpAndExit(c, "enum", 0)
		}
		var domains []string
		var processBar processbar2.ProcessBar = &processbar2.ScreenProcess{}
		var err error

		// Parse Query Types
		queryTypesStr := c.String("qtype")
		queryTypes, err := parseQueryTypes(queryTypesStr)
		if err != nil {
			gologger.Fatalf("Invalid query type string: %s. %v", queryTypesStr, err)
		}

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
			QueryTypes:         queryTypes,
		}
		opt.Check()
		opt.EtherInfo = options.GetDeviceConfig(defaultResolver)
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

func parseQueryTypes(typeStr string) ([]layers.DNSType, error) {
	parts := strings.Split(typeStr, ",")
	var qTypes []layers.DNSType
	seenTypes := make(map[layers.DNSType]bool)

	for _, part := range parts {
		trimmedPart := strings.TrimSpace(strings.ToUpper(part))
		var qType layers.DNSType
		switch trimmedPart {
		case "A":
			qType = layers.DNSTypeA
		case "AAAA":
			qType = layers.DNSTypeAAAA
		case "CNAME":
			qType = layers.DNSTypeCNAME
		case "MX":
			qType = layers.DNSTypeMX
		case "TXT":
			qType = layers.DNSTypeTXT
		case "NS":
			qType = layers.DNSTypeNS
		// Add more types here as needed, e.g. SOA, SRV, CAA
		default:
			return nil, SystemError{Msg: "unsupported DNS query type: " + trimmedPart}
		}
		if !seenTypes[qType] {
			qTypes = append(qTypes, qType)
			seenTypes[qType] = true
		}
	}
	if len(qTypes) == 0 {
		return nil, SystemError{Msg: "no valid query types specified"}
	}
	return qTypes, nil
}

// SystemError 自定义错误类型
type SystemError struct {
	Msg string
}

func (e SystemError) Error() string {
	return e.Msg
}
