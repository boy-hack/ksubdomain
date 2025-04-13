package runner

import (
	"fmt"
	"sync"

	"github.com/boy-hack/ksubdomain/pkg/core/predict"
	"github.com/boy-hack/ksubdomain/pkg/runner/result"
)

func (r *Runner) handleResult(predictChanel chan string) {
	isWildCard := r.options.WildcardFilterMode != "none"
	// cacheResult := make([]result.Result, 0)
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
		// todo: 解决predict模式 go routine 阻塞问题
		// if r.options.Predict {
		// 	r.predict(res, predictChanel)
		// }
	}
	wg.Wait()
}

func (r *Runner) predict(res result.Result, predictChanel chan string) error {

	if r.sender == nil {
		return fmt.Errorf("sender通道未初始化")
	}

	_, err := predict.PredictDomains(res.Subdomain, predictChanel)
	if err != nil {
		return err
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
