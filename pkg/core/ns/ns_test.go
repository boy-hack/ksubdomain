package ns

import "testing"

func TestLookupNS(t *testing.T) {
	ns, ips, err := LookupNS("hacking8.com", "1.1.1.1")
	if err != nil {
		t.Fatalf(err.Error())
	}
	for _, n := range ns {
		t.Log(n)
	}
	for _, ip := range ips {
		t.Log(ip)
	}
}
