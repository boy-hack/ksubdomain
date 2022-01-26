package runner

import (
	"ksubdomain/core/options"
	"testing"
)

func TestRunner(t *testing.T) {
	defaultDns := []string{
		"223.5.5.5",
		"223.6.6.6",
		"180.76.76.76",
		"119.29.29.29",
		"182.254.116.116",
		"114.114.114.115",
	}
	o := options.Options{
		Rate:            100,
		Domain:          []string{"i.hacking8.com", "www.hacking8.com", "xxxx.hacking8.com", "www.hacking8.com", "xxxx.hacking8.com", "www.hacking8.com", "xxxx.hacking8.com", "www.hacking8.com", "xxxx.hacking8.com"},
		FileName:        "",
		Resolvers:       defaultDns,
		Output:          "",
		Test:            false,
		ListNetwork:     false,
		Silent:          false,
		TTL:             false,
		Verify:          true,
		Stdin:           false,
		DomainLevel:     0,
		SkipWildCard:    false,
		SubNameFileName: "",
		FilterWildCard:  false,
		TimeOut:         30,
		Retry:           3,
	}
	runner, err := New(&o)
	if err != nil {
		t.Fatal(err)
	}
	runner.RunEnumeration()
	runner.Close()
}
