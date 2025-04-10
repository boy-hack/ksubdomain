#!/bin/bash

# ksubdomain 压力测试脚本
# 用于测试ksubdomain在高负载下的性能表现

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # 无颜色

# 测试配置
DOMAIN="example.com"
DICT_LARGE="test/stress_dict.txt"
RESOLVERS="test/resolvers.txt"
OUTPUT_DIR="test/results"
BIN="./ksubdomain"

# 确保测试目录存在
mkdir -p "$OUTPUT_DIR"

# 创建DNS解析器文件（如果不存在）
if [ ! -f "$RESOLVERS" ]; then
  echo "创建DNS解析器文件..."
  echo "8.8.8.8" > "$RESOLVERS"
  echo "8.8.4.4" >> "$RESOLVERS"
  echo "1.1.1.1" >> "$RESOLVERS"
  echo "114.114.114.114" >> "$RESOLVERS"
fi

# 创建大型字典（如果不存在）
if [ ! -f "$DICT_LARGE" ]; then
  echo "创建大型字典..."
  for i in $(seq 1 500000); do
    echo "stress$i.$DOMAIN" >> "$DICT_LARGE"
  done
  echo "创建了 500,000 条记录的字典"
fi

# 获取系统信息
echo "========================================"
echo "   系统信息"
echo "========================================"
echo "操作系统: $(uname -s)"
echo "处理器: $(uname -p)"
echo "内核版本: $(uname -r)"

if [ "$(uname -s)" = "Linux" ]; then
  echo "CPU核心数: $(nproc)"
  echo "内存: $(free -h | grep Mem | awk '{print $2}')"
elif [ "$(uname -s)" = "Darwin" ]; then
  echo "CPU核心数: $(sysctl -n hw.ncpu)"
  echo "内存: $(sysctl -n hw.memsize | awk '{print $1/1024/1024/1024 " GB"}')"
fi

echo ""
echo "========================================"
echo "   KSubdomain 压力测试"
echo "========================================"

# 用于每次压力测试的函数
run_stress_test() {
  local rate=$1
  local output="${OUTPUT_DIR}/stress_${rate}.txt"
  local log="${OUTPUT_DIR}/stress_${rate}.log"
  
  echo -e "${YELLOW}测试速率: $rate pps${NC}"
  
  # 清理旧文件
  [ -f "$output" ] && rm "$output"
  [ -f "$log" ] && rm "$log"
  
  # 使用时间命令运行测试，获取总用时
  echo "开始测试..."
  
  # 执行测试并计时
  start_time=$(date +%s.%N)
  
  # 将标准输出和错误输出重定向到日志文件
  $BIN v -f "$DICT_LARGE" -r "$RESOLVERS" -o "$output" -b "$rate" --retry 2 --timeout 4 --np > "$log" 2>&1
  
  end_time=$(date +%s.%N)
  elapsed=$(echo "$end_time - $start_time" | bc)
  
  # 统计结果
  processed_count=$(cat "$log" | grep -o "success:[0-9]*" | tail -1 | grep -o "[0-9]*")
  found_count=$(wc -l < "$output")
  
  # 计算每秒处理的域名数
  domains_per_sec=$(echo "$processed_count / $elapsed" | bc)
  
  echo -e "${GREEN}测试完成${NC}"
  echo "处理域名数: $processed_count"
  echo "找到子域名: $found_count"
  echo "耗时: $elapsed 秒"
  echo "处理速率: $domains_per_sec 域名/秒"
  echo ""
  
  # 输出结果字符串，供后续收集
  echo "$rate,$elapsed,$processed_count,$found_count,$domains_per_sec"
}

# 不同速率的测试
echo -e "${YELLOW}运行不同速率的压力测试...${NC}"
echo ""

# 创建结果CSV文件
RESULT_CSV="${OUTPUT_DIR}/stress_results.csv"
echo "速率(pps),耗时(秒),处理域名数,发现子域名数,实际速率(域名/秒)" > "$RESULT_CSV"

# 逐步提高速率进行测试
for rate in "10k" "50k" "100k" "200k" "500k" "1m"; do
  result=$(run_stress_test "$rate")
  echo "$result" >> "$RESULT_CSV"
  
  # 短暂休息让系统冷却
  sleep 5
done

echo "========================================"
echo "   内存使用测试"
echo "========================================"

# 记录运行时内存使用情况
echo -e "${YELLOW}测试运行时内存使用情况...${NC}"
MEMORY_LOG="${OUTPUT_DIR}/memory_usage.log"
MEM_OUTPUT="${OUTPUT_DIR}/memory_test.txt"

# 清理旧文件
[ -f "$MEMORY_LOG" ] && rm "$MEMORY_LOG"
[ -f "$MEM_OUTPUT" ] && rm "$MEM_OUTPUT"

echo "开始测试..."

# 后台运行ksubdomain
$BIN v -f "$DICT_LARGE" -r "$RESOLVERS" -o "$MEM_OUTPUT" -b "100k" --retry 2 --timeout 4 --np > /dev/null 2>&1 &
PID=$!

# 监控10秒内的内存使用情况
echo "PID: $PID"
echo "监控内存使用情况..."

for i in {1..10}; do
  if [ "$(uname -s)" = "Linux" ]; then
    # Linux下获取RSS内存使用量
    MEM=$(ps -p $PID -o rss= 2>/dev/null)
    if [ ! -z "$MEM" ]; then
      MEM_MB=$(echo "scale=2; $MEM / 1024" | bc)
      echo "内存使用 #$i: ${MEM_MB}MB" | tee -a "$MEMORY_LOG"
    else
      echo "进程已结束"
      break
    fi
  elif [ "$(uname -s)" = "Darwin" ]; then
    # MacOS下获取内存使用量
    MEM=$(ps -p $PID -o rss= 2>/dev/null)
    if [ ! -z "$MEM" ]; then
      MEM_MB=$(echo "scale=2; $MEM / 1024" | bc)
      echo "内存使用 #$i: ${MEM_MB}MB" | tee -a "$MEMORY_LOG"
    else
      echo "进程已结束"
      break
    fi
  fi
  sleep 1
done

# 测试10秒后结束进程
if kill -0 $PID 2>/dev/null; then
  echo "终止进程..."
  kill $PID
fi

# 获取最大内存使用量
if [ -f "$MEMORY_LOG" ]; then
  MAX_MEM=$(cat "$MEMORY_LOG" | grep -o "[0-9]\+\.[0-9]\+MB" | sort -nr | head -1)
  echo -e "${GREEN}最大内存使用量: $MAX_MEM${NC}"
fi

echo ""
echo "压力测试完成!"
echo "结果保存在 $RESULT_CSV" 