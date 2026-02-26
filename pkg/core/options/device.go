package options

import (
	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/device"
)

// GetDeviceConfig 获取网卡配置信息
// 改进版本：优先通过路由表获取网卡信息，不依赖配置文件缓存
func GetDeviceConfig(dnsServer []string) *device.EtherTable {
	// 使用改进的自动识别方法，优先通过路由表获取，不依赖配置文件
	ether, err := device.AutoGetDevicesImproved(dnsServer)
	if err != nil {
		gologger.Fatalf("自动识别外网网卡失败: %v\n", err)
	}

	device.PrintDeviceInfo(ether)
	return ether
}
