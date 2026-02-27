#!/bin/bash
#
# KSubdomain Test Runner
# Purpose: Run all tests and generate a report
#

set -e

# Color output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  KSubdomain Test Suite${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Check Go environment
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go environment not found${NC}"
    exit 1
fi

echo -e "${YELLOW}Go version:${NC} $(go version)"
echo ""

# Create test results directory
mkdir -p test_results

# 1. Unit tests
echo -e "${GREEN}[1/5] Running unit tests...${NC}"
go test -v -cover -coverprofile=test_results/coverage.out ./pkg/... 2>&1 | tee test_results/unit_test.log

if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo -e "${GREEN}✓ Unit tests passed${NC}"
else
    echo -e "${RED}✗ Unit tests failed${NC}"
    exit 1
fi
echo ""

# 2. Coverage report
echo -e "${GREEN}[2/5] Generating coverage report...${NC}"
go tool cover -html=test_results/coverage.out -o test_results/coverage.html
COVERAGE=$(go tool cover -func=test_results/coverage.out | grep total | awk '{print $3}')
echo -e "${YELLOW}Total coverage:${NC} $COVERAGE"

# Check whether coverage meets the threshold
COVERAGE_NUM=$(echo $COVERAGE | sed 's/%//')
if (( $(echo "$COVERAGE_NUM >= 60" | bc -l) )); then
    echo -e "${GREEN}✓ Coverage meets threshold (> 60%)${NC}"
else
    echo -e "${YELLOW}⚠ Coverage is low (< 60%)${NC}"
fi
echo ""

# 3. Benchmark tests
echo -e "${GREEN}[3/5] Running benchmark tests...${NC}"
go test -bench=. -benchmem ./pkg/... 2>&1 | tee test_results/benchmark.log

if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo -e "${GREEN}✓ Benchmark tests completed${NC}"
else
    echo -e "${YELLOW}⚠ Some benchmark tests failed${NC}"
fi
echo ""

# 4. Race condition detection
echo -e "${GREEN}[4/5] Running data race detection...${NC}"
go test -race ./pkg/runner/statusdb/ 2>&1 | tee test_results/race.log

if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo -e "${GREEN}✓ No data races detected${NC}"
else
    echo -e "${RED}✗ Data race detected${NC}"
    exit 1
fi
echo ""

# 5. Static analysis (optional)
echo -e "${GREEN}[5/5] Static code analysis...${NC}"
if command -v golangci-lint &> /dev/null; then
    golangci-lint run ./... 2>&1 | tee test_results/lint.log
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        echo -e "${GREEN}✓ Static analysis passed${NC}"
    else
        echo -e "${YELLOW}⚠ Static analysis found issues${NC}"
    fi
else
    echo -e "${YELLOW}⚠ golangci-lint not installed; skipping static analysis${NC}"
fi
echo ""

# Generate test report
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Test Summary${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "${YELLOW}Unit tests:${NC}    Passed ✓"
echo -e "${YELLOW}Coverage:${NC}      $COVERAGE"
echo -e "${YELLOW}Data races:${NC}    None ✓"
echo -e "${YELLOW}Report location:${NC} test_results/"
echo ""
echo -e "${GREEN}Test results:${NC}"
echo -e "  - coverage.html:  Visual coverage report"
echo -e "  - unit_test.log:  Unit test detailed log"
echo -e "  - benchmark.log:  Benchmark test results"
echo -e "  - race.log:       Race detection log"
echo ""
echo -e "${GREEN}All tests complete! 🎉${NC}"
