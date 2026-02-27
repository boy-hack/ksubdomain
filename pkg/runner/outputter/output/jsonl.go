package output

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/boy-hack/ksubdomain/v2/pkg/gologger"
	"github.com/boy-hack/ksubdomain/v2/pkg/runner/result"
)

// JSONLOutput is the JSONL (JSON Lines) output handler.
// Each line contains one JSON object, suitable for streaming processing and toolchain integration.
// Format: {"domain":"example.com","type":"A","records":["1.2.3.4"],"timestamp":1234567890}
type JSONLOutput struct {
	filename string
	file     *os.File
	mu       sync.Mutex
}

// JSONLRecord is the JSONL record format
type JSONLRecord struct {
	Domain    string   `json:"domain"`           // Subdomain
	Type      string   `json:"type"`             // Record type (A, CNAME, NS, etc.)
	Records   []string `json:"records"`          // Record values
	Timestamp int64    `json:"timestamp"`        // Unix timestamp
	TTL       uint32   `json:"ttl,omitempty"`    // TTL (optional)
	Source    string   `json:"source,omitempty"` // Data source (optional)
}

// NewJSONLOutput creates a JSONL output handler
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

// WriteDomainResult writes a single domain result.
// JSONL format writes one JSON line per call and flushes immediately.
// Benefit: supports streaming processing; results can be read in real time.
func (j *JSONLOutput) WriteDomainResult(r result.Result) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	// Parse record type
	recordType := "A" // default A record
	records := make([]string, 0, len(r.Answers))

	for _, answer := range r.Answers {
		// Parse type (CNAME, NS, PTR, etc.)
		if len(answer) > 0 {
			// Check for special record types
			if len(answer) > 6 && answer[:6] == "CNAME " {
				recordType = "CNAME"
				records = append(records, answer[6:]) // Strip "CNAME " prefix
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
				// IP address (A or AAAA record)
				records = append(records, answer)
			}
		}
	}

	// If no records were parsed, use raw answers
	if len(records) == 0 {
		records = r.Answers
	}

	// Build JSONL record
	record := JSONLRecord{
		Domain:    r.Subdomain,
		Type:      recordType,
		Records:   records,
		Timestamp: time.Now().Unix(),
	}

	// Serialize to JSON
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	// Write one line (JSON + newline)
	_, err = j.file.Write(append(data, '\n'))
	if err != nil {
		return err
	}

	// Flush to disk immediately (supports real-time reading)
	return j.file.Sync()
}

// Close closes the JSONL output handler
func (j *JSONLOutput) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.file != nil {
		gologger.Infof("JSONL output completed: %s\n", j.filename)
		return j.file.Close()
	}
	return nil
}
