package utils

import (
	"github.com/boy-hack/ksubdomain/pkg/runner/result"
)

// WildFilterOutputResult 泛解析过滤结果
func WildFilterOutputResult(outputType string, results []result.Result) []result.Result {
	if outputType == "none" {
		return results
	} else if outputType == "basic" {
		return FilterWildCard(results)
	} else if outputType == "advanced" {
		return FilterWildCardAdvanced(results)
	}
	return nil
}

// FilterWildCard 基础过滤实现
func FilterWildCard(results []result.Result) []result.Result {
	// 这里采用简化实现
	// 实际项目中应根据需要实现更复杂的过滤逻辑
	return results
}

// FilterWildCardAdvanced 高级过滤实现
func FilterWildCardAdvanced(results []result.Result) []result.Result {
	// 这里采用简化实现
	// 实际项目中应根据需要实现更复杂的过滤逻辑
	return results
}
