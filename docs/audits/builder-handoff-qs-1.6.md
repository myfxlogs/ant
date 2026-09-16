# 施工提示词：QS-1.6 read-after-write 确认走 barrier 状态机（ConfirmByAuthoritativeRead）

> **[角色:施工]** — 你是本任务的施工方 agent（见 `.devin/rules/dual-terminal-roles.md`）。
> 严格按 S1–S4 施工，不做决策；超出提示词范围 = 违规，停下转 `[转交决策]`。
> 完成后先过"施工完成自审"（D-012）修至全绿再交回；自报末行带 `[施工完成:QS-1.6] @<commit-hash>`（D-014）。
> 勿部署、勿 push、禁 `--no-verify`；只显式 add 本任务文件；commit 用 `ANT_ROLE=builder git commit` 前缀（D-015 豁免 STATE.md 必更门禁）；**不更新任何交接层文件**（STATE/registry/handover/decisions/spec/adr 归决策方独占，根因答案写进自报）。
>
> **最终决策：Devin CLI（[角色:决策终] 激活）**

## 立项背景（触发 + 证据链）

- registry `QS-1.6`；spec `docs/spec/vm-pipeline-quality-stability-improvement-plan.md` v2 §3.2（设计 SSOT）。
- `backend/internal/connect/strategy/mutation_coordinator.go:330-334`：read-after-write `verify(orders)==true` 但 `NotifyConfirmationEvent` 未迁移 barrier 时，代码直接 `return barrierConfirmed`，barrier 实际停在 `acceptedUnconfirmed` 被 `Release()` 兜底——绕过状态机单一真相源。
- 已否决方案：`Reconcile(true)`——它仅在 `outcomeUnknown` 迁移（`trade_barrier.go:333`），此场景是 no-op。

## 设计 SSOT 声明

- 设计文档：spec v2 §3.2。相关契约：`AGENTS.md` §1 fail-closed；决策 D-009/D-012/D-014。

## 约束与目标

- barrier 状态迁移必须由 `TradeBarrier` 方法完成；coordinator 不得返回 barrier 未达到的状态。
- `ConfirmByAuthoritativeRead` 幂等、持 `b.mu`、`Broadcast`；允许 `{barrierSubmitting, barrierAcceptedUnconfirmed, barrierOutcomeUnknown}`→`barrierConfirmed`；已 confirmed 幂等返回 true；`barrierDeterministicRejected` 返回 false 不覆盖（拒绝是 broker 权威）。
- 不改 push 确认路径、recovery 流程、`Reconcile`/`Notify*`/`Release` 既有语义。

## 边界 / 不做

- 不动 `Reconcile`/`NotifyConfirmationEvent`/`NotifyOutcomeUnknown`/`Release`/`cacheEvent`。
- 不做 open outcomeUnknown 恢复（QS-1.7-INV 另立）。
- 不更新交接层文件；不新增 registry 条目。

## 施工指令

### S1 — 根因确认（只读，答案写进自报）

- **目标**：回答"`mutation_coordinator.go:325` `NotifyConfirmationEvent(ticket, magic, string(action))` 在 `verify(orders)==true` 后为何可能未把 barrier 迁到 confirmed"。
- **坐标**：`trade_barrier.go:218-242`（NotifyConfirmationEvent）、`:185-206`（NotifyBrokerAccepted）、`mutation_coordinator.go:286-335`。
- **落点**：自报给出确切根因（候选：`action==open` 时 `ticket==0` 被 `:219` 早退；`isUpdateTypeCompatible` 不兼容；barrier 仍在 `submitting` 事件仅被缓存）。附代码行证据；根因必须覆盖"verify 成功但状态非 confirmed"的全部路径。
- **验证**：根因与代码分支一一对应，无"可能是"措辞。

### S2 — 新增 `TradeBarrier.ConfirmByAuthoritativeRead`

- **坐标**：`backend/internal/connect/strategy/trade_barrier.go`，加在 `Reconcile`（约 :330-342）之后。
- **落点**（精确形态）：
  ```go
  // ConfirmByAuthoritativeRead transitions the barrier to confirmed based on an
  // authoritative read-after-write result. Used by waitForConfirmation when the
  // OpenedOrders query verified the mutation but no matching push event migrated
  // the state machine. Idempotent; does not override a deterministic rejection.
  // Returns true iff the barrier ends in confirmed.
  func (b *TradeBarrier) ConfirmByAuthoritativeRead() bool {
      b.mu.Lock()
      defer b.mu.Unlock()
      switch b.state {
      case barrierConfirmed:
          return true
      case barrierSubmitting, barrierAcceptedUnconfirmed, barrierOutcomeUnknown:
          b.state = barrierConfirmed
          b.cond.Broadcast()
          return true
      default:
          return false
      }
  }
  ```
- **验证**：见 S4a 单测。

### S3 — `waitForConfirmation` 调用状态机迁移

- **坐标**：`backend/internal/connect/strategy/mutation_coordinator.go:330-334`（`if state == barrierConfirmed { return barrierConfirmed }` 与 `// Force-confirm based on the authoritative read. return barrierConfirmed` 两行）。
- **落点**：
  ```go
  if state == barrierConfirmed {
      return barrierConfirmed
  }
  // Authoritative read verified the mutation — transition via the state
  // machine (QS-1.6) instead of returning a state the barrier never reached.
  barrier.ConfirmByAuthoritativeRead()
  return barrierConfirmed
  ```
- **验证**：S4b 测试断言 `Release` 前 `barrier.State()==barrierConfirmed`。

### S4 — 测试（先红后绿 + mutation）

- **坐标**：`backend/internal/connect/strategy/trade_barrier_test.go`（barrier 单测，复用 `WaitState`/`State` 模式）+ `backend/internal/connect/strategy/mutation_coordinator_test.go`（约 :1237/:1266 已有 `WaitState(waitCtx, barrierSubmitting)` / `State()==barrierAcceptedUnconfirmed` 模式）。
- **落点**：
  a. barrier 单测：`acceptedUnconfirmed`/`submitting`/`outcomeUnknown` → `ConfirmByAuthoritativeRead()` 返回 true 且 `State()==confirmed`；`deterministicRejected` → 返回 false 且状态不变。
  b. coordinator 测试：构造 verify=true 但 push 事件不迁移的场景（以 S1 根因为准，如 ticket 匹配但 updateType 不兼容 / action==open 时 ticket=0），断言 `waitForConfirmation` 返回后 `barrier.State()==barrierConfirmed`。
- **验证**：S4b 在 S3 实施前 RED（`State()==acceptedUnconfirmed`），实施后 GREEN。

## 对抗证明（缺一即未完成）

- mutation：删 `barrier.ConfirmByAuthoritativeRead()` 调用（恢复裸 `return barrierConfirmed`）→ S4b RED（State 非 confirmed）；restore → GREEN。
- mutation：S2 的 switch 中删 `barrierAcceptedUnconfirmed` → S4a 对应用例 RED。

## 验收标准

- [ ] `cd backend && go build ./...` 通过
- [ ] `cd backend && go test -count=1 ./internal/connect/strategy/...` 全过
- [ ] `go test -race -count=3 ./internal/connect/strategy/...` 通过
- [ ] `go run ./tools/check-file-lines --strict` 本任务文件零警告
- [ ] `gofmt -l` / `go vet ./internal/connect/strategy/...` 零输出
- [ ] S1 根因答案 + 对抗证明 RED→restore→GREEN（附命令与输出）
- [ ] diff 无交接层文件、无范围外改动

## 施工完成自审（强制，D-012）

交付自报前必须完成并随报提交：
- [ ] 逐项重跑上方验收标准并贴真实输出
- [ ] 红队自审 diff 三问：更简等价方案 / 边界·nil·并发 / 逆向依赖或重复基础设施
- [ ] 自审发现的缺陷已修复至全绿（自报列出发现项+修复项）
- [ ] 无自审记录 = 复审直接退回

## 交付格式

自报：改动文件清单 + 每条验收项证据（命令+关键输出）+ 施工完成自审记录 + S1 根因答案 + 遗留疑问。
**自报最后一行固定为** `[施工完成:QS-1.6] @<commit-hash>`（D-014：无此行 = 未交付，复审不启动）。
**停手等 Devin CLI 复审；不达标将由决策方开缺陷清单退回。**

[施工完成:QS-1.6] @<本任务最终commit>
