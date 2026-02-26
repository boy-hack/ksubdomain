#!/bin/bash
#
# KSubdomain 测试运行脚本
# 用途: 运行所有测试并生成报告
#

set -e

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  KSubdomain 测试套件${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 检查 Go 环境
if ! command -v go &> /dev/null; then
    echo -e "${RED}错误: 未找到 Go 环境${NC}"
    exit 1
fi

echo -e "${YELLOW}Go 版本:${NC} $(go version)"
echo ""

# 创建测试结果目录
mkdir -p test_results

# 1. 单元测试
echo -e "${GREEN}[1/5] 运行单元测试...${NC}"
go test -v -cover -coverprofile=test_results/coverage.out ./pkg/... 2>&1 | tee test_results/unit_test.log

if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo -e "${GREEN}✓ 单元测试通过${NC}"
else
    echo -e "${RED}✗ 单元测试失败${NC}"
    exit 1
fi
echo ""

# 2. 覆盖率报告
echo -e "${GREEN}[2/5] 生成覆盖率报告...${NC}"
go tool cover -html=test_results/coverage.out -o test_results/coverage.html
COVERAGE=$(go tool cover -func=test_results/coverage.out | grep total | awk '{print $3}')
echo -e "${YELLOW}总覆盖率:${NC} $COVERAGE"

# 检查覆盖率是否达标
COVERAGE_NUM=$(echo $COVERAGE | sed 's/%//')
if (( $(echo "$COVERAGE_NUM >= 60" | bc -l) )); then
    echo -e "${GREEN}✓ 覆盖率达标 (> 60%)${NC}"
else
    echo -e "${YELLOW}⚠ 覆盖率偏低 (< 60%)${NC}"
fi
echo ""

# 3. 性能测试
echo -e "${GREEN}[3/5] 运行性能测试...${NC}"
go test -bench=. -benchmem ./pkg/... 2>&1 | tee test_results/benchmark.log

if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo -e "${GREEN}✓ 性能测试完成${NC}"
else
    echo -e "${YELLOW}⚠ 性能测试部分失败${NC}"
fi
echo ""

# 4. 竞争检测
echo -e "${GREEN}[4/5] 运行数据竞争检测...${NC}"
go test -race ./pkg/runner/statusdb/ 2>&1 | tee test_results/race.log

if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo -e "${GREEN}✓ 无数据竞争${NC}"
else
    echo -e "${RED}✗ 检测到数据竞争${NC}"
    exit 1
fi
echo ""

# 5. 代码静态检查 (可选)
echo -e "${GREEN}[5/5] 代码静态检查...${NC}"
if command -v golangci-lint &> /dev/null; then
    golangci-lint run ./... 2>&1 | tee test_results/lint.log
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        echo -e "${GREEN}✓ 代码检查通过${NC}"
    else
        echo -e "${YELLOW}⚠ 代码检查发现问题${NC}"
    fi
else
    echo -e "${YELLOW}⚠ 未安装 golangci-lint,跳过静态检查${NC}"
fi
echo ""

# 生成测试报告
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  测试总结${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "${YELLOW}单元测试:${NC} 通过 ✓"
echo -e "${YELLOW}覆盖率:${NC}   $COVERAGE"
echo -e "${YELLOW}数据竞争:${NC} 无 ✓"
echo -e "${YELLOW}报告位置:${NC} test_results/"
echo ""
echo -e "${GREEN}测试结果:${NC}"
echo -e "  - coverage.html: 覆盖率可视化报告"
echo -e "  - unit_test.log: 单元测试详细日志"
echo -e "  - benchmark.log: 性能测试结果"
echo -e "  - race.log:      竞争检测日志"
echo ""
echo -e "${GREEN}所有测试完成! 🎉${NC}"
