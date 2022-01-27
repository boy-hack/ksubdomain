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
		currentTime := time.Now().Unix()
		r.hm.Scan(func(key string, v statusdb.Item) error {
			if v.Retry > r.maxRetry {
				r.hm.Del(key)
				atomic.AddUint64(&r.faildIndex, 1)
				return nil
			}
			if currentTime-v.Time >= r.timeout {
				// 重新发送
				newItem := v
				newItem.Time = time.Now().Unix()
				newItem.Retry += 1
				newItem.Dns = r.choseDns()
				r.sender <- newItem
			}
			return nil
		})
		// 延时，map越多延时越大
		length := r.Length()
		time.Sleep(time.Millisecond * time.Duration(length))
	}
}
