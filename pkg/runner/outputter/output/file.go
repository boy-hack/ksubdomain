package output

import (
	"os"
	"strings"

	"github.com/boy-hack/ksubdomain/pkg/runner/result"

	"github.com/boy-hack/ksubdomain/pkg/utils"
)

type FileOutPut struct {
	output         *os.File
	wildFilterMode string
	domains        []result.Result
	filename       string
}

func NewPlainOutput(filename string, wildFilterMode string) (*FileOutPut, error) {
	output, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return nil, err
	}
	f := new(FileOutPut)
	f.output = output
	f.wildFilterMode = wildFilterMode
	f.filename = filename
	return f, err
}
func (f *FileOutPut) WriteDomainResult(domain result.Result) error {
	var msg string
	var domains []string = []string{domain.Subdomain}
	for _, item := range domain.Answers {
		domains = append(domains, item)
	}
	msg = strings.Join(domains, "=>")
	_, err := f.output.WriteString(msg + "\n")
	f.domains = append(f.domains, domain)
	return err
}
func (f *FileOutPut) Close() {
	f.output.Close()
}
func (f *FileOutPut) Finally() error {
	if len(f.domains) > 0 {
		results := utils.WildFilterOutputResult(f.wildFilterMode, f.domains)
		buf := strings.Builder{}
		for _, item := range results {
			buf.WriteString(item.Subdomain + "=>")
			buf.WriteString(strings.Join(item.Answers, "=>"))
			buf.WriteString("\n")
		}
		err := os.WriteFile(f.filename, []byte(buf.String()), 0664)
		return err
	}
	return nil
}
