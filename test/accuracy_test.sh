#!/bin/bash

# ksubdomain accuracy test script
# Tests that the optimized version produces results consistent with the original version

# Color definitions
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No color

# Test configuration
TEST_DOMAIN="example.com"
TEST_DICT="test/accuracy_dict.txt"
RESOLVERS="test/resolvers.txt"
ORIG_BIN="test/ksubdomain_orig"
NEW_BIN="./ksubdomain"
ORIG_OUTPUT="test/results/orig_accuracy.txt"
NEW_OUTPUT="test/results/new_accuracy.txt"
DIFF_OUTPUT="test/results/diff.txt"

# Ensure the test results directory exists
mkdir -p "test/results"

# Check whether the original binary exists
if [ ! -f "$ORIG_BIN" ]; then
  echo -e "${RED}Error: original binary not found at $ORIG_BIN${NC}"
  echo "Please place the original ksubdomain binary in the test/ directory and rename it to ksubdomain_orig"
  exit 1
fi

# Create the DNS resolver file if it does not exist
if [ ! -f "$RESOLVERS" ]; then
  echo "Creating DNS resolver file..."
  echo "8.8.8.8" > "$RESOLVERS"
  echo "8.8.4.4" >> "$RESOLVERS"
  echo "1.1.1.1" >> "$RESOLVERS"
  echo "114.114.114.114" >> "$RESOLVERS"
fi

# Create the test dictionary if it does not exist
if [ ! -f "$TEST_DICT" ]; then
  echo "Creating test dictionary..."
  # Common subdomains that are likely to exist
  echo "www.$TEST_DOMAIN" > "$TEST_DICT"
  echo "mail.$TEST_DOMAIN" >> "$TEST_DICT"
  echo "api.$TEST_DOMAIN" >> "$TEST_DICT"
  echo "blog.$TEST_DOMAIN" >> "$TEST_DICT"
  echo "docs.$TEST_DOMAIN" >> "$TEST_DICT"
  # Randomly generated subdomains
  for i in {1..95}; do
    echo "test$i.$TEST_DOMAIN" >> "$TEST_DICT"
  done
fi

echo "========================================"
echo "   KSubdomain Accuracy Test"
echo "========================================"

# Run the original version
echo -e "${YELLOW}Running original version...${NC}"
$ORIG_BIN v -f "$TEST_DICT" -r "$RESOLVERS" -o "$ORIG_OUTPUT" -b 5m --retry 3 --timeout 6

# Run the optimized version
echo -e "${YELLOW}Running optimized version...${NC}"
$NEW_BIN v -f "$TEST_DICT" -r "$RESOLVERS" -o "$NEW_OUTPUT" -b 5m --retry 3 --timeout 6

# Compare results
echo -e "${YELLOW}Comparing results...${NC}"
# Sort both output files
sort "$ORIG_OUTPUT" > "$ORIG_OUTPUT.sorted"
sort "$NEW_OUTPUT" > "$NEW_OUTPUT.sorted"

# Diff the sorted results
diff "$ORIG_OUTPUT.sorted" "$NEW_OUTPUT.sorted" > "$DIFF_OUTPUT"

if [ -s "$DIFF_OUTPUT" ]; then
  DIFF_COUNT=$(wc -l < "$DIFF_OUTPUT")
  echo -e "${RED}Differences found! $DIFF_COUNT lines differ${NC}"
  echo "Diff details saved to $DIFF_OUTPUT"
  echo "Diff contents:"
  cat "$DIFF_OUTPUT"
else
  echo -e "${GREEN}Test passed! Both versions produce identical results${NC}"
  # Report the number of subdomains found
  FOUND_COUNT=$(wc -l < "$NEW_OUTPUT")
  echo "Found $FOUND_COUNT subdomains"
  rm "$DIFF_OUTPUT" # Remove the empty diff file
fi

# Clean up temporary files
rm "$ORIG_OUTPUT.sorted" "$NEW_OUTPUT.sorted"

echo "Test complete!"
