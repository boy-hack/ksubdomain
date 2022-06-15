package runner

import (
	"fmt"
	"github.com/boy-hack/ksubdomain/core/gologger"
	"math"
	"sort"
	"strconv"
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

// 过滤掉数据的泛解析记录
func FilterWildCardFromFile(origin map[string][]string) (ret map[string][]string) {
	gologger.Debugf("泛解析处理中...\n")
	staticRecords := make(map[string]int) // 统计每个记录的个数

	sum := 0
	for _, records := range origin {
		for _, record := range records {
			_, ok := staticRecords[record]
			if !ok {
				staticRecords[record] = 0
			}
			staticRecords[record] += 1
			sum += 1
		}
	}
	sortRecords := sortMapByValue(staticRecords)

	recordV := make(map[string]int) // 记录每个解析记录的权重值
	index := 0
	for _, v := range sortRecords {
		index += 1
		quan := 0.0
		if index <= 15 && v.Value > 1000 {
			quan, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", 15-float64(index)/15*80), 64)
		}
		if float64(v.Value/sum) > 0.5 {
			quan += float64(v.Value/sum) * 90
		} else {
			quan += float64(v.Value/sum) * 100
		}
		if v.Value > 100 && quan < 10 {
			quan += 10
		}
		if quan > 100 {
			quan = 100
		}
		recordV[v.Key] = int(math.Ceil(quan))
	}
	// 根据权值过滤域名
	ret = make(map[string][]string)
	for domain, records := range origin {
		quan := 0
		for _, r := range records {
			_quan, _ := recordV[r]
			quan += _quan
		}
		avg := quan / len(records)
		if avg > 60 {
			continue
		}
		n1 := domain
		n2 := records
		ret[n1] = n2
	}
	return
}
