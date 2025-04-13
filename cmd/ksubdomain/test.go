package main

import (
	"github.com/boy-hack/ksubdomain/pkg/core/options"
	"github.com/boy-hack/ksubdomain/pkg/runner"
	"github.com/urfave/cli/v2"
)

var testCommand = &cli.Command{
	Name:  string(options.TestType),
	Usage: "测试本地网卡的最大发送速度",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "eth",
			Aliases: []string{"e"},
			Usage:   "指定网卡名称，获取该网卡的详细信息",
		},
	},
	Action: func(c *cli.Context) error {
		ethTable := options.GetDeviceConfig(c.String("eth"))
		runner.TestSpeed(ethTable)
		return nil
	},
}
