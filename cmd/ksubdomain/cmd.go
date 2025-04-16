package main

import (
	"os"

	"github.com/boy-hack/ksubdomain/v2/pkg/core"
	"github.com/boy-hack/ksubdomain/v2/pkg/core/conf"
	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    conf.AppName,
		Version: conf.Version,
		Usage:   conf.Description,
		Commands: []*cli.Command{
			enumCommand,
			verifyCommand,
			testCommand,
			deviceCommand,
		},
	}
	core.ShowBanner()
	err := app.Run(os.Args)
	if err != nil {
		gologger.Fatalf(err.Error())
	}
}
