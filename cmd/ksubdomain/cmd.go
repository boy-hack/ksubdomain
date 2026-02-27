package main

import (
	"github.com/boy-hack/ksubdomain/v2/internal/banner"
	"github.com/boy-hack/ksubdomain/v2/pkg/version"
	"github.com/boy-hack/ksubdomain/v2/pkg/gologger"
	"github.com/urfave/cli/v2"
	"os"
)

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
