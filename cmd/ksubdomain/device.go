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
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "name",
			Aliases: []string{"n"},
			Usage:   "指定网卡名称，获取该网卡的详细信息",
		},
	},
	Action: func(c *cli.Context) error {
		// 如果指定了网卡名称，显示该网卡的详细信息
		if c.String("name") != "" {
			deviceName := c.String("name")
			ether, err := device.GetDevicesByName(deviceName)
			if err != nil {
				gologger.Fatalf("获取网卡信息失败: %v\n", err)
				return err
			}

			gologger.Infof("网卡名称: %s\n", ether.Device)
			gologger.Infof("IP地址: %s\n", ether.SrcIp.String())
			gologger.Infof("MAC地址: %s\n", ether.SrcMac.String())
			gologger.Infof("网关MAC: %s\n", ether.DstMac.String())
			return nil
		}

		// 否则列出所有可用的网卡
		deviceNames, deviceMap := device.GetAllIPv4Devices()

		if len(deviceNames) == 0 {
			gologger.Warningf("未找到可用的IPv4网卡\n")
			return nil
		}

		gologger.Infof("系统发现 %d 个可用的网卡:\n", len(deviceNames))
		fmt.Println("")

		for i, name := range deviceNames {
			ip := deviceMap[name]
			gologger.Infof("[%d] 网卡名称: %s\n", i+1, name)
			gologger.Infof("    IP地址: %s\n", ip.String())
			fmt.Println("")
		}

		gologger.Infof("提示: 可以使用环境变量指定网卡，例如:\n")
		gologger.Infof("export ksubdomain-device=YOUR_DEVICE_NAME\n")

		return nil
	},
}
