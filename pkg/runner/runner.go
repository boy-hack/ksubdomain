package runner

import (
	"context"
	"math"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/core"
	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/core/options"
	"github.com/boy-hack/ksubdomain/v2/pkg/device"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/processbar"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/statusdb"
	"github.com/google/gopacket/pcap"
	"github.com/phayes/freeport"
	"go.uber.org/ratelimit"
)

// Runner 表示子域名扫描的运行时结构
type Runner struct {
	statusDB        *statusdb.StatusDb // 状态数据库
	options         *options.Options   // 配置选项
	rateLimiter     ratelimit.Limiter  // 速率限制器
	pcapHandle      *pcap.Handle       // 网络抓包句柄
	successCount    uint64             // 成功数量
	sendCount       uint64             // 发送数量
	receiveCount    uint64             // 接收数量
	failedCount     uint64             // 失败数量
	domainChan      chan string        // 域名发送通道
	resultChan      chan result.Result // 结果接收通道
	listenPort      int                // 监听端口
	dnsID           uint16             // DNS请求ID
	maxRetryCount   int                // 最大重试次数
	timeoutSeconds  int64              // 超时秒数（固定值，DynamicTimeout=false时使用）
	initialLoadDone chan struct{}      // 初始加载完成信号
	predictLoadDone chan struct{}      // predict加载完成信号
	startTime       time.Time          // 开始时间
	stopSignal      chan struct{}      // 停止信号
	rttTracker      *rttSlidingWindow  // RTT滑动均值追踪器（DynamicTimeout=true时使用）
}

// rttSlidingWindow 基于指数加权移动平均（EWMA）计算RTT滑动均值。
//
// 算法说明：
//   - 使用 alpha=0.125（即 1/8），与 TCP RFC 6298 保持一致
//   - smoothedRTT = (1-alpha)*smoothedRTT + alpha*sample
//   - rttVar      = (1-beta)*rttVar + beta*|sample-smoothedRTT|  (beta=0.25)
//   - dynamicTimeout = smoothedRTT + 4*rttVar（TCP RTO 公式）
//   - 上下界：[minTimeout=1s, maxTimeout=用户配置的 --timeout]
//
// 线程安全：所有字段通过 mu 保护。
type rttSlidingWindow struct {
	mu            sync.Mutex
	smoothedRTT   float64 // 单位：秒（EWMA）
	rttVar        float64 // RTT 方差（EWMA）
	sampleCount   int64   // 已采样数量
	minTimeout    float64 // 动态超时下界（秒）
	maxTimeout    float64 // 动态超时上界 = 用户配置的 --timeout（秒）
}

const (
	rttAlpha = 0.125 // EWMA平滑系数（TCP RFC 6298）
	rttBeta  = 0.25  // 方差平滑系数
)

// newRTTSlidingWindow 创建RTT追踪器，maxTimeout 为用户配置的超时上界（秒）。
func newRTTSlidingWindow(maxTimeout float64) *rttSlidingWindow {
	return &rttSlidingWindow{
		// 初始 smoothedRTT 设为 maxTimeout/2，避免冷启动时过早丢弃域名
		smoothedRTT: maxTimeout / 2,
		rttVar:      maxTimeout / 4,
		minTimeout:  1.0,
		maxTimeout:  maxTimeout,
	}
}

// recordSample 记录一次RTT样本（单位：秒）。
func (w *rttSlidingWindow) recordSample(rttSec float64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.sampleCount == 0 {
		// 第一个样本：直接初始化
		w.smoothedRTT = rttSec
		w.rttVar = rttSec / 2
	} else {
		// RFC 6298 更新公式
		diff := rttSec - w.smoothedRTT
		if diff < 0 {
			diff = -diff
		}
		w.rttVar = (1-rttBeta)*w.rttVar + rttBeta*diff
		w.smoothedRTT = (1-rttAlpha)*w.smoothedRTT + rttAlpha*rttSec
	}
	atomic.AddInt64(&w.sampleCount, 1)
}

// dynamicTimeoutSeconds 返回当前动态超时（秒，整数，向上取整）。
// 公式：smoothedRTT + 4*rttVar，限制在 [minTimeout, maxTimeout] 范围内。
func (w *rttSlidingWindow) dynamicTimeoutSeconds() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()

	timeout := w.smoothedRTT + 4*w.rttVar
	if timeout < w.minTimeout {
		timeout = w.minTimeout
	}
	if timeout > w.maxTimeout {
		timeout = w.maxTimeout
	}
	// 向上取整，最少 1 秒
	result := int64(math.Ceil(timeout))
	if result < 1 {
		result = 1
	}
	return result
}

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

// New 创建一个新的Runner实例
func New(opt *options.Options) (*Runner, error) {
	var err error
	version := pcap.Version()
	r := new(Runner)
	gologger.Infof(version)
	r.options = opt
	r.statusDB = statusdb.CreateMemoryDB()

	// 记录DNS服务器信息
	gologger.Infof("默认DNS服务器: %s\n", core.SliceToString(opt.Resolvers))
	if len(opt.SpecialResolvers) > 0 {
		var keys []string
		for k := range opt.SpecialResolvers {
			keys = append(keys, k)
		}
		gologger.Infof("特殊DNS服务器: %s\n", core.SliceToString(keys))
	}

	// 初始化网络设备
	r.pcapHandle, err = device.PcapInit(opt.EtherInfo.Device)
	if err != nil {
		return nil, err
	}

	// 设置速率限制
	cpuLimit := float64(runtime.NumCPU() * 10000)
	rateLimit := int(math.Min(cpuLimit, float64(opt.Rate)))
	
	// Mac 平台优化: BPF 缓冲区限制较严格
	// 建议速率 < 50000 pps 以避免缓冲区溢出
	if runtime.GOOS == "darwin" && rateLimit > 50000 {
		gologger.Warningf("Mac 平台检测到: 当前速率 %d pps 可能导致缓冲区问题\n", rateLimit)
		gologger.Warningf("建议: 使用 -b 参数限制带宽 (如 -b 5m) 或降低速率\n")
		gologger.Warningf("提示: Mac BPF 缓冲区已优化至 2MB,但仍建议速率 < 50000 pps\n")
	}
	
	r.rateLimiter = ratelimit.New(rateLimit)
	gologger.Infof("速率限制: %d pps\n", rateLimit)

	// 初始化通道
	r.domainChan = make(chan string, 50000)
	r.resultChan = make(chan result.Result, 5000)
	r.stopSignal = make(chan struct{})

	// 获取空闲端口
	freePort, err := freeport.GetFreePort()
	if err != nil {
		return nil, err
	}
	r.listenPort = freePort
	gologger.Infof("监听端口: %d\n", freePort)

	// 设置其他参数
	r.dnsID = 0x2021 // ksubdomain的生日
	r.maxRetryCount = opt.Retry
	r.timeoutSeconds = int64(opt.TimeOut)
	r.initialLoadDone = make(chan struct{})
	r.predictLoadDone = make(chan struct{})
	r.startTime = time.Now()

	// 初始化动态超时追踪器（仅在 DynamicTimeout 开启时）
	if opt.DynamicTimeout {
		r.rttTracker = newRTTSlidingWindow(float64(opt.TimeOut))
		gologger.Infof("动态超时已开启，上界: %ds\n", opt.TimeOut)
	}

	return r, nil
}

// effectiveTimeoutSeconds 返回当前有效的超时秒数。
// 若 DynamicTimeout 已开启且有足够样本，返回动态计算值；否则返回固定配置值。
func (r *Runner) effectiveTimeoutSeconds() int64 {
	if r.options.DynamicTimeout && r.rttTracker != nil && atomic.LoadInt64(&r.rttTracker.sampleCount) > 0 {
		return r.rttTracker.dynamicTimeoutSeconds()
	}
	return r.timeoutSeconds
}

// selectDNSServer 根据域名智能选择DNS服务器
func (r *Runner) selectDNSServer(domain string) string {
	dnsServers := r.options.Resolvers
	specialDNSServers := r.options.SpecialResolvers

	// 根据域名后缀选择特定DNS服务器
	if len(specialDNSServers) > 0 {
		for suffix, servers := range specialDNSServers {
			if strings.HasSuffix(domain, suffix) {
				dnsServers = servers
				break
			}
		}
	}

	// 随机选择一个DNS服务器
	idx := getRandomIndex() % len(dnsServers)
	return dnsServers[idx]
}

// getRandomIndex 获取随机索引
func getRandomIndex() int {
	return int(rand.Int31())
}

// updateStatusBar 更新进度条状态
func (r *Runner) updateStatusBar() {
	if r.options.ProcessBar != nil {
		queueLength := r.statusDB.Length()
		elapsedSeconds := int(time.Since(r.startTime).Seconds())
		data := &processbar.ProcessData{
			SuccessIndex: r.successCount,
			SendIndex:    r.sendCount,
			QueueLength:  queueLength,
			RecvIndex:    r.receiveCount,
			FaildIndex:   r.failedCount,
			Elapsed:      elapsedSeconds,
		}
		r.options.ProcessBar.WriteData(data)
	}
}

// loadDomainsFromSource 从源加载域名
func (r *Runner) loadDomainsFromSource(wg *sync.WaitGroup) {
	defer wg.Done()
	// 从域名源加载域名
	for domain := range r.options.Domain {
		r.domainChan <- domain
	}
	// 通知初始加载完成
	r.initialLoadDone <- struct{}{}
}

// monitorProgress 监控扫描进度
func (r *Runner) monitorProgress(ctx context.Context, cancelFunc context.CancelFunc, wg *sync.WaitGroup) {
	var initialLoadCompleted bool = false
	var initialLoadPredict bool = false
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	defer wg.Done()
	for {
		select {
		case <-ticker.C:
			// 更新状态栏
			r.updateStatusBar()
			// 检查是否完成
			if initialLoadCompleted && initialLoadPredict {
				queueLength := r.statusDB.Length()
				if queueLength <= 0 {
					gologger.Printf("\n")
					gologger.Infof("扫描完毕")
					cancelFunc() // 使用传递的cancelFunc
					return
				}
			}
		case <-r.initialLoadDone:
			// 初始加载完成后启动重试机制
			go r.retry(ctx)
			initialLoadCompleted = true
		case <-r.predictLoadDone:
			initialLoadPredict = true
		case <-ctx.Done():
			return
		}
	}
}

// processPredictedDomains 处理预测的域名
func (r *Runner) processPredictedDomains(ctx context.Context, wg *sync.WaitGroup, predictChan chan string) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case domain := <-predictChan:
			r.domainChan <- domain
		}
	}
}

// RunEnumeration 开始子域名枚举过程
func (r *Runner) RunEnumeration(ctx context.Context) {
	// 创建可取消的上下文
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	// 创建等待组，现在需要等待5个goroutine（添加了sendCycle和handleResult）
	wg := &sync.WaitGroup{}
	wg.Add(5)

	// 启动接收处理
	go r.recvChanel(ctx, wg)

	// 启动发送处理（加入waitgroup管理）
	go r.sendCycleWithContext(ctx, wg)

	// 监控进度
	go r.monitorProgress(ctx, cancelFunc, wg)

	// 创建预测域名通道
	predictChan := make(chan string, 1000)
	if r.options.Predict {
		wg.Add(1)
		// 启动预测域名处理
		go r.processPredictedDomains(ctx, wg, predictChan)
	} else {
		r.predictLoadDone <- struct{}{}
	}

	// 启动结果处理（加入waitgroup管理）
	go r.handleResultWithContext(ctx, wg, predictChan)

	// 从源加载域名
	go r.loadDomainsFromSource(wg)

	// 等待所有协程完成
	wg.Wait()

	// 关闭所有通道
	close(predictChan)
	// 安全关闭通道
	close(r.resultChan)
	close(r.domainChan)
}

// Close 关闭Runner并释放资源
func (r *Runner) Close() {
	// 关闭网络抓包句柄
	if r.pcapHandle != nil {
		r.pcapHandle.Close()
	}

	// 关闭状态数据库
	if r.statusDB != nil {
		r.statusDB.Close()
	}

	// 关闭所有输出器
	for _, out := range r.options.Writer {
		err := out.Close()
		if err != nil {
			gologger.Errorf("关闭输出器失败: %v", err)
		}
	}

	// 关闭进度条
	if r.options.ProcessBar != nil {
		r.options.ProcessBar.Close()
	}
}
