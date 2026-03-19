package options

import (
	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/device"
)

// GetDeviceConfig 获取单张网卡配置信息（原有接口，保持向后兼容）
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

// GetDeviceConfigs 获取一组网卡配置，支持多网卡场景。
//
//   - 若 ethNames 为空，则调用 AutoGetDevicesImproved 自动探测，返回单卡切片（保持原有行为）。
//   - 若 ethNames 非空，则对每个名字调用 getInterfaceByName 获取详细信息，失败时跳过并打警告。
//     若最终没有任何可用网卡则 Fatal。
func GetDeviceConfigs(ethNames []string, dnsServer []string) []*device.EtherTable {
	if len(ethNames) == 0 {
		ether := GetDeviceConfig(dnsServer)
		return []*device.EtherTable{ether}
	}

	var results []*device.EtherTable
	for _, name := range ethNames {
		et, err := device.GetInterfaceByName(name, dnsServer)
		if err != nil {
			gologger.Warningf("跳过网卡 %s: %v\n", name, err)
			continue
		}
		device.PrintDeviceInfo(et)
		results = append(results, et)
	}

	if len(results) == 0 {
		gologger.Fatalf("没有可用网卡，请检查 --eth 参数\n")
	}
	return results
}
