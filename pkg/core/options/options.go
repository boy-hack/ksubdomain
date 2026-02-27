package options

import (
	device2 "github.com/boy-hack/ksubdomain/v2/pkg/device"
	"strconv"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/processbar"
)

type OptionMethod string

const (
	VerifyType OptionMethod = "verify"
	EnumType   OptionMethod = "enum"
	TestType   OptionMethod = "test"
)

type Options struct {
	Rate               int64              // Packet sending rate per second
	Domain             chan string        // Domain input channel
	Resolvers          []string           // DNS resolvers
	Silent             bool               // Silent mode
	TimeOut            int                // Timeout in seconds
	Retry              int                // Maximum retry count
	Method             OptionMethod       // verify / enum / test mode
	Writer             []outputter.Output // Output handlers
	ProcessBar         processbar.ProcessBar
	EtherInfo          *device2.EtherTable // Network interface info
	SpecialResolvers   map[string][]string // DNS resolvers for specific domains
	WildcardFilterMode string              // Wildcard filter mode: "basic", "advanced", "none"
	WildIps            []string
	Predict            bool // Enable prediction mode
}

func Band2Rate(bandWith string) int64 {
	suffix := string(bandWith[len(bandWith)-1])
	rate, _ := strconv.ParseInt(string(bandWith[0:len(bandWith)-1]), 10, 64)
	switch suffix {
	case "G":
		fallthrough
	case "g":
		rate *= 1000000000
	case "M":
		fallthrough
	case "m":
		rate *= 1000000
	case "K":
		fallthrough
	case "k":
		rate *= 1000
	default:
		gologger.Fatalf("unknown bandwith suffix '%s' (supported suffixes are G,M and K)\n", suffix)
	}
	packSize := int64(80) // A DNS packet is approximately 74 bytes
	rate = rate / packSize
	return rate
}
func GetResolvers(resolvers []string) []string {
	// handle resolver
	var rs []string
	if resolvers != nil {
		for _, resolver := range resolvers {
			rs = append(rs, resolver)
		}
	} else {
		defaultDns := []string{
			"1.1.1.1",
			"8.8.8.8",
			"180.76.76.76", // Baidu Public DNS
			"180.184.1.1",  // Volcengine
			"180.184.2.2",
		}
		rs = defaultDns
	}
	return rs
}

func (opt *Options) Check() {
	if opt.Silent {
		gologger.MaxLevel = gologger.Silent
	}
}
