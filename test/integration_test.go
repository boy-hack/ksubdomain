// +build integration

package test

import (
	"context"
	"testing"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/options"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter"
	output2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/outputter/output"
	processbar2 "github.com/boy-hack/ksubdomain/v2/pkg/runner/processbar"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
	"github.com/stretchr/testify/assert"
)

// TestBasicVerification 基础验证测试
func TestBasicVerification(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 已知存在的域名
	domains := []string{
		"www.baidu.com",
		"www.google.com",
		"dns.google",
	}

	domainChan := make(chan string, len(domains))
	for _, domain := range domains {
		domainChan <- domain
	}
	close(domainChan)

	// 收集结果
	results := &testOutputter{results: make([]result.Result, 0)}

	opt := &options.Options{
		Rate:      1000,
		Domain:    domainChan,
		Resolvers: options.GetResolvers(nil),
		Silent:    true,
		TimeOut:   10,
		Retry:     3,
		Method:    options.VerifyType,
		Writer:    []outputter.Output{results},
		ProcessBar: &processbar2.FakeProcess{},
		EtherInfo: options.GetDeviceConfig(options.GetResolvers(nil)),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	r, err := runner.New(opt)
	assert.NoError(t, err)

	r.RunEnumeration(ctx)
	r.Close()

	// 验证结果
	assert.Greater(t, len(results.results), 0, "应该至少找到一个域名")

	for _, res := range results.results {
		assert.NotEmpty(t, res.Subdomain, "域名不应为空")
		assert.Greater(t, len(res.Answers), 0, "应该有至少一个答案")
		t.Logf("找到: %s => %v", res.Subdomain, res.Answers)
	}
}

// TestCNAMEParsing 测试 CNAME 解析正确性
func TestCNAMEParsing(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 已知有 CNAME 记录的域名
	domains := []string{
		"www.github.com", // 通常有 CNAME
		"www.baidu.com",  // 可能有 CNAME
	}

	domainChan := make(chan string, len(domains))
	for _, domain := range domains {
		domainChan <- domain
	}
	close(domainChan)

	results := &testOutputter{results: make([]result.Result, 0)}

	opt := &options.Options{
		Rate:      1000,
		Domain:    domainChan,
		Resolvers: options.GetResolvers(nil),
		Silent:    true,
		TimeOut:   10,
		Retry:     3,
		Method:    options.VerifyType,
		Writer:    []outputter.Output{results},
		ProcessBar: &processbar2.FakeProcess{},
		EtherInfo: options.GetDeviceConfig(options.GetResolvers(nil)),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	r, err := runner.New(opt)
	assert.NoError(t, err)

	r.RunEnumeration(ctx)
	r.Close()

	// 检查 CNAME 记录格式
	for _, res := range results.results {
		for _, answer := range res.Answers {
			// 不应该出现 "comcom" 等错误拼接
			assert.NotContains(t, answer, "comcom", "不应该有错误的字符串拼接")
			assert.NotContains(t, answer, "\x00", "不应该包含空字符")

			t.Logf("%s => %s", res.Subdomain, answer)
		}
	}
}

// TestHighSpeed 高速扫描测试
func TestHighSpeed(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 生成100个测试域名
	domains := make([]string, 100)
	for i := 0; i < 100; i++ {
		if i%2 == 0 {
			domains[i] = "www.baidu.com" // 存在的
		} else {
			domains[i] = "nonexistent12345.baidu.com" // 不存在的
		}
	}

	domainChan := make(chan string, len(domains))
	for _, domain := range domains {
		domainChan <- domain
	}
	close(domainChan)

	results := &testOutputter{results: make([]result.Result, 0)}

	opt := &options.Options{
		Rate:      10000, // 高速
		Domain:    domainChan,
		Resolvers: options.GetResolvers(nil),
		Silent:    true,
		TimeOut:   6,
		Retry:     3,
		Method:    options.VerifyType,
		Writer:    []outputter.Output{results},
		ProcessBar: &processbar2.FakeProcess{},
		EtherInfo: options.GetDeviceConfig(options.GetResolvers(nil)),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	r, err := runner.New(opt)
	assert.NoError(t, err)

	r.RunEnumeration(ctx)
	r.Close()

	// 应该找到大约50个(存在的域名)
	assert.Greater(t, len(results.results), 40, "高速模式应该找到大部分存在的域名")
	assert.Less(t, len(results.results), 60, "不应该有太多误报")

	t.Logf("高速扫描结果: 找到 %d/%d 个域名", len(results.results), len(domains))
}

// TestRetryMechanism 重试机制测试
func TestRetryMechanism(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	domains := []string{"www.example.com"}

	domainChan := make(chan string, len(domains))
	for _, domain := range domains {
		domainChan <- domain
	}
	close(domainChan)

	results := &testOutputter{results: make([]result.Result, 0)}

	// 测试不同的重试次数
	retryCounts := []int{1, 3, 5}

	for _, retryCount := range retryCounts {
		opt := &options.Options{
			Rate:      1000,
			Domain:    domainChan,
			Resolvers: options.GetResolvers(nil),
			Silent:    true,
			TimeOut:   3,
			Retry:     retryCount,
			Method:    options.VerifyType,
			Writer:    []outputter.Output{results},
			ProcessBar: &processbar2.FakeProcess{},
			EtherInfo: options.GetDeviceConfig(options.GetResolvers(nil)),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		r, err := runner.New(opt)
		assert.NoError(t, err)

		startTime := time.Now()
		r.RunEnumeration(ctx)
		elapsed := time.Since(startTime)
		r.Close()

		cancel()

		t.Logf("重试次数 %d: 耗时 %v, 结果数 %d",
			retryCount, elapsed, len(results.results))
	}
}

// TestWildcardDetection 泛解析检测测试
func TestWildcardDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 测试已知的泛解析域名
	// 注意: 这需要一个实际的泛解析域名
	domain := "baidu.com" // 示例

	isWild, ips := runner.IsWildCard(domain)

	if isWild {
		t.Logf("检测到泛解析: %s, IPs: %v", domain, ips)
		assert.Greater(t, len(ips), 0, "泛解析应该返回IP列表")
	} else {
		t.Logf("未检测到泛解析: %s", domain)
	}
}

// testOutputter 测试用输出器
type testOutputter struct {
	results []result.Result
	mu      sync.Mutex
}

func (t *testOutputter) WriteDomainResult(r result.Result) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.results = append(t.results, r)
	return nil
}

func (t *testOutputter) Close() error {
	return nil
}
