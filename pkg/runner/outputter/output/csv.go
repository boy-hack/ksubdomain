package output

import (
	"encoding/csv"
	"os"

	"github.com/boy-hack/ksubdomain/pkg/runner/result"
	"github.com/boy-hack/ksubdomain/pkg/utils"
)

type CsvOutput struct {
	domains        []result.Result
	filename       string
	wildFilterMode string
}

func NewCsvOutput(filename string, wildFilterMode string) *CsvOutput {
	f := new(CsvOutput)
	f.domains = make([]result.Result, 0)
	f.filename = filename
	f.wildFilterMode = wildFilterMode
	return f
}

func (f *CsvOutput) WriteDomainResult(domain result.Result) error {
	f.domains = append(f.domains, domain)
	return nil
}

func (f *CsvOutput) Close() {
}

func (f *CsvOutput) Finally() error {
	results := utils.WildFilterOutputResult(f.wildFilterMode, f.domains)

	// 创建CSV文件
	file, err := os.Create(f.filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// 创建CSV写入器
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入CSV头部
	err = writer.Write([]string{"Subdomain", "Answers"})
	if err != nil {
		return err
	}

	// 写入数据行
	for _, result := range results {
		// 将Answers数组转换为单个字符串，用分号分隔
		answersStr := ""
		if len(result.Answers) > 0 {
			answersStr = result.Answers[0]
			for i := 1; i < len(result.Answers); i++ {
				answersStr += ";" + result.Answers[i]
			}
		}

		err = writer.Write([]string{result.Subdomain, answersStr})
		if err != nil {
			return err
		}
	}

	return nil
}
