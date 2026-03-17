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
