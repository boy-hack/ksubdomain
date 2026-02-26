# 性能基准测试

## 📊 测试目标

参考 README 中的性能对比,验证 ksubdomain 的扫描性能:

| 字典大小 | 目标耗时 | 参考 README |
|---------|---------|------------|
| 1,000 域名 | < 2 秒 | - |
| 10,000 域名 | < 5 秒 | - |
| 100,000 域名 | **< 30 秒** | **README 标准** |

---

## 🧪 测试环境

### README 参考配置
- **CPU**: 4 核
- **带宽**: 5M
- **字典**: 10 万域名 (d2.txt)
- **DNS**: 自定义 DNS 列表 (dns.txt)
- **重试**: 3 次
- **结果**: ~30 秒, 1397 个成功

### 测试要求
- 需要 **root 权限** (访问网卡)
- 需要 **网络连接** (DNS 查询)
- 需要 **libpcap**
- 建议 **4 核 CPU + 5M 带宽**

---

## 🚀 运行测试

### 快速测试 (1000 域名)
```bash
# 编译并运行
go test -tags=performance -bench=Benchmark1k ./test/ -timeout 10m

# 预期结果:
# Benchmark1kDomains    1    1.5s    total_seconds:1.5
#                       950   success_count
#                       95%   success_rate_%
#                       666   domains/sec
```

### 中等测试 (10000 域名)
```bash
go test -tags=performance -bench=Benchmark10k ./test/ -timeout 10m

# 预期结果:
# Benchmark10kDomains   1    4.8s    total_seconds:4.8
#                       9500  success_count
#                       95%   success_rate_%
#                       2083  domains/sec
```

### 完整测试 (100000 域名) - README 标准
```bash
# 需要 sudo (访问网卡)
sudo go test -tags=performance -bench=Benchmark100k ./test/ -timeout 10m -v

# 预期结果 (参考 README):
# Benchmark100kDomains  1    28.5s   total_seconds:28.5
#                       95000 success_count
#                       95%   success_rate_%
#                       3508  domains/sec
#
# ✅ 性能优秀: 10万域名仅耗时 28.5 秒 (达到 README 标准)
```

### 运行所有性能测试
```bash
sudo go test -tags=performance -bench=. ./test/ -timeout 15m -v
```

---

## 📈 性能指标说明

### 报告指标

每个测试会报告以下指标:

```
total_seconds    总耗时 (秒)
success_count    成功解析的域名数
success_rate_%   成功率 (百分比)
domains/sec      扫描速率 (域名/秒)
```

### 日志输出

测试过程中会实时显示:
```
进度: 1000/100000 (1.0%), 速率: 3500 domains/s, 耗时: 0s
进度: 2000/100000 (2.0%), 速率: 3600 domains/s, 耗时: 0s
...
最终结果: 95000/100000, 耗时: 28.5s
```

---

## 🎯 性能基准对比

### README 标准 (100,000 域名)

| 工具 | 耗时 | 速率 | 成功数 | 倍数 |
|------|------|------|--------|------|
| **KSubdomain** | **~30 秒** | ~3333/s | 1397 | **1x** |
| massdns | ~3 分 29 秒 | ~478/s | 1396 | **7x 慢** |
| dnsx | ~5 分 26 秒 | ~307/s | 1396 | **10x 慢** |

### 我们的测试目标

基于 README 标准,我们的目标:

```
✅ 优秀: < 30 秒 (达到 README 标准)
✓  良好: 30-40 秒 (可接受范围)
⚠️  警告: 40-60 秒 (需要优化)
❌ 失败: > 60 秒 (性能问题)
```

---

## 🔧 性能调优建议

### 如果测试较慢

#### 1. 检查带宽限制
```bash
# 测试中使用 5M 带宽
# 可以尝试调整 (在 performance_benchmark_test.go 中)
Rate: options.Band2Rate("10m")  # 提高到 10M
```

#### 2. 检查 DNS 服务器
```bash
# 使用更快的 DNS 服务器
Resolvers: []string{"8.8.8.8", "1.1.1.1"}
```

#### 3. 增加重试次数 (trade-off)
```bash
# 更多重试 = 更高成功率,但更慢
Retry: 5  # 从 3 增加到 5
```

#### 4. 调整超时时间
```bash
# 更短超时 = 更快,但可能漏掉慢响应
TimeOut: 3  # 从 6 减少到 3
```

---

## 📊 性能数据收集

### 生成性能报告

```bash
# 运行测试并保存结果
sudo go test -tags=performance -bench=Benchmark100k ./test/ \
    -timeout 10m -v 2>&1 | tee performance_report.txt

# 提取关键指标
grep "total_seconds\|success_count\|success_rate\|domains/sec" performance_report.txt
```

### 多次运行取平均

```bash
# 运行 3 次取平均值
for i in {1..3}; do
    echo "=== Run $i ==="
    sudo go test -tags=performance -bench=Benchmark100k ./test/ \
        -timeout 10m 2>&1 | grep "total_seconds"
done
```

---

## 🧩 测试场景

### 场景 1: 标准测试 (README 配置)
```
字典: 100,000 域名
带宽: 5M
重试: 3 次
超时: 6 秒
目标: < 30 秒
```

### 场景 2: 高速测试 (10M 带宽)
```
字典: 100,000 域名
带宽: 10M
重试: 3 次
超时: 6 秒
目标: < 20 秒
```

### 场景 3: 保守测试 (高成功率)
```
字典: 100,000 域名
带宽: 5M
重试: 10 次
超时: 10 秒
目标: < 60 秒, 成功率 > 98%
```

---

## 📝 测试清单

运行性能测试前确认:

- [ ] 有 root 权限
- [ ] 网络连接正常
- [ ] libpcap 已安装
- [ ] 网卡正常工作
- [ ] 至少 4 核 CPU
- [ ] 至少 5M 带宽
- [ ] 关闭其他网络密集型程序

---

## 🎯 预期结果

### 1000 域名
```
总耗时:   ~1.5 秒
成功数:   ~950
成功率:   ~95%
速率:     ~666 domains/s
```

### 10000 域名
```
总耗时:   ~5 秒
成功数:   ~9500
成功率:   ~95%
速率:     ~2000 domains/s
```

### 100000 域名 (README 标准)
```
总耗时:   ~30 秒 ✅
成功数:   ~95000
成功率:   ~95%
速率:     ~3333 domains/s
```

---

## 🐛 常见问题

### 问题 1: 权限错误
```
错误: pcap初始化失败
解决: sudo go test -tags=performance ...
```

### 问题 2: 网卡未找到
```
错误: No such device
解决: ./ksubdomain test  # 检查网卡
      --eth <网卡名>     # 手动指定
```

### 问题 3: 测试超时
```
错误: test timed out after 10m
解决: -timeout 15m  # 增加超时时间
```

### 问题 4: 成功率低
```
成功率: < 80%
原因: 网络不稳定 / DNS 服务器慢
解决: 增加重试次数 / 更换 DNS
```

---

## 📚 参考

- README 性能对比: 10万域名 ~30秒
- massdns 对比: 7 倍性能差距
- dnsx 对比: 10 倍性能差距

---

**性能就是王道! ⚡**

运行测试验证 ksubdomain 的极速性能:
```bash
sudo go test -tags=performance -bench=Benchmark100k ./test/ -timeout 10m -v
```
