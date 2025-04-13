package runner

import (
	"fmt"
	"sync"

	"github.com/boy-hack/ksubdomain/pkg/core/predict"
	"github.com/boy-hack/ksubdomain/pkg/runner/result"
)

func (r *Runner) handleResult() {
	isWildCard := r.options.WildcardFilterMode != "none"
	cacheResult := make([]result.Result, 0)
	var wg sync.WaitGroup

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
				resultCopy := make([]result.Result, len(cacheResult))
				copy(resultCopy, cacheResult)

				wg.Add(1)
				go func(results []result.Result) {
					defer wg.Done()
					_ = r.predict(results)
				}(resultCopy)

				cacheResult = make([]result.Result, 0)
			}
		}
	}

	if r.options.Predict && len(cacheResult) > 0 {
		_ = r.predict(cacheResult)
	}

	wg.Wait()
}

type predictWrite struct {
	sender chan string
	mu     sync.Mutex
}

func (o *predictWrite) Write(p []byte) (n int, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	domain := string(p)
	o.sender <- domain
	return len(p), nil
}

var predictMutex sync.Mutex

func (r *Runner) predict(results []result.Result) error {
	predictMutex.Lock()
	defer predictMutex.Unlock()

	if r.sender == nil {
		return fmt.Errorf("sender通道未初始化")
	}

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
