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
| M7 量化基础设施 | 5-7 周 | 23.5 | 🅒 pending | mthub + 因子 DSL + factorsvc + ClickHouse + quantengine（ONNX）+ research 包；沙箱降级为研究专用 | 0013-0016（待写） |

**P0 关键路径（M0.1 → M6）**：全部 ✅ 已完成（2026-05-23）。

**M7 关键路径**：M7.1 mthub → M7.2 因子 DSL → M7.3 factorsvc → M7.4 ClickHouse → M7.5 quantengine → M7.6 research 包 → M7.7 沙箱降级。

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

## M7｜量化基础设施（5-7 周）

**目标**：把 ant 从「会下单的交易系统」拉到「专业量化平台」——参照 alfq 的「研究/生产分离」哲学，引入因子 DSL、ClickHouse 时序库、ONNX 推理通道，并把 Python 沙箱降级为研究专用。

**蓝本**：
- `@/opt/alfq/backend/go/internal/{mthub,factor/dsl,factorsvc,quantengine,mdgateway}`
- `@/opt/alfq/research/alfq_research/`（data/factor/backtest/model）
- `@/opt/alfq/docs/06-Python策略沙箱设计.md`（研究/生产分离哲学）
- `@/opt/alfq/docs/09-因子DSL规范.md`（DSL EBNF 语法）

### 设计决策：沙箱去留

**结论**：保留沙箱代码，但**降级为研究专用**；生产路径切换为 DSL + ONNX。

#### 现状问题（ant）

用户编写的 Python 代码 `validate_strategy_code()` → `RestrictedPython compile` → 在 strategy-service 进程内 `exec()`：

1. **安全风险长期暴露**：AST 白名单 + RestrictedPython 是「黑名单式攻面收敛」，CVE 史上多次被绕（attribute escape、frame walking、subclass injection）。生产实盘资金 + 不可信用户代码 + 共享进程 = 永远的猫鼠游戏。
2. **无法横向扩展**：Python GIL + 进程内执行，单 strategy-service 容器同时跑 N 条策略时互相阻塞；`fork-per-execution`（ADR-0004 推荐方向 B）尚未落地。
3. **回测/实盘语义漂移**：`indicators.py` 230 行硬编码算子，跟 AI 自然语言生成的策略描述对不上，AI 生成的「ema(close, 20)」要由 prompts 翻译为 Python 代码再过沙箱，**两次损耗**。
4. **难以热更新**：策略变更需要发版重启 strategy-service；DSL/ONNX 模式只需 reload 表达式与模型文件。

#### alfq 方案优势

| 维度 | ant 现状（沙箱） | alfq 方案（DSL + ONNX） |
|---|---|---|
| 生产代码语言 | 用户 Python（不可信） | Go 解析 DSL + Go 加载 ONNX（全可信） |
| 安全攻面 | RestrictedPython + AST 白名单（持续维护） | 零（DSL 是受限文法，ONNX 是数据） |
| 性能 | Python 解释 + GIL | Go 增量算子 + onnxruntime native |
| AI 自然语言 → 策略 | NL → Python → 沙箱 exec | NL → DSL 字符串（直接可校验/执行） |
| 热更新 | 需重启 strategy-service | 直接 reload 表达式/模型 |
| 回测/实盘一致 | 双引擎易漂移 | Go DSL 引擎 + Python 研究 DSL 引擎**严格语义对齐**（alfq 已有对齐测试） |
| 多租户隔离 | 进程内共享 | 算子无状态 + 因子值按 tenant_id 写 CH |

#### 沙箱保留位置

- **保留**：`strategy-service/app/engine/sandbox.py` 作为**研究层**入口（notebook、本地回测、AI 实验），仅在用户自己的会话上下文跑，不接触实盘资金
- **不再用于**：实盘信号生成；M2 OMS 之前的「Python 直接发单」路径在 M7.7 完全切断
- **新增护栏**：`internal/connect/strategy_service.go` 增加 `production=true` 标志检测，生产模式拒绝任何 Python 代码路径

### 卡片

| ID | 内容 | 蓝本 | 验收 |
|---|---|---|---|
| M7.1-1 | 移植 `internal/mthub/`：Hub + Session + OrderEventBroker + Service（替代直调 mt4client/mt5client） | `@alfq/backend/go/internal/mthub/` | 同账户多次下单复用同一会话；symbol→bid/ask 实时缓存可被 OMS PreSubmit 取用 |
| M7.1-2 | 抽象统一 `internal/broker/gateway.go` 接口，把 mt4client/mt5client 1146 行重复代码折叠到 `mdgateway/adapter/{mt4,mt5}/client.go` 各 ≤ 80 行 | `@alfq/backend/go/internal/mdgateway/adapter/` | LOC 减少 ≥ 50%；mt4/mt5 各保留 1 份 client，trading/account/stream/subscription 通过接口调用 |
| M7.1-3 | OrderEventBroker：跨账户事件 fan-in，订阅者按 user_id 过滤 | `@alfq/backend/go/internal/mthub/events.go` | 单元测试：3 账户事件 → 1 user 订阅者收到 3 条 |
| M7.2-1 | 移植 `internal/factor/dsl/`：Lexer/Parser/AST/15 算子（SMA/EMA/RSI/BB/ATR/Corr/Cov/统计/震荡） | `@alfq/backend/go/internal/factor/dsl/` | `ema($close, 20) / ema($close, 60) - 1` 可解析+求值，与 alfq 数值对齐误差 < 1e-9 |
| M7.2-2 | 编写 `docs/spec/factor-dsl.md`：EBNF 语法 + 算子表 + 与 alfq 对齐说明 | `@alfq/docs/09-因子DSL规范.md` | 文档评审通过 |
| M7.2-3 | DSL validator：表达式合法性 + 字段白名单（$open/$high/$low/$close/$volume）+ 复杂度上限（深度 ≤ 8） | — | 单测：恶意表达式（递归/巨长）被拒 |
| M7.3-1 | 移植 `internal/factorsvc/`：engine + window_buffer + subscriber + factor_ch_writer | `@alfq/backend/go/internal/factorsvc/` | bar 流入 → 因子值出（in-process 通道 + ClickHouse 持久化） |
| M7.3-2 | migration 096：`factor_definitions` 表（id/expression/symbols/owner_user_id/active） | — | DDL + seed 5 个内置因子 |
| M7.3-3 | 因子值订阅 Connect RPC：`SubscribeFactorValues(factor_name, symbols)` | — | 前端可订阅实时因子值流 |
| M7.4-1 | 引入 ClickHouse 容器（`clickhouse:24-alpine`）+ 网络隔离 + healthcheck | `@alfq/backend/go/internal/mdgateway/clickhouse_conn.go` | `docker compose up -d clickhouse` healthy |
| M7.4-2 | 移植 `chmigrate/` 4 张时序表：`md_ticks` `md_bars` `factor_values` `signals`（按 alfq schema） | `@alfq/backend/go/internal/mdgateway/chmigrate/` | 启动后 4 表存在；写入 1000 条 tick 后查询 ≤ 100ms |
| M7.4-3 | `internal/mdgateway/runner.go`：MT 行情 → bar_aggregator → quality 校验 → CH 持久化 | `@alfq/backend/go/internal/mdgateway/runner.go` | 端到端：MT4 tick → CH `md_bars` 表行 |
| M7.4-4 | spill_replay 容错：CH 写入失败时落本地 spool，恢复后回放 | `@alfq/backend/go/internal/mdgateway/spill_replay.go` | 故障注入：CH 停 30s → 恢复后数据无丢失 |
| M7.5-1 | 移植 `internal/quantengine/`：onnx_runtime + runtime + runner + signal_oms_bridge | `@alfq/backend/go/internal/quantengine/` | ONNX 模型加载 + 推理 + 输出信号 → OMS 提单 |
| M7.5-2 | migration 097：`strategy_models` 表（id/onnx_uri/inputs/outputs/version/owner） | — | DDL + seed 1 个 demo 模型 |
| M7.5-3 | 信号 → OMS 桥接：直接走 M2 已有的 oms.Executor，附 `signal_id` 审计 | — | e2e：模型推理输出 buy 信号 → OMS 状态 SUBMITTED |
| M7.6-1 | 新建 `research/` Python 包（uv + pyproject.toml + ruff strict） | `@alfq/research/` | `cd research && uv run pytest` 通过 |
| M7.6-2 | `research/ant_research/factor/dsl/`：Python 端 DSL 引擎（与 Go 端语义对齐，每个算子单测） | `@alfq/research/alfq_research/factor/dsl/` | Go/Python 对齐测试：100 个表达式 + 1000 bar，最大误差 < 1e-9 |
| M7.6-3 | `research/ant_research/model/`：trainer.py（LightGBM/sklearn）+ exporter.py（ONNX 导出 + MinIO/本地 upload） | `@alfq/research/alfq_research/model/` | 训练 demo → 导出 → quantengine 加载推理一致 |
| M7.6-4 | `research/ant_research/backtest/`：vectorized.py（polars 向量化） + event.py（事件驱动）+ broker_sim.py | `@alfq/research/alfq_research/backtest/` | 同一 DSL 因子在向量化/事件驱动下结果一致 |
| M7.6-5 | DataClient：从 ClickHouse 拉历史 bar/tick/factor_value（只读，按 user_id 过滤） | `@alfq/research/alfq_research/data/` | notebook 一行 `client.bars('BTCUSD', '1d')` 拿数据 |
| M7.7-1 | 沙箱降级：`sandbox.py` 加 `production_mode` 检测，生产模式直接 raise；仅在 `research_mode` 可用 | — | 实盘下单链路 grep 无 sandbox 调用 |
| M7.7-2 | strategy-service 路由分裂：`/research/*` 走沙箱，`/production/*` 仅接受 DSL/model_id 引用 | — | OpenAPI 契约 + 集成测试 |
| M7.7-3 | 老 Python 策略迁移工具：`migrate_legacy_strategy.py` 把现有 `strategies.code`（Python）转 DSL（能转的）+ 标记 unconvertible | — | 50% 内置策略可全自动转 DSL；剩余给出迁移 issue |
| M7.7-4 | AI 生成策略契约切换：`ai_strategy_validator.go` 输出从 Python code → DSL 字符串 + ONNX URI（可选） | M4 已有验证器扩展 | 10 条 NL 样例 → 100% 输出 DSL，0 输出 Python |

### M7.8 — 装配与收尾（迁移评估发现，~10-15 工日）

> **背景**：`85694a1`/`c0a5984`/`e719ff8`/`42921c3` 已把 mdgateway/mthub/factorsvc/factor/research/quantengine 包从 alfq 移植完成且 `go build`/`go test` 全绿。但**包仍是孤岛**——server 启动钩子未调用、ConnectRPC 未暴露、老 `kline_service` 仍在生产路径。M7.8 负责把这些代码真正接入运行时。

#### P0 装配（断点修复）

| ID | 内容 | 蓝本 | 验收 |
|---|---|---|---|
| M7.8-1 | 编写 `internal/mdgateway/runner.go`（等价 alfq `RunGateway`）：从 PG 加载 `mt_accounts` → `manager.AddGateway` → `Connect` → `Subscribe` → 装配 publisher+CHWriter+SpillWriter | `@alfq/.../mdgateway/runner.go` | 启动后 `docker logs ant-backend` 出现 "accounts loaded from PG"，至少 1 个账户 Connected |
| M7.8-2 | `internal/server/start_mdgateway.go`：在 server 启动 hook 调用 `chmigrate.Run(ctx, ch, log)` + `mdgateway.RunGateway(ctx, deps)` | 同上 | `clickhouse-client --query 'SHOW TABLES FROM ant'` 列出 md_ticks/md_bars/factor_values/signals |
| M7.8-3 | `proto/mthub/v1/mthub.proto`：移植 alfq 的 9 个 RPC（EnsureSession/CloseSession/OrderSend/OrderClose/OrderHistory/OpenedOrders/SymbolParamsMany/PriceHistory/SubscribeOrderEvents） | `@alfq/backend/proto/alfq/mthub/v1/mthub.proto` | `make proto` 生成 `gen/proto/mthub/v1/*.connect.go` |
| M7.8-4 | `internal/connect/mthub_service.go`：ConnectRPC handler 把 `*connect.Request` 转 `mthub.MtHubService` 内部签名 | — | 前端可 `client.ensureSession({ accountId })` 拿到 sessionID |
| M7.8-5 | `.env.example` 补 `CH_HOST/CH_PORT/CH_USER/CH_PASSWORD/CH_DATABASE`；`config/config.go` 加 `ClickHouseConfig` | — | `make env-check` 不再缺 CH 项 |
| M7.8-6 | `001_md_ticks.sql` 加回 TTL（90 天）；写 ADR-0014 说明 PG vs CH 边界与 TTL 策略 | `@alfq/.../chmigrate/001_md_ticks.sql` | `SHOW CREATE TABLE md_ticks` 含 `TTL ... INTERVAL 90 DAY` |

#### P1 切流（消除双数据源）

| ID | 内容 | 验收 |
|---|---|---|
| M7.8-7 | `connect/market_service.go`：K 线查询从 `kline_service` 切到 mdgateway（先读 CH md_bars，回退 PG kline_data） | 查询 BTCUSD 1m 1000 根，CH 命中率 ≥ 90% |
| M7.8-8 | `connect/{market_regime,backtest_dataset,python_strategy}_service.go`：同上切流 | 4 个 endpoint 全部走 CH 路径 |
| M7.8-9 | `internal/service/kline_service*.go` 全部加 `// Deprecated: see internal/mdgateway` + 计划下线（M8） | grep 验证；保留只读路径 |
| M7.8-10 | `factorsvc/subscriber.go` 订阅 NATS `md.bar.*` 主题（替代 in-process 通道） | NATS subject 计数 = 因子计算次数 |

#### P1 测试补全

| ID | 内容 | 验收 |
|---|---|---|
| M7.8-11 | 补 `mdgateway/{publisher_test,clickhouse_writer_test,spill_replay_test,runner_test}.go` 4 个文件 | 4 路径覆盖率 ≥ 60% |
| M7.8-12 | 补 `mdgateway/chmigrate/migrate_test.go`（用 dockertest 起临时 CH） | CI 通过 |
| M7.8-13 | OrderEventBroker 跨账户 fan-in e2e 测试：3 账户事件 → 1 user 收 3 条（M7.1-3 验收） | `events_test.go` 加该用例 |

#### P1 历史回填

| ID | 内容 | 蓝本 | 验收 |
|---|---|---|---|
| M7.8-14 | 扩展 `mdgateway/backfill/backfill.go`：除 PG→CH 拷贝外，新增 MT5 QuoteHistory 直拉路径（按 1m/5m/15m/1h/4h/1d/1w/1M） | `@alfq/.../mdgateway/backfill/backfill.go` | 一个账户回填近 30 天 BTCUSD 1m → CH md_bars 行数 ≥ 43000 |

#### P2 可观测增强

| ID | 内容 | 验收 |
|---|---|---|
| M7.8-15 | `quality.go`：把 dropped 原因细分为 `bid_gt_ask / outlier / gap`，作为 Prometheus label | Grafana 看板能区分三类丢弃 |

### 退出标准

- `mthub` 接管所有 MT 账户连接；mt4client/mt5client 不再被 server/service/connect 直接 import
- ClickHouse 启动后 1 小时持续写入 tick/bar/factor_value，无 spill
- 至少 1 个 ONNX 模型（demo 动量策略）端到端跑通：训练 → 导出 → quantengine 推理 → OMS 提单
- 实盘下单链路完全无 Python 代码执行（grep 验证）
- `research/` 包独立可用：`cd research && uv run jupyter lab` 可跑 notebook
- Go DSL 与 Python DSL 数值对齐误差 < 1e-9
- AI 生成策略 100% 输出 DSL（不再生成 Python）

### 待写 ADR

- **ADR-0013 因子 DSL 规范**（Go/Python 双引擎语义对齐契约）
- **ADR-0014 ClickHouse 时序存储**（PG 业务库 vs CH 时序库职责边界）
- **ADR-0015 ONNX 推理通道**（模型版本管理 + signal→OMS 审计）
- **ADR-0016 沙箱降级**（生产路径排除 Python 的决策与回退方案）

---

## 历史版本

- v1.0 (2026-05-23)：基于 DESIGN-REVIEW-2026-05 + AlfQ 迁移计划首版
- v1.1 (2026-05-23)：M0.1-M6 全部 ✅；新增 M7 量化基础设施（mthub + DSL + ClickHouse + ONNX + 沙箱降级）
- v1.2 (2026-05-23)：M7 包移植完成（85694a1/c0a5984/e719ff8/42921c3）；新增 M7.8 装配与收尾 15 张卡片，把孤岛包接入运行时 + 切断老 kline_service 双数据源
