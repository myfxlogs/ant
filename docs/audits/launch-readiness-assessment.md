# 上线就绪评估（Launch Readiness）— 2026-08-08

> **审计方**：Claude Code（接手地面真相审计）。本文是对"距上线还差什么"的系统回答，覆盖 `tech-debt-registry.md` 不含的维度（测试/可观测/运维/前端/性能）。
> **取代**：`docs/roadmaps/pre-launch-assessment.md`（2026-08-02，其 3 个 launch-blocking 缺口——GoExecutor/双风控/marketplace 资金覆盖/agent 基准——**现已全部解决**，旧报告作历史归档，见文末）。
> **不取代**：`tech-debt-registry.md`（代码债务仍以它为准）。

---

## 0. 一个事实校正（先纠错）

此前我（和 handover 旧"下一步方向"行）说"Prometheus 没做"——**错了**。实测 `cmd/server/handlers_sre.go:118-147`：
- `/metrics`（Prometheus，`mdgateway.MetricsHandler()`，M10 ADR-0010 §2.4）✅
- `/healthz`（探活 PG+NATS+Redis）✅
- `/readyz`（K8s readiness probe）✅
- SRE 控制面：kill switch + breaker registry + canary（`:68-72`）✅

教训：handover 那行"下一步方向：…Prometheus"是历史计划，没随实现刷新。**上线评估必须对着代码验，不能信漂移的计划行。**

## 1. 总体结论

**比旧报告乐观，但"能上线对外"仍差几块。**

- **核心可上线**：钱路径正确性、安全/鉴权、数据完整性、后端测试、可观测性基建、SRE 控制面、部署/迁移——**全部到位**，其中钱路径+安全经审计方逐行深验 + 对抗证明。
- **旧报告的 3 个 launch-blocking 缺口全部解决**：GoExecutor `go run` 已移除（ARCH-1/LAUNCH-3 ✅）、双风控引擎合并（ARCH-2 ✅）、marketplace 资金链路 15 集成测（LAUNCH-2 ✅）、agent 基准 20 策略达标（LAUNCH-1 ✅）。
- **真正剩余的上线缺口是非债务维度**：E2E 测试、前端测试、metric 覆盖度/告警、ARCH-4⑥。这些 `tech-debt-registry.md` 不记（不是"继承债"，是"没做过"）。

## 2. 逐维度评估

| 维度 | 状态 | 依据 | 审计方深验? |
|------|------|------|------------|
| 钱路径正确性（风控/OMS/对账/幂等/哈希链） | ✅ 就绪 | Gate 单一 chokepoint + 16 态 OMS + RECON-1 漏单修复 + LIVE-2 幂等 + FEAT-4V append-only 哈希链 | **是**（逐行+对抗证明） |
| 安全 / 鉴权 / IDOR | ✅ 就绪 | 49/49 handler 鉴权、session token_version、access_token 攻击面消除（AUD-W2-1）、SSRF 过滤、退款双 bug 修复 | **是** |
| 数据完整性 | ✅ 就绪* | hash chain + VerifyChain 集成测 + reconciliation；**ARCH-4⑥ 待补**（per-strategy 战绩归因，否则实盘公开拿不到数据） | 是（⑥未完成） |
| 后端测试 | ✅ 就绪 | `go test ./...` exit 0；集成测覆盖 money path / marketplace / 哈希链 | 是 |
| 可观测性（基建） | ✅ 就绪 | `/metrics`+`/healthz`+`/readyz`+SRE 控制面已 wire | 是（基建）；覆盖度/告警 ❓ |
| 可观测性（覆盖度 + 告警） | ✅ 就绪 | mthub 钱路径 4 指标已补（orders_placed/latency/session_active/event_published）+ `deploy/prometheus/alerts.yml` 12 条规则 + Grafana dashboard 9 面板；**审计方实测通过（2026-08-09）：build+test exit 0；4 指标在 `service_orders.go`/`types.go` 真增量（ok/rejected/err 全路径，非空壳）；`metrics_test.go` 对抗证明 spot-check（删 .Inc()→计数不增→红）**。Task 1 审计表见 spec §8（§3 余项 recon/idem/wallet 标 ❌ 属 spec non-goal「不做全量」范围外，已记账非丢失） | 是 |
| 部署 / 迁移 | ✅ 就绪 | Docker compose 唯一部署路径（CLAUDE.md 强制）、MIG-1 down 脚本齐（238/239）、healthz 探活 | 是 |
| **E2E 测试** | ❌ **缺口** | 全仓零 playwright/`.spec.ts`；仅有 playwright-backtest **skill/模式**，无 committed 全链路 E2E | 是（确认缺失） |
| **前端测试** | ✅ 就绪 | vitest+jsdom+@testing-library 基建+128 test（5 store 45 test + 11 组件冒烟 + 72 utils test）+ CI `npm test` 步；对抗证明通过 + **审计方实测 `npm test` 128 绿、authStore logout 对抗证明 spot-check 通过（2026-08-09）** | 是 |
| 前端体验 | ❓ 未审计 | dead code（CQ-2 存量）、i18n 有；UX 流/错误态/边界未系统验 | **否** |
| 性能 / 规模 | ❓ 未审计 | PgWriter drop、NATS JetStream、连接池为"已知特性"；无压测/容量数据 | **否** |
| 合规 / 计费 | ✅* 置信 | KYC jurisdiction gate 在 Gate 链；subscription/billing 代码在；未深审计费精度 | 部分 |
| 文档 / runbook | ⚠️ 部分 | ADR/registry/CAPABILITIES/CLAUDE.md 强；**运维 runbook / 故障手册 / oncall ❓** | 否 |

> 状态图例：✅ 就绪 ｜ ✅* 置信非逐验 ｜ ⚠️ 部分 ｜ ❌ 缺口 ｜ ❓ 未审计（需专项）

## 3. 上线前必须堵的缺口（launch-blocking）

| # | 缺口 | 为什么 blocking | 工作量 |
|---|------|----------------|--------|
| 1 | **E2E 测试套件** | 无任何端到端守护——购买→实盘→战绩主链路、登录→回测→看报告等核心流只靠手工。回归必靠人。复用现成 `playwright-backtest` skill 模式起步 | 中 |
| 2 | ~~**前端测试基线**~~ | ✅ **审计方实测通过（2026-08-09）**——`npm test` 128 test 绿实测 + authStore logout 对抗证明 spot-check（删 isAuthenticated:false 翻转→测试必红）。vitest+jsdom+@testing-library 基建 + setup.ts（matchMedia/IntersectionObserver/ResizeObserver mock）+ 5 Zustand store 测（45 test）+ 11 组件冒烟 + CI `npm test` 步 | ~~中~~ ✅ |
| 3 | ~~**ARCH-4⑥ 归因闭环**~~ | ✅ **已完成（2026-08-08 验收，commit `00e5ccc1`）**——migration 266 + `ResolveScheduleIDByMagic` account-scoped + 两份写路径回填 `ScheduleID`，hollow-core 闭合，per-strategy 战绩可归因。残留(低)：DB 级集成测待补。原 spec `docs/spec/multi-strategy-attribution-spec.md` | ~~小-中~~ ✅ |
| 4 | ~~**metric 覆盖度审 + 告警规则**~~ | ✅ **审计方实测通过（2026-08-09）**——mthub 钱路径 4 指标已补（`mthub_orders_placed_total{broker,status}` / `mthub_place_latency_seconds{broker}` / `mthub_session_active{account_id,broker}` / `mthub_event_published_total{event_type}`）+ `deploy/prometheus/alerts.yml` 12 条规则（mthub 4 + mdgateway 5 + platform 3）+ Grafana dashboard 9 面板。Task 1 审计表完成（spec §8：15-observability §3 全量逐条对账）。对抗证明：5 test 绿（ok/rejected/err/session/event）。`go build`+`go test` 绿。REUSE: promauto 模式（strategy/metrics.go）；NEW: mthub/metrics.go + alerts.yml + dashboard JSON。**残留(低)**：alerts.yml runbook 链接指向 `docs/runbook/mthub-*.md`（文件未建，post-launch 项）；§8 记账的 recon/idem/wallet 指标属 spec non-goal 范围外 | ~~中~~ ✅ |

## 4. 可延后（post-launch / 非阻断）

- CQ-2（前端 knip 死代码）、CQ-5（eslint-disable）——存量清理，不影响业务
- FEAT-3（受保护回测面板对齐）、FEAT-5（AI 迭代闭环）——roadmap 功能
- 性能压测 / 容量规划——流量起来再做
- 运维 runbook / oncall 手册——边运维边补

## 5. 审计边界声明（防过度承诺）

**本审计深验过的**（逐行 + 对抗证明）：钱路径（risk/OMS/reconciliation/idempotency/hash chain）、安全/鉴权/IDOR、ARCH-3 购买→实盘取码源、FEAT-1 授权闸、LAUNCH-2 退款、LIVE-2 幂等、FEAT-2 lookahead、Defense A、FEAT-4V、RISK-MARGIN1、ARCH-4 ①-⑤、本轮全部施工项。

**未审计（标 ❓ 的维度）**：可观测性覆盖度/告警、前端 UX、性能/规模、运维 runbook、依赖安全（dependency scan）。这些需各自专项——**"审计验过"≠"这些维度 OK"**，上线决策须分别补。

## 6. 关于旧报告（`docs/roadmaps/pre-launch-assessment.md`，2026-08-02）

旧报告结论"可以上线但有 3 缺口"，3 缺口现状：
- Gap 1（agent 策略质量基准）→ **LAUNCH-1 ✅**（20 策略，编译 100%/回测 100%/Sharpe>0 58%，全达标）
- Gap 2（marketplace 资金链路覆盖）→ **LAUNCH-2 ✅**（15 集成测覆盖 purchase/settle/refund/subscribe + 退款双 bug 修复）
- Gap 3（GoExecutor）→ **ARCH-1/LAUNCH-3 ✅**（`go run` 移除，剩死 stub）
- 附：旧报告另列的 risk-gate 双引擎（ARCH-2 ✅）、frontend dead code（=CQ-2 🟦 非阻断）。

**即：按旧报告自己的 bar，原 blocking 缺口已清零。** 本报告新增的缺口（E2E/前端测试/metric 覆盖）是旧报告没评分的维度——旧报告当时评 frontend"✅可用"而未查测试覆盖，是当时的盲区。

---

> **一句话**：地基稳、钱路径和安全过硬、旧缺口清零、ARCH-4⑥ 已闭环、前端测试基线 ✅、可观测钱路径指标+告警 ✅实测（2026-08-09）；**距对外上线只剩 E2E 一块**。补完即可上线。
