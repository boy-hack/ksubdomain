package device

import "testing"

func TestAutoGetDevices(t *testing.T) {
	ether := AutoGetDevices()
	PrintDeviceInfo(ether)
}
