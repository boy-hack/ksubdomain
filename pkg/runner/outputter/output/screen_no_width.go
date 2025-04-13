package output

import (
	"strings"

	"github.com/boy-hack/ksubdomain/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/pkg/runner/result"
)

type ScreenOutputNoWidth struct {
	silent bool
}

func NewScreenOutputNoWidth(silent bool) (*ScreenOutputNoWidth, error) {
	return &ScreenOutputNoWidth{silent: silent}, nil
}
func (s *ScreenOutputNoWidth) WriteDomainResult(domain result.Result) error {
	var msg string
	var domains []string = []string{domain.Subdomain}
	for _, item := range domain.Answers {
		domains = append(domains, item)
	}
	msg = strings.Join(domains, " => ")
	if !s.silent {
		gologger.Infof("%s\n", msg)
	} else {
		gologger.Silentf("%s\n", domain.Subdomain)
	}
	return nil
}
func (s *ScreenOutputNoWidth) Close() {

}

func (s *ScreenOutputNoWidth) Finally() error {
	return nil
}
