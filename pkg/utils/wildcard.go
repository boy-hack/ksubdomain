package utils

import (
	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
	"sort"
	"strings"
)

type Pair struct {
	Key   string
	Value int
}
type PairList []Pair

func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value > p[j].Value }

// A function to turn a map into a PairList, then sort and return it.
func sortMapByValue(m map[string]int) PairList {
	p := make(PairList, len(m))
	i := 0
	for k, v := range m {
		p[i] = Pair{k, v}
		i++
	}
	sort.Sort(p)
	return p
}

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

// FilterWildCard 基于Result类型数据过滤泛解析
// 传入参数为[]result.Result，返回过滤后的[]result.Result
// 通过分析整体结果，对解析记录中相同的ip进行阈值判断，超过则丢弃该结果
func FilterWildCard(results []result.Result) []result.Result {
	if len(results) == 0 {
		return results
	}

	gologger.Debugf("泛解析处理中，共 %d 条记录...\n", len(results))

	// 统计每个IP出现的次数
	ipFrequency := make(map[string]int)
	// 记录IP到域名的映射关系
	ipToDomains := make(map[string][]string)
	// 域名计数
	totalDomains := len(results)

	// 第一遍扫描，统计IP频率
	for _, res := range results {
		for _, answer := range res.Answers {
			// 跳过非IP的记录(CNAME等)
			if !strings.HasPrefix(answer, "CNAME ") && !strings.HasPrefix(answer, "NS ") &&
				!strings.HasPrefix(answer, "TXT ") && !strings.HasPrefix(answer, "PTR ") {
				ipFrequency[answer]++
				ipToDomains[answer] = append(ipToDomains[answer], res.Subdomain)
			}
		}
	}

	// 按出现频率排序IP
	sortedIPs := sortMapByValue(ipFrequency)

	// 确定疑似泛解析的IP列表
	// 使用两个标准：
	// 1. IP解析超过总域名数量的特定百分比(动态阈值)
	// 2. 该IP解析的子域名数量超过特定阈值
	suspiciousIPs := make(map[string]bool)

	for _, pair := range sortedIPs {
		ip := pair.Key
		count := pair.Value

		// 计算该IP解析占总体的百分比
		percentage := float64(count) / float64(totalDomains) * 100

		// 动态阈值：根据总域名数量调整
		// 域名数量少时阈值较高，域名数量多时阈值较低
		var threshold float64
		if totalDomains < 100 {
			threshold = 30 // 如果域名总数小于100，阈值设为30%
		} else if totalDomains < 1000 {
			threshold = 20 // 如果域名总数在100-1000，阈值设为20%
		} else {
			threshold = 10 // 如果域名总数超过1000，阈值设为10%
		}

		// 绝对数量阈值
		absoluteThreshold := 70

		// 如果超过阈值，标记为可疑IP
		if percentage > threshold || count > absoluteThreshold {
			gologger.Debugf("发现可疑泛解析IP: %s (解析了 %d 个域名, %.2f%%)\n",
				ip, count, percentage)
			suspiciousIPs[ip] = true
		}
	}

	// 第二遍扫描，过滤结果
	var filteredResults []result.Result

	for _, res := range results {
		// 检查该域名的所有IP是否均为可疑IP
		// 如果有不可疑的IP，保留该记录
		validRecord := false
		var filteredAnswers []string

		for _, answer := range res.Answers {
			// 保留所有非IP记录(如CNAME)
			if strings.HasPrefix(answer, "CNAME ") || strings.HasPrefix(answer, "NS ") ||
				strings.HasPrefix(answer, "TXT ") || strings.HasPrefix(answer, "PTR ") {
				validRecord = true
				filteredAnswers = append(filteredAnswers, answer)
			} else if !suspiciousIPs[answer] {
				// 保留不在可疑IP列表中的IP
				validRecord = true
				filteredAnswers = append(filteredAnswers, answer)
			}
		}

		if validRecord && len(filteredAnswers) > 0 {
			filteredRes := result.Result{
				Subdomain: res.Subdomain,
				Answers:   filteredAnswers,
			}
			filteredResults = append(filteredResults, filteredRes)
		}
	}

	gologger.Infof("泛解析过滤完成，从 %d 条记录中过滤出 %d 条有效记录\n",
		totalDomains, len(filteredResults))

	return filteredResults
}

// FilterWildCardAdvanced 提供更高级的泛解析检测算法
// 使用多种启发式方法和特征检测来识别泛解析
func FilterWildCardAdvanced(results []result.Result) []result.Result {
	if len(results) == 0 {
		return results
	}

	gologger.Debugf("高级泛解析检测开始，共 %d 条记录...\n", len(results))

	// 统计IP出现频率
	ipFrequency := make(map[string]int)
	// 统计每个IP解析的不同子域名前缀数量
	ipPrefixVariety := make(map[string]map[string]bool)
	// 统计IP解析的不同顶级域数量
	ipTLDVariety := make(map[string]map[string]bool)
	// 记录IP到域名的映射
	ipToDomains := make(map[string][]string)
	// 记录CNAME信息
	cnameRecords := make(map[string][]string)

	totalDomains := len(results)

	// 第一轮：收集统计信息
	for _, res := range results {
		subdomain := res.Subdomain
		parts := strings.Split(subdomain, ".")

		// 提取顶级域和前缀
		prefix := ""
		tld := ""
		if len(parts) > 1 {
			prefix = parts[0]
			tld = strings.Join(parts[1:], ".")
		} else {
			prefix = subdomain
			tld = subdomain
		}

		for _, answer := range res.Answers {
			if strings.HasPrefix(answer, "CNAME ") {
				// 提取CNAME目标
				cnameParts := strings.SplitN(answer, " ", 2)
				if len(cnameParts) == 2 {
					cnameTarget := cnameParts[1]
					cnameRecords[subdomain] = append(cnameRecords[subdomain], cnameTarget)
				}
				continue
			}

			// 只处理IP记录
			if !strings.HasPrefix(answer, "NS ") &&
				!strings.HasPrefix(answer, "TXT ") &&
				!strings.HasPrefix(answer, "PTR ") {
				// 计数IP频率
				ipFrequency[answer]++

				// 初始化IP的前缀集合和TLD集合
				if ipPrefixVariety[answer] == nil {
					ipPrefixVariety[answer] = make(map[string]bool)
				}
				if ipTLDVariety[answer] == nil {
					ipTLDVariety[answer] = make(map[string]bool)
				}

				// 记录这个IP解析了哪些不同的前缀和TLD
				ipPrefixVariety[answer][prefix] = true
				ipTLDVariety[answer][tld] = true

				// 记录IP到域名的映射
				ipToDomains[answer] = append(ipToDomains[answer], subdomain)
			}
		}
	}

	// 按照IP频率排序
	sortedIPs := sortMapByValue(ipFrequency)

	// 识别可疑IP列表
	suspiciousIPs := make(map[string]float64) // IP -> 可疑度分数(0-100)

	for _, pair := range sortedIPs {
		ip := pair.Key
		count := pair.Value

		// 初始可疑度分数
		suspiciousScore := 0.0

		// 因子1: IP频率百分比
		freqPercentage := float64(count) / float64(totalDomains) * 100

		// 因子2: 前缀多样性
		prefixVariety := len(ipPrefixVariety[ip])
		prefixVarietyRatio := float64(prefixVariety) / float64(count) * 100

		// 因子3: TLD多样性
		tldVariety := len(ipTLDVariety[ip])

		// 计算可疑度分数
		// 1. 频率因子
		if freqPercentage > 30 {
			suspiciousScore += 40
		} else if freqPercentage > 10 {
			suspiciousScore += 20
		} else if freqPercentage > 5 {
			suspiciousScore += 10
		}

		// 2. 前缀多样性因子
		// 如果一个IP解析了大量不同前缀的域名，可能是CDN或者泛解析
		if prefixVarietyRatio > 90 && prefixVariety > 10 {
			suspiciousScore += 30
		} else if prefixVarietyRatio > 70 && prefixVariety > 5 {
			suspiciousScore += 20
		}

		// 3. 绝对数量因子
		if count > 100 {
			suspiciousScore += 20
		} else if count > 50 {
			suspiciousScore += 10
		} else if count > 20 {
			suspiciousScore += 5
		}

		// 4. TLD多样性因子 - 如果一个IP解析了多个不同TLD，更可能是合法的
		if tldVariety > 3 {
			suspiciousScore -= 20
		} else if tldVariety > 1 {
			suspiciousScore -= 10
		}

		// 只有当可疑度分数超过阈值时，才标记为可疑IP
		if suspiciousScore >= 35 {
			gologger.Debugf("可疑IP: %s (解析域名数: %d, 占比: %.2f%%, 前缀多样性: %d/%d, 可疑度: %.2f)\n",
				ip, count, freqPercentage, prefixVariety, count, suspiciousScore)
			suspiciousIPs[ip] = suspiciousScore
		}
	}

	// 第二轮：过滤结果
	var filteredResults []result.Result

	// CNAME聚类分析：检测指向相同目标的多个CNAME记录
	cnameTargetCount := make(map[string]int)
	for _, targets := range cnameRecords {
		for _, target := range targets {
			cnameTargetCount[target]++
		}
	}

	// 识别可疑CNAME目标
	suspiciousCnames := make(map[string]bool)
	for cname, count := range cnameTargetCount {
		if count > 5 && float64(count)/float64(totalDomains)*100 > 10 {
			gologger.Debugf("可疑CNAME目标: %s (指向次数: %d)\n", cname, count)
			suspiciousCnames[cname] = true
		}
	}

	// 过滤结果
	for _, res := range results {
		// 检查是否含有可疑CNAME
		hasSuspiciousCname := false
		if targets, ok := cnameRecords[res.Subdomain]; ok {
			for _, target := range targets {
				if suspiciousCnames[target] {
					hasSuspiciousCname = true
					break
				}
			}
		}

		validRecord := !hasSuspiciousCname
		var filteredAnswers []string

		// 处理所有回答
		for _, answer := range res.Answers {
			isIP := !strings.HasPrefix(answer, "CNAME ") &&
				!strings.HasPrefix(answer, "NS ") &&
				!strings.HasPrefix(answer, "TXT ") &&
				!strings.HasPrefix(answer, "PTR ")

			// 保留所有非IP记录但排除可疑CNAME
			if !isIP {
				if strings.HasPrefix(answer, "CNAME ") {
					cnameParts := strings.SplitN(answer, " ", 2)
					if len(cnameParts) == 2 && suspiciousCnames[cnameParts[1]] {
						continue // 跳过可疑CNAME
					}
				}
				validRecord = true
				filteredAnswers = append(filteredAnswers, answer)
			} else {
				// 针对IP记录，根据可疑度评分过滤
				suspiciousScore, isSuspicious := suspiciousIPs[answer]

				// 如果不在可疑IP列表中，或者可疑度较低，则保留
				if !isSuspicious || suspiciousScore < 50 {
					validRecord = true
					filteredAnswers = append(filteredAnswers, answer)
				}
			}
		}

		// 只添加有效记录
		if validRecord && len(filteredAnswers) > 0 {
			filteredRes := result.Result{
				Subdomain: res.Subdomain,
				Answers:   filteredAnswers,
			}
			filteredResults = append(filteredResults, filteredRes)
		}
	}

	gologger.Infof("高级泛解析过滤完成，从 %d 条记录中过滤出 %d 条有效记录\n",
		totalDomains, len(filteredResults))

	return filteredResults
}
