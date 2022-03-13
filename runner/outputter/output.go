package outputter

import "github.com/boy-hack/ksubdomain/runner"

type Output interface {
	WriteDomainResult(domain runner.Result) error
	Close()
}
