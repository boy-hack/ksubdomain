package runner

import (
	"context"
	"ksubdomain/runner/statusdb"
	"sync/atomic"
	"time"
)

func (r *runner) retry(ctx context.Context) {
	for {
		// 循环检测超时的队列
		now := time.Now()
		r.hm.Scan(func(key string, v statusdb.Item) error {
			if v.Retry > r.maxRetry {
				r.hm.Del(key)
				atomic.AddUint64(&r.faildIndex, 1)
				return nil
			}
			if int64(now.Sub(v.Time)) >= r.timeout {
				// 重新发送
				r.sender <- key
			}
			return nil
		})
		length := 1000
		time.Sleep(time.Millisecond * time.Duration(length))
	}
}
