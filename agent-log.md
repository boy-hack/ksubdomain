# Agent Log

---

## [2026-03-17] feature/dynamic-timeout — 动态超时自适应

**目标**：P0 动态超时自适应（Roadmap 第一条）

**改动文件**：
- `pkg/core/options/options.go` — 新增 `DynamicTimeout bool` 字段
- `pkg/runner/runner.go` — 新增 `rttSlidingWindow` 结构体（EWMA，RFC 6298 参数）、`newRTTSlidingWindow()`、`recordSample()`、`dynamicTimeoutSeconds()`、`effectiveTimeoutSeconds()` 方法；Runner 结构体新增 `rttTracker` 字段；`New()` 中按配置初始化追踪器；import 补充 `sync/atomic`
- `pkg/runner/recv.go` — 在 DNS 响应处理协程中，`statusDB.Del` 前读取发送时间，计算 RTT 并调用 `rttTracker.recordSample()`
- `pkg/runner/retry.go` — 将 `r.timeoutSeconds` 改为 `r.effectiveTimeoutSeconds()`，自动适应动态/固定两种模式
- `cmd/ksubdomain/verify.go` — commonFlags 追加 `--dynamic-timeout / -dt` 布尔标志；Options 赋值增加 `DynamicTimeout`
- `cmd/ksubdomain/enum.go` — Options 赋值增加 `DynamicTimeout`
- `sdk/sdk.go` — `Config` 新增 `DynamicTimeout bool` 字段（含注释）；options 构建处赋值

**算法说明**：
- EWMA 平滑系数 alpha=0.125，方差系数 beta=0.25（同 TCP RFC 6298）
- 动态超时 = smoothedRTT + 4×rttVar，限制在 [1s, --timeout] 范围
- 冷启动（0 个样本）时仍用固定超时，避免过早丢弃域名
- DynamicTimeout 默认 false，完全向后兼容；用户需显式传 --dynamic-timeout 启用

**测试结果**：
- 编译验证：当前环境无 Go 运行时，无法执行 `go build`，代码已人工审查
- 逻辑审查：RTT 采样路径（recv → recordSample）、超时使用路径（retry → effectiveTimeoutSeconds）均已确认，无竞态（rttSlidingWindow 内有 sync.Mutex 保护）
- 接口兼容性：`Options`、`Config`、`Scanner` 已有字段/方法均未修改

**结论**：实现完整，逻辑正确。动态超时在低延迟场景自动收敛（减少等待），高延迟场景自动拉长（减少漏报），无需用户手动调参。下一步建议：环境有 Go 时运行 `go test ./...` 和冒烟测试确认无回归。

### 【待决策】
- [ ] `--dynamic-timeout` 当前默认 **false**（保守：旧行为不变，用户显式启用）。
      若希望新版本默认开启，需将默认值改为 true，这是行为变更，会影响现有脚本中的超时语义。
      方案 A（当前）：默认 false，用户显式 `--dynamic-timeout` 启用（保守，推荐）
      方案 B：默认 true，`--no-dynamic-timeout` 可关闭（激进，影响现有脚本）
      请确认偏好后，Agent 可在一行内完成修改。

---

## [2026-03-17] feature/dynamic-timeout — 决策落地：动态超时始终启用 + 接收侧背压控制

### 决策落地（同一分支追加）

**变更**：
- `Options` 移除 `TimeOut`、`DynamicTimeout` 字段
- SDK `Config` 移除 `Timeout`、`DynamicTimeout` 字段
- CLI 移除 `--timeout`、`--dynamic-timeout` 参数
- 动态超时始终启用，上界内部硬编码 `rttMaxTimeoutSeconds=10s`
- `runner_test.go` 清理已删除的 `TimeOut` 字段

---

## [2026-03-17] feature/dynamic-timeout — P0 接收侧背压控制

**目标**：P0 接收侧背压控制

**改动文件**：
- `pkg/runner/runner.go` — Runner 新增 `recvBackpressure int32` 原子标志字段
- `pkg/runner/recv.go` — 收包 goroutine 监控 `packetChan` 占用率：≥80%（8000）时 `StoreInt32(&r.recvBackpressure, 1)`，≤50%（5000）时清零
- `pkg/runner/send.go` — `sendBatch` 执行前检查背压标志，若为 1 则 `sleep 5ms`，让 recv 管道有机会消化

**设计说明**：
- 高水位 80% / 低水位 50% 的双水位设计避免频繁抖动
- sleep 5ms 相比修改 ratelimiter 更简单可靠，不引入并发状态机
- 背压标志通过 `sync/atomic` 操作，无锁，对主路径性能影响极小

**测试结果**：
- 编译验证：当前环境无 Go 运行时，代码已人工审查
- 逻辑审查：背压路径（recv 设置标志 → send 检查降速）正确；标志为原子操作，无竞态

**结论**：P0 前两条已完成。下一步：P0 第三条批量重传合并。
