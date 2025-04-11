package runner

import (
	"github.com/boy-hack/ksubdomain/pkg/core/predict"
	"github.com/boy-hack/ksubdomain/pkg/runner/result"
)

func (r *Runner) handleResult() {
	isWildCard := r.options.WildcardFilterMode != "none"
	cacheResult := make([]result.Result, 0)
	for res := range r.recver {
		if isWildCard {
			if checkWildIps(r.options.WildIps, res.Answers) {
				continue
			}
		}
		for _, out := range r.options.Writer {
			_ = out.WriteDomainResult(res)
		}
		r.printStatus()
		if r.options.Predict {
			cacheResult = append(cacheResult, res)
			if len(cacheResult) > 300 {
				go r.predict(cacheResult)
				cacheResult = make([]result.Result, 0)
			}
		}
	}
	if r.options.Predict && len(cacheResult) > 0 {
		r.predict(cacheResult)
	}
}

type predictWrite struct {
	sender chan string
}

func (o *predictWrite) Write(p []byte) (n int, err error) {
	domain := string(p)
	o.sender <- domain
	return len(p), nil
}

func (r *Runner) predict(results []result.Result) error {
	buf := predictWrite{
		sender: r.sender,
	}
	for _, res := range results {
		_, err := predict.PredictDomains(res.Subdomain, &buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkWildIps(wildIps []string, ip []string) bool {
	for _, w := range wildIps {
		for _, i := range ip {
			if w == i {
				return true
			}
		}
	}
	return false
}
