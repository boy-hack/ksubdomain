package output

import (
	"strings"

	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
)

type BuffOutput struct {
	sb strings.Builder
}

func NewBuffOutput() (*BuffOutput, error) {
	s := &BuffOutput{}
	s.sb = strings.Builder{}
	return s, nil
}

func (b *BuffOutput) WriteDomainResult(domain result.Result) error {
	var domains []string = []string{domain.Subdomain}
	for _, item := range domain.Answers {
		domains = append(domains, item)
	}
	msg := strings.Join(domains, "=>")
	b.sb.WriteString(msg + "\n")
	return nil
}

func (b *BuffOutput) Close() error {
	b.sb.Reset()
	return nil
}

func (b *BuffOutput) Strings() string {
	return b.sb.String()
}
