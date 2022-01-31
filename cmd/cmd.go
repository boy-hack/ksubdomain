package main

import (
	"github.com/urfave/cli/v2"
	"ksubdomain/core/conf"
	"ksubdomain/core/gologger"
	"os"
)

func main() {
	app := &cli.App{
		Name:        conf.AppName,
		Version:     conf.Version,
		Description: conf.Description,
		Commands: []*cli.Command{
			enumCommand,
			verifyCommand,
			testCommand,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		gologger.Fatalf(err.Error())
	}
}
