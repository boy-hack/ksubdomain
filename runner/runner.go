package runner

import (
	"bufio"
	"context"
	"fmt"
	"github.com/boy-hack/ksubdomain/core"
	"github.com/boy-hack/ksubdomain/core/device"
	"github.com/boy-hack/ksubdomain/core/gologger"
	options2 "github.com/boy-hack/ksubdomain/core/options"
	"github.com/boy-hack/ksubdomain/runner/statusdb"
	"github.com/google/gopacket/pcap"
	"github.com/phayes/freeport"
	"go.uber.org/ratelimit"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"
)

type runner struct {
	ether           *device.EtherTable //本地网卡信息
	hm              *statusdb.StatusDb
	options         *options2.Options
	limit           ratelimit.Limiter
	handle          *pcap.Handle
	successIndex    uint64
	sendIndex       uint64
	recvIndex       uint64
	faildIndex      uint64
	sender          chan string
	recver          chan result
	freeport        int
	dnsid           uint16 // dnsid 用于接收的确定ID
	maxRetry        int    // 最大重试次数
	timeout         int64  // 超时xx秒后重试
	ctx             context.Context
	fisrtloadChanel chan string // 数据加载完毕的chanel
	startTime       time.Time
	domains         []string
}

func GetDeviceConfig() *device.EtherTable {
	filename := "ksubdomain.yaml"
	var ether *device.EtherTable
	var err error
	if core.FileExists(filename) {
		ether, err = device.ReadConfig(filename)
		if err != nil {
			gologger.Fatalf("读取配置失败:%v", err)
		}
		gologger.Infof("读取配置%s成功!\n", filename)
	} else {
		ether = device.AutoGetDevices()
		err = ether.SaveConfig(filename)
		if err != nil {
			gologger.Fatalf("保存配置失败:%v", err)
		}
	}
	gologger.Infof("Use Device: %s\n", ether.Device)
	gologger.Infof("Use IP:%s\n", ether.SrcIp.String())
	gologger.Infof("Local Mac: %s\n", ether.SrcMac.String())
	gologger.Infof("GateWay Mac: %s\n", ether.DstMac.String())
	return ether
}
func New(options *options2.Options) (*runner, error) {
	var err error
	version := pcap.Version()
	r := new(runner)
	gologger.Infof(version + "\n")

	r.options = options
	r.ether = GetDeviceConfig()
	r.hm = statusdb.CreateMemoryDB()

	gologger.Infof("DNS:%s\n", core.SliceToString(options.Resolvers))
	r.handle, err = device.PcapInit(r.ether.Device)
	if err != nil {
		return nil, err
	}

	// 根据发包总数和timeout时间来分配每秒速度
	allPacket := r.loadTargets()
	if options.Level > 2 {
		allPacket = allPacket * int(math.Pow(float64(len(options.LevelDomains)), float64(options.Level-2)))
	}
	calcLimit := float64(allPacket/options.TimeOut) * 0.85
	if calcLimit < 1000 {
		calcLimit = 1000
	}
	limit := int(math.Min(calcLimit, float64(options.Rate)))
	r.limit = ratelimit.New(limit) // per second

	gologger.Infof("Rate:%dpps\n", limit)

	r.sender = make(chan string, 99) // 多个协程发送
	r.recver = make(chan result, 99) // 多个协程接收

	freePort, err := freeport.GetFreePort()
	if err != nil {
		return nil, err
	}
	r.freeport = freePort
	gologger.Infof("FreePort:%d\n", freePort)
	r.dnsid = 0x2021 // set dnsid 65500
	r.maxRetry = r.options.Retry
	r.timeout = int64(r.options.TimeOut)
	r.ctx = context.Background()
	r.fisrtloadChanel = make(chan string)
	r.startTime = time.Now()

	go func() {
		for _, msg := range r.domains {
			r.sender <- msg
			if options.Method == "enum" && options.Level > 2 {
				r.iterDomains(options.Level, msg)
			}
		}
		r.domains = nil
		r.fisrtloadChanel <- "ok"
	}()
	return r, nil
}
func (r *runner) iterDomains(level int, domain string) {
	if level == 2 {
		return
	}
	for _, levelMsg := range r.options.LevelDomains {
		tmpDomain := fmt.Sprintf("%s.%s", levelMsg, domain)
		r.sender <- tmpDomain
		r.iterDomains(level-1, tmpDomain)
	}
}
func (r *runner) choseDns() string {
	dns := r.options.Resolvers
	return dns[rand.Intn(len(dns))]
}

func (r *runner) loadTargets() int {
	// get targets
	var reader *bufio.Reader
	options := r.options
	if options.Method == "verify" {
		if options.Stdin {
			reader = bufio.NewReader(os.Stdin)

		} else {
			f2, err := os.Open(options.FileName)
			if err != nil {
				gologger.Fatalf("打开文件:%s 出现错误:%s", options.FileName, err.Error())
			}
			defer f2.Close()
			reader = bufio.NewReader(f2)
		}
	} else if options.Method == "enum" {
		if options.Stdin {
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				options.Domain = append(options.Domain, scanner.Text())
			}
		}
		// 读取字典
		if options.FileName == "" {
			subdomainDict := core.GetDefaultSubdomainData()
			reader = bufio.NewReader(strings.NewReader(strings.Join(subdomainDict, "\n")))
		} else {
			subdomainDict, err := core.LinesInFile(options.FileName)
			if err != nil {
				gologger.Fatalf("打开文件:%s 错误:%s", options.FileName, err.Error())
			}
			reader = bufio.NewReader(strings.NewReader(strings.Join(subdomainDict, "\n")))
		}

		if options.SkipWildCard && len(options.Domain) > 0 {
			var tmpDomains []string
			gologger.Infof("检测泛解析\n")
			for _, domain := range options.Domain {
				if !core.IsWildCard(domain) {
					tmpDomains = append(tmpDomains, domain)
				} else {
					gologger.Warningf("域名:%s 存在泛解析记录,已跳过\n", domain)
				}
			}
			options.Domain = tmpDomains
		}
	}

	if len(options.Domain) > 0 {
		p := core.SliceToString(options.Domain)
		gologger.Infof("检测域名:%s\n", p)
	}

	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		msg := string(line)
		if r.options.Method == "verify" {
			// send msg
			r.domains = append(r.domains, msg)
		} else {
			for _, tmpDomain := range r.options.Domain {
				newDomain := msg + "." + tmpDomain
				r.domains = append(r.domains, newDomain)
			}
		}
	}
	return len(r.domains)
}
func (r *runner) PrintStatus() {
	queue := r.hm.Length()
	tc := int(time.Since(r.startTime).Seconds())
	gologger.Printf("\rSuccess:%d Send:%d Queue:%d Accept:%d Fail:%d Elapsed:%ds", r.successIndex, r.sendIndex, queue, r.recvIndex, r.faildIndex, tc)
}
func (r *runner) RunEnumeration() {
	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()
	go r.recvChanel(ctx) // 启动接收线程
	for i := 0; i < 3; i++ {
		go r.sendCycle(ctx) // 发送线程
	}
	go r.handleResult(ctx) // 处理结果，打印输出

	var isLoadOver bool = false // 是否加载文件完毕
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			r.PrintStatus()
			if isLoadOver {
				if r.hm.Length() == 0 {
					gologger.Printf("\n")
					gologger.Infof("扫描完毕")
					return
				}
			}
		case <-r.fisrtloadChanel:
			go r.retry(ctx) // 遍历hm，依次重试
			isLoadOver = true
		}
	}
}

func (r *runner) Close() {
	close(r.recver)
	close(r.sender)
	r.handle.Close()
	r.hm.Close()
}
