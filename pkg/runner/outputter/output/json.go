package output

import (
	"encoding/json"
	"os"

	"github.com/boy-hack/ksubdomain/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/pkg/runner/result"

	"github.com/boy-hack/ksubdomain/pkg/utils"
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

func (f *JsonOutPut) Close() error {
	gologger.Infof("写入json文件:%s count:%d", f.filename, len(f.domains))
	if len(f.domains) > 0 {
		results := utils.WildFilterOutputResult(f.wildFilterMode, f.domains)
		jsonBytes, err := json.Marshal(results)
		if err != nil {
			return err
		}
		err = os.WriteFile(f.filename, jsonBytes, 0664)
		return err
	}
	return nil
}
