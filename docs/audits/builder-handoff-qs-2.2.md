# 施工提示词：QS-2.2 goleak 集成 + WaitConfirmed 无泄漏证明

> **[角色:施工]** — 你是本任务的施工方 agent（见 `.devin/rules/dual-terminal-roles.md`）。
> 严格按 S1–S4 施工，不做决策；超出提示词范围 = 违规，停下转 `[转交决策]`。
> 完成后先过自审（D-012）修至全绿再交回；自报末行带 `[施工完成:QS-2.2] @<commit-hash>`（D-014）。
> 勿部署、勿 push、禁 `--no-verify`；只显式 add 本任务文件；commit 用 `ANT_ROLE=builder git commit` 前缀（D-015）；**不更新任何交接层文件**。
>
> **最终决策：Devin CLI（[角色:决策终] 激活）**

## 立项背景（触发 + 证据链）

- registry `QS-2.2`；spec `docs/spec/vm-pipeline-quality-stability-improvement-plan.md` v2 §4.1；决策 D-009（QS-2.1 watcher 重构已否决——watcher 有界生命周期 `trade_barrier.go:298-308`，goleak 测试即是否决证据）。
- **决策方预核实的关键事实**（施工时复核，若不符 `[转交决策]`）：
  a. `go.mod` 无 `go.uber.org/goleak`（仅有 zap/atomic/multierr）——需新增依赖，按项目规则选 ≥7 天前发布的版本（`go.uber.org/goleak v1.3.0`，2023-12 发布，达标）。
  b. `tools/mql2go` 与 `internal/connect/strategy` 两包**均无 `func TestMain`**——各新建 `main_test.go`。
  c. `WaitConfirmed` watcher（`trade_barrier.go:290-321`）：`stopWatcher` chan + `defer close` + ctx.Done 分支，生命周期绑定单次调用。
  d. 既有依赖约束：新增依赖必须 `go get go.uber.org/goleak@v1.3.0` 显式钉版，禁浮动 latest。

## 设计 SSOT 声明

- 设计文档：spec v2 §4.1。契约：AGENTS.md §1 fail-closed；决策 D-009/D-012/D-014/D-015。

## 约束与目标

- 两包 `TestMain` 调 `goleak.VerifyTestMain(m)`；`TestWaitConfirmed_NoGoroutineLeak` 用 `goleak.VerifyNone(t)` 证 watcher 调用后无残留。
- **pre-existing 泄漏处理裁定**：`VerifyTestMain` 若暴露包内既有泄漏（非本任务引入），施工方**不得**自行 `IgnoreTopFunction` 掩盖——逐条列出泄漏 goroutine 的 top function + 源码 file:line + 归属判断（库自有/测试残留/生产泄漏）在自报中，由决策方裁定 ignore 或另立项；本任务可提交的最小豁免仅限"已证明为第三方库后台常驻"（如 grpc/runtime）且逐条注明理由。
- 依赖操作只准 `go get go.uber.org/goleak@v1.3.0` + `go mod tidy`；go.mod/go.sum 变更纳入本任务 commit。

## 边界 / 不做

- 不改 `WaitConfirmed`/watcher 实现（QS-2.1 否决保持）；不修任何暴露出的 pre-existing 泄漏（只报告）；不动其他包的 TestMain；不升级其它依赖。

## 施工指令

### S1 — 事实复核（只读）

- **目标**：复核 a–d 四条事实 + 确认 `goleak` 最新稳定版发布日期 ≥7 天。
- **验证**：自报逐条"属实/不符 + 证据"。

### S2 — 依赖引入

- **落点**：`cd backend && go get go.uber.org/goleak@v1.3.0 && go mod tidy`。
- **验证**：`go.mod` 出现 goleak 钉版；`go build ./...` 通过。

### S3 — 两包 TestMain + watcher 无泄漏测试

- **坐标**：新建 `tools/mql2go/main_test.go` 与 `internal/connect/strategy/main_test.go`；`internal/connect/strategy/` 既有测试文件内加 `TestWaitConfirmed_NoGoroutineLeak`。
- **落点**：
  a. `main_test.go`：`func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }`。
  b. `TestWaitConfirmed_NoGoroutineLeak`：构造 `TradeBarrier`（复用 `trade_barrier_test.go` 既有构造模式）→ ctx cancel 路径与 terminal 路径各跑一次 `WaitConfirmed` 返回 → `goleak.VerifyNone(t)`。
- **验证**：先红后绿——先确认 `VerifyTestMain`/`VerifyNone` 在无 watcher 残留时绿；若 RED，区分"本任务测试代码自身泄漏"（必须修）与"包内 pre-existing 泄漏"（报告见约束）。

### S4 — pre-existing 泄漏处置

- **落点**：若 `VerifyTestMain` RED 且根因为包内既有 goroutine（非本任务引入）：自报逐条列 top function + file:line + 归属；仅对"已证明第三方库常驻"项加 `goleak.IgnoreTopFunction`/`IgnoreCurrent` 并在代码注释写明理由；其余情况 `[转交决策]`。
- **验证**：最终 `go test -count=1` 两包全绿。

## 对抗证明

- mutation：临时在 `TestWaitConfirmed_NoGoroutineLeak` 内 `go func() { select{} }()` 注入泄漏 → `VerifyNone` RED；restore → GREEN。
- mutation：`stopWatcher` 的 `defer close` 注释掉模拟泄漏场景（**改后必须恢复**，仅本地验证不提交）→ watcher 残留被 goleak 捕获 → RED；restore → GREEN。

## 验收标准

- [ ] `go build ./...` 通过；`go.mod` goleak=v1.3.0 钉版
- [ ] `go test -count=1 ./tools/mql2go/... ./internal/connect/strategy/...` 全过
- [ ] `go test -race -count=3 ./internal/connect/strategy/` 通过
- [ ] `go run ./tools/check-file-lines --strict` 本任务文件零新增警告
- [ ] `gofmt -l` / `go vet` 本任务文件零输出
- [ ] pre-existing 泄漏清单（若有）逐条带 file:line + 处置
- [ ] 对抗证明 RED→restore→GREEN（附输出）
- [ ] diff 无交接层文件、无范围外改动

## 施工完成自审（强制，D-012）

- [ ] 逐项重跑验收标准贴真实输出；diff 三问；发现项修复至全绿；无自审记录 = 复审退回。

## 交付格式

自报：改动文件清单 + 验收项证据 + 自审记录 + S1 复核 + pre-existing 泄漏清单 + 遗留疑问。
**自报最后一行固定为** `[施工完成:QS-2.2] @<commit-hash>`。
**停手等 Devin CLI 复审。**

[施工完成:QS-2.2] @<本任务最终commit>
