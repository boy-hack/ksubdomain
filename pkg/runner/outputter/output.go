package outputter

import (
	"github.com/boy-hack/ksubdomain/pkg/runner"
	"github.com/boy-hack/ksubdomain/pkg/runner/result"
)

type Output interface {
	WriteDomainResult(domain result.Result) error
	Finally() error
	Close()
}

func WildFilterOutputResult(outputType string, results []result.Result) []result.Result {
	if outputType == "none" {
		return results
	} else if outputType == "basic" {
		return runner.FilterWildCard(results)
	} else if outputType == "advanced" {
		return runner.FilterWildCardAdvanced(results)
	}
	return nil
}
