package outputter

import (
	"github.com/boy-hack/ksubdomain/core"
	"github.com/boy-hack/ksubdomain/core/gologger"
	"github.com/boy-hack/ksubdomain/runner"
	"strings"
)

type ScreenOutput struct {
	windowsWidth int
}

func NewScreenOutput() (*ScreenOutput, error) {
	windowsWidth := core.GetWindowWith()
	s := new(ScreenOutput)
	s.windowsWidth = windowsWidth
	return s, nil
}
func (s *ScreenOutput) WriteDomainResult(domain runner.Result) error {
	var domains []string = []string{domain.Subdomain}
	for _, item := range domain.Answers {
		domains = append(domains, item)
	}
	msg := strings.Join(domains, "=>")
	screenWidth := s.windowsWidth - len(msg) - 1
	if s.windowsWidth > 0 && screenWidth > 0 {
		gologger.Silentf("\r%s% *s\n", msg, screenWidth, "")
	} else {
		gologger.Silentf("\r%s\n", msg)
	}
	return nil
}
func (s *ScreenOutput) Close() {

}
