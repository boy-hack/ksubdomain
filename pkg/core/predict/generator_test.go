package predict

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type output struct {
}

func (o *output) Write(p []byte) (n int, err error) {
	fmt.Println(string(p))
	return len(p), nil
}

func TestRealConfigFiles(t *testing.T) {
	var buf output
	count, err := PredictDomains("test.example.com", &buf)
	if err != nil {
		t.Fatalf("使用实际配置文件进行域名预测失败: %v", err)
	}
	t.Log(count)
	assert.Greater(t, count, 0)
}
