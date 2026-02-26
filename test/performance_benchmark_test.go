// +build performance

package test

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/options"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter"
	output2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter/output"
	processbar2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/processbar"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
)

// Benchmark100kDomains 10万域名性能基准测试
// 参考 README 中的对比测试:
// - 测试环境: 4核CPU, 5M 带宽
// - 字典大小: 10万域名
// - 目标: ~30秒完成扫描
// - 成功率: > 95%
func Benchmark100kDomains(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过性能基准测试 (使用 -tags=performance 运行)")
	}

	// 创建 10 万域名字典
	dictFile := createBenchmarkDict(b, 100000)
	defer os.Remove(dictFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runBenchmark(b, dictFile, 100000)
	}
}

// Benchmark10kDomains 1万域名快速测试
func Benchmark10kDomains(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过性能基准测试")
	}

	dictFile := createBenchmarkDict(b, 10000)
	defer os.Remove(dictFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runBenchmark(b, dictFile, 10000)
	}
}

// Benchmark1kDomains 1千域名基础测试
func Benchmark1kDomains(b *testing.B) {
	dictFile := createBenchmarkDict(b, 1000)
	defer os.Remove(dictFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runBenchmark(b, dictFile, 1000)
	}
}

// createBenchmarkDict 创建测试字典
func createBenchmarkDict(b *testing.B, count int) string {
	b.Helper()

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("ksubdomain_bench_%d.txt", count))
	f, err := os.Create(tmpFile)
	if err != nil {
		b.Fatalf("创建字典文件失败: %v", err)
	}
	defer f.Close()

	writer := bufio.NewWriter(f)

	// 生成域名列表
	// 使用一些真实存在的域名模式,提高测试真实性
	baseDomains := []string{
		"example.com",
		"test.com",
		"demo.org",
		"sample.net",
	}

	prefixes := []string{
		"www", "mail", "ftp", "blog", "shop", "admin",
		"api", "dev", "test", "staging", "prod", "app",
		"web", "mobile", "cdn", "static", "img", "media",
	}

	for i := 0; i < count; i++ {
		var domain string

		if i < len(prefixes)*len(baseDomains) {
			// 使用常见前缀
			prefix := prefixes[i%len(prefixes)]
			base := baseDomains[i/len(prefixes)%len(baseDomains)]
			domain = fmt.Sprintf("%s.%s", prefix, base)
		} else {
			// 生成随机子域名
			base := baseDomains[i%len(baseDomains)]
			domain = fmt.Sprintf("subdomain%d.%s", i, base)
		}

		_, err := writer.WriteString(domain + "\n")
		if err != nil {
			b.Fatalf("写入字典失败: %v", err)
		}
	}

	err = writer.Flush()
	if err != nil {
		b.Fatalf("刷新字典失败: %v", err)
	}

	b.Logf("创建字典: %s (%d 个域名)", tmpFile, count)
	return tmpFile
}

// runBenchmark 运行性能测试
func runBenchmark(b *testing.B, dictFile string, expectedCount int) {
	b.Helper()

	// 打开字典文件
	file, err := os.Open(dictFile)
	if err != nil {
		b.Fatalf("打开字典失败: %v", err)
	}
	defer file.Close()

	// 读取所有域名到通道
	domainChan := make(chan string, 10000)
	go func() {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			domainChan <- scanner.Text()
		}
		close(domainChan)
	}()

	// 收集结果
	results := &perfOutputter{
		results:     make([]result.Result, 0, expectedCount),
		startTime:   time.Now(),
		totalDomains: expectedCount,
	}

	// 配置扫描参数 (参考 README 的测试配置)
	opt := &options.Options{
		Rate:       options.Band2Rate("5m"), // 5M 带宽
		Domain:     domainChan,
		Resolvers:  options.GetResolvers(nil),
		Silent:     true,
		TimeOut:    6,
		Retry:      3,
		Method:     options.VerifyType,
		Writer:     []outputter.Output{results},
		ProcessBar: &processbar2.FakeProcess{},
		EtherInfo:  options.GetDeviceConfig(options.GetResolvers(nil)),
	}

	// 创建上下文 (5分钟超时,足够10万域名)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 记录开始时间
	startTime := time.Now()

	// 运行扫描
	r, err := runner.New(opt)
	if err != nil {
		b.Fatalf("创建 runner 失败: %v", err)
	}

	r.RunEnumeration(ctx)
	r.Close()

	// 计算性能指标
	elapsed := time.Since(startTime)
	successCount := len(results.results)
	successRate := float64(successCount) / float64(expectedCount) * 100
	domainsPerSecond := float64(expectedCount) / elapsed.Seconds()

	// 报告性能指标
	b.ReportMetric(elapsed.Seconds(), "total_seconds")
	b.ReportMetric(float64(successCount), "success_count")
	b.ReportMetric(successRate, "success_rate_%")
	b.ReportMetric(domainsPerSecond, "domains/sec")

	// 日志输出
	b.Logf("性能测试结果:")
	b.Logf("  - 字典大小: %d 个域名", expectedCount)
	b.Logf("  - 总耗时:   %v", elapsed)
	b.Logf("  - 成功数:   %d", successCount)
	b.Logf("  - 成功率:   %.2f%%", successRate)
	b.Logf("  - 速率:     %.0f domains/s", domainsPerSecond)

	// 性能基准检查 (参考 README: 10万域名 ~30秒)
	if expectedCount == 100000 {
		// 10万域名应该在 60 秒内完成 (给予一定容差)
		if elapsed.Seconds() > 60 {
			b.Logf("⚠️  性能警告: 10万域名耗时 %.1f 秒 (目标 < 60秒)", elapsed.Seconds())
		} else if elapsed.Seconds() <= 30 {
			b.Logf("✅ 性能优秀: 10万域名仅耗时 %.1f 秒 (达到 README 标准)", elapsed.Seconds())
		} else {
			b.Logf("✓  性能良好: 10万域名耗时 %.1f 秒", elapsed.Seconds())
		}
	}
}

// perfOutputter 性能测试输出器
type perfOutputter struct {
	results      []result.Result
	mu           sync.Mutex
	startTime    time.Time
	totalDomains int
	lastReport   time.Time
}

func (p *perfOutputter) WriteDomainResult(r result.Result) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.results = append(p.results, r)

	// 每1000个结果报告一次进度
	if len(p.results)%1000 == 0 {
		elapsed := time.Since(p.startTime)
		rate := float64(len(p.results)) / elapsed.Seconds()
		progress := float64(len(p.results)) / float64(p.totalDomains) * 100

		// 避免频繁输出
		if time.Since(p.lastReport) > time.Second {
			fmt.Printf("\r进度: %d/%d (%.1f%%), 速率: %.0f domains/s, 耗时: %v",
				len(p.results), p.totalDomains, progress, rate, elapsed.Round(time.Second))
			p.lastReport = time.Now()
		}
	}

	return nil
}

func (p *perfOutputter) Close() error {
	elapsed := time.Since(p.startTime)
	fmt.Printf("\n最终结果: %d/%d, 耗时: %v\n",
		len(p.results), p.totalDomains, elapsed.Round(time.Millisecond))
	return nil
}
