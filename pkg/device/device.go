package device

import (
	"net"
	"os"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/google/gopacket/pcap"
	"gopkg.in/yaml.v3"
)

// EtherTable 存储网卡信息的数据结构
type EtherTable struct {
	SrcIp  net.IP  `yaml:"src_ip"`  // 源IP地址
	Device string  `yaml:"device"`  // 网卡设备名称
	SrcMac SelfMac `yaml:"src_mac"` // 源MAC地址
	DstMac SelfMac `yaml:"dst_mac"` // 目标MAC地址（通常是网关）
}

// ReadConfig 从文件读取EtherTable配置
func ReadConfig(filename string) (*EtherTable, error) {
	data, err := os.ReadFile(filename)
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

// SaveConfig 保存EtherTable配置到文件
func (e *EtherTable) SaveConfig(filename string) error {
	data, err := yaml.Marshal(e)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0666)
}

// PcapInit 初始化pcap句柄
func PcapInit(devicename string) (*pcap.Handle, error) {
	var (
		snapshot_len int32         = 1024
		timeout      time.Duration = -1 * time.Second
	)
	handle, err := pcap.OpenLive(devicename, snapshot_len, false, timeout)
	if err != nil {
		gologger.Fatalf("pcap初始化失败:%s\n", err.Error())
		return nil, err
	}
	return handle, nil
}
