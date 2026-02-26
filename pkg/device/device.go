package device

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
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

// isWSL 检测是否在 WSL/WSL2 环境中运行
func isWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	version := strings.ToLower(string(data))
	return strings.Contains(version, "microsoft") || strings.Contains(version, "wsl")
}

// isDeviceUp 检查网卡是否处于激活状态
func isDeviceUp(devicename string) bool {
	iface, err := net.InterfaceByName(devicename)
	if err != nil {
		return false
	}
	// 检查 UP 标志位
	return iface.Flags&net.FlagUp != 0
}

// PcapInit 初始化pcap句柄
// 修复 Issue #68: 增强错误提示,特别是 WSL2 环境
func PcapInit(devicename string) (*pcap.Handle, error) {
	var (
		snapshot_len int32         = 1024
		timeout      time.Duration = -1 * time.Second
	)
	
	handle, err := pcap.OpenLive(devicename, snapshot_len, false, timeout)
	if err != nil {
		// 修复 Issue #68: 提供详细的错误信息和解决方案
		errMsg := err.Error()
		
		// 情况1: 网卡未激活
		if strings.Contains(errMsg, "not up") {
			var solution string
			if isWSL() {
				// WSL/WSL2 特殊提示
				solution = fmt.Sprintf(
					"网卡 %s 未激活 (WSL/WSL2 环境检测到)\n\n"+
					"解决方案:\n"+
					"  1. 激活网卡: sudo ip link set %s up\n"+
					"  2. 或使用其他网卡: ksubdomain --eth <网卡名>\n"+
					"  3. 查看可用网卡: ip link show\n"+
					"  4. WSL2 通常使用 eth0,尝试: --eth eth0\n",
					devicename, devicename,
				)
			} else {
				solution = fmt.Sprintf(
					"网卡 %s 未激活\n\n"+
					"解决方案:\n"+
					"  1. Linux:   sudo ip link set %s up\n"+
					"  2. 或使用其他网卡: ksubdomain --eth <网卡名>\n"+
					"  3. 查看可用网卡: ip link show 或 ifconfig -a\n",
					devicename, devicename,
				)
			}
			gologger.Fatalf(solution)
			return nil, fmt.Errorf("网卡未激活: %s", devicename)
		}
		
		// 情况2: 权限不足
		if strings.Contains(errMsg, "permission denied") || strings.Contains(errMsg, "Operation not permitted") {
			solution := fmt.Sprintf(
				"权限不足,无法访问网卡 %s\n\n"+
				"解决方案:\n"+
				"  运行: sudo %s [参数...]\n",
				devicename, os.Args[0],
			)
			gologger.Fatalf(solution)
			return nil, fmt.Errorf("权限不足: %s", devicename)
		}
		
		// 情况3: 网卡不存在
		if strings.Contains(errMsg, "No such device") || strings.Contains(errMsg, "doesn't exist") {
			solution := fmt.Sprintf(
				"网卡 %s 不存在\n\n"+
				"解决方案:\n"+
				"  1. 查看可用网卡:\n"+
				"     Linux/WSL: ip link show\n"+
				"     macOS:     ifconfig -a\n"+
				"  2. 使用正确的网卡名: ksubdomain --eth <网卡名>\n"+
				"  3. 常见网卡名:\n"+
				"     Linux: eth0, ens33, wlan0\n"+
				"     macOS: en0, en1\n"+
				"     WSL2:  eth0\n",
				devicename,
			)
			gologger.Fatalf(solution)
			return nil, fmt.Errorf("网卡不存在: %s", devicename)
		}
		
		// 其他错误
		gologger.Fatalf("pcap初始化失败: %s\n详细错误: %s\n", devicename, errMsg)
		return nil, err
	}
	
	return handle, nil
}
