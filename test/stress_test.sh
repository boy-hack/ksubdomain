#!/bin/bash

# ksubdomain stress test script
# Tests ksubdomain performance under high load

# Color definitions
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No color

# Test configuration
DOMAIN="example.com"
DICT_LARGE="test/stress_dict.txt"
RESOLVERS="test/resolvers.txt"
OUTPUT_DIR="test/results"
BIN="./ksubdomain"

# Ensure test directories exist
mkdir -p "$OUTPUT_DIR"

# Create the DNS resolver file if it does not exist
if [ ! -f "$RESOLVERS" ]; then
  echo "Creating DNS resolver file..."
  echo "8.8.8.8" > "$RESOLVERS"
  echo "8.8.4.4" >> "$RESOLVERS"
  echo "1.1.1.1" >> "$RESOLVERS"
  echo "114.114.114.114" >> "$RESOLVERS"
fi

# Create the large dictionary if it does not exist
if [ ! -f "$DICT_LARGE" ]; then
  echo "Creating large dictionary..."
  for i in $(seq 1 500000); do
    echo "stress$i.$DOMAIN" >> "$DICT_LARGE"
  done
  echo "Created dictionary with 500,000 entries"
fi

# Print system information
echo "========================================"
echo "   System Information"
echo "========================================"
echo "OS:             $(uname -s)"
echo "Processor:      $(uname -p)"
echo "Kernel version: $(uname -r)"

if [ "$(uname -s)" = "Linux" ]; then
  echo "CPU cores: $(nproc)"
  echo "Memory:    $(free -h | grep Mem | awk '{print $2}')"
elif [ "$(uname -s)" = "Darwin" ]; then
  echo "CPU cores: $(sysctl -n hw.ncpu)"
  echo "Memory:    $(sysctl -n hw.memsize | awk '{print $1/1024/1024/1024 " GB"}')"
fi

echo ""
echo "========================================"
echo "   KSubdomain Stress Test"
echo "========================================"

# Function to run a single stress test at a given rate
run_stress_test() {
  local rate=$1
  local output="${OUTPUT_DIR}/stress_${rate}.txt"
  local log="${OUTPUT_DIR}/stress_${rate}.log"
  
  echo -e "${YELLOW}Testing rate: $rate pps${NC}"
  
  # Clean up old files
  [ -f "$output" ] && rm "$output"
  [ -f "$log" ] && rm "$log"
  
  echo "Starting test..."
  
  # Run and time the test
  start_time=$(date +%s.%N)
  
  # Redirect stdout and stderr to the log file
  $BIN v -f "$DICT_LARGE" -r "$RESOLVERS" -o "$output" -b "$rate" --retry 2 --timeout 4 --np > "$log" 2>&1
  
  end_time=$(date +%s.%N)
  elapsed=$(echo "$end_time - $start_time" | bc)
  
  # Parse results
  processed_count=$(cat "$log" | grep -o "success:[0-9]*" | tail -1 | grep -o "[0-9]*")
  found_count=$(wc -l < "$output")
  
  # Calculate throughput
  domains_per_sec=$(echo "$processed_count / $elapsed" | bc)
  
  echo -e "${GREEN}Test complete${NC}"
  echo "Domains processed: $processed_count"
  echo "Subdomains found:  $found_count"
  echo "Elapsed time:      $elapsed sec"
  echo "Throughput:        $domains_per_sec domains/sec"
  echo ""
  
  # Return result string for collection
  echo "$rate,$elapsed,$processed_count,$found_count,$domains_per_sec"
}

# Run tests at different rates
echo -e "${YELLOW}Running stress tests at various rates...${NC}"
echo ""

# Create the results CSV file
RESULT_CSV="${OUTPUT_DIR}/stress_results.csv"
echo "rate(pps),elapsed(sec),processed,found,throughput(domains/sec)" > "$RESULT_CSV"

# Gradually increase the rate
for rate in "10k" "50k" "100k" "200k" "500k" "1m"; do
  result=$(run_stress_test "$rate")
  echo "$result" >> "$RESULT_CSV"
  
  # Brief pause to let the system cool down
  sleep 5
done

echo "========================================"
echo "   Memory Usage Test"
echo "========================================"

# Monitor runtime memory usage
echo -e "${YELLOW}Testing runtime memory usage...${NC}"
MEMORY_LOG="${OUTPUT_DIR}/memory_usage.log"
MEM_OUTPUT="${OUTPUT_DIR}/memory_test.txt"

# Clean up old files
[ -f "$MEMORY_LOG" ] && rm "$MEMORY_LOG"
[ -f "$MEM_OUTPUT" ] && rm "$MEM_OUTPUT"

echo "Starting test..."

# Run ksubdomain in the background
$BIN v -f "$DICT_LARGE" -r "$RESOLVERS" -o "$MEM_OUTPUT" -b "100k" --retry 2 --timeout 4 --np > /dev/null 2>&1 &
PID=$!

# Monitor memory usage for 10 seconds
echo "PID: $PID"
echo "Monitoring memory usage..."

for i in {1..10}; do
  if [ "$(uname -s)" = "Linux" ]; then
    # Get RSS memory usage on Linux
    MEM=$(ps -p $PID -o rss= 2>/dev/null)
    if [ ! -z "$MEM" ]; then
      MEM_MB=$(echo "scale=2; $MEM / 1024" | bc)
      echo "Memory usage #$i: ${MEM_MB} MB" | tee -a "$MEMORY_LOG"
    else
      echo "Process has ended"
      break
    fi
  elif [ "$(uname -s)" = "Darwin" ]; then
    # Get memory usage on macOS
    MEM=$(ps -p $PID -o rss= 2>/dev/null)
    if [ ! -z "$MEM" ]; then
      MEM_MB=$(echo "scale=2; $MEM / 1024" | bc)
      echo "Memory usage #$i: ${MEM_MB} MB" | tee -a "$MEMORY_LOG"
    else
      echo "Process has ended"
      break
    fi
  fi
  sleep 1
done

# Terminate the process after 10 seconds
if kill -0 $PID 2>/dev/null; then
  echo "Terminating process..."
  kill $PID
fi

# Report peak memory usage
if [ -f "$MEMORY_LOG" ]; then
  MAX_MEM=$(cat "$MEMORY_LOG" | grep -o "[0-9]\+\.[0-9]\+ MB" | sort -nr | head -1)
  echo -e "${GREEN}Peak memory usage: $MAX_MEM${NC}"
fi

echo ""
echo "Stress test complete!"
echo "Results saved to $RESULT_CSV"
