package runner

import (
	"context"
	"math"
	"math/rand"
	"runtime"
	"strings"
	"sync"
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
	timeoutSeconds  int64              // 超时秒数
	initialLoadDone chan struct{}      // 初始加载完成信号
	predictLoadDone chan struct{}      // predict加载完成信号
	startTime       time.Time          // 开始时间
	stopSignal      chan struct{}      // 停止信号
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
	return r, nil
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

	// 创建等待组
	wg := &sync.WaitGroup{}
	wg.Add(3)

	// 启动接收处理
	go r.recvChanel(ctx, wg)

	// 启动发送处理
	go r.sendCycle()

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

	// 启动结果处理
	go r.handleResult(predictChan)

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
