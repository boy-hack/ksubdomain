package main

import (
	"context"
	"fmt"
	"net"
	"time"
)

func DnsLookUp(address string, dnserver string) ([]string, error) {
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(10000),
			}
			return d.DialContext(ctx, network, dnserver+":53")
		},
	}
	return r.LookupHost(context.Background(), address)
}

func main() {

	defaultDns := []string{
		"223.5.5.5",
		"223.6.6.6",
		"180.76.76.76",
		"119.29.29.29",
		"182.254.116.116",
		"114.114.114.115",
		"8.8.8.8",
		"1.1.1.1",
	}
	for _, dns := range defaultDns {
		s, err := DnsLookUp("www.google.com", dns)
		if err != nil {
			_ = fmt.Errorf("dns server:%s error", dns)
		} else {
			fmt.Println(dns, s)
		}
	}

}
