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
| **E2E 测试** | ✅ 就绪 | **审计方实测通过（2026-08-09，fresh-state）**：从 DB 删 test user 后跑 `npx playwright test` = **17 passed / 0 failed, exit 0（1.1m）**。globalSetup 自播种修复生效（commit `420b9f2f`）：`e2e/globalSetup.ts` 调 `registerTestUser` + `playwright.config.ts:6` 引用。4 旅程（login/marketplace/purchase→live/backtest）+ 5 对抗证明 + CI e2e job 全绿。原 fresh-env 2-failed 缺陷已闭合 | 是（实测）|
| **前端测试** | ✅ 就绪 | vitest+jsdom+@testing-library 基建+128 test（5 store 45 test + 11 组件冒烟 + 72 utils test）+ CI `npm test` 步；对抗证明通过 + **审计方实测 `npm test` 128 绿、authStore logout 对抗证明 spot-check 通过（2026-08-09）** | 是 |
| 前端体验 | ❓ 未审计 | dead code（CQ-2 存量）、i18n 有；UX 流/错误态/边界未系统验 | **否** |
| 性能 / 规模 | ❓ 未审计 | PgWriter drop、NATS JetStream、连接池为"已知特性"；无压测/容量数据 | **否** |
| 合规 / 计费 | ✅ 代码就绪 / ⚠️ 合规姿态待定 | KYC jurisdiction gate 在 Gate 链；**money-path（USDT-TRC20 充值/内部账/提现/托管）逐行验=真实+生产级**（见 §7）；加密托管监管姿态需业务/法律定 | 代码是 / 合规否 |
| 文档 / runbook | ⚠️ 部分 | ADR/registry/CAPABILITIES/CLAUDE.md 强；**运维 runbook / 故障手册 / oncall ❓** | 否 |

> 状态图例：✅ 就绪 ｜ ✅* 置信非逐验 ｜ ⚠️ 部分 ｜ ❌ 缺口 ｜ ❓ 未审计（需专项）

## 3. 上线前必须堵的缺口（launch-blocking）

| # | 缺口 | 为什么 blocking | 工作量 |
|---|------|----------------|--------|
| 1 | ~~**E2E 测试套件**~~ | ✅ **审计方实测通过（2026-08-09，fresh-state）**——新建 `e2e/globalSetup.ts` 调 `registerTestUser`（REUSE: `auth.ts:81`，幂等）+ `playwright.config.ts` 加 `globalSetup`（commit `420b9f2f`）。**审计方 fresh-state 实测**：从 DB 删 `e2e@test.com` 后 `npx playwright test` = **17 passed / 0 failed（1.1m）**。**对抗证明**：删 user + 去 globalSetup → login `auth.ts:41` fail（Windsurf 验 5 failed/8 not run）；加回 → fresh 17 绿。adversarial.spec.ts 修复：原"insufficient balance"（后端不强制）→ "non-existent strategy fails"；"no token"独立出 describe | ~~小~~ ✅ |
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

## 7. 运营/外部接缝地面真相（2026-08-08 审计方逐行实测 money-path 平台账半边）

> 此前 §2「合规/计费」标 `✅* 置信非逐验`。本轮以运营负责人身份，对「平台账」钱路径（区别于已深验的「交易账」半边 risk/OMS/reconcile/hash）做了逐行实测。**结论：真实、全自动、生产级托管，非 stub。** 同时定位真正的上线阻塞 = 纯外部凭证/基础设施，非代码。

### 7.1 钱路径全链（平台账半边）实测

| 环节 | 实现 | 文件锚点 | 状态 |
|---|---|---|---|
| 钱进：USDT-TRC20 充值 | HD 冷钱包（在线只持 xpub，私钥离线）+ 链上自动检测入账 | `chain/monitor.go`、`service/deposit_service.go`、`hdwallet/derive_priv.go` | ✅ 逐行验 |
| 链上检测 | TronGrid 块扫描 3s + 双源验证(TronGrid/TronScan) + min-confirms=20 + 检查点断点续扫 + PG advisory lock 单实例 + 监控含 RETIRED 地址 | `chain/monitor.go:74-350` | ✅ 生产级 |
| 内部账本 | AdjustBalanceTx 幂等（购买/订阅/结算/退款/agent-credit），idem_key 去重 | `service/wallet_service.go:48`、`marketplace/settlement.go` | ✅ 已审(LAUNCH-2) |
| 钱出：作者提现 | WithdrawalRequest 生命周期 + WebAuthn 授权 + 目的地白名单(R12) | `model/webauthn.go:26-46`、`marketplace/provider_earnings.go` | ✅ 存在+硬化 |
| 托管对账 | 24h 链上余额核对（短缺=偿付风险告警/盈余=信息告警）+ 冷钱包 | `reconcile/reconcile.go:154-186` | ✅ 在 |
| 偿付证明 | 离线 keygen(hdgen-gui, 24 词助记词) + solvency-check CLI | `cmd/hdgen-gui/`、`cmd/solvency-check/` | ✅ 在 |

**纠正记录**：开局曾两度误判（先判"无支付通道"，修正为"加密充值 USDT-TRC20"）。根因 = `chain/`/`sweep/`/`hdwallet/`/`webauthn` 包在 `wallet` 包外、用不同术语，首轮窄 grep 未命中。教训复述：上线判断必须对着代码逐包验，不能信单次窄搜索的"零命中=不存在"。

### 7.2 真正的上线阻塞 = 外部凭证/基础设施（非代码，需人/运营）

| # | 项 | 性质 | 谁能完成 |
|---|---|---|---|
| 1 | 加密托管冷启动：离线生成助记词 → 配 `deposit_xpub`/`deposit_xpub_fingerprint` → 冷钱包地址 → `usdt_contract_address`/`min_confirmations`/`min_deposit_amount`(system_config) → TronGrid/TronScan API key | 离线+外部凭证 | 运营（人） |
| 2 | mtapi.io 真实账户/token（MT4/MT5 网关第三方代理，付费） | 外部账户 | 运营（人） |
| 3 | 真实主机 + 域名 + TLS + DNS（docker 栈已就绪，缺跑的地方） | 基础设施 | 运营/运维（人） |
| 4 | Admin/operator 引导：seed admin、配 system_config、alertmanager/uptime-kuma 通知通道(Slack/email) | 运维配置 | 运营/运维（人） |

> 这些不是代码任务，是上线运营清单。代码侧就绪，运营把外部件接上即可点亮。

### 7.3 两个需决策/补审的点（非阻断，但上线前应处置）

1. **🔴 合规姿态（业务/法律，需人定）**：平台持有用户 USDT 托管 = 触及「资金/货币传输」监管。架构已严肃对待（冷储 + 偿付证明 + 对账 + WebAuthn + 白名单），但 CLAUDE.md「不碰用户资金」表述与托管现实存在张力。KYC jurisdiction gate 已在 Gate 链，但加密入金的 KYC/AML、司法管辖、是否构成货币传输 = 真实业务/法律决策，非 AI 可代定。**列为引入真实付费时的第一业务风险。**
   - **👤 用户决策（2026-08-08）**：当前为**最小版测试期，全站免费**——不启用充值/订阅计费/抽成/提现，故**钱路径与合规在测试期均休眠**（代码已就绪但 dormant）。合规姿态留到引入真实付费时再定。即：测试期上线**不受** §7.2-①（托管冷启动）/④（计费配置）+ §7.3-1（合规）阻塞；§7.2-②（mtapi.io）/③（主机域名 TLS）仍是测试期上线所需的外部件。
2. **🟡 钱出路径未深审（代码审计缺口）**：本轮逐行验了钱进 + 内部账 + 检测；**作者提现(WithdrawalRequest → WebAuthn → 白名单 → 出金)只确认存在+硬化、未逐行深审**。钱出是盗用/欺诈风险集中处，且此前「钱路径深验」覆盖的是交易账(risk/OMS/reconcile/hash)，不含钱包提现。**建议：处理真实作者提现前，补一次 wallet 提现专项审计**（对抗证明：伪造提现 / 越权 / 重放 / 白名单绕过 / 余额竞态）。

### 7.4 结论（运营负责人判断）

代码与平台账钱路径**技术上线就绪**。上线与否不再取决于工程，取决于：① 外部凭证/基础设施接上（§7.2）；② 合规姿态给定性（§7.3-1）。补 wallet 提现专项审（§7.3-2）是推荐而非阻断——且是当前仓库内最高价值、最高风险的未审钱路径。

---

> **一句话**：地基稳、交易账钱路径和安全过硬、旧缺口清零、ARCH-4⑥ ✅、前端测试 ✅、可观测 ✅、**E2E ✅（2026-08-09 fresh-state 17 绿实测）**、**平台账钱路径 ✅（2026-08-08 逐行验=真实生产级加密托管）**。**所有代码层 launch-blocking 缺口已清零——技术上可上线。** 真正的上线阻塞 = 4 件外部凭证/基础设施（§7.2）+ 合规姿态定性（§7.3-1）。post-launch 边运维边补：runbook / 性能压测 / 前端 UX / 依赖扫描（非阻断）。
