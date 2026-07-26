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

### A1-3 ✅ FIXED — connect 层业务逻辑已提取到 service 层

**修复**: 提取 `share_service.go`，包含 `BuildSharePerformance`、`FormatSharedTrades`、`FormatSharedPositions`、`summarizeTrades`、`computeSharpe`。

`share_handler.go` 从 294→196 行，handler 现在只做请求/响应映射。
`share_og_image.go` 从 260→212 行，使用共享 `BuildSharePerformance`。

### A1-4 🟡 MEDIUM — connect/strategy 包过大（待处理）

`backend/internal/connect/strategy/` 包含 20+ 文件，涵盖回测、实盘执行、实验、CRUD、调度。虽然每个文件在 300 行以内，但包内耦合度高。

**建议**: 考虑按功能域拆分为 `strategy/backtest/`、`strategy/live/`、`strategy/experiment/`、`strategy/crud/` 子包。

**状态**: 暂缓 — 大重构，高回归风险，建议在专门重构窗口处理。

### A1-5 ✅ Push-first 架构

SSE 流 + PG LISTEN 推送是默认模式。无轮询、无 cron、无 `time.Ticker`（除 VM 执行上下文超时）。

---

## 2. 最优性

### O2-1 ✅ FIXED — 硬编码 balance fallback 10000 已移除

**修复**: `buildStateFromSnapshot` 和 `buildStateFromOrders` 中移除 10000 fallback，改为 fail-closed（返回 nil → gate 阻止交易）。新增 `SetBalance` 方法供 ProfitUpdate 事件注入真实 balance。

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

### F3-4 ✅ NON-ISSUE — 策略执行 context.WithoutCancel

经核实，`submitOrder`（被 `dispatchMarketOrder` 和 `dispatchPendingOrder` 调用）已在 `live_dispatch.go:312` 使用 `context.WithoutCancel(ctx)`。所有订单提交路径均正确分离了请求生命周期。

---

## 4. 代码整洁

### C4-1 ✅ 无 TODO/FIXME/HACK 标记

代码中无遗留的 TODO/FIXME/HACK 注释。所有匹配项均为函数名（`UpdateToDone`）或 LLM prompt 文本。

### C4-2 ✅ 文件大小合规

`check-file-lines --strict` 通过：0 errors, 29 warnings, 74 info。无文件超过 450 行硬性红线。

### C4-3 ✅ FIXED — 重复的 Sharpe ratio 计算已合并

**修复**: `share_handler.go` 和 `share_og_image.go` 现在共享 `computeSharpe` 函数（位于 `share_service.go`）。

### C4-4 ✅ 命名一致性

包命名、文件命名、函数命名风格一致。`Set*` 注入模式统一使用。

### C4-5 ✅ 错误处理一致

所有 handler 使用 `connect.NewError()` 包装错误，错误码语义正确（`CodeUnauthenticated`、`CodeInvalidArgument`、`CodeInternal`）。

---

## 5. 技术债

### T5-1 ✅ FIXED — float64 在 MT proto 边界

`mthub/types.go:131` 的 `ProfitPercent float64` 和 `broker_types.go:100` 的 `Volume float64` 是 MT API proto 要求的 float64。这是外部约束，非技术债。

**修复**: `service_orders_modify.go` 日志已改为 `zap.String("sl", sl.String())` 和 `zap.String("tp", tp.String())`，避免不必要的精度损失。

### T5-2 🟢 LOW — 已知缺口（来自 go-native-strategy-pipeline.md §8）

以下为已记录的已知缺口，非新增技术债：
- barsDropped notification (P2)
- per-bar OpenedOrders query 应使用 PositionSnapshotBroker subscription (P2)
- AccountService 仍使用 float64 (P2)
- MTAccountInfo 仍使用 float64 (P2)
- iCustom custom indicators (P3)
- Bytecode cache persistence to DB (P3)
- Live consistency verification (VM signals vs backtest results) — 未验证

### T5-3 ✅ FIXED — NATS 认证已启用

**修复**: `nats.conf` 添加 `authorization` 块，docker-compose 传递 `NATS_USER`/`NATS_PASSWORD` 环境变量，后端 `NATS_URL` 包含凭据。

### T5-4 ✅ FIXED — migration 已包装事务

**修复**: `docker-entrypoint.sh` 中每个 migration 文件包装 `BEGIN/COMMIT`，防部分应用。

### T5-5 ✅ 无 deprecated/legacy 代码标记

代码中无 `// Deprecated:`、`// legacy`、`// workaround` 标记。Python 策略运行时已完全移除。

---

## 修复状态

| ID | 严重度 | 描述 | 状态 |
|---|---|---|---|
| O2-1 | 🟡 MEDIUM | 硬编码 balance 10000 fallback | ✅ FIXED — fail-closed |
| W4-1 | 🟡 MEDIUM | SSRF — AI provider base_url 无私有 IP 过滤 | ✅ FIXED — isPrivateOrLoopbackHost |
| W3-2 | 🟡 MEDIUM | NATS 无认证 | ✅ FIXED — authorization 配置 |
| A1-3 | 🟡 MEDIUM | share_handler 业务逻辑过重 | ✅ FIXED — 提取 share_service.go |
| A1-4 | 🟡 MEDIUM | connect/strategy 包过大 | ⏸️ DEFERRED — 高回归风险 |
| F3-4 | 🟢 LOW | dispatchMarketOrder 未用 WithoutCancel | ✅ NON-ISSUE — submitOrder 已使用 |
| C4-3 | 🟢 LOW | 重复的 Sharpe ratio 计算 | ✅ FIXED — 合并到 share_service.go |
| T5-1 | 🟢 LOW | service_orders_modify.go float64 日志 | ✅ FIXED — 改用 String() |
| W3-1 | 🟢 LOW | migration 未包装事务 | ✅ FIXED — BEGIN/COMMIT |

## Reuse Preflight

- **O2-1**: REUSE: `AccountStateProvider` interface @ `account_provider.go:83-85` (修改现有实现)
- **A1-3**: REUSE: `computeSharpe` @ `share_og_image.go:112` → 提取到 `share_service.go`
- **C4-3**: REUSE: `computeSharpe` @ `share_service.go` (统一实现)
- **W4-1**: NEW: 无现成能力（已搜: SSRF protection, private IP check）
- **W3-1**: NEW: 无现成能力（已搜: migration transaction wrapper）
- **W3-2**: NEW: 无现成能力（已搜: NATS auth）
