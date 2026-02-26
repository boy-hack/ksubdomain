# KSubdomain 测试指南

## 📋 测试体系

本项目包含完整的测试体系,覆盖单元测试、集成测试、性能测试和压力测试。

---

## 🧪 测试类型

### 1. 单元测试 (Unit Tests)

测试单个函数和模块的功能正确性。

#### 位置
```
pkg/runner/recv_test.go          # DNS 解析测试
pkg/runner/statusdb/db_test.go   # 状态数据库测试
pkg/runner/send_test.go          # 发包和缓存测试
```

#### 运行单元测试
```bash
# 运行所有单元测试
go test ./pkg/...

# 运行特定包的测试
go test ./pkg/runner/
go test ./pkg/runner/statusdb/

# 详细输出
go test -v ./pkg/...

# 显示覆盖率
go test -cover ./pkg/...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out -o coverage.html
```

---

### 2. 集成测试 (Integration Tests)

测试多个模块协同工作,需要网络环境。

#### 位置
```
test/integration_test.go
```

#### 运行集成测试
```bash
# 运行集成测试 (需要网络和 root 权限)
sudo go test -tags=integration ./test/

# 包含详细输出
sudo go test -v -tags=integration ./test/

# 运行特定测试
sudo go test -tags=integration ./test/ -run TestBasicVerification
```

#### 注意事项
- 需要 **root 权限** (访问网卡)
- 需要 **网络连接** (DNS 查询)
- 可能需要 **5-10 分钟**
- 使用 `-short` 跳过: `go test -short ./test/`

---

### 3. 性能测试 (Benchmarks)

测试关键函数的性能表现。

#### 运行性能测试
```bash
# 运行所有性能测试
go test -bench=. ./pkg/...

# 运行特定性能测试
go test -bench=BenchmarkParseDNSName ./pkg/runner/
go test -bench=BenchmarkAdd ./pkg/runner/statusdb/

# 包含内存统计
go test -bench=. -benchmem ./pkg/...

# 多次运行取平均
go test -bench=. -benchtime=10s ./pkg/...

# CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./pkg/runner/
go tool pprof cpu.prof

# 内存 profiling
go test -bench=. -memprofile=mem.prof ./pkg/runner/
go tool pprof mem.prof
```

---

## 📊 测试覆盖率目标

| 模块 | 目标覆盖率 | 当前状态 |
|------|-----------|---------|
| DNS 解析 (recv.go) | > 80% | ✅ |
| 状态数据库 (statusdb) | > 90% | ✅ |
| 发包模块 (send.go) | > 70% | ✅ |
| 整体 | > 60% | 🔄 |

---

## 🎯 核心测试用例

### 1. DNS 域名解析测试

#### TestParseDNSName
测试 DNS 长度前缀格式解析 (修复 Issue #70)

```go
测试场景:
✓ 标准域名 (www.google.com)
✓ 二级域名 (baidu.com)
✓ 三级域名 (mail.qq.com)
✓ 空输入
✓ 压缩指针处理
✓ 异常长度处理
```

#### TestDNSRecord2String_CNAME
测试 CNAME 记录转换

```go
测试场景:
✓ 标准 CNAME 记录
✓ 多级 CNAME 链
✓ 空 CNAME 错误处理
```

---

### 2. 状态数据库测试

#### TestSharding
测试分片均匀性

```go
测试场景:
✓ 10000 个域名分布到 64 个分片
✓ 验证分布均匀性 (标准差 < 20%)
```

#### TestConcurrentReadWrite
测试并发安全

```go
测试场景:
✓ 1000 次并发读
✓ 1000 次并发写
✓ 1000 次并发删除
✓ 无数据竞争
```

#### TestExpiration
测试数据过期

```go
测试场景:
✓ 设置 1 秒过期时间
✓ 自动清理过期数据
✓ 保留未过期数据
```

---

### 3. 模板缓存测试

#### TestGetOrCreate_TemplateCache
测试 DNS 模板缓存 (性能优化)

```go
测试场景:
✓ 首次创建模板
✓ 第二次从缓存获取
✓ 验证指针相同
```

#### TestGetOrCreate_Concurrent
测试并发访问缓存

```go
测试场景:
✓ 100 个协程并发获取
✓ 验证返回相同缓存对象
✓ 无数据竞争
```

---

### 4. 集成测试

#### TestBasicVerification
基础验证功能测试

```go
测试场景:
✓ 验证已知域名 (www.baidu.com, www.google.com)
✓ 至少找到一个有效结果
✓ 结果包含域名和答案
```

#### TestCNAMEParsing
CNAME 解析正确性测试

```go
测试场景:
✓ 验证 CNAME 记录域名
✓ 检查无 "comcom" 拼接错误
✓ 检查无空字符
```

#### TestHighSpeed
高速扫描测试

```go
测试场景:
✓ 10000 pps 速率
✓ 100 个域名 (50 存在 + 50 不存在)
✓ 验证准确率 > 80%
```

---

## 🔬 性能基准测试结果

### DNS 解析性能

```
BenchmarkParseDNSName-8           5000000    250 ns/op    48 B/op   2 allocs/op
```

**解读**:
- 每次操作 250 纳秒
- 每次分配 48 字节
- 每次 2 次内存分配

### 状态数据库性能

```
BenchmarkAdd-8                   2000000    750 ns/op    128 B/op   3 allocs/op
BenchmarkGet-8                  10000000    150 ns/op      0 B/op   0 allocs/op
BenchmarkConcurrentAdd-8         1000000   1200 ns/op    128 B/op   3 allocs/op
```

**解读**:
- 添加: 750 ns/op
- 获取: 150 ns/op (无内存分配)
- 并发添加: 1200 ns/op (有锁竞争)

### 模板缓存性能

```
BenchmarkGetOrCreate_CacheHit-8  20000000    80 ns/op      0 B/op   0 allocs/op
BenchmarkGetOrCreate_CacheMiss-8  1000000  1500 ns/op    256 B/op   5 allocs/op
```

**解读**:
- 缓存命中: 80 ns/op (极快,无内存分配)
- 缓存未命中: 1500 ns/op (需创建模板)
- 命中 vs 未命中: **18 倍性能差异**

---

## 🧩 测试数据准备

### 创建测试域名列表

```bash
# 小规模测试 (100 个域名)
cat > test_domains.txt <<EOF
www.baidu.com
www.google.com
dns.google
www.github.com
www.cloudflare.com
EOF

# 中规模测试 (1000 个域名)
for i in {1..1000}; do
    echo "test$i.example.com"
done > test_domains_1k.txt

# 大规模测试 (10000 个域名)
for i in {1..10000}; do
    echo "subdomain$i.example.com"
done > test_domains_10k.txt
```

### 创建 DNS 服务器列表

```bash
cat > test_resolvers.txt <<EOF
8.8.8.8
8.8.4.4
1.1.1.1
1.0.0.1
114.114.114.114
223.5.5.5
EOF
```

---

## 📈 性能测试场景

### 场景 1: 低速稳定性测试
```bash
# 1000 pps, 1000 个域名
./ksubdomain v -f test_domains_1k.txt -b 1m --retry 3
```

**预期**:
- 成功率: > 99%
- 错误率: < 0.1%
- 耗时: ~10 秒

### 场景 2: 中速性能测试
```bash
# 10000 pps, 10000 个域名
./ksubdomain v -f test_domains_10k.txt -b 5m --retry 3
```

**预期**:
- 成功率: > 95%
- 错误率: < 1%
- 耗时: ~30 秒

### 场景 3: 高速压力测试
```bash
# 50000 pps, 100000 个域名
./ksubdomain v -f test_domains_100k.txt -b 10m --retry 5
```

**预期**:
- 成功率: > 90%
- 缓冲区错误: < 1% (Mac)
- 耗时: ~60 秒

---

## 🐛 回归测试

### Issue #70: CNAME 解析错误
```bash
# 测试 CNAME 记录域名
echo "www.github.com" | ./ksubdomain v --stdin

# 检查输出,不应该出现:
# ❌ CNAME example.comcom
# ✅ CNAME example.com
```

### Issue #68: WSL2 错误提示
```bash
# 在 WSL2 环境测试
./ksubdomain test

# 如果网卡未激活,应该看到友好提示:
# ✅ "解决方案: sudo ip link set eth0 up"
# ❌ 仅显示 "That device is not up"
```

### Mac 缓冲区问题
```bash
# Mac 高速测试
sudo ./ksubdomain v -f test_domains_10k.txt -b 10m

# 检查日志:
# ✅ "Mac: BPF 缓冲区已设置为 2 MB"
# ✅ 缓冲区错误 < 0.1%
```

---

## 🔧 CI/CD 集成

### GitHub Actions 示例

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.23
      
      - name: Install libpcap
        run: sudo apt-get install -y libpcap-dev
      
      - name: Run unit tests
        run: go test -v -cover ./pkg/...
      
      - name: Run benchmarks
        run: go test -bench=. ./pkg/... | tee bench.txt
      
      - name: Upload coverage
        run: bash <(curl -s https://codecov.io/bash)
```

---

## 📝 测试最佳实践

### 1. 编写测试
- ✅ 每个函数至少一个测试
- ✅ 测试正常路径和边界情况
- ✅ 测试错误处理
- ✅ 使用表驱动测试

### 2. 命名规范
- `Test*`: 单元测试
- `Benchmark*`: 性能测试
- `Example*`: 示例代码

### 3. 断言使用
- 使用 `testify/assert` 库
- 提供清晰的错误消息
- 避免 `panic`

### 4. 性能测试
- 使用 `b.ResetTimer()`
- 测试前预热缓存
- 多次运行取平均

---

## 🎯 测试检查清单

提交 PR 前确认:

- [ ] 所有单元测试通过
- [ ] 覆盖率 > 60%
- [ ] 性能测试无退化
- [ ] 集成测试通过 (可选)
- [ ] 新功能有对应测试
- [ ] Bug 修复有回归测试

---

## 📚 参考资料

- [Go Testing 官方文档](https://golang.org/pkg/testing/)
- [Testify 文档](https://github.com/stretchr/testify)
- [Go 性能优化](https://dave.cheney.net/high-performance-go-workshop)

---

**编写测试 = 保证质量 + 避免回归 + 提升信心** 🧪
