# Performance Benchmark Tests

## 📊 Test Objectives

Validate ksubdomain's scanning performance as described in the README:

| Dictionary Size | Target Time | Reference (README) |
|----------------|-------------|-------------------|
| 1,000 domains  | < 2 sec     | -                 |
| 10,000 domains | < 5 sec     | -                 |
| 100,000 domains | **< 30 sec** | **README standard** |

---

## 🧪 Test Environment

### README Reference Configuration
- **CPU**: 4 cores
- **Bandwidth**: 5 M
- **Dictionary**: 100k domains (d2.txt)
- **DNS**: Custom DNS list (dns.txt)
- **Retry**: 3 times
- **Result**: ~30 sec, 1397 successes

### Requirements
- **Root privileges** (for NIC access)
- **Network connectivity** (for DNS queries)
- **libpcap** installed
- Recommended: **4-core CPU + 5 M bandwidth**

---

## 🚀 Running the Tests

### Quick Test (1,000 domains)
```bash
# Build and run
go test -tags=performance -bench=Benchmark1k ./test/ -timeout 10m

# Expected output:
# Benchmark1kDomains    1    1.5s    total_seconds:1.5
#                       950   success_count
#                       95%   success_rate_%
#                       666   domains/sec
```

### Medium Test (10,000 domains)
```bash
go test -tags=performance -bench=Benchmark10k ./test/ -timeout 10m

# Expected output:
# Benchmark10kDomains   1    4.8s    total_seconds:4.8
#                       9500  success_count
#                       95%   success_rate_%
#                       2083  domains/sec
```

### Full Test (100,000 domains) — README Standard
```bash
# Requires sudo (NIC access)
sudo go test -tags=performance -bench=Benchmark100k ./test/ -timeout 10m -v

# Expected output (per README):
# Benchmark100kDomains  1    28.5s   total_seconds:28.5
#                       95000 success_count
#                       95%   success_rate_%
#                       3508  domains/sec
#
# ✅ Excellent performance: 100k domains in 28.5 sec (meets README standard)
```

### Run All Performance Tests
```bash
sudo go test -tags=performance -bench=. ./test/ -timeout 15m -v
```

---

## 📈 Performance Metrics

### Reported Metrics

Each test reports the following metrics:

```
total_seconds    Total elapsed time (seconds)
success_count    Number of successfully resolved domains
success_rate_%   Success rate (percentage)
domains/sec      Scan throughput (domains per second)
```

### Log Output

During the test, real-time progress is displayed:
```
Progress: 1000/100000 (1.0%), rate: 3500 domains/s, elapsed: 0s
Progress: 2000/100000 (2.0%), rate: 3600 domains/s, elapsed: 0s
...
Final result: 95000/100000, elapsed: 28.5s
```

---

## 🎯 Performance Benchmark Comparison

### README Standard (100,000 domains)

| Tool | Time | Rate | Success | Ratio |
|------|------|------|---------|-------|
| **KSubdomain** | **~30 sec** | ~3333/s | 1397 | **1×** |
| massdns | ~3 min 29 sec | ~478/s | 1396 | **7× slower** |
| dnsx | ~5 min 26 sec | ~307/s | 1396 | **10× slower** |

### Our Test Targets

Based on the README standard:

```
✅ Excellent: < 30 sec  (meets README standard)
✓  Good:      30–40 sec (acceptable range)
⚠️  Warning:   40–60 sec (optimization needed)
❌ Fail:      > 60 sec  (performance issue)
```

---

## 🔧 Performance Tuning Tips

### If the Test Is Slow

#### 1. Check Bandwidth Limit
```bash
# Tests use 5 M bandwidth by default
# Try increasing it (in performance_benchmark_test.go)
Rate: options.Band2Rate("10m")  # Raise to 10 M
```

#### 2. Check DNS Servers
```bash
# Use faster DNS servers
Resolvers: []string{"8.8.8.8", "1.1.1.1"}
```

#### 3. Increase Retry Count (trade-off)
```bash
# More retries = higher success rate, but slower
Retry: 5  # Increase from 3 to 5
```

#### 4. Adjust Timeout
```bash
# Shorter timeout = faster, but may miss slow responses
TimeOut: 3  # Decrease from 6 to 3
```

---

## 📊 Collecting Performance Data

### Generate a Performance Report

```bash
# Run the test and save output
sudo go test -tags=performance -bench=Benchmark100k ./test/ \
    -timeout 10m -v 2>&1 | tee performance_report.txt

# Extract key metrics
grep "total_seconds\|success_count\|success_rate\|domains/sec" performance_report.txt
```

### Average Over Multiple Runs

```bash
# Run 3 times and average
for i in {1..3}; do
    echo "=== Run $i ==="
    sudo go test -tags=performance -bench=Benchmark100k ./test/ \
        -timeout 10m 2>&1 | grep "total_seconds"
done
```

---

## 🧩 Test Scenarios

### Scenario 1: Standard Test (README configuration)
```
Dictionary: 100,000 domains
Bandwidth:  5 M
Retry:      3 times
Timeout:    6 sec
Target:     < 30 sec
```

### Scenario 2: High-Speed Test (10 M bandwidth)
```
Dictionary: 100,000 domains
Bandwidth:  10 M
Retry:      3 times
Timeout:    6 sec
Target:     < 20 sec
```

### Scenario 3: Conservative Test (high success rate)
```
Dictionary: 100,000 domains
Bandwidth:  5 M
Retry:      10 times
Timeout:    10 sec
Target:     < 60 sec, success rate > 98%
```

---

## 📝 Pre-Test Checklist

Before running performance tests, confirm:

- [ ] Root privileges available
- [ ] Network connectivity is stable
- [ ] libpcap is installed
- [ ] Network adapter is functioning
- [ ] At least 4-core CPU
- [ ] At least 5 M bandwidth
- [ ] Other network-intensive programs are closed

---

## 🎯 Expected Results

### 1,000 domains
```
Total time:    ~1.5 sec
Success count: ~950
Success rate:  ~95%
Throughput:    ~666 domains/s
```

### 10,000 domains
```
Total time:    ~5 sec
Success count: ~9500
Success rate:  ~95%
Throughput:    ~2000 domains/s
```

### 100,000 domains (README standard)
```
Total time:    ~30 sec ✅
Success count: ~95000
Success rate:  ~95%
Throughput:    ~3333 domains/s
```

---

## 🐛 Common Issues

### Issue 1: Permission Error
```
Error: pcap initialization failed
Fix:   sudo go test -tags=performance ...
```

### Issue 2: Network Adapter Not Found
```
Error: No such device
Fix:   ./ksubdomain test       # Check available adapters
       --eth <adapter-name>    # Specify manually
```

### Issue 3: Test Timeout
```
Error: test timed out after 10m
Fix:   -timeout 15m            # Increase timeout
```

### Issue 4: Low Success Rate
```
Success rate: < 80%
Cause: Unstable network / slow DNS servers
Fix:   Increase retry count / switch DNS servers
```

---

## 📚 References

- README performance comparison: 100k domains in ~30 sec
- vs massdns: 7× performance gap
- vs dnsx: 10× performance gap

---

**Performance is everything! ⚡**

Run the test to validate ksubdomain's blazing speed:
```bash
sudo go test -tags=performance -bench=Benchmark100k ./test/ -timeout 10m -v
```
