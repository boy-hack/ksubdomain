package main

import (
	"bufio"
	"context"
	"os"

	"github.com/boy-hack/ksubdomain/v2/pkg/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/options"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter"
	output2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter/output"
	processbar2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/processbar"
	"github.com/urfave/cli/v2"
)

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
		// Read domains from file
		go func() {
			for _, line := range domains {
				render <- line
			}
			if c.String("filename") != "" {
				f2, err := os.Open(c.String("filename"))
				if err != nil {
					gologger.Fatalf("Failed to open file %s: %s", c.String("filename"), err.Error())
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

		// Output to screen
		if c.Bool("quiet") {
			processBar = nil
		}

		var screenWriter outputter.Output
		var err error

		// Beautified output mode
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
				gologger.Fatalf("Unsupported output type: %s (supported: txt, json, csv, jsonl)", outputType)
			}
		}
		resolver := options.GetResolvers(c.StringSlice("resolvers"))

		// Support both old (band) and new (bandwidth) parameter names
		bandwidthValue := c.String("bandwidth")
		if bandwidthValue == "" || bandwidthValue == "3m" {
			// Fallback to old parameter for compatibility
			bandwidthValue = c.String("band")
		}

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
			EtherInfo:          options.GetDeviceConfig(resolver),
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
