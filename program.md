# KSubdomain — Agent Program

> 本文件是面向 AI Agent 的项目上下文与任务指引。Agent 应首先通读此文件，再开始任何工作。
> 类似 autoresearch 的 `program.md`，这里定义了项目的"研究组织逻辑"。

---

## 项目概述

**KSubdomain** 是一个无状态子域名枚举工具，核心思路来自 Masscan：绕过内核协议栈，直接在网卡层面收发 raw DNS 包，配合内置轻量状态表做重传，从而实现极速扫描。

当前版本 **v2.5**，模块路径 `github.com/boy-hack/ksubdomain/v2`，使用 Go 1.23+。

---

## 代码地图（必读）

```
cmd/ksubdomain/          CLI 入口，4 个子命令：enum / verify / test / device
pkg/
  options/options.go     核心配置结构体 Options（所有功能的控制面板）
  runner/runner.go       扫描主循环，RunEnumeration() 启动 5 条并发 goroutine
  runner/send.go         发包：模板缓存 + 批量发送 + 指数退避重试
  runner/recv.go         收包：BPF filter + CPU*2 并行解析 + CPU*2 并行处理
  runner/statusdb/db.go  64 分片锁状态表，记录发送状态与重传计数
  runner/mempool.go      全局 sync.Pool，复用 DNS 结构体减少 GC
  runner/result.go       结果处理 + predict 预测触发
  runner/wildcard.go     泛解析过滤逻辑
  runner/outputter/      Output 接口 + txt/json/csv/jsonl/screen 实现
  device/                网卡初始化（pcap），构建 EtherTable
  sdk/sdk.go             Go SDK 封装（Enum / Verify / *WithContext）
  ns/ns.go               NS 记录查询（--ns 模式辅助）
internal/
  predict/generator.go   基于 regular.cfg + regular.dict 的预测生成器
  assets/data/           内置字典 subdomain.txt / subnext.txt
docs/
  OUTPUT_FORMATS.md      输出格式完整说明
```

---

## 核心数据流

```
Domain 输入 (chan string)
    │
    ▼
loadDomainsFromSource  ──►  domainChan
                                │
                    ┌───────────┘
                    ▼
            sendCycleWithContext
             （模板缓存 + 批量发包）
                    │
                    ▼
             statusdb 登记 {Domain, Dns, Time, Retry}
                    │
      ┌─────────────┴──────────────────────────────────┐
      ▼                                                 ▼
 recvChan → packetChan → dnsChan → resultChan     monitorProgress
                                       │             + retry()
                                       ▼
                              handleResultWithContext
                                       │
                          ┌────────────┴──────────────┐
                          ▼                            ▼
                   Output.WriteDomainResult()    predict() → predictChan
```

---

## 关键指标（以此衡量改动是否有效）

| 指标 | 当前基准 | 目标方向 |
|------|---------|---------|
| **检测速率** | 100k 域名约 30 秒（5M 带宽，4 核） | 同等带宽继续提升吞吐 |
| **漏报率** | 对比 massdns 结果几乎一致 | 减少 timeout 造成的漏报 |
| **误报率** | 泛解析过滤后接近 0 | advanced 模式覆盖更多泛解析场景 |
| **内存占用** | 百万级域名仍维持低内存 | 不引入新的大块堆分配 |
| **SDK 易用性** | NewScanner(config).Enum(domain) 3 行接入 | 任何改动不破坏 SDK 接口兼容性 |

每次修改后，执行以下命令验证基准不退步：

```bash
# 单元测试（无需网卡）
go test ./...

# 集成测试（需要网卡权限）
sudo go test ./test/ -v -run Integration

# 发包速率基准
sudo ./ksubdomain test
```

---

## 当前优先任务（Roadmap）

按优先级从高到低排列，Agent 应按序处理，每次专注一个任务。

**Roadmap 维护规则（Agent 可直接修改此文件）**：
- 完成一个任务 → 将 `[ ]` 改为 `[x]`，并在 `agent-log.md` 记录结果
- 发现新问题或新改进点 → 自行在对应优先级下追加条目，注明发现来源
- 评估某条任务不再必要 → 将 `[ ]` 改为 `[~]` 并在条目后用括号注明原因
- **需要用户决策的事项不在此处等待**，而是写入 `agent-log.md` 的 `【待决策】` 区块，由用户查阅日志后自行处理

### P0 — 检测效率

- [x] **动态超时自适应**：RTT EWMA滑动均值，上界内部固定10s，始终启用
- [x] **接收侧背压控制**：packetChan双水位监控（80%触发/50%恢复），send侧背压时sleep 5ms降速
- [x] **批量重传合并**：按DNS server分组批量send()，去掉channel中转层，复用slice/map减少GC

### P1 — 开发者集成

- [x] **流式结果回调**：EnumStream/VerifyStream，streamCollector 实时回调，无缓冲
- [x] **自定义 Output 接入**：Config.ExtraWriters []outputter.Output，追加到内部 writer 之后
- [x] **错误类型化**：pkg/core/errors 包，ErrPermissionDenied/ErrDeviceNotFound/ErrDeviceNotActive/ErrPcapInit/ErrDomainChanNil，sdk 重导出
- [x] **dev.md 重写**：架构图、goroutine 表、SDK/Options 字段表、错误处理、背压说明、构建命令

### P1.5 — 工具联动兼容性审查

> 参考「工具联动兼容性矩阵」章节，逐条验证每个集成场景是否真实可用。

- [x] **httpx 管道**：移除 screen.go 的 \r 前缀，--od --silent 输出干净 domain\n
- [x] **JSONL 下游兼容**：bufio 替换 Sync()，parseAnswers() 共享解析，字段名 domain/type/records 稳定
- [x] **退出码语义**：SuccessCount()==0 时 os.Exit(1)，文档化在 faq.md

### P2 — 文档完善

- [x] **docs/quickstart.md**：安装、首次扫描、常用参数表、输出格式、故障排查
- [x] **docs/api.md**：完整 SDK API 参考，Config/Result/Scanner 方法、错误处理、自定义 sink
- [x] **docs/best-practices.md**：带宽选择表、DNS resolver 指南、泛解析、管道输出、大规模扫描
- [x] **docs/faq.md**：sudo/CAP_NET_RAW、网卡错误、macOS BPF、WSL2、泛解析、httpx 乱码、退出码
- [x] **内联注释**：RunEnumeration goroutine 拓扑图，sendCycleWithContext 批量+背压说明

### P3 — 工程健康

- [x] **`simple` 二进制缺失**：确认为编译产物，加入 .gitignore，更新 examples 代码修复 API
- [x] **CI 矩阵**：ci.yml 五平台矩阵（Linux amd64/arm64、macOS amd64/arm64、Windows amd64）+ lint job
- [x] **版本号自动化**：Version 改为 var，build.yml/build.sh 通过 ldflags 注入 git tag

---

## 修改约束

1. **不修改 `pkg/options/options.go` 中 `Options` 的已有字段名**：下游用户代码依赖此结构体，字段重命名视为破坏性变更
2. **不修改 `pkg/sdk/sdk.go` 中 `Config`、`Scanner`、`Result` 的已有字段和方法签名**：SDK 公开 API，需向后兼容
3. **`Output` 接口只可扩展，不可修改已有方法**：现有 `WriteDomainResult` 和 `Close` 签名不变
4. **所有平台文件（`*_darwin.go`、`*_linux.go`、`*_windows.go`）需同步更新**
5. **Go 规则**：不使用 `++`/`--`，字符串用反引号或双引号（Go 惯例），接口定义只放在 `pkg/` 下

---

## 上手流程（Agent 首次运行）

```bash
# 1. 查看项目状态
git status
go build ./...

# 2. 运行现有测试，确认基线通过
go test ./pkg/... -v

# 3. 检查缺失文档（README 中引用但未创建的文件）
ls docs/

# 4. 选择 Roadmap 中最高优先级未完成项，新建分支后开始实现
# 每次只聚焦一个任务，完成后运行 go test ./... 确认无回归
```

---

## 测试规范（内网环境）

> **当前为内网环境，禁止发起大批量外部 DNS 请求。**

Agent 在实现每个功能后，必须自行运行以下测试，验证参数行为正确、无崩溃、无明显 bug：

```bash
# 编译验证（无需权限，最快检查）
go build ./...

# 单元测试（无需网卡，全部通过为准入条件）
go test ./... -timeout 30s

# 本地轻量功能冒烟测试（只需几条域名，不做大批量扫描）
# 构建二进制
go build -o ksubdomain_dev ./cmd/ksubdomain

# verify 模式：验证 1 条已知域名，检查 -d / -o / --oy / --silent / --np 参数
sudo ./ksubdomain_dev verify -d www.baidu.com --timeout 5 --retry 2 -b 1m --np
sudo ./ksubdomain_dev verify -d www.baidu.com --oy jsonl --silent
sudo ./ksubdomain_dev verify -d www.baidu.com -o /tmp/ksub_test.txt && cat /tmp/ksub_test.txt

# enum 模式：只枚举 1 个域名、新建一个小字典,不超过10行，检查基本流程不崩溃
sudo ./ksubdomain_dev enum -d baidu.com --timeout 5 --retry 1 -b 1m --np -f subdomain.dic

# test 模式：测试本地网卡发包速率，无外部流量
sudo ./ksubdomain_dev test

# device 模式：列出可用网卡
sudo ./ksubdomain_dev device
```

**测试要点**：
- 每个 CLI 参数至少覆盖一次，确认有无 panic 或非预期输出
- 输出文件格式正确（txt 每行一条、jsonl 每行合法 JSON）
- `--silent` 模式下无多余输出，`--np` 模式下无域名打印
- 测试完毕后删除临时文件：`rm -f /tmp/ksub_test* ksubdomain_dev`

---

## 分支管理规范

> **main 分支只读，任何功能改动都在独立分支上进行。**

每次实现一个 Roadmap 任务，遵循以下流程：

```bash
# 1. 从 main 新建功能分支，命名格式：feature/<简短描述>
git checkout main
git checkout -b feature/dynamic-timeout      # P0 示例
git checkout -b feature/stream-sdk           # P1 示例
git checkout -b docs/quickstart              # P2 示例
git checkout -b fix/simple-binary            # P3 示例

# 2. 实现功能，提交粒度：每个逻辑单元一次 commit
git add .
git commit -m "feat(runner): add RTT sliding window for dynamic timeout"

# 3. 功能完成后在分支上运行完整测试
go test ./... -timeout 30s

# 4. 将分支结论记录到运行日志（见下文），不合并到 main
# main 分支由人工决定何时合并
```

**分支命名前缀约定**：

| 前缀 | 用途 |
|------|------|
| `feature/` | 新功能实现（对应 Roadmap P0/P1） |
| `docs/` | 文档补充（对应 Roadmap P2） |
| `fix/` | Bug 修复（对应 Roadmap P3 或冒烟测试发现的问题） |
| `refactor/` | 重构（不改变外部行为） |
| `exp/` | 实验性改动（不确定是否保留） |

---

## 运行日志规范

Agent 每完成一个任务，必须在 `agent-log.md` 文件中追加一条记录。该文件位于项目根目录，不存在则自行创建。

**记录格式**：

```markdown
## [YYYY-MM-DD] <分支名> — <任务简述>

**目标**：对应 Roadmap 中的哪一条
**改动文件**：列出修改的文件
**测试结果**：
- go test ./... → PASS / FAIL（附错误摘要）
- 冒烟测试：列出执行的命令及输出摘要
**结论**：改动是否有效，是否引入新问题，下一步建议

<!-- 若存在需要用户决策的事项，追加以下区块；无则省略 -->
### 【待决策】
```

**说明**：
- `【待决策】` 区块是 Agent 与用户沟通的唯一通道。Agent 不中断工作等待回复，而是记录后继续处理下一任务。
- 用户查看 `agent-log.md` 时，搜索 `【待决策】` 即可找到所有待处理事项，勾选 `[x]` 或在条目下方回复意见后，Agent 下次运行时读取并执行。
- 典型的待决策场景：破坏性 API 变更是否接受、新增外部依赖是否引入、某功能是否值得实现、发现潜在安全或性能风险需要人工确认。

**示例**：

```markdown
## [2026-03-17] feature/dynamic-timeout — 动态超时自适应

**目标**：P0 动态超时自适应
**改动文件**：pkg/runner/runner.go, pkg/runner/send.go, pkg/options/options.go
**测试结果**：
- go test ./... → PASS
- 冒烟测试：`sudo ./ksubdomain_dev verify -d www.baidu.com --timeout 5 --retry 2`
  输出：`www.baidu.com => 110.242.68.66`，耗时约 2s，正常
**结论**：RTT 滑动均值计算正常，低延迟场景下超时提前收敛，无漏报。
  建议后续在 P0 背压控制任务中复用 RTT 数据。

### 【待决策】
      若希望新版本默认开启动态超时，需将默认值改为 true，这属于行为变更，请确认。
      方案 A：默认 false，用户显式 --dynamic-timeout 启用（保守）
      方案 B：默认 true，--no-dynamic-timeout 可关闭（激进，影响现有脚本）
```

---

## 输出格式速查

| 格式 | 写入时机 | 适用场景 |
|------|---------|---------|
| `txt` | 实时（每条结果） | 人工查阅、管道 grep |
| `json` | 完成后一次性 | 离线分析、脚本处理 |
| `csv` | 完成后一次性 | Excel/数据库导入 |
| `jsonl` | 实时流式 | 管道链（httpx/nuclei）、监控系统 |
| screen | 实时彩色（stdout） | 交互式终端 |

---

## 工具联动兼容性矩阵

> Agent 在审查 P1.5 任务时，以此为检查清单。每条给出期望行为、验证命令和当前状态。

### ProjectDiscovery 工具链

| 工具 | 联动方式 | 期望行为 | 关键参数 | 当前状态 |
|------|---------|---------|---------|---------|
| **httpx** | stdout 管道 | 每行一个域名，httpx 自动探活 | `--od --silent` | 待验证 |
| **nuclei** | stdout 管道或临时文件 | 域名作为目标，模板正常扫描 | `--od` + nuclei `-l /dev/stdin` | 待验证 |
| **naabu** | stdout 管道 | 域名列表作为端口扫描输入 | `--od` + naabu `-iL -` | 待验证 |
| **dnsx** | stdout 管道 | ksubdomain 初筛 → dnsx 多记录精查 | `--od --silent` + dnsx `-a -cname` | 待验证 |
| **subfinder** | subfinder → ksubdomain stdin | subfinder 发现域名 → ksubdomain verify 存活确认 | ksubdomain `v --stdin` | 待验证 |
| **alterx** | alterx → ksubdomain stdin | 排列生成 → 批量验证，大量输入不丢包 | ksubdomain `v --stdin -b 10m` | 待验证 |
| **katana** | ksubdomain → katana | 子域名列表作为爬取种子 | `--od` + katana `-list -` | 待验证 |
| **chaos** | chaos → ksubdomain verify | chaos 数据集导入验证存活 | ksubdomain `v -f chaos_output.txt` | 待验证 |

### 验证命令参考（内网用小字典，不发大批量请求）

```bash
# 1. httpx 联动：枚举 → HTTP 探活
sudo ./ksubdomain_dev enum -d baidu.com -f subdomain.dic --od --silent | httpx -silent -title

# 2. dnsx 联动：ksubdomain 初筛 → dnsx 多类型查询
sudo ./ksubdomain_dev verify -d www.baidu.com --od --silent | dnsx -a -cname -resp

# 3. naabu 联动：枚举 → 端口扫描
sudo ./ksubdomain_dev enum -d baidu.com -f subdomain.dic --od --silent | naabu -silent -p 80,443

# 4. subfinder → ksubdomain：二阶段发现
subfinder -d baidu.com -silent | sudo ./ksubdomain_dev verify --stdin --silent --od

# 5. alterx → ksubdomain：排列验证
echo 'www.baidu.com' | alterx -silent | sudo ./ksubdomain_dev verify --stdin --np

# 6. JSONL 流式过滤后交给 httpx
sudo ./ksubdomain_dev enum -d baidu.com -f subdomain.dic --oy jsonl --silent \
  | jq -r 'select(.type=="A") | .domain' \
  | httpx -silent

# 7. nuclei 漏洞扫描
sudo ./ksubdomain_dev enum -d baidu.com -f subdomain.dic --od --silent \
  | nuclei -l /dev/stdin -t technologies/ -silent
```

### 联动验证要点

- **`--od` 输出必须是纯域名单列**，无 `=>` 箭头、无 IP、无空行，否则下游工具解析失败
- **`--silent` 必须屏蔽进度条和 banner**，只保留结果行，否则污染管道数据
- **`--stdin` 必须正确处理 EOF**，subfinder/alterx 结束后 ksubdomain 应正常退出而非挂起
- **退出码**：有结果退出 0，无结果退出非 0，让 shell `&&` 链路能正确短路
- **JSONL 字段名**必须稳定（`domain`、`type`、`records`、`timestamp`），jq 脚本依赖此约定

---

## 集成速查

```bash
# 完整侦察链：子域名枚举 → HTTP 探活 → 漏洞扫描
sudo ./ksubdomain enum -d example.com -f subdomain.dic --od --silent \
  | httpx -silent \
  | nuclei -l /dev/stdin -silent

# 二阶段发现：subfinder 粗扫 → ksubdomain 精筛存活
subfinder -d example.com -silent \
  | sudo ./ksubdomain verify --stdin --od --silent

# 流式过滤（仅 A 记录）后端口扫描
sudo ./ksubdomain enum -d example.com --oy jsonl --silent \
  | jq -r 'select(.type=="A") | .domain' \
  | naabu -silent -p 80,443,8080,8443

# Go SDK 最简集成
scanner := sdk.NewScanner(sdk.DefaultConfig)
results, err := scanner.Enum('example.com')
```

---

*`program.md` 由 Agent 和人工共同维护：Roadmap 条目由 Agent 自主更新，需要人工判断的事项统一写入 `agent-log.md` 的 `【待决策】` 区块。*
