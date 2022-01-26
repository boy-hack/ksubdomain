package runner

import (
	"bytes"
	"context"
	"encoding/gob"
	"ksubdomain/core"
	"time"
)

func (r *runner) retry(ctx context.Context, chanel chan string) {
	for {
		// 循环检测超时的队列
		r.hm.Scan(func(_ []byte, buff []byte) error {
			out := core.StatusTable{}
			dec := gob.NewDecoder(bytes.NewReader(buff))
			err := dec.Decode(&out)
			if err != nil {
				return err
			}
			currentTime := time.Now().Unix()
			if currentTime-out.Time >= r.timeout {
				// 重新发送
				out.Time = time.Now().Unix()
				r.sender <- out
			}
			return nil
		})
	}
}
