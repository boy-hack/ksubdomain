package runner

import (
	"github.com/boy-hack/ksubdomain/core/options"
	"path/filepath"
	"testing"
)

func TestVerify(t *testing.T) {
	filename, _ := filepath.Abs("../test/data/verify.txt")

	opt := &options.Options{
		Rate:         options.Band2Rate("1m"),
		Domain:       nil,
		FileName:     filename,
		Resolvers:    options.GetResolvers(""),
		Output:       "",
		Silent:       false,
		Stdin:        false,
		SkipWildCard: false,
		TimeOut:      3,
		Retry:        3,
		Method:       "verify",
	}
	opt.Check()
	r, err := New(opt)
	if err != nil {
		t.Fatal(err)
	}
	r.RunEnumeration()
	r.Close()
}

func TestEnum(t *testing.T) {
	opt := &options.Options{
		Rate:         options.Band2Rate("1m"),
		Domain:       []string{"baidu.com"},
		FileName:     "",
		Resolvers:    options.GetResolvers(""),
		Output:       "",
		Silent:       false,
		Stdin:        false,
		SkipWildCard: false,
		TimeOut:      3,
		Retry:        3,
		Method:       "enum",
	}
	opt.Check()
	r, err := New(opt)
	if err != nil {
		t.Fatal(err)
	}
	r.RunEnumeration()
	r.Close()
}
