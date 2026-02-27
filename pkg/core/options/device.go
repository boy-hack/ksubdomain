package options

import (
	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/device"
)

// GetDeviceConfig retrieves network interface configuration.
// Improved version: prioritizes getting interface info via routing table, no dependency on config file cache.
func GetDeviceConfig(dnsServer []string) *device.EtherTable {
	// Use improved auto-detection method: routing table first, no config file dependency
	ether, err := device.AutoGetDevicesImproved(dnsServer)
	if err != nil {
		gologger.Fatalf("Failed to auto-detect external network interface: %v\n", err)
	}

	device.PrintDeviceInfo(ether)
	return ether
}
