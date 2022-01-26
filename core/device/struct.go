package device

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net"
)

type EtherTable struct {
	SrcIp  net.IP           `yaml:"src_ip"`
	Device string           `yaml:"device"`
	SrcMac net.HardwareAddr `yaml:"src_mac"`
	DstMac net.HardwareAddr `yaml:"dst_mac"`
}

func ReadConfig(filename string) (*EtherTable, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var ether EtherTable
	err = yaml.Unmarshal(data, &ether)
	if err != nil {
		return nil, err
	}
	return &ether, nil
}
func (e *EtherTable) SaveConfig(filename string) error {
	data, err := yaml.Marshal(e)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0666)
}
