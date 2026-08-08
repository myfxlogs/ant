# 可观测性补缺施工 Spec（钱路径指标 + 告警落地）

> **定位**：本 spec 是 **`docs/spec/15-observability.md` 的补缺**，不重定义指标体系。15-observability 是设计权威（命名规范 + 全量清单 + §6 告警规则），本 spec 只管"specified-but-not-implemented"的落地。
> **关联**：`launch-readiness-assessment.md`（上线缺口 ④）、ADR-0010（SLO/alert/DLQ/trace）。**状态**：🏗 待施工。**日期**：2026-08-08

---

## 1. 背景（实测，已纠正"❓"）

可观测**地基已有**：promhttp `/metrics`（`handlers_sre.go:118`）+ `/healthz`（探 PG/NATS/Redis）+ `/readyz` + SRE 控制面（kill switch/breaker/canary）+ 前端 `@sentry/react`。代码里已有 7 个 metric 向量（`strategy/metrics.go`：backtest_runs_total / duration / experiment_runs / sse_connections 等）。

**实测缺口**（对着 15-observability 清单验代码）：
- **mthub 钱路径指标全部 specified-but-not-implemented**：15-observability §3.2 的 `mthub_orders_placed_total{broker,status}` / `mthub_place_latency_seconds` / `mthub_session_active` / `mthub_event_published_total` —— grep 代码**全空**。钱路径（下单/拒单/延迟）当前**无指标**。
- **`deploy/prometheus/alerts.yml` 似未创建**：15-observability §6 指向它（含 `MtHubOrderRejectRateHigh`/`MtHubPlaceLatencyP99High` 等规则），但仓库里无此文件 → **告警未落地**。
- 全仓仅 7 个 metric 向量，远少于 15-observability 清单。

> 即：可观测的"设计"完整，但**钱路径的"实现"和"告警"是空的**。本 spec 补这两块。

## 2. 目标 / 非目标

**目标**：① 实现钱路径核心指标（下单/拒单/延迟/会话）；② 落地 `deploy/prometheus/alerts.yml`（§6 规则）；③ 一个钱路径 + 平台健康 dashboard；④ 与 ADR-0010 SLO 对齐。使"钱路径可测、关键异常可告警"。

**非目标**：不重写 15-observability；不上新监控栈（Prometheus/Sentry 已在）；不做全量指标（只补钱路径 + 平台健康关键项）。

## 3. 实现任务

| # | 任务 | 锚点/产出 |
|----|------|----------|
| 1 | **指标落地审计**：逐条对 15-observability §3 清单 vs 代码，输出"已实现/缺"表（本 spec 的 task 0，做完贴回本 spec） | 审计表 |
| 2 | **实现 mthub 钱路径指标**（§3.2）：`mthub_orders_placed_total{broker,status}`（status∈ok/rejected/err）、`mthub_place_latency_seconds{broker}`、`mthub_session_active{account_id,broker}`、`mthub_event_published_total{event_type}`。埋点在 `submitToBroker`/`PlaceOrder`（`mthub/service_orders.go`）。REUSE `strategy/metrics.go` 的 `promauto.NewCounterVec` 模式 | `backend/internal/mthub/metrics.go`（NEW） |
| 3 | **补 wallet/marketplace/recon 指标**（若 15-observability 清单有）：wallet_tx_total、marketplace_purchase/refund_total、recon_ghost_positions / orphan_trades / sync_failures_total。埋点在各 service | 对应包 metrics.go |
| 4 | **创建 `deploy/prometheus/alerts.yml`**：按 15-observability §6 落地（MtHubOrderRejectRateHigh、MtHubPlaceLatencyP99High、MdGateway* 系列）+ ADR-0010 M10 告警。每条带 runbook 链接 | `deploy/prometheus/alerts.yml` |
| 5 | **Dashboard**：Grafana JSON（或文档）覆盖钱路径（下单/拒单/延迟/PnL）+ 平台健康（连接/断线/circuit/SSE）。REUSE healthz 探的维度 | `deploy/grafana/*.json` 或 doc |
| 6 | **SLO 对齐**：核对 ADR-0010 SLO 目标（下单延迟、SSE 新鲜度、回测队列深）是否可由现有指标算出，缺则补 | 引用 ADR-0010 |
| 7 | **部署接线**：Prometheus scrape `/metrics` + Alertmanager 加载 alerts.yml；docker-compose 或 k8s 配置 | `deploy/` |

> **复用核对**：`strategy/metrics.go` promauto 模式（REUSE，勿重造）、15-observability §6 告警定义（直接落地，勿改规则名/阈值除非有据）、healthz 探活维度、ADR-0010 SLO。动工前 `bash scripts/cap.sh metrics`。

## 4. 验收 + 对抗证明

- `/metrics` 暴露 mthub 钱路径指标；`deploy/prometheus/alerts.yml` 存在且 `promtool check rules` 通过。
- **对抗证明**：构造一单下单 → `mthub_orders_placed_total{status="ok"}` +1；构造 Gate 拒单 → `status="rejected"` +1 且 `MtHubOrderRejectRateHigh` 规则在测试期可触发；kill switch on → 对应告警条件成立。埋点删了则指标不增 = 测试必红。
- runbook：每条 alert 有处置链接（至少占位），避免告警来了没人懂。

## 5. 边界

- **禁高基数 label**（15-observability §2）：`user_id × symbol` 这类会爆 Prometheus——只用 `account_id, broker, canonical, period` 选用维度。`mthub_session_active{account_id}` 注意账户基数（零售账户量级可接受；若爆炸改用 `broker` 聚合）。
- 指标命名严守 15-observability §2（`<domain>_<subject>_<verb>_<unit>`，Counter `_total`，Histogram `_seconds`）。
- Trace（OTLP）15-observability 标"M8 接入可选"，本 spec 不做。

## 6. 审计方边界声明

我（审计方）**未做**全量"15-observability 清单 vs 代码"逐条对账——那是本 spec task 1（交给施工方产出审计表）。我只验了关键样：mthub 钱路径指标 grep 全空、alerts.yml 缺失、全仓仅 7 向量。施工方 task 1 的审计表若发现清单已部分实现，据实调整 task 2-3 范围（**不扩大**，只补缺）。

## 7. 完工回填

`launch-readiness-assessment.md` 缺口 ④ 划掉 + handover 变更日志 + 对抗证明（含 task 1 审计表）+ `promtool check rules` 绿。不自行宣告完成。
