# ADR-0001 Connect RPC + SSE 通信协议

## Status
Accepted (2026-05-23) — 既定状态，承袭自 anttrader 现有实现

## Context
ant 需要一个统一的对外通信协议。候选：REST、gRPC、Connect RPC、tRPC、GraphQL。需要满足：① 强类型 schema；② 浏览器原生可用（不依赖 grpc-web 网关）；③ 服务流式（AI 辩论、行情推送）；④ 单一 proto 源同时出 Go / TS / Python stub。

## Options
- **A. Connect RPC + SSE**（已选）：proto 单源 + buf 生成；Connect 协议浏览器直连；单向流用 SSE（Server Streaming）
- B. gRPC + grpc-web：需额外网关；浏览器流式体验弱
- C. REST + OpenAPI：类型生成质量低；流式靠 SSE 但失去 schema 一致性
- D. WebSocket：双向流；但 SSE 在 HTTP/2 下已足够；WebSocket 增加运维复杂度

## Decision
**A**。proto 在 `proto/` 单源，`buf generate` 出 `backend/gen/proto/` 与 `frontend/src/gen/`；流式 RPC 一律用 Server Streaming（SSE 在 Connect 下原生支持）；**禁止** REST 新接口（healthz / metrics 例外）与 WebSocket。

## Consequences
- **+** 单源类型安全；浏览器直连；流式开箱即用
- **+** 与 AGENT.md 硬性规则一致
- **−** 工具链依赖 buf；Agent 需熟悉 Connect 错误码（已有 `connect.Code*`，与本仓 errs 包需要适配，见 ADR-0010）
- 迁移：M0.3 前不动；M0.3 后所有新接口走 Connect RPC

## Related
- AGENT.md 硬性规则「协议」章节
- DESIGN-REVIEW MR-04（错误码体系与 Connect Code 协调）

## History
- 2026-05-23 Accepted（沿用既定实现）
