# ADR-0011 可观测性：trace_id / health / metrics

## Status
**Accepted** (2026-05-23) — 按推荐方向 B (OpenTelemetry 兼容)；M0.3 实施 trace_id + health metrics

## Context
DESIGN-REVIEW MR-05：① `trace_id` 全仓 0 命中（违反 AGENT.md「日志必带 trace_id」）；② `/healthz` 仅 1 处误用，docker-compose 三业务容器无 healthcheck（MR-07）；③ 已有 zap 结构化日志（521 处）但未贯穿请求链路；④ 无 metrics 端点。

## Options
- A. 自建 `trace_id`（UUID）+ 自建 metrics 格式
- **B. OpenTelemetry 兼容**（推荐）：`trace_id` / `span_id` 走 OTel 语义约定；后期可接 Jaeger / Tempo；metrics 走 Prometheus exporter
- C. 仅做 trace_id，metrics 暂不做

## Decision
**B (OpenTelemetry 兼容)**：trace_id / span_id 走 OTel 语义约定；后期可接 Jaeger / Tempo；metrics 走 Prometheus exporter。M0.3 实施。

## 实施细则（按 B 推演，落地 M0.3）

### trace_id 注入
- Connect interceptor（server side）：从 incoming header `traceparent` 解析；缺失则生成新 W3C Trace Context；写入 `context.Context`
- zap logger 通过 `WithContext(ctx)` 自动带 `trace_id` `span_id` 字段
- 跨服务调用：backend → strategy-service 必须透传 traceparent header

### Health endpoints
- `/healthz`：进程存活（200 即可）
- `/readyz`：依赖就绪（PG 可连 + Redis 可连 + strategy-service 可达）
- `/metrics`：Prometheus 格式
- docker-compose 三业务容器 healthcheck 用 `/healthz`

### Metrics 必备指标
- `ant_http_requests_total{method, code}`
- `ant_http_request_duration_seconds_bucket{...}`
- `ant_orders_submitted_total{state}`
- `ant_strategy_executions_total{result}`
- `ant_ai_calls_total{model, status}`

## Consequences
- **+** 与 ADR-0010 错误码联动后线上排障极快
- **+** Prometheus + Grafana 单机部署成熟方案
- **−** 引入 OTel + Prometheus exporter 两个依赖
- **−** strategy-service Python 侧也要装 OTel SDK（M0.3 后期）

## Related
- DESIGN-REVIEW MR-05, MR-07
- AGENT.md「日志」「部署形态」
- ADR-0010 错误码

## History
- 2026-05-23 Proposed
