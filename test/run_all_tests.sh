#!/bin/bash

# ksubdomain test runner script
# Runs all tests

# Color definitions
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No color

# Check for root privileges
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}Warning: this test script requires root privileges to run correctly${NC}"
  echo "Please run with sudo"
  exit 1
fi

# Ensure the test results directory exists
mkdir -p "test/results"

# Clean up old test results
echo -e "${YELLOW}Cleaning up old test results...${NC}"
rm -rf test/results/*

# Set execute permissions
chmod +x test/benchmark.sh
chmod +x test/accuracy_test.sh
chmod +x test/stress_test.sh

echo "========================================"
echo -e "${BLUE}KSubdomain Test Suite${NC}"
echo "========================================"
echo ""

# Ask whether an original binary is available for comparison
read -p "Is an original ksubdomain binary available for comparison testing? (y/n): " has_original
if [ "$has_original" = "y" ] || [ "$has_original" = "Y" ]; then
  read -p "Enter the path to the original ksubdomain binary: " orig_path
  if [ -f "$orig_path" ]; then
    echo "Copying original binary to test directory..."
    cp "$orig_path" "test/ksubdomain_orig"
    chmod +x "test/ksubdomain_orig"
  else
    echo -e "${RED}Error: the specified path does not exist${NC}"
    exit 1
  fi
fi

# Menu: select which tests to run
echo ""
echo "Select the tests to run:"
echo "1) Performance benchmark (test various dictionary sizes)"
echo "2) Accuracy test (compare result consistency)"
echo "3) Stress test (test performance under high load)"
echo "4) Run all tests"
echo "0) Exit"
echo ""

read -p "Enter option [0-4]: " choice

case $choice in
  1)
    echo -e "${YELLOW}Running performance benchmark...${NC}"
    test/benchmark.sh
    ;;
  2)
    if [ -f "test/ksubdomain_orig" ]; then
      echo -e "${YELLOW}Running accuracy test...${NC}"
      test/accuracy_test.sh
    else
      echo -e "${RED}Error: accuracy test requires the original ksubdomain binary${NC}"
      exit 1
    fi
    ;;
  3)
    echo -e "${YELLOW}Running stress test...${NC}"
    test/stress_test.sh
    ;;
  4)
    echo -e "${YELLOW}Running all tests...${NC}"
    
    echo -e "${BLUE}1. Performance benchmark${NC}"
    test/benchmark.sh
    
    if [ -f "test/ksubdomain_orig" ]; then
      echo -e "${BLUE}2. Accuracy test${NC}"
      test/accuracy_test.sh
    else
      echo -e "${RED}Skipping accuracy test: original binary not found${NC}"
    fi
    
    echo -e "${BLUE}3. Stress test${NC}"
    test/stress_test.sh
    ;;
  0)
    echo "Exiting"
    exit 0
    ;;
  *)
    echo -e "${RED}Invalid option${NC}"
    exit 1
    ;;
esac

echo ""
echo -e "${GREEN}All tests complete!${NC}"
echo "Test results saved in test/results/"
