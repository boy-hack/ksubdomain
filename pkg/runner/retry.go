package runner

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/runner/statusdb"
	"github.com/google/gopacket/layers"
)

// retry 重试机制。
//
// 设计要点：
//  1. 每 200ms 扫描一次 statusDB，比超时周期更频繁，配合动态超时自适应
//  2. 空扫描优化：上次为空且队列仍为 0 时跳过，节省 CPU
//  3. 批量重传合并：按 DNS server 分组，对同一 server 的多个域名连续调用
//     send()，减少重复的 selectDNSServer + map lookup 开销
//  4. 直接调用 send()：重传不再通过 domainChan 中转（避免两次 channel 传递），
//     但仍更新 statusDB 中的 Retry/Time 字段保持状态一致
func (r *Runner) retry(ctx context.Context) {
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()

	const batchCap = 256

	// dnsBatches: dns server -> []domain，按 server 分组收集需重传域名
	// 复用 map 以减少 GC
	type retryItem struct {
		domain string
		dns    string
	}
	items := make([]retryItem, 0, batchCap)

	// 按 DNS server 分组的 map，key=dnsServer, value=域名列表
	dnsBatches := make(map[string][]string, 16)

	lastScanEmpty := false

	for {
		select {
		case <-ctx.Done():
			return

		case <-t.C:
			// 空扫描快速跳过
			if lastScanEmpty && r.statusDB.Length() == 0 {
				continue
			}

			now := time.Now()
			items = items[:0]
			// 清空分组缓冲（复用已有 key 的 slice）
			for k := range dnsBatches {
				dnsBatches[k] = dnsBatches[k][:0]
			}

			// 扫描 statusDB，收集超时域名并分组
			effectiveTimeout := r.effectiveTimeoutSeconds()
			r.statusDB.Scan(func(key string, v statusdb.Item) error {
				// 超过最大重试次数则放弃
				if r.maxRetryCount > 0 && v.Retry > r.maxRetryCount {
					r.statusDB.Del(key)
					atomic.AddUint64(&r.failedCount, 1)
					return nil
				}

				// 检查是否超时
				if int64(now.Sub(v.Time).Seconds()) < effectiveTimeout {
					return nil
				}

				dns := r.selectDNSServer(key)
				items = append(items, retryItem{domain: key, dns: dns})

				if dnsBatches[dns] == nil {
					dnsBatches[dns] = make([]string, 0, 32)
				}
				dnsBatches[dns] = append(dnsBatches[dns], key)
				return nil
			})

			lastScanEmpty = len(items) == 0
			if lastScanEmpty {
				continue
			}

			// 更新 statusDB：批量更新 Retry/Time，然后按 DNS server 分组批量发包
			// 先更新状态，再发包，保证 statusDB 状态一致
			for _, item := range items {
				v, ok := r.statusDB.Get(item.domain)
				if !ok {
					continue // 可能已被 recv 侧删除，跳过
				}
				v.Retry += 1
				v.Time = time.Now()
				v.Dns = item.dns
				r.statusDB.Set(item.domain, v)
			}

			// 按 DNS server 分组批量调用 send()
			// 同一 server 的域名连续发送，减少 pcap handle 竞争和函数调用开销
			for dns, domains := range dnsBatches {
				if len(domains) == 0 {
					continue
				}
				for _, domain := range domains {
					send(domain, dns, r.options.EtherInfo, r.dnsID,
						uint16(r.listenPort), r.pcapHandle, layers.DNSTypeA)
				}
			}
		}
	}
}
