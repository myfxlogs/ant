# ADR-0012 Broker Adapter 抽象与多 broker 拓展

## Status
**Accepted** (2026-05-23) — 按推荐方向 B (MT4/MT5 + 加密交易所)；接口契约 freeze；M1 canonical symbol + M2 OMS 共用此前置

## Context
ant 当前直接调 MT4/MT5（`backend/internal/mt4client/`、`mt5client/`），与策略调度器紧耦合（`strategy_schedule_runner.go`）。AlfQ 迁移计划 §M2 要求引入 `BrokerAdapter` 接口让 broker 可换。**此 ADR 是 M1 canonical symbol + M2 OMS 状态机的公共前置**。

## Options
- A. 仅 MT4/MT5（保持现状抽象）
- **B. MT4/MT5 + 加密交易所（币安/OKX 等）**（推荐路径）
- C. + 国内 A 股券商：合规成本高，短期不接

## Decision
**B (MT4/MT5 + 加密交易所)**：引入统一 `BrokerAdapter` 接口；M1 canonical symbol + M2 OMS 共用此前置。接口契约见下文。

## 接口契约（无关 D-10 选项）

```go
// internal/broker/adapter.go
type BrokerAdapter interface {
    Submit(ctx context.Context, req *pb.OrderRequest) (*BrokerResp, error)
    Cancel(ctx context.Context, ticket string) error
    Modify(ctx context.Context, ticket string, price, stopPrice float64) error
    Query(ctx context.Context, ticket string) (*pb.Order, error)
}
```

实现：
- `internal/broker/mt4_adapter.go`：包装现有 `mt4client/`
- `internal/broker/mt5_adapter.go`：包装现有 `mt5client/`
- 未来：`internal/broker/binance_adapter.go` 等按 D-10 决定

## Consequences
- **+** OMS（M2）可面向接口编程；新增 broker 不改 OMS / strategy_schedule_runner
- **+** symbol 体系（M1）可按 broker 拉取归一化
- **−** 现有直接调 mt4/5 的代码必须在 M2 完成前替换（避免双路径）

## 实施约束
- M1 完成前接口 freeze
- M2 完成时 `grep -r "directly call mt[45]" backend/internal/service/` = 0
- 单测：每个 adapter 用 mock/sim broker 走通 Submit/Cancel/Modify/Query

## Related
- AlfQ 迁移计划 §M2.2
- DESIGN-REVIEW CR-01（OMS 缺失）
- ADR-0006（sqlc 方向影响 broker 持久化层）

## History
- 2026-05-23 Proposed

- 2026-05-23 Accepted（按推荐方向落地，详见 Decision）
