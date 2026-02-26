package output

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
)

// JSONLOutput JSONL (JSON Lines) 输出器
// 每行一个 JSON 对象,便于流式处理和工具链集成
// 格式: {"domain":"example.com","type":"A","records":["1.2.3.4"],"timestamp":1234567890}
type JSONLOutput struct {
	filename string
	file     *os.File
	mu       sync.Mutex
}

// JSONLRecord JSONL 记录格式
type JSONLRecord struct {
	Domain    string   `json:"domain"`              // 子域名
	Type      string   `json:"type"`                // 记录类型 (A, CNAME, NS, etc.)
	Records   []string `json:"records"`             // 记录值列表
	Timestamp int64    `json:"timestamp"`           // Unix 时间戳
	TTL       uint32   `json:"ttl,omitempty"`       // TTL (可选)
	Source    string   `json:"source,omitempty"`    // 数据来源 (可选)
}

// NewJSONLOutput 创建 JSONL 输出器
func NewJSONLOutput(filename string) (*JSONLOutput, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return nil, err
	}

	gologger.Infof("JSONL output file: %s\n", filename)

	return &JSONLOutput{
		filename: filename,
		file:     file,
	}, nil
}

// WriteDomainResult 写入单个域名结果
// JSONL 格式每次写入一行 JSON,立即刷新
// 优点: 支持流式处理,可以实时读取
func (j *JSONLOutput) WriteDomainResult(r result.Result) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	// 解析记录类型
	recordType := "A" // 默认 A 记录
	records := make([]string, 0, len(r.Answers))

	for _, answer := range r.Answers {
		// 解析类型 (CNAME, NS, PTR 等)
		if len(answer) > 0 {
			// 检查是否为特殊记录类型
			if len(answer) > 6 && answer[:6] == "CNAME " {
				recordType = "CNAME"
				records = append(records, answer[6:]) // 去掉 "CNAME " 前缀
			} else if len(answer) > 3 && answer[:3] == "NS " {
				recordType = "NS"
				records = append(records, answer[3:])
			} else if len(answer) > 4 && answer[:4] == "PTR " {
				recordType = "PTR"
				records = append(records, answer[4:])
			} else if len(answer) > 4 && answer[:4] == "TXT " {
				recordType = "TXT"
				records = append(records, answer[4:])
			} else {
				// IP 地址 (A 或 AAAA 记录)
				records = append(records, answer)
			}
		}
	}

	// 如果没有解析出记录,使用原始 answers
	if len(records) == 0 {
		records = r.Answers
	}

	// 构造 JSONL 记录
	record := JSONLRecord{
		Domain:    r.Subdomain,
		Type:      recordType,
		Records:   records,
		Timestamp: time.Now().Unix(),
	}

	// 序列化为 JSON
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	// 写入一行 (JSON + 换行符)
	_, err = j.file.Write(append(data, '\n'))
	if err != nil {
		return err
	}

	// 立即刷新到磁盘 (支持实时读取)
	return j.file.Sync()
}

// Close 关闭 JSONL 输出器
func (j *JSONLOutput) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.file != nil {
		gologger.Infof("JSONL output completed: %s\n", j.filename)
		return j.file.Close()
	}
	return nil
}
