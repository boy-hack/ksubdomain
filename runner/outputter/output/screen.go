package output

import (
	"encoding/json"
	"github.com/boy-hack/ksubdomain/core"
	"github.com/boy-hack/ksubdomain/core/gologger"
	"github.com/boy-hack/ksubdomain/runner/result"
	"strings"
)

type ScreenOutput struct {
	windowsWidth int
	onlyDomain   bool
}

func NewScreenOutput(onlyDomain bool) (*ScreenOutput, error) {
	windowsWidth := core.GetWindowWith()
	s := new(ScreenOutput)
	s.windowsWidth = windowsWidth
	s.onlyDomain = onlyDomain
	return s, nil
}

func (s *ScreenOutput) WriteDomainResult(domain result.Result, jsonFormat bool) error {
	var msg strings.Builder
	if jsonFormat {
		content, err := json.Marshal(domain)
		if err != nil {
			return err
		}
		msg.Write(content)
	} else {
		msg.WriteString(domain.Subdomain)
		var domains []string = []string{domain.Subdomain}
		for _, item := range domain.Answers {
			domains = append(domains, item)
		}
		msg.WriteString(strings.Join(domains, " => "))
	}

	gologger.Silentf("\r%s\n", msg.String())
	return nil
}
func (s *ScreenOutput) Close() {

}
