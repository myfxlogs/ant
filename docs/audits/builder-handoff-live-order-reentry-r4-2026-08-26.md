# Builder Handoff — LIVE-ORDER-REENTRY-1 R4 复审阻断解决（P0 实盘重复开仓）

> 日期：2026-08-26 ｜ 审计方：Devin CLI（项目第一负责人） ｜ 施工方：Devin IDE
> 设计 SSOT：`docs/spec/live-order-reentry-r4-spec.md`
> 根因与证据链：`docs/audits/tech-debt-registry.md` 条目 `LIVE-ORDER-REENTRY-1`（:70）、`LIVE-ORDER-REENTRY-1-R4-REVIEW`（:71）

## 0. 立项背景

**触发**：LIVE-ORDER-REENTRY-1 经 4 轮复审，R4（2026-08-21）仍有 3 个阻断未解决。

**证据链**：
- R4 复审记录（registry :71）：目标测试/race/build/file-lines 均通过，但 3 个阻断未解
- 阻断一：`mutation_coordinator.go:258-267` 对 open mutation 也启动 recovery，违反"open mutation fail-closed"边界
- 阻断二：pipeline 测试用 `publishOrderUpdate` helper 直接注入 label，绕过真实 adapter `parseMt4OrderUpdate` label 接线
- 阻断三：6 处 `time.Sleep(300ms)` 违反确定性测试纪律

**设计 SSOT 声明**：`docs/spec/live-order-reentry-r4-spec.md`（D1-D3 设计决策）

**约束与目标**：
- 解决 3 个 R4 阻断，使 LIVE-ORDER-REENTRY-1 可验收
- 不改 `trade_barrier.go` 状态机设计（已通过 3 轮复审）
- 不改 `LIVE-ORDER-REENTRY-1-BROKER-REJECT` 的 sentinel 检查顺序（已 ✅done + 部署验证）

**边界/不做**：
- 不改已通过复审的 round 1-3 修复
- 不部署（D-COMMIT-SCOPE-001 部署闸仍有效）
- 不 commit/push/deploy
- 禁 `--no-verify`

---

## S1 — 阻断一：open mutation recovery 边界（D1）

**目标**：open mutation outcome unknown 时不启动 recovery，直接 fail-closed。

**精确坐标**：
- 文件：`backend/internal/connect/strategy/mutation_coordinator.go:258-267`
- 当前代码（:258-267）：
```go
// ④-②: For known-ticket mutations, start background reconciliation.
// Open mutations (ticket=0 at spec creation, but effectiveTicket is
// now known from RPC) also get recovery since we have the ticket.
verify := spec.verifyReadAfterWrite
if verify == nil {
    verify = verifyTicketPresent(effectiveTicket)
}
go s.recoverFromOutcomeUnknown(cfg, activeSess, barrier, effectiveTicket, spec.action, verify, conf)
```
- 落点：recovery 启动条件加 `spec.action != ActionOpen`（或等效的 action 判断）
```go
// ④-②: For known-ticket mutations, start background reconciliation.
// Open mutations are excluded: read-after-write cannot prove the new order
// has been processed by the broker (processing latency may cause the order
// to not yet appear in OpenedOrders). Open mutation outcome unknown = fail-closed.
if spec.action != ActionOpen {
    verify := spec.verifyReadAfterWrite
    if verify == nil {
        verify = verifyTicketPresent(effectiveTicket)
    }
    go s.recoverFromOutcomeUnknown(cfg, activeSess, barrier, effectiveTicket, spec.action, verify, conf)
}
```

**注意**：确认 `ActionOpen` 的确切常量名——检查 `mutation_coordinator.go` 中 `spec.action` 的类型和可能的值。

**对抗证明**：
- 构造 open mutation outcome unknown 场景 → 验证 `recoverFromOutcomeUnknown` 不被调用（可用 mock 计数或 channel）
- 将条件改回"open 也启动 recovery" → `recoverFromOutcomeUnknown` 被调用 → RED
- 恢复 → GREEN

**同样检查 :195-201 的 outcome_unknown 路径**：
- `mutation_coordinator.go:195-201` 也有 `go s.recoverFromOutcomeUnknown(...)` 调用
- 确认该路径是否也需要加 `spec.action != ActionOpen` 条件

---

## S2 — 阻断二：adapter pipeline 测试改用真实 label 接线（D2）

**目标**：pipeline 测试通过真实 MT4/MT5 adapter 的 `parseMt4OrderUpdate`/`parseMt5OrderUpdate` 生成 label，而非直接注入预定义 label。

**精确坐标**：
- 当前测试：`backend/internal/connect/strategy/mutation_coordinator_test.go:156`（`func publishOrderUpdate`）直接注入 label
- 真实 adapter：
  - `backend/internal/mdgateway/adapter/mt4/order_stream.go:121`（`func parseMt4OrderUpdate`）
  - `backend/internal/mdgateway/adapter/mt5/order_stream.go:121`（`func parseMt5OrderUpdate`）
- 测试位置：`mutation_coordinator_test.go:1203` 附近的 FullBrokerPath 测试

**落点**：
1. 新增测试 `TestLIVE_ORDER_REENTRY_1_AdapterLabelPipeline_MT4`：
   - 构造 MT4 `OnOrderUpdate` protobuf 消息（`pb.OrderUpdateSummary`）
   - 调用真实 `parseMt4OrderUpdate` 生成 `mdtick.OrderUpdate`
   - 通过真实 broker publish → coordinator 接收
   - 验证 coordinator 正确处理 label
2. 突变 `parseMt4OrderUpdate` 的 label 映射（如把 `open` 映射为 `close`）→ 测试 RED
3. 恢复 → GREEN

**关键**：测试必须通过 `parseMt4OrderUpdate` 的真实 label 生成逻辑，不能绕过它直接注入 label。

---

## S3 — 阻断三：time.Sleep → channel 同步（D3）

**目标**：6 处 `time.Sleep` 改为 channel/condition variable 同步。

**精确坐标**：
- `backend/internal/connect/strategy/mutation_coordinator_test.go`:
  - `:1226` — `time.Sleep(50 * time.Millisecond) // wait for barrier to enter submitting`
  - `:1313` — `time.Sleep(300 * time.Millisecond)`
  - `:1361` — `time.Sleep(300 * time.Millisecond)`
  - `:1407` — `time.Sleep(300 * time.Millisecond)`
  - `:1454` — `time.Sleep(300 * time.Millisecond)`
  - `:1503` — `time.Sleep(300 * time.Millisecond)`

**落点**：
- 每处 `time.Sleep` 改为确定性同步：
  - barrier 状态变化时通过 channel 通知测试（如 `barrier.StateChanged() <-chan BarrierState`）
  - 或使用 `barrier.WaitState(state)` helper（如不存在则新增到 `trade_barrier.go`）
  - 或使用 `sync.Cond` + `barrier.State()` 轮询（最后手段）

**对抗证明**：
- 将 channel 同步改回 `time.Sleep(300ms)` → 在 CI 慢机器上测试 flaky → 证明 channel 同步是必要的
- `grep "time.Sleep" mutation_coordinator_test.go` 返回 0 行（注释行除外）

---

## 验收标准

1. **对抗证明 3 项**（S1/S2/S3），每项 RED→restore→GREEN
2. **门禁全绿**：
   - `go build ./...`
   - `go test ./internal/connect/strategy -count=1`
   - `go test -race ./internal/connect/strategy -count=1` ×3
   - `go vet ./...`
   - `go run ./tools/check-file-lines --strict`（0 errors）
   - `git diff --check`
3. **不变量 I1-I8 全部保持**（registry :70）
4. **`grep "time.Sleep" mutation_coordinator_test.go`** 返回 0 行（注释行除外）
5. **pipeline 测试**通过真实 `parseMt4OrderUpdate` label 接线，突变 label 映射 → RED

## 红队自审（施工方完工前必答）

1. `ActionOpen` 的确切常量名是什么？`spec.action` 的类型是什么？
2. `mutation_coordinator.go:195-201` 的 outcome_unknown 路径是否也需要加 open mutation 条件？
3. open mutation fail-closed 后，barrier 状态是什么？`outcomeUnknown` 还是 `idle`？circuit open 是否正确？
4. pipeline 测试是否覆盖 MT5 的 `parseMt5OrderUpdate`？
5. channel 同步是否有 goroutine 泄漏？（测试结束后 channel 是否关闭？）
6. `barrier.WaitState` helper 如新增，是否会影响生产代码？（应只用于测试）

## 回填纪律

1. registry `LIVE-ORDER-REENTRY-1-R4-REVIEW`（:71）：状态改为 `🟦open（施工完成，待独立复审）` + 真实实现 + 对抗证明结果
2. `handover-audit-plan.md` 变更日志加一行
3. **不自行宣告完成**——停手等 Devin CLI 复审

## 范围约束

One task = one scope：只动 3 个 R4 阻断——`backend/internal/connect/strategy/mutation_coordinator.go`（S1）+ `mutation_coordinator_test.go`（S2/S3）+ 可能的 `trade_barrier.go`（S3 helper）。不顺手重构、不改 round 1-3 修复、不动 broker/handler 业务语义。

## 固定尾部

**勿部署，停手等 Devin CLI 复审。** 禁 `--no-verify`。禁 commit/push/deploy。只 add 本任务文件，禁 `git add -A`。
