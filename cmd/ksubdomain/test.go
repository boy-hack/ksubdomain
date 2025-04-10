package main

import (
	"github.com/boy-hack/ksubdomain/pkg/core/options"
	runner2 "github.com/boy-hack/ksubdomain/pkg/runner"
	"github.com/urfave/cli/v2"
)

var testCommand = &cli.Command{
	Name:  runner2.TestType,
	Usage: "测试本地网卡的最大发送速度",
	Action: func(c *cli.Context) error {
		ether := options.GetDeviceConfig()
		runner2.TestSpeed(ether)
		return nil
	},
}
