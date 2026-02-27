package main

import (
	"fmt"
	"github.com/boy-hack/ksubdomain/v2/pkg/device"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/urfave/cli/v2"
)

var deviceCommand = &cli.Command{
	Name:  "device",
	Usage: "List all available network interfaces on the system",
	Flags: []cli.Flag{},
	Action: func(c *cli.Context) error {
		// List all available network interfaces
		deviceNames, deviceMap := device.GetAllIPv4Devices()

		if len(deviceNames) == 0 {
			gologger.Warningf("No available IPv4 network interfaces found\n")
			return nil
		}

		gologger.Infof("Found %d available network interface(s):\n", len(deviceNames))

		for i, name := range deviceNames {
			ip := deviceMap[name]
			gologger.Infof("[%d] Interface: %s\n", i+1, name)
			gologger.Infof("    IP Address: %s\n", ip.String())
			fmt.Println("")
		}
		ether, err := device.AutoGetDevices([]string{"1.1.1.1", "8.8.8.8"})
		if err != nil {
			gologger.Errorf("Failed to get network interface info: %s\n", err.Error())
			return nil
		}
		device.PrintDeviceInfo(ether)
		return nil
	},
}
