package main

import (
	"github.com/urfave/cli/v2"
	"ksubdomain/runner"
)

var testCommand = &cli.Command{
	Name:  "test",
	Usage: "测试本地网卡的最大发送速度",
	Action: func(c *cli.Context) error {
		ether := runner.GetDeviceConfig()
		runner.TestSpeed(ether)
		return nil
	},
}
