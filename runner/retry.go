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
		var preSend []string
		r.hm.Scan(func(key string, v statusdb.Item) error {
			// Scan自带锁，不要调用其他r.hm下的函数。。
			if v.Retry > r.maxRetry {
				delete(r.hm.Items, key)
				atomic.AddUint64(&r.faildIndex, 1)
				return nil
			}
			if currentTime-v.Time >= r.timeout {
				// 重新发送
				preSend = append(preSend, key)
			}
			return nil
		})
		for _, d := range preSend {
			r.sender <- d
		}
		// 延时，map越多延时越大
		length := r.hm.Length()
		if length < 100 {
			length = 1000
		} else if length < 1000 {
			length = 1500
		} else if length < 5000 {
			length = 3500
		} else if length < 10000 {
			length = 4500
		} else {
			length = 5500
		}
		time.Sleep(time.Millisecond * time.Duration(length))
		r.firstRetryChanel <- "ok"
	}
}
