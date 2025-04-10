#!/bin/bash

# ksubdomain 准确性测试脚本
# 用于测试优化后的版本结果与原始版本的一致性

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # 无颜色

# 测试配置
TEST_DOMAIN="example.com"
TEST_DICT="test/accuracy_dict.txt"
RESOLVERS="test/resolvers.txt"
ORIG_BIN="test/ksubdomain_orig"
NEW_BIN="./ksubdomain"
ORIG_OUTPUT="test/results/orig_accuracy.txt"
NEW_OUTPUT="test/results/new_accuracy.txt"
DIFF_OUTPUT="test/results/diff.txt"

# 确保测试目录存在
mkdir -p "test/results"

# 检查原始版本是否存在
if [ ! -f "$ORIG_BIN" ]; then
  echo -e "${RED}错误: 找不到原始版本二进制文件 $ORIG_BIN${NC}"
  echo "请将原始的ksubdomain放到test目录，重命名为ksubdomain_orig"
  exit 1
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
if [ ! -f "$TEST_DICT" ]; then
  echo "创建测试字典..."
  # 常见子域名，可能存在的
  echo "www.$TEST_DOMAIN" > "$TEST_DICT"
  echo "mail.$TEST_DOMAIN" >> "$TEST_DICT"
  echo "api.$TEST_DOMAIN" >> "$TEST_DICT"
  echo "blog.$TEST_DOMAIN" >> "$TEST_DICT"
  echo "docs.$TEST_DOMAIN" >> "$TEST_DICT"
  # 随机生成的子域名
  for i in {1..95}; do
    echo "test$i.$TEST_DOMAIN" >> "$TEST_DICT"
  done
fi

echo "========================================"
echo "   KSubdomain 准确性测试"
echo "========================================"

# 运行原始版本
echo -e "${YELLOW}运行原始版本...${NC}"
$ORIG_BIN v -f "$TEST_DICT" -r "$RESOLVERS" -o "$ORIG_OUTPUT" -b 5m --retry 3 --timeout 6

# 运行优化版本
echo -e "${YELLOW}运行优化版本...${NC}"
$NEW_BIN v -f "$TEST_DICT" -r "$RESOLVERS" -o "$NEW_OUTPUT" -b 5m --retry 3 --timeout 6

# 比较结果
echo -e "${YELLOW}比较结果...${NC}"
# 对结果文件进行排序
sort "$ORIG_OUTPUT" > "$ORIG_OUTPUT.sorted"
sort "$NEW_OUTPUT" > "$NEW_OUTPUT.sorted"

# 使用diff比较排序后的结果
diff "$ORIG_OUTPUT.sorted" "$NEW_OUTPUT.sorted" > "$DIFF_OUTPUT"

if [ -s "$DIFF_OUTPUT" ]; then
  DIFF_COUNT=$(wc -l < "$DIFF_OUTPUT")
  echo -e "${RED}发现差异! $DIFF_COUNT 行不同${NC}"
  echo "差异详情保存在 $DIFF_OUTPUT"
  echo "以下是差异内容:"
  cat "$DIFF_OUTPUT"
else
  echo -e "${GREEN}测试通过! 两个版本的结果完全一致${NC}"
  # 获取找到的子域名数量
  FOUND_COUNT=$(wc -l < "$NEW_OUTPUT")
  echo "找到了 $FOUND_COUNT 个子域名"
  rm "$DIFF_OUTPUT" # 删除空的差异文件
fi

# 清理临时文件
rm "$ORIG_OUTPUT.sorted" "$NEW_OUTPUT.sorted"

echo "测试完成!" 