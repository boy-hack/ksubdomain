# Output Formats Guide

KSubdomain supports multiple output formats for different use cases.

## 📋 Supported Formats

| Format | Extension | Use Case | Streaming | Beautified |
|--------|-----------|----------|-----------|------------|
| **TXT** | .txt | Default, human-readable | ✅ | ❌ |
| **JSON** | .json | Structured data | ❌ | ❌ |
| **CSV** | .csv | Spreadsheet compatible | ❌ | ❌ |
| **JSONL** | .jsonl | Tool chaining, streaming | ✅ | ❌ |
| **Beautified** | - | Enhanced terminal output | ✅ | ✅ |

---

## 1. TXT Format (Default)

Simple text format, one result per line.

### Usage
```bash
./ksubdomain enum -d example.com -o results.txt
# or
./ksubdomain enum -d example.com --oy txt -o results.txt
```

### Output
```
www.example.com => 93.184.216.34
mail.example.com => CNAME mail.google.com
api.example.com => 93.184.216.35
```

### Best For
- Quick viewing
- Manual analysis
- Simple text processing

---

## 2. JSON Format

Structured JSON output, all results in one object.

### Usage
```bash
./ksubdomain enum -d example.com --oy json -o results.json
```

### Output
```json
{
  "domains": [
    {
      "subdomain": "www.example.com",
      "answers": ["93.184.216.34"]
    },
    {
      "subdomain": "mail.example.com",
      "answers": ["CNAME mail.google.com"]
    }
  ]
}
```

### Best For
- Structured data processing
- Web API integration
- Complete result sets

---

## 3. CSV Format

Comma-separated values for spreadsheet applications.

### Usage
```bash
./ksubdomain enum -d example.com --oy csv -o results.csv
```

### Output
```csv
subdomain,type,record
www.example.com,A,93.184.216.34
mail.example.com,CNAME,mail.google.com
api.example.com,A,93.184.216.35
```

### Best For
- Excel/Google Sheets
- Data analysis
- Reporting

---

## 4. JSONL Format (JSON Lines) 🆕

One JSON object per line, perfect for streaming.

### Usage
```bash
./ksubdomain enum -d example.com --oy jsonl -o results.jsonl
```

### Output
```jsonl
{"domain":"www.example.com","type":"A","records":["93.184.216.34"],"timestamp":1709011200}
{"domain":"mail.example.com","type":"CNAME","records":["mail.google.com"],"timestamp":1709011201}
{"domain":"api.example.com","type":"A","records":["93.184.216.35"],"timestamp":1709011202}
```

### Processing with jq
```bash
# Extract domains
./ksubdomain enum -d example.com --oy jsonl | jq -r '.domain'

# Filter A records only
./ksubdomain enum -d example.com --oy jsonl | jq -r 'select(.type == "A") | .domain'

# Extract CNAME targets
./ksubdomain enum -d example.com --oy jsonl | jq -r 'select(.type == "CNAME") | .records[0]'

# Filter by timestamp
./ksubdomain enum -d example.com --oy jsonl | jq -r 'select(.timestamp > 1709011000) | .domain'
```

### Integration Examples

#### With httpx
```bash
./ksubdomain enum -d example.com --oy jsonl | \
  jq -r '.domain' | \
  httpx -silent
```

#### With nuclei
```bash
./ksubdomain enum -d example.com --oy jsonl | \
  jq -r '.domain' | \
  nuclei -l /dev/stdin
```

#### In Python
```python
import subprocess
import json

proc = subprocess.Popen(
    ['ksubdomain', 'enum', '-d', 'example.com', '--oy', 'jsonl'],
    stdout=subprocess.PIPE,
    text=True
)

for line in proc.stdout:
    data = json.loads(line)
    print(f"{data['domain']} => {data['records']}")
```

#### In Node.js
```javascript
const { spawn } = require('child_process');
const readline = require('readline');

const proc = spawn('ksubdomain', ['enum', '-d', 'example.com', '--oy', 'jsonl']);
const rl = readline.createInterface({ input: proc.stdout });

rl.on('line', (line) => {
  const data = JSON.parse(line);
  console.log(`${data.domain} => ${data.records}`);
});
```

### Best For
- **Streaming processing** (real-time)
- **Tool chaining** (pipes)
- **Log aggregation**
- **Time-series analysis**

---

## 5. Beautified Output 🎨

Enhanced terminal output with colors, emojis, and summary.

### Usage
```bash
# Enable colors
./ksubdomain enum -d example.com --color

# Full beautified mode
./ksubdomain enum -d example.com --beautify

# Beautified with file output
./ksubdomain enum -d example.com --beautify -o results.txt
```

### Output Example
```
✓ www.example.com                          93.184.216.34
✓ mail.example.com                [CNAME]  mail.google.com
✓ api.example.com                          93.184.216.35
✓ ftp.example.com                          93.184.216.36

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Scan Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Total Found:  125
  Time Elapsed: 5.2s
  Speed:        2403 domains/s

  Record Types:
    A: 120 (96.0%)
    CNAME: 4 (3.2%)
    NS: 1 (0.8%)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### Color Scheme
- ✅ **Green checkmark** - Success
- 🔵 **Blue [CNAME]** - CNAME records
- 🟡 **Yellow [NS]** - NS records
- 🔴 **Red [WARN]** - Wildcards or warnings

### Best For
- Terminal viewing
- Presentations
- Demos
- Human-readable output

---

## 🔄 Format Comparison

### Which format to use?

| Scenario | Recommended Format |
|----------|-------------------|
| Quick manual check | TXT or Beautified |
| Piping to other tools | JSONL |
| Data analysis | CSV or JSON |
| Real-time processing | JSONL |
| Web API response | JSON |
| Reporting | CSV or Beautified |
| Tool integration | JSONL |

---

## 📚 Examples

### Save to multiple formats
```bash
# Save TXT and JSON
./ksubdomain enum -d example.com -o results.txt
./ksubdomain enum -d example.com --oy json -o results.json

# Save JSONL for processing
./ksubdomain enum -d example.com --oy jsonl -o results.jsonl

# Beautified terminal + JSON file
./ksubdomain enum -d example.com --beautify --oy json -o results.json
```

### Real-time monitoring
```bash
# Stream to terminal with beautification
./ksubdomain enum -d example.com --beautify

# Stream to file in JSONL format
./ksubdomain enum -d example.com --oy jsonl -o results.jsonl &
tail -f results.jsonl | jq -r '.domain'
```

---

## 🎨 Beautified Output Details

### Features

1. **Color-coded types**
   - A records: Green ✓
   - CNAME: Blue [CNAME]
   - NS: Yellow [NS]

2. **Aligned output**
   - Domain names aligned to 40 chars
   - Clean table-like appearance

3. **Summary statistics**
   - Total count
   - Time elapsed
   - Speed (domains/s)
   - Type distribution

4. **Emoji indicators**
   - ✅ Success
   - 📊 Summary
   - ⚡ Speed
   - 📋 Types

### Disable colors
```bash
# Force disable (for piping/redirecting)
./ksubdomain enum -d example.com --beautify --no-color
```

---

**Choose the right format for your workflow! 🎯**
