package runner

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/google/gopacket/pcap"
	"github.com/phayes/freeport"
	_ "github.com/projectdiscovery/fdmax/autofdmax"
	"go.uber.org/ratelimit"
	"io"
	"ksubdomain/core"
	"ksubdomain/core/device"
	"ksubdomain/core/gologger"
	options2 "ksubdomain/core/options"
	"ksubdomain/runner/statusdb"
	"math/rand"
	"os"
	"strings"
	"sync"
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
	sender          chan statusdb.Item
	recver          chan core.RecvResult
	freeport        int
	dnsid           uint16 // dnsid 用于接收的确定ID
	maxRetry        int    // 最大重试次数
	timeout         int64  // 超时xx秒后重试
	lock            sync.RWMutex
	ctx             context.Context
	fisrtloadChanel chan string // 数据加载完毕的chanel
}

func New(options *options2.Options) (*runner, error) {
	var err error
	version := pcap.Version()
	r := new(runner)
	r.options = options
	gologger.Infof(version + "\n")
	if options.ListNetwork {
		device.GetIpv4Devices()
		os.Exit(0)
	}
	ether := device.AutoGetDevices()
	r.ether = ether
	gologger.Infof("Use Device: %s\n", ether.Device)
	gologger.Infof("Use IP:%s\n", ether.SrcIp.String())
	gologger.Infof("Local Mac:%s\n", ether.SrcMac.String())
	gologger.Infof("GateWay Mac:%s\n", ether.DstMac.String())

	if options.Test {
		TestSpeed(ether)
		os.Exit(0)
	}

	r.hm = statusdb.CreateMemoryDB()

	// get targets
	var f io.Reader
	if options.Stdin {
		if options.Verify {
			f = os.Stdin
		} else {
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				options.Domain = append(options.Domain, scanner.Text())
			}
		}
	}
	if len(options.Domain) > 0 {
		if options.Verify {
			f = strings.NewReader(strings.Join(options.Domain, "\n"))
		} else if options.FileName == "" {
			subdomainDict := core.GetDefaultSubdomainData()
			gologger.Infof("加载内置字典:%d\n", len(subdomainDict))
			f = strings.NewReader(strings.Join(subdomainDict, "\n"))
		}
	} else {
		f2, err := os.Open(options.FileName)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("打开文件:%s 错误:%s", options.FileName, err.Error()))
		}
		defer f2.Close()
		f = f2
	}
	if options.Verify && options.FileName != "" {
		f2, err := os.Open(options.FileName)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("打开文件:%s 出现错误:%s", options.FileName, err.Error()))
		}
		defer f2.Close()
		f = f2
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
	if len(options.Domain) > 0 {
		gologger.Infof("检测域名:%s\n", options.Domain)
	}
	gologger.Infof("设置rate:%dpps\n", options.Rate)
	gologger.Infof("DNS:%s\n", options.Resolvers)
	r.handle, err = device.PcapInit(ether.Device)
	if err != nil {
		return nil, err
	}
	r.limit = ratelimit.New(int(options.Rate)) // per second
	r.sender = make(chan statusdb.Item, 999)   // 可多个协程发送
	r.recver = make(chan core.RecvResult)      // 只用一个协程接收，这里不会影响性能
	tmpFreeport, err := freeport.GetFreePort()
	if err != nil {
		return nil, err
	}
	r.freeport = tmpFreeport
	gologger.Infof("获取FreePort:%d\n", tmpFreeport)
	r.dnsid = 0x2021 // set dnsid 65500
	r.maxRetry = r.options.Retry
	r.timeout = int64(r.options.TimeOut)
	r.lock = sync.RWMutex{}
	r.ctx = context.Background()
	r.fisrtloadChanel = make(chan string)
	go r.loadTargets(f)
	return r, nil
}
func (r *runner) choseDns() string {
	dns := r.options.Resolvers
	return dns[rand.Intn(len(dns))]
}

func (r *runner) loadTargets(f io.Reader) {
	reader := bufio.NewReader(f)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		msg := string(line)
		if r.options.Verify {
			// send msg
			item := statusdb.Item{
				Domain:      msg,
				Dns:         r.choseDns(),
				Time:        0,
				Retry:       0,
				DomainLevel: 0,
			}
			r.sender <- item
		} else {
			for _, tmpDomain := range r.options.Domain {
				newDomain := msg + "." + tmpDomain
				item := statusdb.Item{
					Domain:      newDomain,
					Dns:         r.choseDns(),
					Time:        0,
					Retry:       0,
					DomainLevel: 0,
				}
				r.sender <- item
			}
		}
	}
	r.fisrtloadChanel <- "ok"
}
func (r *runner) PrintStatus() {
	gologger.Printf("\rSuccess:%d Sent:%d Recved:%d Faild:%d", r.successIndex, r.sendIndex, r.recvIndex, r.faildIndex)
}
func (r *runner) RunEnumeration() {
	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()

	go r.recvChanel(ctx)   // 启动接收线程
	go r.sendCycle(ctx)    // 发送线程
	go r.handleResult(ctx) // 处理结果，打印输出

	var isLoadOver int = 0 // 是否加载文件完毕
	t := time.NewTicker(300 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			r.PrintStatus()
			if isLoadOver >= 1 {
				t.Reset(time.Second)
				if r.Length() == 0 {
					isLoadOver += 1
				}
				if isLoadOver >= 5 {
					gologger.Infof("扫描完毕")
					return
				}
			}
		case <-r.fisrtloadChanel:
			go r.retry(ctx) // 遍历hm，依次重试
			isLoadOver = 1
		}
	}
}
func (r *runner) Length() int {
	r.hm.Mu.Lock()
	defer r.hm.Mu.Unlock()
	length := len(r.hm.Items)
	return length
}
func (r *runner) Close() {
	close(r.recver)
	close(r.sender)
	r.handle.Close()
	r.hm.Close()
}
