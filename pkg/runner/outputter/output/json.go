package output

import (
	"encoding/json"
	"github.com/boy-hack/ksubdomain/pkg/runner/outputter"
	"github.com/boy-hack/ksubdomain/pkg/runner/result"
	"os"
)

type JsonOutPut struct {
	domains        []result.Result
	filename       string
	wildFilterMode string
}

func NewJsonOutput(filename string, wildFilterMode string) *JsonOutPut {
	f := new(JsonOutPut)
	f.domains = make([]result.Result, 0)
	f.filename = filename
	f.wildFilterMode = wildFilterMode
	return f
}

func (f *JsonOutPut) WriteDomainResult(domain result.Result) error {
	f.domains = append(f.domains, domain)
	return nil
}

func (f *JsonOutPut) Close() {
}

func (f *JsonOutPut) Finally() error {
	results := outputter.WildFilterOutputResult(f.wildFilterMode, f.domains)
	v, err := json.Marshal(results)
	if err != nil {
		return err
	}
	err = os.WriteFile(f.filename, v, 0664)
	return err
}
