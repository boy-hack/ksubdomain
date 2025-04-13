package runner

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/boy-hack/ksubdomain/pkg/runner/statusdb"
)

// retry 优化的重试机制
// 使用超时检测和批量发送以提高效率
func (r *Runner) retry(ctx context.Context) {
	// 检测间隔，太频繁会浪费CPU资源
	t := time.NewTicker(time.Duration(r.timeout) * time.Second)
	defer t.Stop()

	// 用于批量发送的域名缓冲区
	const batchSize = 100
	retryDomains := make([]string, 0, batchSize)

	// 记录上次扫描时间，当数据库为空时可以更节约资源
	lastScanEmpty := false

	// 启动多个worker用于处理重试
	workerCount := 4
	retryDomainCh := make(chan string, batchSize*2)
	var wg sync.WaitGroup
	wg.Add(workerCount)

	// 工作协程，用于发送重试请求
	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case domain, ok := <-retryDomainCh:
					if !ok {
						return
					}
					// 重新发送
					r.sender <- domain
				}
			}
		}()
	}

	// 为域名分组的批处理域名缓冲
	dnsBatches := make(map[string][]string)

	for {
		select {
		case <-ctx.Done():
			close(retryDomainCh)
			wg.Wait()
			return
		case <-t.C:
			// 如果上次扫描为空且长度仍为0，可跳过
			currentLength := r.hm.Length()
			if lastScanEmpty && currentLength == 0 {
				continue
			}

			// 当前时间
			now := time.Now()
			// 清空域名缓冲
			retryDomains = retryDomains[:0]

			// 清空分组缓冲
			for k := range dnsBatches {
				dnsBatches[k] = dnsBatches[k][:0]
			}

			// 收集需要重试的域名
			r.hm.Scan(func(key string, v statusdb.Item) error {
				// 超过最大重试次数则放弃
				if r.maxRetry > 0 && v.Retry > r.maxRetry {
					r.hm.Del(key)
					atomic.AddUint64(&r.faildIndex, 1)
					return nil
				}

				// 检查是否超时
				if int64(now.Sub(v.Time).Seconds()) >= r.timeout {
					// 将域名添加到重试列表，或者使用批量发送通道
					retryDomains = append(retryDomains, key)

					// 根据DNS服务器分组，以便批量发送
					dns := r.choseDns(key)
					if _, ok := dnsBatches[dns]; !ok {
						dnsBatches[dns] = make([]string, 0, batchSize)
					}
					dnsBatches[dns] = append(dnsBatches[dns], key)
				}
				return nil
			})

			// 记录扫描状态
			lastScanEmpty = len(retryDomains) == 0

			// 如果有需要重试的域名
			if len(retryDomains) > 0 {
				// 向工作协程发送重试域名
				for _, domain := range retryDomains {
					// 非阻塞发送
					select {
					case retryDomainCh <- domain:
						// 发送成功
					default:
						// 通道满了，直接发送
						r.sender <- domain
					}
				}
			}
		}
	}
}
