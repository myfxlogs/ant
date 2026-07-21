# api-gateway — API 层

> ConnectRPC handlers + SSE 流 + proto 定义。

## 代码位置

```
proto/ant/v1/                    ← 104 个 proto 文件，52 个 gRPC 服务，335 个 RPC
backend/internal/connect/        ← 14 个 handler 目录
```

## 服务清单（主要）

- `AccountService`（12 RPC）、`AuthService`（8 RPC）
- `AIService`、`AIGatewayService`、`AgentGatewayService`
- `StrategyService`（17 RPC）、`StrategyRuntimeService`（15 RPC）
- `MarketplaceService`（23 RPC）
- `MtHubService`（11 RPC）
- `StreamService`（6 SSE 流）
- 10 个 Admin 服务 + 20+ 其他服务（Wallet、Deposit、Share、Subscription…）

## 关键设计

- 所有对外 API 通过 ConnectRPC
- 实时推送使用 server-stream SSE
- 禁止 REST（除 healthz/readyz/livez/metrics）
- [plans/sse-connection-limit.md](plans/sse-connection-limit.md) — SSE 连接数限制
