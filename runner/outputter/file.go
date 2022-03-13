package outputter

import (
	"github.com/boy-hack/ksubdomain/runner"
	"os"
	"strings"
)

type FileOutPut struct {
	output *os.File
}

func NewFileOutput(filename string) (*FileOutPut, error) {
	output, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return nil, err
	}
	f := new(FileOutPut)
	f.output = output
	return f, err
}
func (f *FileOutPut) WriteDomainResult(domain runner.Result) error {
	var domains []string = []string{domain.Subdomain}
	for _, item := range domain.Answers {
		domains = append(domains, item)
	}
	msg := strings.Join(domains, "=>")
	_, err := f.output.WriteString(msg + "\n")
	return err
}
func (f *FileOutPut) Close() error {
	return f.output.Close()
}
