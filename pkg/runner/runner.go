package runner

import (
	"context"
	"math"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/boy-hack/ksubdomain/pkg/core"
	"github.com/boy-hack/ksubdomain/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/pkg/core/options"
	"github.com/boy-hack/ksubdomain/pkg/device"
	"github.com/boy-hack/ksubdomain/pkg/runner/processbar"
	"github.com/boy-hack/ksubdomain/pkg/runner/result"
	"github.com/boy-hack/ksubdomain/pkg/runner/statusdb"
	"github.com/google/gopacket/pcap"
	"github.com/phayes/freeport"
	"go.uber.org/ratelimit"
)

type Runner struct {
	hm              *statusdb.StatusDb
	options         *options.Options
	limit           ratelimit.Limiter
	handle          *pcap.Handle
	successIndex    uint64
	sendIndex       uint64
	recvIndex       uint64
	faildIndex      uint64
	sender          chan string
	recver          chan result.Result
	freeport        int
	dnsid           uint16
	maxRetry        int
	timeout         int64
	fisrtloadChanel chan string
	startTime       time.Time
	stopSignal      chan struct{}
	workerWg        sync.WaitGroup
}

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

func New(opt *options.Options) (*Runner, error) {
	var err error
	version := pcap.Version()
	r := new(Runner)
	gologger.Infof(version + "\n")
	r.options = opt
	r.hm = statusdb.CreateMemoryDB()
	gologger.Infof("Default DNS:%s\n", core.SliceToString(opt.Resolvers))
	if len(opt.SpecialResolvers) > 0 {
		var keys []string
		for k := range opt.SpecialResolvers {
			keys = append(keys, k)
		}
		gologger.Infof("Special DNS:%s\n", core.SliceToString(keys))
	}
	r.handle, err = device.PcapInit(opt.EtherInfo.Device)
	if err != nil {
		return nil, err
	}

	cpuLimit := float64(runtime.NumCPU() * 10000)
	limit := int(math.Min(cpuLimit, float64(opt.Rate)))
	r.limit = ratelimit.New(limit)
	gologger.Infof("Rate:%dpps\n", limit)

	r.sender = make(chan string, 50000)
	r.recver = make(chan result.Result, 5000)
	r.stopSignal = make(chan struct{})

	freePort, err := freeport.GetFreePort()
	if err != nil {
		return nil, err
	}
	r.freeport = freePort
	gologger.Infof("FreePort:%d\n", freePort)
	r.dnsid = 0x2021 // birthday of ksubdomain
	r.maxRetry = opt.Retry
	r.timeout = int64(opt.TimeOut)
	r.fisrtloadChanel = make(chan string)
	r.startTime = time.Now()
	return r, nil
}

// choseDns 智能选择DNS服务器
func (r *Runner) choseDns(domain string) string {
	dns := r.options.Resolvers
	specialDns := r.options.SpecialResolvers

	// 根据域名后缀选择特定DNS服务器
	if len(specialDns) > 0 {
		for k, v := range specialDns {
			if strings.HasSuffix(domain, k) {
				dns = v
				break
			}
		}
	}

	// 随机选择DNS服务器
	idx := fastrand() % len(dns)
	return dns[idx]
}

func fastrand() int {
	return int(rand.Int31())
}

func (r *Runner) printStatus() {
	queue := r.hm.Length()
	tc := int(time.Since(r.startTime).Seconds())
	data := &processbar.ProcessData{
		SuccessIndex: r.successIndex,
		SendIndex:    r.sendIndex,
		QueueLength:  queue,
		RecvIndex:    r.recvIndex,
		FaildIndex:   r.faildIndex,
		Elapsed:      tc,
	}
	if r.options.ProcessBar != nil {
		r.options.ProcessBar.WriteData(data)
	}
}

func (r *Runner) RunEnumeration(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	wg := &sync.WaitGroup{}
	wg.Add(3)
	// 接收线程
	go r.recvChanel(ctx, wg)
	// 发送线程
	go r.sendCycle()
	// 处理结果
	predictChanel := make(chan string)
	go func() {
		defer func() {
			r.options.Predict = false
		}()
		if r.options.Predict {
			for domain := range predictChanel {
				r.sender <- domain
			}
		}
	}()
	go r.handleResult(predictChanel)
	// 加载域名
	go func() {
		defer wg.Done()
		batchSize := 1000
		batch := make([]string, 0, batchSize)
		for domain := range r.options.Domain {
			batch = append(batch, domain)
			if len(batch) >= batchSize {
				for _, d := range batch {
					r.sender <- d
				}
				batch = batch[:0]
			}
		}
		for _, d := range batch {
			r.sender <- d
		}
		r.fisrtloadChanel <- "ok"
	}()
	var isLoadOver bool = false
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()
	go func() {
		defer wg.Done()
		for {
			select {
			case <-t.C:
				r.printStatus()
				if isLoadOver {
					length := r.hm.Length()
					if length <= 0 {
						gologger.Printf("\n")
						gologger.Infof("扫描完毕")
						cancel()
					}
				}
			case <-r.fisrtloadChanel:
				go r.retry(ctx)
				isLoadOver = true
			case <-ctx.Done():
				return
			}
		}
	}()
	wg.Wait()
	if r.options.Predict {
		gologger.Infof("预测模式暂时下线了\n")
	}
	close(predictChanel)
	close(r.recver)
	close(r.sender)
}

func (r *Runner) Close() {
	if r.handle != nil {
		r.handle.Close()
	}
	if r.hm != nil {
		r.hm.Close()
	}
	for _, out := range r.options.Writer {
		err := out.Close()
		if err != nil {
			gologger.Errorf("关闭输出器失败: %v", err)
		}
	}
	if r.options.ProcessBar != nil {
		r.options.ProcessBar.Close()
	}
}
