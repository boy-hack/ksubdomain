package output

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/core/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
)

// JSONLOutput writes one JSON object per line (JSON Lines format).
//
// Each line contains exactly the fields below, making it directly
// consumable by jq, grep, and other line-oriented tools:
//
//	{"domain":"sub.example.com","type":"A","records":["1.2.3.4"],"timestamp":1700000000}
//
// Use `--oy jsonl` on the CLI, or ExtraWriters in the SDK, to activate.
type JSONLOutput struct {
	filename string
	file     *os.File
	bw       *bufio.Writer // buffered writer for throughput
	mu       sync.Mutex
}

// JSONLRecord is the schema for each output line.
// Field names are intentionally short and stable — do not rename.
type JSONLRecord struct {
	Domain    string   `json:"domain"`           // resolved subdomain
	Type      string   `json:"type"`             // A, CNAME, NS, PTR, TXT, …
	Records   []string `json:"records"`          // record values (IPs, target names, …)
	Timestamp int64    `json:"timestamp"`        // Unix epoch seconds
	TTL       uint32   `json:"ttl,omitempty"`    // TTL when available
}

// NewJSONLOutput creates a JSONL output writer that appends to filename
// (truncates on open).
func NewJSONLOutput(filename string) (*JSONLOutput, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return nil, err
	}
	gologger.Infof("JSONL output: %s\n", filename)
	return &JSONLOutput{
		filename: filename,
		file:     file,
		bw:       bufio.NewWriterSize(file, 64*1024),
	}, nil
}

// WriteDomainResult appends a JSON line for r.
// The write is buffered; data is flushed on Close().
func (j *JSONLOutput) WriteDomainResult(r result.Result) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	recordType, records := parseAnswers(r.Answers)

	rec := JSONLRecord{
		Domain:    r.Subdomain,
		Type:      recordType,
		Records:   records,
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}

	if _, err = j.bw.Write(data); err != nil {
		return err
	}
	return j.bw.WriteByte('\n')
}

// Close flushes buffered data and closes the underlying file.
func (j *JSONLOutput) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.file == nil {
		return nil
	}

	if err := j.bw.Flush(); err != nil {
		_ = j.file.Close()
		return err
	}
	gologger.Infof("JSONL output complete: %s\n", j.filename)
	return j.file.Close()
}

// parseAnswers extracts the record type and cleaned record values from
// the raw answer strings produced by the DNS layer.
func parseAnswers(answers []string) (recordType string, records []string) {
	recordType = "A"
	records = make([]string, 0, len(answers))

	for _, answer := range answers {
		switch {
		case strings.HasPrefix(answer, "CNAME "):
			recordType = "CNAME"
			records = append(records, answer[6:])
		case strings.HasPrefix(answer, "NS "):
			recordType = "NS"
			records = append(records, answer[3:])
		case strings.HasPrefix(answer, "PTR "):
			recordType = "PTR"
			records = append(records, answer[4:])
		case strings.HasPrefix(answer, "TXT "):
			recordType = "TXT"
			records = append(records, answer[4:])
		case strings.HasPrefix(answer, "AAAA "):
			recordType = "AAAA"
			records = append(records, answer[5:])
		default:
			records = append(records, answer) // plain IP → A record
		}
	}

	if len(records) == 0 {
		records = answers
	}
	return
}
