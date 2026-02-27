package main

import (
	"github.com/boy-hack/ksubdomain/v2/pkg/options"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner"
	"github.com/urfave/cli/v2"
)

var testCommand = &cli.Command{
	Name:  string(options.TestType),
	Usage: "Test the maximum sending speed of the local network interface",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "eth",
			Aliases: []string{"e"},
			Usage:   "Specify network interface name to get its detailed information",
		},
	},
	Action: func(c *cli.Context) error {
		ethTable := options.GetDeviceConfig(nil)
		runner.TestSpeed(ethTable)
		return nil
	},
}
