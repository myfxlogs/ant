# 施工交接：D-VM-LIVE-001-P1B（Phase 1b 旁路清理）

> 审计方（Claude）2026-08-26 派工，施工方（GLM-5.2）任务：**one task = one scope，只做本文件 S1–S3**。不重新审计、不自由发挥、不扩大范围。
> **基线 HEAD=`b42f2cfa`，工作树干净**。开工前先核验 SSOT 指纹（见下），再整读本文件 + `AGENTS.md §0` + registry「D-VM-LIVE-001 范围重定」段 + P1/P1-R1 复审记录 + Phase 2 重估裁决。
> **SSOT 核验**：本文件是 registry `D-VM-LIVE-001-P1B:BEGIN/END` 区块（SSOT 权威文本）的只读副本。对 registry 区块执行
> `sed -n '/^<!-- D-VM-LIVE-001-P1B:BEGIN -->$/,/^<!-- D-VM-LIVE-001-P1B:END -->$/p' docs/audits/tech-debt-registry.md | sed '1d;$d' | sha256sum`
> 必须等于 `ed1663029ae774978c7dc939d533461f8b74b59fa6d757316f5dcc86734e06b1`。不匹配立即停止返回 Claude。**任何修订只改 registry 区块并重算指纹，本文件不单独修订。**

---

<!-- D-VM-LIVE-001-P1B:BEGIN -->
## D-VM-LIVE-001-P1B 施工提示词（Phase 1b 旁路清理；施工者 GLM-5.2，设计/验收者 Claude）

> **指纹与核验（协议 v2，2026-08-25）**：SSOT SHA256 见本区块 END 标记之后一行（指纹行在 marker 外，天然不含于提取结果）。核验命令（**无任何排除/删除操作**，提取 marker 两行之间的区块原文整体哈希）：
> `sed -n '/^<!-- D-VM-LIVE-001-P1B:BEGIN -->$/,/^<!-- D-VM-LIVE-001-P1B:END -->$/p' docs/audits/tech-debt-registry.md | sed '1d;$d' | sha256sum`
> 与 SSOT 值比对。不匹配说明 prompt 被改动，立即停止并返回 Claude。

> **先整读** `AGENTS.md §0`、本 registry 的「D-VM-LIVE-001 范围重定」段、P1/P1-R1 复审记录、Phase 2 重估裁决和本节，再动手。只做本节 S1–S3。

### 立项背景（证据链）

P1（public `ExecuteLive` compile 前拒 live）已验收：`validateExecuteLiveRequestMode` 在 handler（strategy_execution_handlers.go:97-99）拒绝 live/未知 mode，T1/T2/T3/T5b/T5 + P1-P3 + R1-P1 对抗全有效。因此 **`injectServerSideAccountTruth` 与 `dispatchVMLive` 的 live 分支成为不可达死代码**（`executeVMLive`/`executePythonVMLive` 仅被 public `ExecuteLive` handler 调用，live 请求在 compile 前即被拒；生产调度路径走 `VMLiveSession.Start`→自身 `dispatch`，不经 `dispatchVMLive`）。P1 prompt 边界 #3 明确："清理是 Phase 1b 的独立任务。理由：先证明拒绝生效，再删旁路"——拒绝已生效并验收，1b 可开工。

**⚠️ scope 修正（Claude 复审 Phase 1 时发现，覆盖 P1 边界 #3 字面）**：P1 边界 #3 称"`accountTradeAllowedLookup`/`accountIsInvestorLookup` 等变死代码"——**仅适用于 `injectServerSideAccountTruth` 函数体内**。5 个 lookup 字段本身与装配（strategy_execution_handler.go:109-139 字段声明 / :213-243 `Set*Lookup` 装配）以及 `resolveBrokerCompanyErr` 是**生产调度路径 `buildLiveContext`（live_context_build.go:48-108）的 active 依赖**——**禁止删除**（删 = 实盘调度全停）。本任务只删两个死代码目标：① `injectServerSideAccountTruth` 整函数；② `dispatchVMLive` 的 live 分支。

### 🔴 绝对边界（违反 = 直接判失败）

1. **只改** `vm_live_dispatch.go` + `vm_trade_context6_round5_test.go`（+ 因删除编译失败的任何引用点，需在回填中披露）。**禁止删/改**：5 个 lookup 字段与装配（strategy_execution_handler.go:109-139/213-243）、`resolveBrokerCompanyErr`、`buildLiveContext`（live_context_build.go）、`backfillContextStrings`（live_context.go）、`validateExecuteLiveRequestMode`/`validateBarContextWithMode`/`validateLiveFinancialFields`（vm_live_validators.go）、`VMLiveSession`（vm_live_session.go）、`execute_live_mode_reject_test.go`、`live_indicator_freeze_test.go`。
2. `dispatchVMLive` 函数**保留**（paper 路径仍使用）；`validateBarContextWithMode` 的 live 金融字段校验**保留**（VMLiveSession 调度路径仍用，防御性）。
3. 不碰 proto / DB schema / 部署 / 其他功能块；`modePaper` 与 `modeLive` 常量保留。

### 施工步骤（目标 + 精确坐标）

- **S1**：删除 `injectServerSideAccountTruth` 整函数（vm_live_dispatch.go:161-221，含函数体与 :157-160 的文档注释）。
- **S2**：删除 `dispatchVMLive` 的 live 分支（vm_live_dispatch.go:94-106 整个 `if bctx.Mode == "live" { ... }` 块）；同步修正 :129-131 引用该函数的注释（"server-side lookups (injectServerSideAccountTruth)" → 删除该短语，保留 VM-TRADE-CONTEXT-5/VM-API-TRUTH-3 的注入说明）。
- **S3**：删除因此失效的测试（测死代码）：
  - `TestDispatchVMLive_RejectsLiveModeWithoutAccountID`（vm_trade_context6_round5_test.go:149-165 区域，含 "dispatchVMLive should reject live mode without account_id" 断言）——测被删分支；
  - `TestDispatchVMLive_LiveModeWithAccountIDOverridesClientIdentity`（vm_trade_context6_round5_test.go:167-232）——测被删函数；
  - 全仓 grep `injectServerSideAccountTruth` 清零（测试与生产均不得残留引用）。

### 测试与对抗证明（缺一即未完成）

- **T1**：删除完成后 `go test ./internal/connect/strategy -count=1` 全绿（含 `execute_live_mode_reject_test.go` 的 T1/T2/T3/T4/T5b/T5 —— 拒绝逻辑在 handler/validator，**不得**依赖旁路；若变红 = 测试意外依赖死代码，停下报告，不得自行改测试绕过）。
- **P1**：`rtk grep -rn "injectServerSideAccountTruth" backend/ --include="*.go"` 输出为空 + `go build ./...` 通过（删除后编译依赖自然验证）。
- **P2**：`go test ./internal/connect/strategy -run 'TestExecuteLive|TestVMLiveSession_LiveModeStillWorks' -count=1` 全绿——证明 P1 拒绝逻辑与旁路删除无关。
- **P3（红队，书面）**：给出调用链证据证明 `dispatchVMLive` live 分支确实不可达（public `ExecuteLive` → `validateExecuteLiveRequestMode` compile 前拒 live → `executeVMLive` 仅此入口调用；生产调度路径 `VMLiveSession.Start` → 自身 `dispatch`（vm_live_session.go:175）不经 `dispatchVMLive`）。任何"应该不可达"的无证据断言不算。
- 每项记录命令、RED/GREEN 输出摘要。**nil panic、另一条错误、"任意 error" 均不算证据。**

### 红队自审（施工后切换怀疑者视角，逐条书面回答）

1. 删除 live 分支后，任何路径能让 `Mode:"live"` 请求到达 `dispatchVMLive`？给出完整调用链。
2. 5 个 lookup 字段为什么不能删？引用 `buildLiveContext` 的依赖位置。
3. paper 模式下 `dispatchVMLive` 的行为是否改变？（不应改变——只删 live 分支。）
4. `validateBarContextWithMode` 的 live 金融校验还必要吗？（保留——VMLiveSession 调度路径使用。）
5. 本改动让哪些既有测试失败？失败是"测死代码"还是"我改坏了"？

### 验收门禁（逐条贴真实输出）

`gofmt -l` 本次改动文件为空；`go build ./...`；`go vet ./internal/connect/strategy/...`；`go test ./internal/connect/strategy -count=1`；`go test -race ./internal/connect/strategy -run 'TestVMLiveSession_LiveModeStillWorks|TestExecuteLive' -count=1` **连跑 3 次**；`go run ./tools/check-file-lines --strict` 必须 `0 errors, 0 warnings`（info 需披露）；`git diff --check`。（buf lint 不涉及——proto 零改动。）

### 回填与收尾

registry 本条回填真实实现 + REUSE/NEW 结论 + T1/P1/P2/P3 结果 + 红队自审 5 问答；`handover-audit-plan.md` 追加一行。**状态填 `⚠️待Claude复审`，不得自标 ✅done。** 后续排队任务（**禁止并行施工，本任务验收后再派**）：D-CODE-HYGIENE-001 逐文件 manifest 补齐（P0 验收收口，120 新文件缺 H2 要求的 manifest）。

> **勿部署、勿 push、停手等 Claude 复审。禁止 `--no-verify`。收工只显式 `git add` 本任务涉及的文件（预期仅 `vm_live_dispatch.go` + `vm_trade_context6_round5_test.go` + 两个文档），禁止 `git add -A`／`git add .`（本仓多 agent 并发）。**

<!-- D-VM-LIVE-001-P1B:END -->
