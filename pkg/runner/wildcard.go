package runner

import (
	"github.com/boy-hack/ksubdomain/v2/pkg/core"
	"net"
)

func IsWildCard(domain string) (bool, []string) {
	var ret []string
	for i := 0; i < 4; i++ {
		subdomain := core.RandomStr(6) + "." + domain
		ips, err := net.LookupIP(subdomain)
		if err != nil {
			continue
		}
		for _, ip := range ips {
			if ip.To4() != nil {
				ret = append(ret, ip.String())
			}
		}
	}
	if len(ret) == 0 {
		return true, nil
	}
	return false, ret
}
