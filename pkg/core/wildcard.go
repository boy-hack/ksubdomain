package core

import "net"

func IsWildCard(domain string) (bool, []net.IP) {
	ret := []net.IP{}
	for i := 0; i < 4; i++ {
		subdomain := RandomStr(8) + "." + domain
		ips, err := net.LookupIP(subdomain)
		if err != nil {
			continue
		}
		ret = append(ret, ips...)
	}
	if len(ret) == 0 {
		return true, nil
	}
	return false, ret
}
