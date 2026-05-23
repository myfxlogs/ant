# ROADMAP — ant 实施路线图

**Version**: v2.0 Completed · 2026-05-23
**Source of Truth**: 本文档 + `@/opt/ant/docs/adr/` + `@/opt/ant/AGENT.md`
**关联**: `@/opt/ant/docs/audit/DESIGN-REVIEW-2026-05.md`（cards 来源） · `@/opt/ant/docs/AlfQ功能迁移计划.md`（功能蓝本）

> **路线图原则**
>
> 1. **现状定位**：ant 不是空仓，而是「功能基本齐全 + 工程纪律待补」的 M0.5 状态
> 2. **M0.x 工程基线先行**：在动 AlfQ 迁移前先把 git/CI/测试/复杂度红线落齐 ✅ 已完成
> 3. **M1-M6 对齐 AlfQ 迁移计划**：但每个里程碑由 ant 独立验收口径
> 4. **每张卡片必须**：commit hash + 实测日志 + 回归测试 + ADR 关联（如改架构）
> 5. **不跳步**：上一里程碑未拿到 ✅ 不开下一里程碑

---

## 0. 里程碑总览

| Milestone | 持续 | 累计 | 状态 | 关键交付 | 依赖 ADR |
|---|---|---|---|---|---|
| M0.0 现状基线 | — | — | ✅ 已完成 | git 化、docker 构建通、版本基线（PG18/Go1.26/Py3.14/Node24/TS5.9） | 0009 |
| M0.1 工程基线 | 1 周 | 1 | ✅ `f113ff0` | 入仓二进制清理、CI workflow、healthcheck、migration .down 政策 | 0011 |
| M0.2 文档与 ADR | 1 周 | 2 | ✅ `439cb76` | 12 篇 ADR 转 Accepted、`docs/README.md` 导航、AGENT.md 校正 | 0001-0012 |
| M0.3 复杂度与品质 | 1 周 | 3 | ✅ `13ef4a2` | baseline/lint/errs 包/trace_id interceptor/vitest | 0010, 0011 |
| M1 canonical symbol | 2 周 | 5 | ✅ `730b234` | symbol 体系（canonical/broker/strategy_symbols）+ resolver + seed | 0012 |
| M2 OMS 状态机 | 2 周 | 7 | ✅ `9e88e0b` | OMS state machine + BrokerAdapter + risk_events | 0006, 0012 |
| M3 风控引擎 | 1.5 周 | 8.5 | ✅ `1ec6af2` | 6 条规则插件化 + user_risk_profiles + PreSubmit hook | — |
| M4 AI → canonical | 1 周 | 9.5 | ✅ `d057c3a` | SymbolExtractor + SymbolValidator + SymbolDetection 前端 | 0007 |
| M5 策略市场 | 4-6 周 | 14.5 | ✅ `bfb8988` | marketplace_strategies + sandbox_scan + AI 合规检查 | — |
| M6 部署上线 | 2 周 | 16.5 | ✅ `332360c` | backup/bench/deploy/status Makefile targets | — |

**P0 关键路径**：M0.1 → M0.2 → M0.3 → M1 → M2 → M3 → M4 → M5 → M6，全部 ✅ 已完成（2026-05-23）。

---

## M0.1｜工程基线（1 周）

**目标**：让任何新 Agent 进仓库 30 分钟内能跑通完整工作流；防偷懒 7 条全部能起作用。

### 卡片

| ID | 内容 | 关联 finding | 验收 |
|---|---|---|---|
| M0.1-1 | `git rm --cached backend/main backend/server backend/gen backend/mt4/*.pb.go backend/mt5/*.pb.go frontend/src/gen` | CR-02 | `git ls-files \| xargs -I{} stat -c '%s {}' {} \| sort -rn \| head -3` Top 3 < 5 MB |
| M0.1-2 | 更新 `.gitignore` 屏蔽生成代码 + 二进制 | CR-02 | `make proto && git status` 无新增追踪文件 |
| M0.1-3 | `.github/workflows/ci.yml`：lint + test + build + buf breaking + migrate dry-run | CR-03 | PR 触发 CI 5 个 job 全绿 |
| M0.1-4 | docker-compose 三业务容器加 healthcheck（backend `/healthz`、strategy-service `/healthz`、frontend `/health`） | MR-07 | `docker compose ps` 全部 healthy |
| M0.1-5 | CI 加 migration lint：任意 `*.up.sql` 必须配 `*.down.sql`（仅对 PR 新增） | MR-06 | 故意 PR 仅 .up → CI red |
| M0.1-6 | `.env.example` 全量 placeholder + `make env-check` | MN-04 | `make env-check` 列出所有缺失 key |

### 退出标准

- `make verify` 在 fresh clone 后一次跑通
- `docker compose up -d` → 5 容器全部 healthy
- CI matrix 全绿且耗时 ≤ 10 分钟

---

## M0.2｜文档与 ADR（1 周）

**目标**：把所有架构决策从口头/AGENT.md 散落处沉淀到 ADR；新 Agent 看 ADR 索引即知项目宪法。

### 卡片

| ID | 内容 | 关联 |
|---|---|---|
| M0.2-1 | 12 篇 ADR 全部从 `Proposed` 转 `Accepted`，待决策项需用户拍板 | D-01..D-10 |
| M0.2-2 | `docs/adr/README.md` 索引页（按 status / 按主题） | DOC-5 |
| M0.2-3 | `docs/README.md` 导航：domain / spec / plan / adr / audit / runbook 五大区 | DOC-2 |
| M0.2-4 | `docs/remediation-2026-05/` 14 文件归档到 `docs/_archive/` 或删除 | DOC-3 |
| M0.2-5 | `AGENT.md` 校正：M0 → M0.5；补充本审查关键 finding 引用 | DOC-1 |
| M0.2-6 | 建立 `docs/tasks/AGENT-RUNBOOK.md` 执行入口 | AGENT.md 顶部声明 |

### 退出标准

- 12 篇 ADR `Status: Accepted` 全数完成
- `docs/README.md` 一页可达所有关键文档
- 新 Agent 进仓 → 读 AGENT.md → 读 RUNBOOK → 读 ADR 索引 → 能开始干活

---

## M0.3｜复杂度与品质红线（1 周）

**目标**：把 AGENT.md 既有的硬性规则真正在 CI 强制起来。**不做全量一次性重构**（反 ROI）；存量 254 处违规走 baseline 豁免 + 分层策略。

### 存量违规数据（2026-05-23 实测）

| 类别 | 数量 | 处理策略 |
|---|---|---|
| Go 文件 >300 行 | 52 | baseline 豁免；Tier 1 随 M1-M5 顺手拆；Tier 2/3 不动 |
| TS 文件 >250 行 | 37 | 同上 |
| Go 函数 >50 行 | 165 | 同上 |
| Python >400 行 | 0 | — |

### 分层策略

- **Tier 1**（必拆）：M1-M5 热路径 + 文件 >500 行 或 函数 >100 行；进入对应 milestone 时**作为前置卡片**拆到上限以内（列入各 milestone 工作量，不单独立项）
- **Tier 2**（Boy Scout）：在热路径上但 300–500 行；任何卡片触碰时顺手拆，合并到该卡片提交
- **Tier 3**（豁免）：不在热路径的文件 / i18n 资源文件 / proto 生成代码 / 数据与常量文件 / model struct 集中定义 → 永久豁免

### Tier 1 热路径清单（~16 个文件，分散到 M1-M5）

**M2 随手拆**：`execution_gateway.go` 531、`strategy_schedule_runner_loop.go` 503、`strategy_schedule_service.go` 454、`trading_service_orders.go` 459、`StrategySchedulePage.tsx` 683

**M4 随手拆**：`debate_v2_service.go` 722、`debate_v2_prompts.go` 455、`StrategyTemplatePage.tsx` 750、`StrategyTemplateScheduleLaunchForm.tsx` 661、`SystemAI.tsx` 646

**M1 随手拆**：`python_strategy_service.go` 622、`python_strategy_service_backtest_runs.go` 616、`BindAccount.tsx` 498

**Dashboard 高频**：`Summary.tsx` 641、`LogManagement.tsx` 475（服务于 M0.3 本身的用户画面体验）

### 卡片

| ID | 内容 | 关联 |
|---|---|---|
| M0.3-1 | 生成 baseline 文件 `tools/lint/baseline.json`：冻结当前 254 处违规点（Go 52 文件 + TS 37 文件 + Go 165 函数） | MR-03 |
| M0.3-2 | golangci-lint 配置：file-len ≤ 300、func-len ≤ 50、cyclo ≤ 10、参数 ≤ 5；读 baseline 豁免现有列表，新增/修改后超限则 CI red | AGENT.md §复杂度 |
| M0.3-3 | eslint-plugin-max-lines + custom rule：读同一 baseline；TS 同级零容忍 | MR-03 |
| M0.3-4 | eslint 加 `@typescript-eslint/no-explicit-any: error`；存量 522 处违规加 baseline waiver；新代码零容忍 | MN-01 |
| M0.3-5 | Tier 3 豁免规则落到 lint 配置：`**/i18n/**` `**/gen/**` `**/model/models.go` `**/*defaults.ts` 等彻底跳过 | MR-03 |
| M0.3-6 | 建 `internal/errs/` 包：错误码 enum + 中/英 message map；先映射 Top-50 高频错误 | MR-04, ADR-0010 |
| M0.3-7 | Connect interceptor：注入 `trace_id`（OTel 兼容） + 透传到 zap logger fields | MR-05, ADR-0011 |
| M0.3-8 | 前端引入 vitest + @testing-library；先覆盖 auth/login、accounts/list、dashboard 三个核心页面 | CR-03 |
| M0.3-9 | 后端 fat-service 加 happy-path 测试，优先覆盖 Tier 1 清单中的文件（为 M1-M5 拆分提供回归网） | CR-03 |

### 退出标准

- `make lint` Go + TS 全绿（带 baseline，仅限初始 254 处违规）
- 任何新 PR 动过的文件如果发生超限必须在同 PR 拆到上限以内（CI 强制，baseline 中检查“是否还仅是该列表原本”）
- Tier 1 清单中 16 个文件在 M1-M5 进度条中处于指派状态
- 后端 happy-path 覆盖率 ≥ 20%（从 < 5% 起步）
- 前端关键页面有冒烟测试
- `trace_id` 在所有日志行可见

---

## M1｜canonical symbol 基础设施（2 周）

**蓝本**：`@/opt/ant/docs/AlfQ功能迁移计划.md` §M1（不复制，按 ADR-0012 适配 ant 架构）

### 卡片

| ID | 内容 | 验收口径 |
|---|---|---|
| M1-1 | migrations 087/088/089/090：`canonical_symbols` `broker_symbols` `strategy_symbols` + 数据迁移 | 任意 broker 账户绑定后 `SELECT * FROM broker_symbols WHERE canonical='BTCUSD'` ≥ 1 行 |
| M1-2 | 移植 `internal/symbolsync/`（删 RLS、适配 ant PG 抽象） | broker 连接事件触发后 broker_symbols 更新 |
| M1-3 | seed `canonical_symbols` ~50 主流商品 | 字典覆盖 BTCUSD / EURUSD / XAUUSD / ... |
| M1-4 | `internal/symbol/resolver.go`：`ResolveCanonical(accountID, canonical) → (symbolRaw, tradeMode, ok)` | 单测覆盖：BTCUSDm/BTCUSD.x/BTCUSDpro/BTCUSD# |
| M1-5 | 历史 strategies.symbol → strategy_symbols 反向回填 | 老策略全部能解析到 canonical |
| M1-6 | 集成测：绑账户→symbolsync→canonical 可解析→trade_mode 正确 | e2e |

---

## M2｜OMS 状态机（2 周）

**蓝本**：AlfQ §M2 + ADR-0006（sqlc 引入）+ ADR-0012（broker adapter）

### 卡片

| ID | 内容 | 验收 |
|---|---|---|
| M2-1 | migration 091：`orders.state` + `orders.broker_symbol_raw`；老订单默认 `SUBMITTED` | schema 兼容 |
| M2-2 | 移植 `internal/oms/`：executor + state machine + repo（删 RLS） | 状态机转移合法性单测 100% |
| M2-3 | 新建 `internal/broker/{mt4,mt5}_adapter.go` 实现 `oms.BrokerAdapter` 接口 | 抽象层就绪 |
| M2-4 | 改 `strategy_schedule_runner` 走 OMS（删除直接调 broker 路径） | `grep "directly call mt4/5 from runner"` = 0 |
| M2-5 | risk_events 审计表 + Insert 链路 | 故意触发风控拒单 → state=REJECTED + risk_events 有行 |
| M2-6 | 全量回归：所有下单路径替换后 e2e 跑通 | 模拟盘 1 小时不间断 |

---

## M3｜风控引擎（1.5 周）

**蓝本**：AlfQ §M3

| ID | 内容 | 验收 |
|---|---|---|
| M3-1 | 移植 `internal/risksvc/` 8 条规则（max_position / daily_loss / drawdown / session / margin / canonical_auth / ...） | 每条规则单测 ≥ 2（通过 + 拒单） |
| M3-2 | migration：`user_risk_profiles` 表 | DDL + seed |
| M3-3 | 数据迁移：`Strategy.RiskControl JSONB` → `UserRiskProfile` | 100% 老策略迁移行为不变 |
| M3-4 | OMS executor 接入 risk engine（PreSubmit hook） | 拒单经审计 |
| M3-5 | 前端：风控偏好编辑页 | 用户可改 daily loss / max drawdown / 白名单 |

---

## M4｜AI 自然语言 → canonical 闭环（1 周）

**蓝本**：AlfQ §M4 + ADR-0007

| ID | 内容 | 验收 |
|---|---|---|
| M4-1 | `debate_v2_prompts.go` 加约束：symbol 必须命中 `canonical_symbols` 字典 | prompt diff 通过 review |
| M4-2 | `internal/service/ai_strategy_validator.go`：AI 输出 symbol 不命中则反馈重生成（≤ 3 次） | 单测：刻意污染输入 → 重试触发 |
| M4-3 | `strategy-service/app/engine/runner.py` 信号过滤：symbol 不在用户白名单 → 丢弃 + 告警 | e2e |
| M4-4 | 端到端：用户说「BTC 跌 3% 后买入」→ AI 生成 → 校验 → 实盘走 OMS | 10 条 NL 样例 100% pass |

---

## M5｜策略市场（4-6 周）

**蓝本**：AlfQ §M5（独立大模块，可并行）

### 数据模型

migration 092：`strategy_listings` / `strategy_subscriptions` / `strategy_performance_snapshots`（详见 AlfQ 计划 §5.1）

### 卡片（高层）

| ID | 内容 |
|---|---|
| M5-1 | RPC：上架 / 下架 / 订阅 / 我的订阅 |
| M5-2 | 平台冷锁回测 worker：跑足样本 → 写 `verified_metrics` + `locked_hash` |
| M5-3 | 自动 delist worker：连续 N 天回撤超阈值 → delisted + 通知订阅者 |
| M5-4 | 跟单 dispatcher：作者 strategy 信号 → fan-out 到订阅者 OMS（异步队列） |
| M5-5 | 前端：市场列表 / 详情 / 订阅按钮 / 我的订阅 / 作者收入面板 |
| M5-6 | 计费：接入支付通道（首期月租，按 D-09 决策） |

### 验收

三测试用户：A 作者、B/C 订阅者；A 发信号 → B/C 跟单成功；B 订阅期满后停止跟单。

---

## M6｜anttrader 退役与数据迁移（2 周）

| ID | 内容 |
|---|---|
| M6-1 | 写 anttrader → ant 数据迁移脚本（users / mt_accounts / strategies / orders） |
| M6-2 | ant 加「从 anttrader 导入」一键功能 |
| M6-3 | 灰度切量：邀请用户 → 全量 → anttrader 只读 → 关停 |
| M6-4 | 用户切换通告 |

---

## 全局工程纪律（执行约束）

适用于上述所有卡片：

1. 不跳步：上一里程碑未拿到 ✅ 不开下一
2. 每张卡片提交前必跑：`make verify`
3. 每个 ☑ 卡片必带：commit short-sha + 实测日志（`docs/handover/RS-final-verify.log`） + 关联 ADR
4. **复杂度分层策略**（M0.3 设立，贯穿 M1-M6）：
   - 任何 milestone 进入 Tier 1 热路径文件时，**拆拆到上限以内作为该 milestone 工作量的一部分**，不单独立项也不跳过
   - 任何卡片触碰 Tier 2（300–500 行 或 50–100 行函数）时 Boy Scout 顺手拆
   - Tier 3 永久豁免：不在热路径且未被迫修改的文件不动
5. 详细约束见 `@/opt/ant/AGENT.md` §防偷懒约束（7 条） + §复杂度硬上限（含分层豁免规则）

---

## 历史版本

- v1.0 (2026-05-23)：基于 DESIGN-REVIEW-2026-05 + AlfQ 迁移计划首版
