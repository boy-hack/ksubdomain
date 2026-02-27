#!/bin/bash

# ksubdomain benchmark script
# Tests the performance of the optimized ksubdomain

# Color definitions
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No color

# Test configuration
TEST_DOMAIN="baidu.com"
SMALL_DICT="test/dict_small.txt"   # 1,000 subdomains
MEDIUM_DICT="test/dict_medium.txt" # 10,000 subdomains
LARGE_DICT="test/dict_large.txt"   # 100,000 subdomains
RESOLVERS="test/resolvers.txt"
OUTPUT_DIR="test/results"
ORIG_BIN="test/ksubdomain_orig"
NEW_BIN="./ksubdomain"

# Ensure test directories exist
mkdir -p "$OUTPUT_DIR"
mkdir -p "test"

# Check whether the original binary exists
if [ ! -f "$ORIG_BIN" ]; then
  echo -e "${YELLOW}Original binary not found; skipping comparison tests${NC}"
  SKIP_COMPARE=true
else
  SKIP_COMPARE=false
fi

# Create the DNS resolver file if it does not exist
if [ ! -f "$RESOLVERS" ]; then
  echo "Creating DNS resolver file..."
  echo "8.8.8.8" > "$RESOLVERS"
  echo "8.8.4.4" >> "$RESOLVERS"
  echo "1.1.1.1" >> "$RESOLVERS"
  echo "114.114.114.114" >> "$RESOLVERS"
fi

# Create a test dictionary if it does not exist
create_test_dict() {
  local file=$1
  local size=$2
  
  if [ ! -f "$file" ]; then
    echo "Creating test dictionary $file ($size entries)..."
    for i in $(seq 1 $size); do
      echo "sub$i.$TEST_DOMAIN" >> "$file"
    done
  fi
}

create_test_dict "$SMALL_DICT" 1000
create_test_dict "$MEDIUM_DICT" 10000
create_test_dict "$LARGE_DICT" 100000

# Run a single test
run_test() {
  local bin=$1
  local mode=$2
  local dict=$3
  local output=$4
  local extra_params=$5
  local dict_size=$(wc -l < "$dict")
  
  echo "Testing $bin in $mode mode, dictionary size: $dict_size $extra_params"
  
  # Remove old output file
  if [ -f "$output" ]; then
    rm "$output"
  fi
  
  # Run and time the test
  local start_time=$(date +%s.%N)
  
  if [ "$mode" == "verify" ]; then
    $bin v -f "$dict" -r "$RESOLVERS" -o "$output" $extra_params --np
  else
    $bin e -d "$TEST_DOMAIN" -f "$dict" -r "$RESOLVERS" -o "$output" $extra_params --np
  fi
  
  local end_time=$(date +%s.%N)
  local elapsed=$(echo "$end_time - $start_time" | bc)
  local found=$(wc -l < "$output")
  
  echo -e "${GREEN}Done! Time: $elapsed sec, Found: $found subdomains${NC}"
  echo ""
  
  # Return results
  echo "$elapsed,$found"
}

# Run all tests for a given binary
run_all_tests() {
  local bin=$1
  local prefix=$2
  
  # Small dictionary, verify mode
  small_verify=$(run_test "$bin" "verify" "$SMALL_DICT" "${OUTPUT_DIR}/${prefix}_small_verify.txt" "-b 5m")
  
  # Small dictionary, enum mode
  small_enum=$(run_test "$bin" "enum" "$SMALL_DICT" "${OUTPUT_DIR}/${prefix}_small_enum.txt" "-b 5m")
  
  # Medium dictionary, verify mode
  medium_verify=$(run_test "$bin" "verify" "$MEDIUM_DICT" "${OUTPUT_DIR}/${prefix}_medium_verify.txt" "-b 5m")
  
  # Large dictionary, verify mode
  large_verify=$(run_test "$bin" "verify" "$LARGE_DICT" "${OUTPUT_DIR}/${prefix}_large_verify.txt" "-b 5m")
  
  # Test with different timeout and retry parameters
  retry_test=$(run_test "$bin" "verify" "$MEDIUM_DICT" "${OUTPUT_DIR}/${prefix}_retry_test.txt" "-b 5m --retry 5 --timeout 8")
  
  # Return all results
  echo "$small_verify|$small_enum|$medium_verify|$large_verify|$retry_test"
}

# Main function
main() {
  echo "========================================"
  echo "   KSubdomain Performance Test"
  echo "========================================"
  
  # Test the new (optimized) version
  echo -e "${YELLOW}Testing optimized version...${NC}"
  new_results=$(run_all_tests "$NEW_BIN" "new")
  
  # If the original binary is available, run a comparison
  if [ "$SKIP_COMPARE" = false ]; then
    echo -e "${YELLOW}Testing original version...${NC}"
    orig_results=$(run_all_tests "$ORIG_BIN" "orig")
    
    # Parse results
    IFS='|' read -r new_small_verify new_small_enum new_medium_verify new_large_verify new_retry_test <<< "$new_results"
    IFS='|' read -r orig_small_verify orig_small_enum orig_medium_verify orig_large_verify orig_retry_test <<< "$orig_results"
    
    # Extract time and found counts
    IFS=',' read -r new_small_verify_time new_small_verify_found <<< "$new_small_verify"
    IFS=',' read -r orig_small_verify_time orig_small_verify_found <<< "$orig_small_verify"
    
    IFS=',' read -r new_small_enum_time new_small_enum_found <<< "$new_small_enum"
    IFS=',' read -r orig_small_enum_time orig_small_enum_found <<< "$orig_small_enum"
    
    IFS=',' read -r new_medium_verify_time new_medium_verify_found <<< "$new_medium_verify"
    IFS=',' read -r orig_medium_verify_time orig_medium_verify_found <<< "$orig_medium_verify"
    
    IFS=',' read -r new_large_verify_time new_large_verify_found <<< "$new_large_verify"
    IFS=',' read -r orig_large_verify_time orig_large_verify_found <<< "$orig_large_verify"
    
    IFS=',' read -r new_retry_test_time new_retry_test_found <<< "$new_retry_test"
    IFS=',' read -r orig_retry_test_time orig_retry_test_found <<< "$orig_retry_test"
    
    # Calculate speed-up percentages
    small_verify_speedup=$(echo "scale=2; ($orig_small_verify_time - $new_small_verify_time) / $orig_small_verify_time * 100" | bc)
    small_enum_speedup=$(echo "scale=2; ($orig_small_enum_time - $new_small_enum_time) / $orig_small_enum_time * 100" | bc)
    medium_verify_speedup=$(echo "scale=2; ($orig_medium_verify_time - $new_medium_verify_time) / $orig_medium_verify_time * 100" | bc)
    large_verify_speedup=$(echo "scale=2; ($orig_large_verify_time - $new_large_verify_time) / $orig_large_verify_time * 100" | bc)
    retry_test_speedup=$(echo "scale=2; ($orig_retry_test_time - $new_retry_test_time) / $orig_retry_test_time * 100" | bc)
    
    # Print comparison results
    echo ""
    echo "========================================"
    echo "   Performance Comparison Results"
    echo "========================================"
    echo "Small dictionary, verify mode:"
    echo "  Original:  $orig_small_verify_time sec, found: $orig_small_verify_found domains"
    echo "  Optimized: $new_small_verify_time sec, found: $new_small_verify_found domains"
    echo -e "  Speed-up:  ${GREEN}$small_verify_speedup%${NC}"
    echo ""
    
    echo "Small dictionary, enum mode:"
    echo "  Original:  $orig_small_enum_time sec, found: $orig_small_enum_found domains"
    echo "  Optimized: $new_small_enum_time sec, found: $new_small_enum_found domains"
    echo -e "  Speed-up:  ${GREEN}$small_enum_speedup%${NC}"
    echo ""
    
    echo "Medium dictionary, verify mode:"
    echo "  Original:  $orig_medium_verify_time sec, found: $orig_medium_verify_found domains"
    echo "  Optimized: $new_medium_verify_time sec, found: $new_medium_verify_found domains"
    echo -e "  Speed-up:  ${GREEN}$medium_verify_speedup%${NC}"
    echo ""
    
    echo "Large dictionary, verify mode:"
    echo "  Original:  $orig_large_verify_time sec, found: $orig_large_verify_found domains"
    echo "  Optimized: $new_large_verify_time sec, found: $new_large_verify_found domains"
    echo -e "  Speed-up:  ${GREEN}$large_verify_speedup%${NC}"
    echo ""
    
    echo "Retry parameter test:"
    echo "  Original:  $orig_retry_test_time sec, found: $orig_retry_test_found domains"
    echo "  Optimized: $new_retry_test_time sec, found: $new_retry_test_found domains"
    echo -e "  Speed-up:  ${GREEN}$retry_test_speedup%${NC}"
    echo ""
    
    # Calculate average speed-up
    avg_speedup=$(echo "scale=2; ($small_verify_speedup + $small_enum_speedup + $medium_verify_speedup + $large_verify_speedup + $retry_test_speedup) / 5" | bc)
    echo -e "Average speed-up: ${GREEN}$avg_speedup%${NC}"
  else
    echo ""
    echo "========================================"
    echo "   Test Results (optimized version only)"
    echo "========================================"
    
    # Parse results
    IFS='|' read -r new_small_verify new_small_enum new_medium_verify new_large_verify new_retry_test <<< "$new_results"
    
    # Extract time and found counts
    IFS=',' read -r new_small_verify_time new_small_verify_found <<< "$new_small_verify"
    IFS=',' read -r new_small_enum_time new_small_enum_found <<< "$new_small_enum"
    IFS=',' read -r new_medium_verify_time new_medium_verify_found <<< "$new_medium_verify"
    IFS=',' read -r new_large_verify_time new_large_verify_found <<< "$new_large_verify"
    IFS=',' read -r new_retry_test_time new_retry_test_found <<< "$new_retry_test"
    
    echo "Small dictionary, verify mode:  $new_small_verify_time sec, found: $new_small_verify_found domains"
    echo "Small dictionary, enum mode:    $new_small_enum_time sec, found: $new_small_enum_found domains"
    echo "Medium dictionary, verify mode: $new_medium_verify_time sec, found: $new_medium_verify_found domains"
    echo "Large dictionary, verify mode:  $new_large_verify_time sec, found: $new_large_verify_found domains"
    echo "Retry parameter test:           $new_retry_test_time sec, found: $new_retry_test_found domains"
  fi
  
  echo ""
  echo "Test results saved in $OUTPUT_DIR"
  echo "Test complete!"
}

# Execute main function
main
