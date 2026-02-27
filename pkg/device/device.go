package device

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/gologger"
	"github.com/google/gopacket/pcap"
	"gopkg.in/yaml.v3"
)

// EtherTable stores network interface information
type EtherTable struct {
	SrcIp  net.IP  `yaml:"src_ip"`  // Source IP address
	Device string  `yaml:"device"`  // Network interface device name
	SrcMac SelfMac `yaml:"src_mac"` // Source MAC address
	DstMac SelfMac `yaml:"dst_mac"` // Destination MAC address (usually the gateway)
}

// ReadConfig reads EtherTable configuration from a file
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

// SaveConfig saves EtherTable configuration to a file
func (e *EtherTable) SaveConfig(filename string) error {
	data, err := yaml.Marshal(e)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0666)
}

// isWSL detects whether running in a WSL/WSL2 environment
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

// isDeviceUp checks whether a network interface is active
func isDeviceUp(devicename string) bool {
	iface, err := net.InterfaceByName(devicename)
	if err != nil {
		return false
	}
	// Check the UP flag
	return iface.Flags&net.FlagUp != 0
}

// PcapInit initializes a pcap handle.
// Fix Issue #68: enhanced error messages, especially for WSL2 environments.
// Fix Mac buffer issue: use InactiveHandle to set a larger buffer size.
func PcapInit(devicename string) (*pcap.Handle, error) {
	// Using InactiveHandle allows setting parameters before activation.
	// This is especially important for Mac BPF buffer optimization.
	inactive, err := pcap.NewInactiveHandle(devicename)
	if err != nil {
		gologger.Fatalf("Failed to create pcap handle: %s\n", err.Error())
		return nil, err
	}
	defer inactive.CleanUp()

	// Set snapshot length to 64KB (the original 1024 was too small).
	// DNS packets are usually < 512 bytes, but full Ethernet frames may be larger.
	err = inactive.SetSnapLen(65536)
	if err != nil {
		gologger.Warningf("Failed to set SnapLen: %v\n", err)
	}

	// Set timeout to blocking mode
	err = inactive.SetTimeout(-1 * time.Second)
	if err != nil {
		gologger.Warningf("Failed to set Timeout: %v\n", err)
	}

	// Mac-specific optimization: increase BPF buffer size.
	// Mac's default BPF buffer is small (typically 32KB), which can overflow during high-speed sending.
	// Setting it to 2MB significantly reduces "No buffer space available" errors.
	if runtime.GOOS == "darwin" {
		bufferSize := 2 * 1024 * 1024 // 2MB
		err = inactive.SetBufferSize(bufferSize)
		if err != nil {
			gologger.Warningf("Mac: Failed to set BPF buffer size: %v (will use default)\n", err)
		} else {
			gologger.Infof("Mac: BPF buffer set to %d MB\n", bufferSize/(1024*1024))
		}
	}

	// Set immediate mode (reduce latency).
	// Failure is non-fatal; some platforms may not support it.
	err = inactive.SetImmediateMode(true)
	if err != nil {
		gologger.Debugf("Failed to set immediate mode: %v (non-fatal)\n", err)
	}

	// Activate the handle
	handle, err := inactive.Activate()
	if err != nil {
		// Fix Issue #68: provide detailed error messages and solutions
		errMsg := err.Error()

		// Case 1: Interface is not up
		if strings.Contains(errMsg, "not up") {
			var solution string
			if isWSL() {
				// WSL/WSL2 specific hint
				solution = fmt.Sprintf(
					"Interface %s is not up (WSL/WSL2 environment detected)\n\n"+
						"Solutions:\n"+
						"  1. Bring up the interface: sudo ip link set %s up\n"+
						"  2. Or use another interface: ksubdomain --eth <interface>\n"+
						"  3. List available interfaces: ip link show\n"+
						"  4. WSL2 usually uses eth0, try: --eth eth0\n",
					devicename, devicename,
				)
			} else {
				solution = fmt.Sprintf(
					"Interface %s is not up\n\n"+
						"Solutions:\n"+
						"  1. Linux:   sudo ip link set %s up\n"+
						"  2. Or use another interface: ksubdomain --eth <interface>\n"+
						"  3. List available interfaces: ip link show or ifconfig -a\n",
					devicename, devicename,
				)
			}
			gologger.Fatalf(solution)
			return nil, fmt.Errorf("interface not up: %s", devicename)
		}

		// Case 2: Permission denied
		if strings.Contains(errMsg, "permission denied") || strings.Contains(errMsg, "Operation not permitted") {
			solution := fmt.Sprintf(
				"Permission denied accessing interface %s\n\n"+
					"Solution:\n"+
					"  Run: sudo %s [args...]\n",
				devicename, os.Args[0],
			)
			gologger.Fatalf(solution)
			return nil, fmt.Errorf("permission denied: %s", devicename)
		}

		// Case 3: Interface does not exist
		if strings.Contains(errMsg, "No such device") || strings.Contains(errMsg, "doesn't exist") {
			solution := fmt.Sprintf(
				"Interface %s does not exist\n\n"+
					"Solutions:\n"+
					"  1. List available interfaces:\n"+
					"     Linux/WSL: ip link show\n"+
					"     macOS:     ifconfig -a\n"+
					"  2. Use the correct interface name: ksubdomain --eth <interface>\n"+
					"  3. Common interface names:\n"+
					"     Linux: eth0, ens33, wlan0\n"+
					"     macOS: en0, en1\n"+
					"     WSL2:  eth0\n",
				devicename,
			)
			gologger.Fatalf(solution)
			return nil, fmt.Errorf("interface not found: %s", devicename)
		}

		// Other errors
		gologger.Fatalf("pcap initialization failed: %s\nDetails: %s\n", devicename, errMsg)
		return nil, err
	}

	return handle, nil
}
