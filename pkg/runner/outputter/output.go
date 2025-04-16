package outputter

import (
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
)

type Output interface {
	WriteDomainResult(domain result.Result) error
	Close() error
}
