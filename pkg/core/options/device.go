package options

import (
	core2 "github.com/boy-hack/ksubdomain/v2/pkg/core"
	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/device"
	"os"
)

// GetDeviceConfig 获取网卡配置信息
func GetDeviceConfig(dnsServer []string) *device.EtherTable {
	// 读取配置文件路径环境变量
	var filename string
	filename, ok := os.LookupEnv("ksubdomain-config")
	if !ok {
		filename = "ksubdomain.yaml"
	}
	var ether *device.EtherTable
	var err error

	if core2.FileExists(filename) {
		ether, err = device.ReadConfig(filename)
		if err == nil {
			gologger.Infof("读取配置 %s 成功!\n", filename)
			device.PrintDeviceInfo(ether)
			return ether
		}
	}
	// 自动发现外网网卡
	gologger.Infof("正在自动识别外网网卡...\n")
	ether, err = device.AutoGetDevices(dnsServer)
	if err != nil {
		gologger.Fatalf("自动识别外网网卡失败: %v\n", err)
	}
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
