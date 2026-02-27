package output

import (
	"strings"

	"github.com/boy-hack/ksubdomain/v2/pkg/core"
	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
)

type ScreenOutput struct {
	windowsWidth int
	silent       bool
	onlyDomain   bool // Fix Issue #67: output domain name only
}

// NewScreenOutput creates a screen output handler.
// Fix Issue #67: supports optional onlyDomain parameter.
func NewScreenOutput(silent bool, onlyDomain ...bool) (*ScreenOutput, error) {
	windowsWidth := core.GetWindowWith()
	s := new(ScreenOutput)
	s.windowsWidth = windowsWidth
	s.silent = silent
	// Support optional onlyDomain parameter (backward compatible)
	if len(onlyDomain) > 0 {
		s.onlyDomain = onlyDomain[0]
	}
	return s, nil
}

func (s *ScreenOutput) WriteDomainResult(domain result.Result) error {
	var msg string

	// Fix Issue #67: support domain-only output mode
	if s.onlyDomain {
		// Output domain name only, without IP or other records
		msg = domain.Subdomain
	} else {
		// Full output: domain => record1 => record2
		var domains []string = []string{domain.Subdomain}
		for _, item := range domain.Answers {
			domains = append(domains, item)
		}
		msg = strings.Join(domains, " => ")
	}

	if !s.silent {
		screenWidth := s.windowsWidth - len(msg) - 1
		gologger.Silentf("\r%s% *s\n", msg, screenWidth, "")
	} else {
		gologger.Silentf("\r%s\n", msg)
	}
	return nil
}

func (s *ScreenOutput) Close() error {
	return nil
}
