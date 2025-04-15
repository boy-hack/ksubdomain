package main

import (
	"fmt"
	"github.com/boy-hack/ksubdomain/pkg/device"

	"github.com/boy-hack/ksubdomain/pkg/core/gologger"
	"github.com/urfave/cli/v2"
)

var deviceCommand = &cli.Command{
	Name:  "device",
	Usage: "列出系统所有可用的网卡信息",
	Flags: []cli.Flag{},
	Action: func(c *cli.Context) error {
		// 否则列出所有可用的网卡
		deviceNames, deviceMap := device.GetAllIPv4Devices()

		if len(deviceNames) == 0 {
			gologger.Warningf("未找到可用的IPv4网卡\n")
			return nil
		}

		gologger.Infof("系统发现 %d 个可用的网卡:\n", len(deviceNames))

		for i, name := range deviceNames {
			ip := deviceMap[name]
			gologger.Infof("[%d] 网卡名称: %s\n", i+1, name)
			gologger.Infof("    IP地址: %s\n", ip.String())
			fmt.Println("")
		}
		ether, err := device.AutoGetDevices([]string{"1.1.1.1", "8.8.8.8"})
		if err != nil {
			gologger.Errorf("获取网卡信息失败: %s\n", err.Error())
			return nil
		}
		device.PrintDeviceInfo(ether)
		return nil
	},
}
