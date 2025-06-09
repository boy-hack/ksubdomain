package runner

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/runner/statusdb"
)

// retry 优化的重试机制
// 使用超时检测和批量发送以提高效率
func (r *Runner) retry(ctx context.Context) {
	// 检测间隔，太频繁会浪费CPU资源
	t := time.NewTicker(time.Duration(r.timeoutSeconds) * time.Second)
	defer t.Stop()

	// 用于批量发送的域名缓冲区
	const batchSize = 100
	// retryDomains will store the base domain names that need retrying.
	// We will fetch the full Item from statusDB to get the base domain.
	itemsToRetry := make([]statusdb.Item, 0, batchSize)

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
					r.domainChan <- domain
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
			currentLength := r.statusDB.Length()
			if lastScanEmpty && currentLength == 0 {
				continue
			}

			// 当前时间
			now := time.Now()
			// 清空待重试项目缓冲
			itemsToRetry = itemsToRetry[:0]

			// 清空分组缓冲
			for k := range dnsBatches {
				dnsBatches[k] = dnsBatches[k][:0]
			}

			// 收集需要重试的域名
			r.statusDB.Scan(func(key string, v statusdb.Item) error {
				// 超过最大重试次数则放弃
				if r.maxRetryCount > 0 && v.Retry > r.maxRetryCount {
					r.statusDB.Del(key)
					atomic.AddUint64(&r.failedCount, 1)
					return nil
				}

				// 检查是否超时
				if int64(now.Sub(v.Time).Seconds()) >= r.timeoutSeconds {
					// 将项目添加到重试列表
					itemsToRetry = append(itemsToRetry, v) // v is statusdb.Item

					// The dnsBatches logic might need re-evaluation if it's essential.
					// For now, ensuring correct domain goes to domainChan is priority.
					// If selectDNSServer needs the base domain, it should be v.Domain.
					dns := r.selectDNSServer(v.Domain) // Use base domain for DNS server selection
					if _, ok := dnsBatches[dns]; !ok {
						dnsBatches[dns] = make([]string, 0, batchSize)
					}
					// dnsBatches currently stores keys ("domain:type"). This might be fine if
					// the consumer of dnsBatches is aware or if it's just for logging/grouping.
					dnsBatches[dns] = append(dnsBatches[dns], key)
				}
				return nil
			})

			// 记录扫描状态
			lastScanEmpty = len(itemsToRetry) == 0

			// 如果有需要重试的域名
			if len(itemsToRetry) > 0 {
				// 向工作协程发送重试域名 (base domain names)
				for _, item := range itemsToRetry {
					// 非阻塞发送
					select {
					case retryDomainCh <- item.Domain: // Send base domain name
						// 发送成功
					default:
						// 通道满了，直接发送
						r.domainChan <- item.Domain // Send base domain name
					}
				}
			}
		}
	}
}
