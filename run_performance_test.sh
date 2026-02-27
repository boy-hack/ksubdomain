#!/bin/bash
#
# KSubdomain Performance Benchmark Script
# Purpose: Run the 100k-domain performance test to validate the metrics described in the README.
#
# README reference:
# - Dictionary: 100k domains
# - Bandwidth:  5 M
# - Time:       ~30 sec
# - Success:    1397 domains
#

set -e

# Color output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  KSubdomain Performance Benchmark${NC}"
echo -e "${GREEN}  README reference: 100k domains ~30 sec${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Check for root privileges
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}Error: root privileges are required to run performance tests${NC}"
    echo -e "${YELLOW}Please run: sudo $0${NC}"
    exit 1
fi

# Check Go environment
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go environment not found${NC}"
    exit 1
fi

echo -e "${YELLOW}Go version:${NC} $(go version)"
echo ""

# Check network connectivity
echo -e "${BLUE}[1/4] Checking network connectivity...${NC}"
if ping -c 1 8.8.8.8 &> /dev/null; then
    echo -e "${GREEN}✓ Network connectivity OK${NC}"
else
    echo -e "${YELLOW}⚠ Network connectivity may have issues; test results may be inaccurate${NC}"
fi
echo ""

# Quick test (1,000 domains)
echo -e "${BLUE}[2/4] Quick test (1,000 domains)...${NC}"
echo -e "${YELLOW}Target: < 2 sec${NC}"

go test -tags=performance -bench=Benchmark1k ./test/ -timeout 5m -v 2>&1 | \
    grep -E "Benchmark1k|total_seconds|success|domains/sec" | \
    tee /tmp/ksubdomain_1k.log

if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo -e "${GREEN}✓ 1,000-domain test completed${NC}"
else
    echo -e "${RED}✗ 1,000-domain test failed${NC}"
fi
echo ""

# Medium test (10,000 domains)
echo -e "${BLUE}[3/4] Medium test (10,000 domains)...${NC}"
echo -e "${YELLOW}Target: < 5 sec${NC}"

go test -tags=performance -bench=Benchmark10k ./test/ -timeout 5m -v 2>&1 | \
    grep -E "Benchmark10k|total_seconds|success|domains/sec" | \
    tee /tmp/ksubdomain_10k.log

if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo -e "${GREEN}✓ 10,000-domain test completed${NC}"
else
    echo -e "${RED}✗ 10,000-domain test failed${NC}"
fi
echo ""

# Full test (100,000 domains) — README standard
echo -e "${BLUE}[4/4] Full test (100,000 domains) — README standard${NC}"
echo -e "${YELLOW}Target: < 30 sec (per README)${NC}"
echo -e "${YELLOW}Note: This will take a few minutes, please be patient...${NC}"
echo ""

go test -tags=performance -bench=Benchmark100k ./test/ -timeout 10m -v 2>&1 | \
    tee /tmp/ksubdomain_100k.log

if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✓ 100,000-domain test completed${NC}"
else
    echo ""
    echo -e "${RED}✗ 100,000-domain test failed${NC}"
fi
echo ""

# Extract performance data
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Performance Test Summary${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 100k test results
if [ -f /tmp/ksubdomain_100k.log ]; then
    echo -e "${BLUE}100,000-domain test (README standard):${NC}"
    
    TOTAL_SEC=$(grep "total_seconds" /tmp/ksubdomain_100k.log | tail -1 | awk '{print $2}')
    SUCCESS=$(grep "success_count" /tmp/ksubdomain_100k.log | tail -1 | awk '{print $2}')
    RATE_PCT=$(grep "success_rate" /tmp/ksubdomain_100k.log | tail -1 | awk '{print $2}')
    SPEED=$(grep "domains/sec" /tmp/ksubdomain_100k.log | tail -1 | awk '{print $2}')
    
    echo -e "  - Total time:    ${YELLOW}${TOTAL_SEC} sec${NC}"
    echo -e "  - Success count: ${YELLOW}${SUCCESS}${NC}"
    echo -e "  - Success rate:  ${YELLOW}${RATE_PCT}${NC}"
    echo -e "  - Throughput:    ${YELLOW}${SPEED} domains/s${NC}"
    echo ""
    
    # Performance rating
    if (( $(echo "$TOTAL_SEC <= 30" | bc -l) )); then
        echo -e "${GREEN}✅ Rating: Excellent (meets README standard: ≤30 sec)${NC}"
    elif (( $(echo "$TOTAL_SEC <= 40" | bc -l) )); then
        echo -e "${YELLOW}✓  Rating: Good (30–40 sec)${NC}"
    elif (( $(echo "$TOTAL_SEC <= 60" | bc -l) )); then
        echo -e "${YELLOW}⚠  Rating: Fair (40–60 sec, optimization may be needed)${NC}"
    else
        echo -e "${RED}❌ Rating: Slow (>60 sec, please check network and configuration)${NC}"
    fi
    echo ""
    
    # Comparison with README
    echo -e "${BLUE}Comparison with README:${NC}"
    echo -e "  - README standard: ~30 sec, 1397 successes"
    echo -e "  - This run:        ${TOTAL_SEC} sec, ${SUCCESS} successes"
    echo ""
    
    # Comparison with other tools (README data)
    echo -e "${BLUE}Comparison with other tools (README data):${NC}"
    echo -e "  - massdns: ~3 min 29 sec (209 sec) → ksubdomain is ${YELLOW}$(echo "scale=1; 209/$TOTAL_SEC" | bc)×${NC} faster"
    echo -e "  - dnsx:    ~5 min 26 sec (326 sec) → ksubdomain is ${YELLOW}$(echo "scale=1; 326/$TOTAL_SEC" | bc)×${NC} faster"
fi

echo ""
echo -e "${BLUE}Detailed logs:${NC}"
echo -e "  - /tmp/ksubdomain_1k.log"
echo -e "  - /tmp/ksubdomain_10k.log"
echo -e "  - /tmp/ksubdomain_100k.log"
echo ""

echo -e "${GREEN}Performance tests complete! 🎉${NC}"
echo ""
echo -e "${YELLOW}Tips:${NC}"
echo -e "  - If performance is below target, check network bandwidth and DNS servers."
echo -e "  - See test/PERFORMANCE_TEST.md for performance tuning guidance."
