package outputter

import (
	"github.com/boy-hack/ksubdomain/pkg/runner/result"
)

type Output interface {
	WriteDomainResult(domain result.Result) error
	Close()
}
