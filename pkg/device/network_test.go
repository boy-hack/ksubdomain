package device

import (
	"github.com/boy-hack/ksubdomain/pkg/core"
	"github.com/boy-hack/ksubdomain/pkg/core/gologger"
	"testing"
	"time"
)

func TestAutoGetDevices(t *testing.T) {
	ether := AutoGetDevices()
	PrintDeviceInfo(ether)
}

func TestLookupIP(t *testing.T) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	count := 0
	for {
		select {
		case <-ticker.C:
			// 每秒尝试一次DNS查询
			domain := core.RandomStr(4) + ".baidu.com"
			err := LookUpIP(domain, "1.1.1.1")
			if err != nil {
				gologger.Debugf("DNS查询失败: %s\n", err.Error())
			}
			t.Log(count)
			count++
		}
		if count > 100 {
			break
		}
	}
}
