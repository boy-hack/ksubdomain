package output

import (
	"strings"

	"github.com/boy-hack/ksubdomain/pkg/core"
	"github.com/boy-hack/ksubdomain/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/pkg/runner/result"
)

type ScreenOutput struct {
	windowsWidth int
	silent       bool
}

func NewScreenOutput(silent bool) (*ScreenOutput, error) {
	windowsWidth := core.GetWindowWith()
	s := new(ScreenOutput)
	s.windowsWidth = windowsWidth
	s.silent = silent
	return s, nil
}

func (s *ScreenOutput) WriteDomainResult(domain result.Result) error {
	var msg string
	var domains []string = []string{domain.Subdomain}
	for _, item := range domain.Answers {
		domains = append(domains, item)
	}
	msg = strings.Join(domains, " => ")
	if !s.silent {
		screenWidth := s.windowsWidth - len(msg) - 1
		if s.windowsWidth > 0 && screenWidth > 0 {
			gologger.Silentf("\r%s% *s\n", msg, screenWidth, "")
		} else {
			gologger.Silentf("\r%s\n", domain.Subdomain)
		}
	}
	return nil
}

func (s *ScreenOutput) Close() {

}

func (s *ScreenOutput) Finally() error {
	return nil
}
