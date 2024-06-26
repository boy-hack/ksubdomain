package output

import (
	"encoding/json"
	"github.com/boy-hack/ksubdomain/runner/result"
	"strings"
)

type BuffOutput struct {
	sb strings.Builder
}

func NewBuffOutput() (*BuffOutput, error) {
	s := &BuffOutput{}
	s.sb = strings.Builder{}
	return s, nil
}

func (b *BuffOutput) WriteDomainResult(domain result.Result, jsonFormat bool) error {
	if jsonFormat {
		content, err := json.Marshal(domain)
		if err != nil {
			return err
		}
		b.sb.Write(content)
		b.sb.Write([]byte("\n"))
	} else {
		var domains []string = []string{domain.Subdomain}
		for _, item := range domain.Answers {
			domains = append(domains, item)
		}
		msg := strings.Join(domains, "=>")
		b.sb.WriteString(msg + "\n")
	}

	return nil
}
func (b *BuffOutput) Close() {
	b.sb.Reset()
}
func (b *BuffOutput) Strings() string {
	return b.sb.String()
}
