#!/bin/bash

# ksubdomain 测试运行脚本
# 用于运行所有测试

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # 无颜色

# 检查权限
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}警告: 测试脚本需要root权限才能正常运行${NC}"
  echo "请使用 sudo 运行此脚本"
  exit 1
fi

# 确保测试目录存在
mkdir -p "test/results"

# 清理旧的测试结果
echo -e "${YELLOW}清理旧的测试结果...${NC}"
rm -rf test/results/*

# 设置权限
chmod +x test/benchmark.sh
chmod +x test/accuracy_test.sh
chmod +x test/stress_test.sh

echo "========================================"
echo -e "${BLUE}KSubdomain 测试套件${NC}"
echo "========================================"
echo ""

# 询问是否有旧版本可用于比较
read -p "是否有原始版本的 ksubdomain 可用于比较测试? (y/n): " has_original
if [ "$has_original" = "y" ] || [ "$has_original" = "Y" ]; then
  read -p "请输入原始版本 ksubdomain 的路径: " orig_path
  if [ -f "$orig_path" ]; then
    echo "复制原始版本到测试目录..."
    cp "$orig_path" "test/ksubdomain_orig"
    chmod +x "test/ksubdomain_orig"
  else
    echo -e "${RED}错误: 指定的路径不存在${NC}"
    exit 1
  fi
fi

# 菜单选择要运行的测试
echo ""
echo "请选择要运行的测试:"
echo "1) 性能基准测试 (测试不同规模的字典)"
echo "2) 准确性测试 (比较结果一致性)"
echo "3) 压力测试 (测试高负载下的性能)"
echo "4) 运行所有测试"
echo "0) 退出"
echo ""

read -p "请输入选项 [0-4]: " choice

case $choice in
  1)
    echo -e "${YELLOW}运行性能基准测试...${NC}"
    test/benchmark.sh
    ;;
  2)
    if [ -f "test/ksubdomain_orig" ]; then
      echo -e "${YELLOW}运行准确性测试...${NC}"
      test/accuracy_test.sh
    else
      echo -e "${RED}错误: 准确性测试需要原始版本的 ksubdomain${NC}"
      exit 1
    fi
    ;;
  3)
    echo -e "${YELLOW}运行压力测试...${NC}"
    test/stress_test.sh
    ;;
  4)
    echo -e "${YELLOW}运行所有测试...${NC}"
    
    echo -e "${BLUE}1. 性能基准测试${NC}"
    test/benchmark.sh
    
    if [ -f "test/ksubdomain_orig" ]; then
      echo -e "${BLUE}2. 准确性测试${NC}"
      test/accuracy_test.sh
    else
      echo -e "${RED}跳过准确性测试，原始版本不存在${NC}"
    fi
    
    echo -e "${BLUE}3. 压力测试${NC}"
    test/stress_test.sh
    ;;
  0)
    echo "退出测试"
    exit 0
    ;;
  *)
    echo -e "${RED}无效选项${NC}"
    exit 1
    ;;
esac

echo ""
echo -e "${GREEN}所有测试完成!${NC}"
echo "测试结果保存在 test/results 目录中" 