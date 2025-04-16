package runner

import (
	"fmt"
	"sync"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/predict"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
)

// handleResult 处理扫描结果
func (r *Runner) handleResult(predictChan chan string) {
	isWildCard := r.options.WildcardFilterMode != "none"
	var wg sync.WaitGroup
	var predictSignal bool = false

	for res := range r.resultChan {
		// 过滤通配符域名
		if isWildCard {
			if checkWildIps(r.options.WildIps, res.Answers) {
				continue
			}
		}

		// 将结果写入输出器
		for _, out := range r.options.Writer {
			_ = out.WriteDomainResult(res)
		}

		// 预测域名处理
		if r.options.Predict {
			wg.Add(1)
			go func(domain string) {
				defer wg.Done()
				r.predict(res, predictChan)
				if !predictSignal {
					r.predictLoadDone <- struct{}{}
					predictSignal = true
				}
			}(res.Subdomain)
		}
	}
	wg.Wait()
}

// predict 根据已知域名预测新的子域名
func (r *Runner) predict(res result.Result, predictChan chan string) error {
	if r.domainChan == nil {
		return fmt.Errorf("域名通道未初始化")
	}
	_, err := predict.PredictDomains(res.Subdomain, predictChan)
	if err != nil {
		return err
	}
	return nil
}

// checkWildIps 检查是否为通配符IP
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
