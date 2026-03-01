package main

import (
	"github.com/boy-hack/ksubdomain/v2/internal/banner"
	"github.com/boy-hack/ksubdomain/v2/pkg/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/version"
	"github.com/urfave/cli/v2"
	"os"
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
		Usage:    "Output format: txt (default), json, csv, jsonl",
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
		Name:  "color",
		Usage: "Enable colorized output (beautified mode)",
		Value: false,
	},

	// Beautified output
	&cli.BoolFlag{
		Name:  "beautify",
		Usage: "Enable beautified output with colors and summary statistics",
		Value: false,
	},
	&cli.BoolFlag{
		Name:  "only-domain",
		Usage: "Only output domain names, do not display IP (Fix Issue #67)",
		Value: false,
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
		Aliases: []string{"q"},
		Usage:   "Suppress screen output (save to file only) [Recommended: use --quiet or -q]",
		Value:   false,
	},

	// Network interface
	// Internationalization: interface (recommended) replaces eth (kept for compatibility)
	&cli.StringFlag{
		Name:    "interface",
		Aliases: []string{"eth", "e", "i"},
		Usage:   "Network interface name (e.g., eth0, en0, wlan0) [Recommended: use --interface]",
	},

	// Wildcard filter
	// Internationalization: wildcard-filter (recommended) replaces wild-filter-mode
	&cli.StringFlag{
		Name:    "wildcard-filter",
		Aliases: []string{"wf"},
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

func main() {
	app := &cli.App{
		Name:    version.AppName,
		Version: version.Version,
		Usage:   version.Description,
		Commands: []*cli.Command{
			enumCommand,
			verifyCommand,
			testCommand,
			deviceCommand,
		},
		Before: func(c *cli.Context) error {
			silent := false
			for _, arg := range os.Args {
				if arg == "--silent" {
					silent = true
					break
				}
			}
			if silent {
				gologger.MaxLevel = gologger.Silent
			}
			banner.ShowBanner(silent)
			return nil
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		gologger.Fatalf(err.Error())
	}
}
