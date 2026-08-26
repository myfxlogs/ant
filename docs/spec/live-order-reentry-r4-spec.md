# Spec：LIVE-ORDER-REENTRY-1 R4 复审阻断解决（P0 实盘重复开仓）

> **状态**：🟦open（R4 复审阻断待解，待施工 + 验收）
> **registry 条目**：`LIVE-ORDER-REENTRY-1`（`docs/audits/tech-debt-registry.md:70`）、`LIVE-ORDER-REENTRY-1-R4-REVIEW`（:71）
> **优先级**：P0（真金白银风险——实盘重复开仓）

## 1. 问题陈述

`live_dispatch.go` 的 `submitOrder` 用 goroutine fire-and-forget 下单，违反 MT4 EA `OrderSend` 单线程语义。VM 在 broker 确认前继续执行 → `OrdersTotal()==0` → 下一 tick/bar 再次开仓 → 数秒内连续生成多个同方向订单。

**已完成的修复**（round 1-3 + BROKER-REJECT）：
- `trade_barrier.go` — session-scoped execution barrier 状态机
- `mutation_coordinator.go` — 5 路径共享协议（同步替换 fire-and-forget）
- `position_cache.go` — freshness 拆分
- `mutation_outcome.go` — typed MutationError + ClassifyMutationError
- `broker_types.go` — PositionsCapturedAt/Source
- `LIVE-ORDER-REENTRY-1-BROKER-REJECT` — broker 应用层拒绝正确分类为 `deterministic_rejected`（✅done，已部署验证）

**R4 复审阻断（2026-08-21，未解决）**：

### 阻断一：open mutation recovery 边界冲突

**位置**：`mutation_coordinator.go:260-267`

**问题**：对 broker RPC 成功但确认 outcome unknown 的 **open** mutation 也启动 `recoverFromOutcomeUnknown`，与"open mutation fail-closed"边界冲突。代码注释说"Open mutations also get recovery since we have the ticket"，但 R4 审计认为单次 `OpenedOrders` 不能证明新订单已执行——broker 可能还没处理完，read-after-write 看到的是旧状态。

**不变量冲突**：I5 要求 transport timeout = outcome unknown → fail-closed 不重下。对 open mutation 启动 recovery 意味着系统会尝试确认订单是否成功，如果 recovery 误判成功，barrier 释放后下一事件可能重复下单。

### 阻断二：adapter pipeline 测试覆盖不足

**问题**：R4 pipeline 测试直接调用导出的 label 函数和测试 `publishOrderUpdate` helper，隔离突变 MT4 `parseMt4OrderUpdate` 的真实 label 接线为错误标签后，R4 pipeline 测试仍全 GREEN。测试没有覆盖真实 adapter → broker → coordinator 的完整链路。

### 阻断三：time.Sleep 违反确定性测试纪律

**位置**：`mutation_coordinator_test.go:1226/1313/1361/1407/1454/1503`

**问题**：6 处 `time.Sleep(300ms)` 用于等待 barrier 状态变化，违反确定性测试纪律。应改用 channel 同步或 condition variable。

## 2. 设计决策

### D1：open mutation recovery 边界——区分 open vs close/modify/delete

**决策**：open mutation outcome unknown 时**不启动 recovery**，直接 fail-closed（barrier 锁定 + circuit open）。close/modify/delete mutation outcome unknown 时**启动 recovery**（因为这些操作有明确的 ticket，read-after-write 可以确认状态）。

**理由**：
- open mutation 的 ticket 是 RPC 返回的，但订单可能还在 broker 处理队列中，read-after-write 看到的 `OpenedOrders` 可能不包含新订单（处理延迟）→ 误判"订单不存在"→ barrier 释放 → 重复下单
- close/modify/delete 的 ticket 是已知的（订单已存在），read-after-write 可以确认订单状态是否已变更
- fail-closed 对 open mutation 是安全的：即使订单实际已执行，barrier 锁定只是阻止后续下单，不会导致资金损失；而误释放 barrier 可能导致重复下单

**实现**：`mutation_coordinator.go:260-267` 的 recovery 启动条件加 `spec.action != ActionOpen`（或等效的 action 判断）。

**对抗证明**：
- 构造 open mutation outcome unknown 场景 → 验证不启动 recovery（`recoverFromOutcomeUnknown` 不被调用）
- 将条件改回"open 也启动 recovery" → 验证 recovery 被调用 → RED
- 恢复后 GREEN

### D2：adapter pipeline 测试——改用真实 adapter label 接线

**决策**：pipeline 测试应通过真实 MT4/MT5 adapter 的 `parseMt4OrderUpdate`/`parseMt5OrderUpdate` 生成 label，而非直接调用 `publishOrderUpdate` helper 注入预定义 label。

**理由**：真实链路的 bug 点在 adapter 的 label 生成逻辑（`parseMt4OrderUpdate` 把 broker update 映射为 `open`/`close`/`modify` 等 label）。如果测试绕过这一步直接注入 label，就无法发现 label 生成逻辑的 bug。

**实现**：
- 新增测试：构造 MT4 `OnOrderUpdate` protobuf 消息 → 调用真实 `parseMt4OrderUpdate` → 验证生成的 label → 通过真实 broker publish → coordinator 接收
- 突变 `parseMt4OrderUpdate` 的 label 映射（如把 `open` 映射为 `close`）→ 验证 pipeline 测试 RED
- 恢复后 GREEN

### D3：time.Sleep → channel 同步

**决策**：6 处 `time.Sleep` 改为 channel/condition variable 同步。

**实现**：
- barrier 状态变化时通过 channel 通知测试
- 或使用 `barrier.WaitState(state)` helper（如不存在则新增）
- 或使用 `sync.Cond` + `barrier.State()` 轮询（最后手段，不如 channel）

**对抗证明**：将 channel 同步改回 `time.Sleep(300ms)` → 在慢机器上测试 flaky → 证明 channel 同步是必要的

## 3. 验收标准

1. **阻断一**：open mutation outcome unknown 不启动 recovery，对抗证明 RED→GREEN
2. **阻断二**：pipeline 测试通过真实 adapter label 接线，突变 label 映射 → RED
3. **阻断三**：6 处 time.Sleep 全部替换为 channel 同步，`grep "time.Sleep" mutation_coordinator_test.go` 返回 0 行（注释行除外）
4. **门禁全绿**：`go build ./...` / `go test ./internal/connect/strategy -count=1` / `go test -race ./internal/connect/strategy -count=1` / `go vet ./...` / `check-file-lines --strict`（0 errors）/ `git diff --check`
5. **不变量 I1-I8 全部保持**

## 4. 不做

- 不改 `trade_barrier.go` 状态机设计（已通过 3 轮复审）
- 不改 `LIVE-ORDER-REENTRY-1-BROKER-REJECT` 的 sentinel 检查顺序（已 ✅done + 部署验证）
- 不部署（D-COMMIT-SCOPE-001 部署闸仍有效）
- 不扩大 scope 到其他 ID
