package runner

import (
	"github.com/boy-hack/ksubdomain/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/pkg/core/options"
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
			processBufferedResults(r, bufferedResults)
			// 清空缓冲区
			bufferedResults = []result.Result{}
		}
	}

	// 处理最后剩余的结果
	if len(bufferedResults) > 0 {
		processBufferedResults(r, bufferedResults)
	}
}

// processBufferedResults 处理缓冲的结果
func processBufferedResults(r *Runner, results []result.Result) {
	// 根据泛解析过滤选项处理结果
	var filteredResults []result.Result

	if r.options.Method == options.EnumType && len(results) > 0 {
		// 枚举模式下才应用泛解析过滤
		wildFilterMode := r.options.WildcardFilterMode

		switch wildFilterMode {
		case "advanced":
			gologger.Debugf("使用高级泛解析过滤处理 %d 个结果...", len(results))
			filteredResults = FilterWildCardAdvanced(results)
		case "basic":
			gologger.Debugf("使用基础泛解析过滤处理 %d 个结果...", len(results))
			filteredResults = FilterWildCard(results)
		case "none":
			gologger.Debugf("跳过泛解析过滤，共 %d 个结果", len(results))
			filteredResults = results
		default:
			// 默认使用基础过滤
			gologger.Debugf("使用默认泛解析过滤处理 %d 个结果...", len(results))
			filteredResults = FilterWildCard(results)
		}
	} else {
		// 非枚举模式不过滤泛解析
		filteredResults = results
	}

	// 输出过滤后的结果
	for _, res := range filteredResults {
		for _, out := range r.options.Writer {
			_ = out.WriteDomainResult(res)
		}
		r.printStatus()
	}
}
