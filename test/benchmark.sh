#!/bin/bash

# ksubdomain 性能测试脚本
# 用于测试优化后的ksubdomain性能

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # 无颜色

# 测试配置
TEST_DOMAIN="baidu.com"
SMALL_DICT="test/dict_small.txt"  # 1000个子域名
MEDIUM_DICT="test/dict_medium.txt" # 10000个子域名
LARGE_DICT="test/dict_large.txt"  # 100000个子域名
RESOLVERS="test/resolvers.txt"
OUTPUT_DIR="test/results"
ORIG_BIN="test/ksubdomain_orig"
NEW_BIN="./ksubdomain"

# 确保测试目录存在
mkdir -p "$OUTPUT_DIR"
mkdir -p "test"

# 检查原始版本是否存在
if [ ! -f "$ORIG_BIN" ]; then
  echo -e "${YELLOW}找不到原始版本二进制文件，跳过比较测试${NC}"
  SKIP_COMPARE=true
else
  SKIP_COMPARE=false
fi

# 创建DNS解析器文件（如果不存在）
if [ ! -f "$RESOLVERS" ]; then
  echo "创建DNS解析器文件..."
  echo "8.8.8.8" > "$RESOLVERS"
  echo "8.8.4.4" >> "$RESOLVERS"
  echo "1.1.1.1" >> "$RESOLVERS"
  echo "114.114.114.114" >> "$RESOLVERS"
fi

# 创建测试字典（如果不存在）
create_test_dict() {
  local file=$1
  local size=$2
  
  if [ ! -f "$file" ]; then
    echo "创建测试字典 $file ($size 条记录)..."
    for i in $(seq 1 $size); do
      echo "sub$i.$TEST_DOMAIN" >> "$file"
    done
  fi
}

create_test_dict "$SMALL_DICT" 1000
create_test_dict "$MEDIUM_DICT" 10000
create_test_dict "$LARGE_DICT" 100000

# 运行单次测试
run_test() {
  local bin=$1
  local mode=$2
  local dict=$3
  local output=$4
  local extra_params=$5
  local dict_size=$(wc -l < "$dict")
  
  echo "测试 $bin $mode 模式，字典大小: $dict_size $extra_params"
  
  # 清理输出文件
  if [ -f "$output" ]; then
    rm "$output"
  fi
  
  # 执行测试并计时
  local start_time=$(date +%s.%N)
  
  if [ "$mode" == "verify" ]; then
    $bin v -f "$dict" -r "$RESOLVERS" -o "$output" $extra_params --np
  else
    $bin e -d "$TEST_DOMAIN" -f "$dict" -r "$RESOLVERS" -o "$output" $extra_params --np
  fi
  
  local end_time=$(date +%s.%N)
  local elapsed=$(echo "$end_time - $start_time" | bc)
  local found=$(wc -l < "$output")
  
  echo -e "${GREEN}完成！用时: $elapsed 秒，发现: $found 个子域名${NC}"
  echo ""
  
  # 返回结果
  echo "$elapsed,$found"
}

# 执行所有测试
run_all_tests() {
  local bin=$1
  local prefix=$2
  
  # 小字典，验证模式
  small_verify=$(run_test "$bin" "verify" "$SMALL_DICT" "${OUTPUT_DIR}/${prefix}_small_verify.txt" "-b 5m")
  
  # 小字典，枚举模式
  small_enum=$(run_test "$bin" "enum" "$SMALL_DICT" "${OUTPUT_DIR}/${prefix}_small_enum.txt" "-b 5m")
  
  # 中等字典，验证模式
  medium_verify=$(run_test "$bin" "verify" "$MEDIUM_DICT" "${OUTPUT_DIR}/${prefix}_medium_verify.txt" "-b 5m")
  
  # 大字典，验证模式
  large_verify=$(run_test "$bin" "verify" "$LARGE_DICT" "${OUTPUT_DIR}/${prefix}_large_verify.txt" "-b 5m")
  
  # 测试不同超时和重试参数
  retry_test=$(run_test "$bin" "verify" "$MEDIUM_DICT" "${OUTPUT_DIR}/${prefix}_retry_test.txt" "-b 5m --retry 5 --timeout 8")
  
  # 返回所有结果
  echo "$small_verify|$small_enum|$medium_verify|$large_verify|$retry_test"
}

# 主函数
main() {
  echo "========================================"
  echo "   KSubdomain 性能测试"
  echo "========================================"
  
  # 测试新版本
  echo -e "${YELLOW}测试优化后的版本...${NC}"
  new_results=$(run_all_tests "$NEW_BIN" "new")
  
  # 如果原始版本存在，则测试比较
  if [ "$SKIP_COMPARE" = false ]; then
    echo -e "${YELLOW}测试原始版本...${NC}"
    orig_results=$(run_all_tests "$ORIG_BIN" "orig")
    
    # 解析结果
    IFS='|' read -r new_small_verify new_small_enum new_medium_verify new_large_verify new_retry_test <<< "$new_results"
    IFS='|' read -r orig_small_verify orig_small_enum orig_medium_verify orig_large_verify orig_retry_test <<< "$orig_results"
    
    # 提取时间和发现数量
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
    
    # 计算性能提升百分比
    small_verify_speedup=$(echo "scale=2; ($orig_small_verify_time - $new_small_verify_time) / $orig_small_verify_time * 100" | bc)
    small_enum_speedup=$(echo "scale=2; ($orig_small_enum_time - $new_small_enum_time) / $orig_small_enum_time * 100" | bc)
    medium_verify_speedup=$(echo "scale=2; ($orig_medium_verify_time - $new_medium_verify_time) / $orig_medium_verify_time * 100" | bc)
    large_verify_speedup=$(echo "scale=2; ($orig_large_verify_time - $new_large_verify_time) / $orig_large_verify_time * 100" | bc)
    retry_test_speedup=$(echo "scale=2; ($orig_retry_test_time - $new_retry_test_time) / $orig_retry_test_time * 100" | bc)
    
    # 输出比较结果
    echo ""
    echo "========================================"
    echo "   性能比较结果"
    echo "========================================"
    echo "小字典验证模式："
    echo "  原始版本: $orig_small_verify_time 秒, 发现: $orig_small_verify_found 个域名"
    echo "  优化版本: $new_small_verify_time 秒, 发现: $new_small_verify_found 个域名"
    echo -e "  速度提升: ${GREEN}$small_verify_speedup%${NC}"
    echo ""
    
    echo "小字典枚举模式："
    echo "  原始版本: $orig_small_enum_time 秒, 发现: $orig_small_enum_found 个域名"
    echo "  优化版本: $new_small_enum_time 秒, 发现: $new_small_enum_found 个域名"
    echo -e "  速度提升: ${GREEN}$small_enum_speedup%${NC}"
    echo ""
    
    echo "中等字典验证模式："
    echo "  原始版本: $orig_medium_verify_time 秒, 发现: $orig_medium_verify_found 个域名"
    echo "  优化版本: $new_medium_verify_time 秒, 发现: $new_medium_verify_found 个域名"
    echo -e "  速度提升: ${GREEN}$medium_verify_speedup%${NC}"
    echo ""
    
    echo "大字典验证模式："
    echo "  原始版本: $orig_large_verify_time 秒, 发现: $orig_large_verify_found 个域名"
    echo "  优化版本: $new_large_verify_time 秒, 发现: $new_large_verify_found 个域名"
    echo -e "  速度提升: ${GREEN}$large_verify_speedup%${NC}"
    echo ""
    
    echo "重试参数测试："
    echo "  原始版本: $orig_retry_test_time 秒, 发现: $orig_retry_test_found 个域名"
    echo "  优化版本: $new_retry_test_time 秒, 发现: $new_retry_test_found 个域名"
    echo -e "  速度提升: ${GREEN}$retry_test_speedup%${NC}"
    echo ""
    
    # 计算平均性能提升
    avg_speedup=$(echo "scale=2; ($small_verify_speedup + $small_enum_speedup + $medium_verify_speedup + $large_verify_speedup + $retry_test_speedup) / 5" | bc)
    echo -e "平均性能提升: ${GREEN}$avg_speedup%${NC}"
  else
    echo ""
    echo "========================================"
    echo "   测试结果 (仅优化版本)"
    echo "========================================"
    
    # 解析结果
    IFS='|' read -r new_small_verify new_small_enum new_medium_verify new_large_verify new_retry_test <<< "$new_results"
    
    # 提取时间和发现数量
    IFS=',' read -r new_small_verify_time new_small_verify_found <<< "$new_small_verify"
    IFS=',' read -r new_small_enum_time new_small_enum_found <<< "$new_small_enum"
    IFS=',' read -r new_medium_verify_time new_medium_verify_found <<< "$new_medium_verify"
    IFS=',' read -r new_large_verify_time new_large_verify_found <<< "$new_large_verify"
    IFS=',' read -r new_retry_test_time new_retry_test_found <<< "$new_retry_test"
    
    echo "小字典验证模式: $new_small_verify_time 秒, 发现: $new_small_verify_found 个域名"
    echo "小字典枚举模式: $new_small_enum_time 秒, 发现: $new_small_enum_found 个域名"
    echo "中等字典验证模式: $new_medium_verify_time 秒, 发现: $new_medium_verify_found 个域名"
    echo "大字典验证模式: $new_large_verify_time 秒, 发现: $new_large_verify_found 个域名"
    echo "重试参数测试: $new_retry_test_time 秒, 发现: $new_retry_test_found 个域名"
  fi
  
  echo ""
  echo "测试结果保存在 $OUTPUT_DIR 目录"
  echo "测试完成！"
}

# 执行主函数
main 