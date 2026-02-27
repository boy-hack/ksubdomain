package output

import (
	"encoding/csv"
	"os"

	"github.com/boy-hack/ksubdomain/v2/pkg/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
	"github.com/boy-hack/ksubdomain/v2/internal/utils"
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

func (f *CsvOutput) Close() error {
	gologger.Infof("Writing CSV file: %s\n", f.filename)

	// Check result count
	if len(f.domains) == 0 {
		gologger.Infof("No subdomain results found, CSV file will be empty\n")
		return nil
	}

	results := utils.WildFilterOutputResult(f.wildFilterMode, f.domains)
	gologger.Infof("Filtered result count: %d\n", len(results))

	// Check filtered results
	if len(results) == 0 {
		gologger.Infof("No valid results after wildcard filtering, CSV file will be empty\n")
		return nil
	}

	// Create CSV file
	file, err := os.Create(f.filename)
	if err != nil {
		gologger.Errorf("Failed to create CSV file: %v", err)
		return err
	}
	defer file.Close()

	// Create CSV writer
	writer := csv.NewWriter(file)

	// Write CSV header
	err = writer.Write([]string{"Subdomain", "Answers"})
	if err != nil {
		gologger.Errorf("Failed to write CSV header: %v", err)
		return err
	}

	// Write data rows
	for _, result := range results {
		// Convert Answers array to a single string separated by semicolons
		answersStr := ""
		if len(result.Answers) > 0 {
			answersStr = result.Answers[0]
			for i := 1; i < len(result.Answers); i++ {
				answersStr += ";" + result.Answers[i]
			}
		}

		err = writer.Write([]string{result.Subdomain, answersStr})
		if err != nil {
			gologger.Errorf("Failed to write CSV data row: %v", err)
			continue
		}
	}
	writer.Flush()
	gologger.Infof("CSV file written successfully, %d records written", len(results))
	return nil
}
