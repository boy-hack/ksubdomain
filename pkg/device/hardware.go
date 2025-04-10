package device

import (
	"github.com/boy-hack/ksubdomain/pkg/core/gologger"
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
	gologger.Infof("使用网卡: %s\n", ether.Device)
	gologger.Infof("IP地址: %s\n", ether.SrcIp.String())
	gologger.Infof("本地MAC: %s\n", ether.SrcMac.String())
	gologger.Infof("网关MAC: %s\n", ether.DstMac.String())
}
