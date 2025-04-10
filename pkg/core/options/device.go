package options

import (
	core2 "github.com/boy-hack/ksubdomain/pkg/core"
	"github.com/boy-hack/ksubdomain/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/pkg/device"
	"os"
)

// GetDeviceConfig 获取网卡配置信息
// 优先级：
// 1. 读取配置文件
// 2. 指定的网卡名
// 3. 自动发现外网网卡
func GetDeviceConfig(deviceName string) *device.EtherTable {
	// 读取配置文件路径环境变量
	var filename string
	filename, ok := os.LookupEnv("ksubdomain-config")
	if !ok {
		filename = "ksubdomain.yaml"
	}

	var ether *device.EtherTable
	var err error

	// 1. 优先读取配置文件
	if core2.FileExists(filename) {
		ether, err = device.ReadConfig(filename)
		if err != nil {
			gologger.Warningf("读取配置失败: %v\n", err)
		} else {
			gologger.Infof("读取配置 %s 成功!\n", filename)
			device.PrintDeviceInfo(ether)
			return ether
		}
	}

	// 2. 检查环境变量指定的网卡名
	if deviceName != "" {
		gologger.Infof("从环境变量读取到网卡名: %s\n", deviceName)
		ether, err = device.GetDevicesByName(deviceName)
		if err != nil {
			gologger.Warningf("使用指定网卡失败: %v\n", err)
		} else {
			saveConfig(ether, filename)
			device.PrintDeviceInfo(ether)
			return ether
		}
	}

	// 3. 自动发现外网网卡
	gologger.Infof("正在自动识别外网网卡...\n")
	ether = device.AutoGetDevices()
	saveConfig(ether, filename)
	device.PrintDeviceInfo(ether)
	return ether
}

// 保存配置到文件
func saveConfig(ether *device.EtherTable, filename string) {
	err := ether.SaveConfig(filename)
	if err != nil {
		gologger.Warningf("保存配置失败: %v\n", err)
	} else {
		gologger.Infof("配置已保存到 %s\n", filename)
	}
}
