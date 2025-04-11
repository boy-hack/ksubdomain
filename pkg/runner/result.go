package runner

import (
	"github.com/boy-hack/ksubdomain/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/pkg/core/predict"
	"github.com/boy-hack/ksubdomain/pkg/runner/result"
)

func (r *Runner) handleResult() {
	// 缓冲结果，用于泛解析过滤
	var bufferedResults []result.Result

	// 处理接收到的DNS响应
	for res := range r.recver {
		// 添加到缓冲区
		bufferedResults = append(bufferedResults, res)

		// 当缓冲区足够大或接收完毕时，进行泛解析过滤
		if len(bufferedResults) >= 1000 {
			// 处理当前批次的结果
			r.processBufferedResults(bufferedResults)
			// 清空缓冲区
			bufferedResults = []result.Result{}
		}
	}

	// 处理最后剩余的结果
	if len(bufferedResults) > 0 {
		r.processBufferedResults(bufferedResults)
	}
}

// processBufferedResults 处理缓冲的结果
func (r *Runner) processBufferedResults(results []result.Result) {
	// 输出过滤后的结果
	isWildCard := r.options.WildcardFilterMode != "none"
	for _, res := range results {
		if isWildCard {
			if checkWildIps(r.options.WildIps, res.Answers) {
				continue
			}
		}
		for _, out := range r.options.Writer {
			_ = out.WriteDomainResult(res)
		}
		r.printStatus()
	}
	if r.options.Predict {
		err := r.predict(results)
		if err != nil {
			gologger.Errorf("predict failed: %v", err)
		}
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

func (r *Runner) predict(ress []result.Result) error {
	buf := predictWrite{
		sender: r.sender,
	}
	for _, res := range ress {
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
