#!/bin/bash
#
# KSubdomain 性能基准测试脚本
# 用途: 运行 10 万域名性能测试,验证 README 中的性能指标
#
# 参考 README:
# - 字典: 10 万域名
# - 带宽: 5M
# - 耗时: ~30 秒
# - 成功: 1397 个
#

set -e

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  KSubdomain 性能基准测试${NC}"
echo -e "${GREEN}  参考 README: 10万域名 ~30秒${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 检查 root 权限
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}错误: 需要 root 权限运行性能测试${NC}"
    echo -e "${YELLOW}请使用: sudo $0${NC}"
    exit 1
fi

# 检查 Go 环境
if ! command -v go &> /dev/null; then
    echo -e "${RED}错误: 未找到 Go 环境${NC}"
    exit 1
fi

echo -e "${YELLOW}Go 版本:${NC} $(go version)"
echo ""

# 检查网络
echo -e "${BLUE}[1/4] 检查网络连接...${NC}"
if ping -c 1 8.8.8.8 &> /dev/null; then
    echo -e "${GREEN}✓ 网络连接正常${NC}"
else
    echo -e "${YELLOW}⚠ 网络连接可能有问题,测试结果可能不准确${NC}"
fi
echo ""

# 快速测试 (1000 域名)
echo -e "${BLUE}[2/4] 快速测试 (1000 域名)...${NC}"
echo -e "${YELLOW}目标: < 2 秒${NC}"

go test -tags=performance -bench=Benchmark1k ./test/ -timeout 5m -v 2>&1 | \
    grep -E "Benchmark1k|total_seconds|success|domains/sec" | \
    tee /tmp/ksubdomain_1k.log

if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo -e "${GREEN}✓ 1000 域名测试完成${NC}"
else
    echo -e "${RED}✗ 1000 域名测试失败${NC}"
fi
echo ""

# 中等测试 (10000 域名)
echo -e "${BLUE}[3/4] 中等测试 (10000 域名)...${NC}"
echo -e "${YELLOW}目标: < 5 秒${NC}"

go test -tags=performance -bench=Benchmark10k ./test/ -timeout 5m -v 2>&1 | \
    grep -E "Benchmark10k|total_seconds|success|domains/sec" | \
    tee /tmp/ksubdomain_10k.log

if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo -e "${GREEN}✓ 10000 域名测试完成${NC}"
else
    echo -e "${RED}✗ 10000 域名测试失败${NC}"
fi
echo ""

# 完整测试 (100000 域名) - README 标准
echo -e "${BLUE}[4/4] 完整测试 (100000 域名) - README 标准${NC}"
echo -e "${YELLOW}目标: < 30 秒 (参考 README)${NC}"
echo -e "${YELLOW}提示: 这将需要几分钟,请耐心等待...${NC}"
echo ""

go test -tags=performance -bench=Benchmark100k ./test/ -timeout 10m -v 2>&1 | \
    tee /tmp/ksubdomain_100k.log

if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✓ 100000 域名测试完成${NC}"
else
    echo ""
    echo -e "${RED}✗ 100000 域名测试失败${NC}"
fi
echo ""

# 提取性能数据
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  性能测试总结${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 100k 测试结果
if [ -f /tmp/ksubdomain_100k.log ]; then
    echo -e "${BLUE}100,000 域名测试 (README 标准):${NC}"
    
    TOTAL_SEC=$(grep "total_seconds" /tmp/ksubdomain_100k.log | tail -1 | awk '{print $2}')
    SUCCESS=$(grep "success_count" /tmp/ksubdomain_100k.log | tail -1 | awk '{print $2}')
    RATE_PCT=$(grep "success_rate" /tmp/ksubdomain_100k.log | tail -1 | awk '{print $2}')
    SPEED=$(grep "domains/sec" /tmp/ksubdomain_100k.log | tail -1 | awk '{print $2}')
    
    echo -e "  - 总耗时:   ${YELLOW}${TOTAL_SEC} 秒${NC}"
    echo -e "  - 成功数:   ${YELLOW}${SUCCESS}${NC}"
    echo -e "  - 成功率:   ${YELLOW}${RATE_PCT}${NC}"
    echo -e "  - 扫描速率: ${YELLOW}${SPEED} domains/s${NC}"
    echo ""
    
    # 性能评估
    if (( $(echo "$TOTAL_SEC <= 30" | bc -l) )); then
        echo -e "${GREEN}✅ 性能评估: 优秀 (达到 README 标准: ≤30秒)${NC}"
    elif (( $(echo "$TOTAL_SEC <= 40" | bc -l) )); then
        echo -e "${YELLOW}✓  性能评估: 良好 (30-40秒)${NC}"
    elif (( $(echo "$TOTAL_SEC <= 60" | bc -l) )); then
        echo -e "${YELLOW}⚠  性能评估: 一般 (40-60秒,可能需要优化)${NC}"
    else
        echo -e "${RED}❌ 性能评估: 较慢 (>60秒,需要检查网络和配置)${NC}"
    fi
    echo ""
    
    # README 对比
    echo -e "${BLUE}与 README 对比:${NC}"
    echo -e "  - README 标准: ~30 秒, 1397 个成功"
    echo -e "  - 本次测试:    ${TOTAL_SEC} 秒, ${SUCCESS} 个成功"
    echo ""
    
    # 与其他工具对比
    echo -e "${BLUE}与其他工具对比 (README 数据):${NC}"
    echo -e "  - massdns: ~3分29秒 (209秒) → ksubdomain 快 ${YELLOW}$(echo "scale=1; 209/$TOTAL_SEC" | bc)x${NC}"
    echo -e "  - dnsx:    ~5分26秒 (326秒) → ksubdomain 快 ${YELLOW}$(echo "scale=1; 326/$TOTAL_SEC" | bc)x${NC}"
fi

echo ""
echo -e "${BLUE}详细日志:${NC}"
echo -e "  - /tmp/ksubdomain_1k.log"
echo -e "  - /tmp/ksubdomain_10k.log"
echo -e "  - /tmp/ksubdomain_100k.log"
echo ""

echo -e "${GREEN}性能测试完成! 🎉${NC}"
echo ""
echo -e "${YELLOW}提示:${NC}"
echo -e "  - 如果性能不达标,请检查网络带宽和 DNS 服务器"
echo -e "  - 参考: test/PERFORMANCE_TEST.md 了解性能调优"
