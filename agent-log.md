
---

## [2026-03-18] feature/stream-sdk — 所有 roadmap 任务完成

### 本次完成（接昨日进度）

**P2-2 httpx 管道** (commit 44eeef2)
- screen.go 移除所有输出路径的 `\r` 前缀
- 非 silent 时防止 padding 为负值

**P2-3 JSONL 下游兼容** (commit 44eeef2，同一次提交)
- 用 `bufio.Writer`（64 KiB）替换每条 `file.Sync()`
- 提取 `parseAnswers()` 共享函数，beautified.go 也复用
- 新增 AAAA 记录类型检测
- 字段名稳定：`domain` / `type` / `records` / `timestamp`

**P2-4 退出码语义** (commit 21c99a7)
- Runner 新增 `SuccessCount() uint64` 方法
- enum.go / verify.go：SuccessCount==0 时 os.Exit(1)

**P3-1 docs/quickstart.md** (commit a15ba1a)
**P3-2 docs/api.md** (commit a15ba1a)
**P3-3 docs/best-practices.md** (commit a15ba1a)
**P3-4 docs/faq.md** (commit a15ba1a)

**P3-5 内联注释** (commit e8dd7d1)
- RunEnumeration：goroutine 拓扑图
- sendCycleWithContext：批量/背压设计说明

**P3-6 `simple` 二进制** (commit 31d0301)
- 确认是编译产物，加 .gitignore
- examples/simple、examples/advanced 修复旧 API 引用

**P3-7 CI 矩阵** (commit 286460f)
- .github/workflows/ci.yml：5平台构建矩阵 + lint job

**P3-8 版本号自动化** (commit f9d107e)
- Version const → var，支持 ldflags 注入
- build.yml / build.sh 全部加 ldflags

### 结论
**全部 19 项 roadmap 任务已完成。**
- P0（3项）：feature/dynamic-timeout 分支
- P1-P3（16项）：feature/stream-sdk 分支
