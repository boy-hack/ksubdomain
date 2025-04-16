package device

import (
	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"gopkg.in/yaml.v3"
	"net"
)

type SelfMac net.HardwareAddr

func (d SelfMac) String() string {
	n := (net.HardwareAddr)(d)
	return n.String()
}
func (d SelfMac) MarshalYAML() (interface{}, error) {
	n := (net.HardwareAddr)(d)
	return n.String(), nil
}
func (d SelfMac) HardwareAddr() net.HardwareAddr {
	n := (net.HardwareAddr)(d)
	return n
}
func (d *SelfMac) UnmarshalYAML(value *yaml.Node) error {
	v := value.Value
	v2, err := net.ParseMAC(v)
	if err != nil {
		return err
	}
	n := SelfMac(v2)
	*d = n
	return nil
}

// 打印设备信息
func PrintDeviceInfo(ether *EtherTable) {
	gologger.Infof("Device: %s\n", ether.Device)
	gologger.Infof("IP: %s\n", ether.SrcIp.String())
	gologger.Infof("Local Mac: %s\n", ether.SrcMac.String())
	gologger.Infof("Gateway Mac: %s\n", ether.DstMac.String())
}
