# 逐项深度审计报告

## 审计维度
1. 架构合理性
2. 最优性
3. 第一性原则
4. 代码整洁
5. 技术债

---

## 1. 架构合理性

### A1-1 ✅ 分层边界清晰

依赖方向正确：
- `connect/` (handler) → `service/` → `repository/` → `model/`
- `connect/` → `mthub/` (业务 facade)
- `service/` 不导入 `connect/`
- `repository/` 不导入 `service/`

无循环依赖。`mthub/` 作为业务 facade 隔离了 `connect/` 与 `mdgateway/adapter/` 的直接耦合。

### A1-2 ✅ ConnectRPC + SSE 架构合规

无 REST endpoint（除 healthz/readyz），无 WebSocket，无 JSON 持久化。所有跨进程通信走 ConnectRPC + SSE。

### A1-3 🟡 MEDIUM — connect 层包含业务逻辑

**文件**: `backend/internal/connect/user/share_handler.go:69-185`

`GetSharedPerformance` handler 包含大量业务逻辑：Sharpe ratio 计算、交易统计、持仓快照管理。这些应提取到 `service/` 层。

**影响**: handler 职责过重，难以测试和复用。如果另一个 handler 需要相同的统计逻辑，会导致代码重复。

**建议**: 提取 `SharePerformanceService`，handler 只做请求/响应映射。

### A1-4 🟡 MEDIUM — connect/strategy 包过大

`backend/internal/connect/strategy/` 包含 20+ 文件，涵盖回测、实盘执行、实验、CRUD、调度。虽然每个文件在 300 行以内，但包内耦合度高。

**建议**: 考虑按功能域拆分为 `strategy/backtest/`、`strategy/live/`、`strategy/experiment/`、`strategy/crud/` 子包。

### A1-5 ✅ Push-first 架构

SSE 流 + PG LISTEN 推送是默认模式。无轮询、无 cron、无 `time.Ticker`（除 VM 执行上下文超时）。

---

## 2. 最优性

### O2-1 🟡 MEDIUM — 硬编码 balance fallback 10000

**文件**: `backend/internal/connect/strategy/account_provider.go:118, 180`

当 `PositionCache` 和 `balanceCache` 都没有数据时，balance 默认为 10000。这个值会影响 risk gate 的决策（max lot size、margin check）。

**风险**: 如果用户账户实际余额远小于 10000（如 500），risk gate 会允许超出实际承受能力的仓位。反之如果远大于 10000（如 100万），gate 会过度限制。

**建议**: 当无真实 balance 数据时，应 fail-closed（返回 nil → gate 阻止交易），而非使用虚假 balance。已有 fail-closed 逻辑（`exec == nil` 时返回 nil），但 `buildStateFromSnapshot` 和 `buildStateFromOrders` 中的 10000 fallback 绕过了它。

### O2-2 🟢 LOW — float64 用于统计计算

**文件**: `backend/internal/connect/system/analytics_compute.go`、`backend/internal/connect/user/share_handler.go`

Sharpe ratio、Sortino、VaR 等统计指标使用 float64 计算。虽然项目规则禁止"price calculations"使用 float64，但统计指标是近似值，float64 精度足够。这是合理的工程权衡。

### O2-3 🟢 LOW — per-SSE-stream PG LISTEN 消耗连接池

**已知问题**（来自 memory）: 每个 SSE stream 调用 `pgListen.Listen` → 消耗一个 pool 连接。大规模时应使用共享 listener + fan-out 模式。

**当前状态**: `DBMaxConns` 已从默认 4 调整为 25，暂时缓解。长期需要架构改进。

### O2-4 ✅ VM 执行沙箱完善

- `MaxTicks: 10M` — 防无限循环
- `MaxCallDepth: 256` — 防栈溢出
- `MaxStackDepth: 4096` — 防 OOM
- `MaxSourceSize: 500KB` — 防资源耗尽
- `safeRun()` + `defer recover()` — 防 panic

---

## 3. 第一性原则

### F3-1 🟡 MEDIUM — OMS 事件广播修复后的架构一致性

W1-2 修复了 `OrderEventBroker.PublishEvent` 的跨用户广播问题，改为按 `userID` 过滤。这是正确的安全修复，但从第一性原则看，`PublishEvent` 的签名从 `(_ string, ev)` 改为 `(userID string, ev)` 引入了一个隐含约束：调用者必须传入正确的 userID。

**当前状态**: `oms_writer.go` 从 context 提取 userID 并传递。这是正确的，但如果未来有新的调用者忘记传 userID（传空字符串），事件将不会被投递到任何用户——这是 fail-closed 的安全行为，可接受。

### F3-2 ✅ Risk gate 单一 chokepoint

`mthub.MtHubService` 的 `PlaceOrder`/`CloseOrder`/`ModifyOrder` 都经过 `guard` + `gate` 的双重检查。这是从第一性原则出发的正确设计——所有订单路径必须经过同一安全检查点。

### F3-3 ✅ Token 生命周期管理

前端 token 设计从第一性原则出发：
- accessToken 在内存中（不持久化）→ 刷新后旧 token 立即失效
- refreshToken 在 httpOnly cookie 中 → JS 不可访问
- 主动刷新（tokenLifecycle）+ 被动刷新（401 interceptor）→ 双保险
- `SameSite=Strict` → CSRF 防护

### F3-4 🟢 LOW — 策略执行 context.WithoutCancel

`live_dispatch.go` 中 `dispatchCloseAll` 和 `dispatchCloseOrder` 使用 `context.WithoutCancel(ctx)` 来分离订单执行与请求生命周期。这是正确的——订单提交不应因客户端断开而取消。但 `dispatchMarketOrder` 和 `dispatchPendingOrder` 没有使用 `WithoutCancel`，存在不一致。

**建议**: 统一所有订单提交路径使用 `WithoutCancel`，或明确文档化为何市价单需要跟随请求生命周期。

---

## 4. 代码整洁

### C4-1 ✅ 无 TODO/FIXME/HACK 标记

代码中无遗留的 TODO/FIXME/HACK 注释。所有匹配项均为函数名（`UpdateToDone`）或 LLM prompt 文本。

### C4-2 ✅ 文件大小合规

`check-file-lines --strict` 通过：0 errors, 29 warnings, 74 info。无文件超过 450 行硬性红线。

### C4-3 🟢 LOW — 重复的 Sharpe ratio 计算

`share_handler.go:116-150` 和 `share_og_image.go:112-130` 都有 Sharpe ratio 计算逻辑，实现方式不同（一个用 `decimal.Decimal`，一个用 `float64`）。

**建议**: 提取到 `service/analytics/` 包，统一实现。

### C4-4 ✅ 命名一致性

包命名、文件命名、函数命名风格一致。`Set*` 注入模式统一使用。

### C4-5 ✅ 错误处理一致

所有 handler 使用 `connect.NewError()` 包装错误，错误码语义正确（`CodeUnauthenticated`、`CodeInvalidArgument`、`CodeInternal`）。

---

## 5. 技术债

### T5-1 🟡 MEDIUM — float64 在 MT proto 边界

`mthub/types.go:131` 的 `ProfitPercent float64` 和 `broker_types.go:100` 的 `Volume float64` 是 MT API proto 要求的 float64。这是外部约束，非技术债。

但 `service_orders_modify.go:61-62` 将 `decimal.Decimal` 转为 `float64` 仅用于日志：
```go
slFloat, _ := sl.Float64()
tpFloat, _ := tp.Float64()
```
应直接用 `zap.String("sl", sl.String())` 避免不必要的精度损失。

### T5-2 🟢 LOW — 已知缺口（来自 go-native-strategy-pipeline.md §8）

以下为已记录的已知缺口，非新增技术债：
- barsDropped notification (P2)
- per-bar OpenedOrders query 应使用 PositionSnapshotBroker subscription (P2)
- AccountService 仍使用 float64 (P2)
- MTAccountInfo 仍使用 float64 (P2)
- iCustom custom indicators (P3)
- Bytecode cache persistence to DB (P3)
- Live consistency verification (VM signals vs backtest results) — 未验证

### T5-3 🟢 LOW — NATS 无认证（W3-2 已记录）

Docker 网络内无 NATS 认证。纵深防御缺口，非直接漏洞。

### T5-4 🟢 LOW — migration 未包装事务（W3-1 已记录）

多语句 migration 可能部分应用。建议引入 `golang-migrate` 或在 entrypoint 中包装 `BEGIN/COMMIT`。

### T5-5 ✅ 无 deprecated/legacy 代码标记

代码中无 `// Deprecated:`、`// legacy`、`// workaround` 标记。Python 策略运行时已完全移除。

---

## 修复优先级

| ID | 严重度 | 描述 | 建议操作 |
|---|---|---|---|
| O2-1 | 🟡 MEDIUM | 硬编码 balance 10000 fallback 可能导致 risk gate 误判 | 改为 fail-closed |
| A1-3 | 🟡 MEDIUM | share_handler 业务逻辑过重 | 提取到 service 层 |
| A1-4 | 🟡 MEDIUM | connect/strategy 包过大 | 按功能域拆子包 |
| F3-4 | 🟢 LOW | dispatchMarketOrder 未用 WithoutCancel | 统一或文档化 |
| C4-3 | 🟢 LOW | 重复的 Sharpe ratio 计算 | 提取公共函数 |
| T5-1 | 🟢 LOW | service_orders_modify.go float64 仅用于日志 | 改用 String() |

## Reuse Preflight

- **O2-1**: REUSE: `AccountStateProvider` interface @ `account_provider.go:83-85` (修改现有实现)
- **A1-3**: NEW: 无现成能力（已搜: share performance service, analytics service）
- **C4-3**: REUSE: `computeRiskMetrics` @ `analytics_compute.go:142` (可统一到此处)
