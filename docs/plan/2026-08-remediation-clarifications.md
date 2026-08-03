# AlphaForge 整改方案 — 待澄清问题清单

> 来源：通读 `2026-08-remediation-plan.md` 后提出，按 Phase 排列。
> 请编写者逐条回复，回复后标记 ✅。

---

## Phase 0

### 0.1 备份
- [x] ✅ **对象存储选型**：选 **Cloudflare R2**。项目已在 Cloudflare 生态（tunnel），且 R2 无出口流量费；用 S3 兼容 API + `rclone`（cron 里最简单，无需 SDK 依赖）。
- [x] ✅ **热钱包助记词备份**：接受降级。先执行一次链上余额核查（`solvency-check`），若余额为 0 → 移到 Phase 4 前置条件；若已有余额 → 保持 P0。

### 0.2 mtapi 降级
- [x] ✅ **VM 断连检测机制**：现有 `sdk.Broker` 接口（`backend/strategy/sdk/broker.go`）无连接状态语义，**需新增**。方案：在 mthub 会话层维护熔断状态（连接断开/RPC 连续失败即 open），LiveRunner 持有该状态，在调用 `Broker` 下单方法前检查——即门禁做在 **runner 层**，不改 VM/bytecode，不改 Broker 接口签名。熔断状态变更事件进 Prometheus。
- [x] ✅ **`position_snapshot` 复用**：经查 migrations 中**无此表**——`positionSnapshot` 是 SSE 事件概念（内存态），非 PG 表。计划原文表述有误，需**新建表** `mt_position_snapshots(account_id, payload_proto BYTEA, captured_at TIMESTAMPTZ)`，由 mthub 事件管道按节流（如 30s）落库，断连时前端读最后一条并展示 `captured_at`。

### 0.3 覆盖率
- [x] ✅ **`internal/oms` 路径确认**：存在，`backend/internal/oms/`（含 `broker.go`、`statemachine.go`、`fill_model.go`、`pnl_calculator.go` 等，且已有部分 `_test.go`）。覆盖率闸门路径按 `backend/internal/oms` 配置。

### 0.4 仓库卫生
- [x] ✅ **`reference/` 去向**：**直接从 git 移除 + 本地保留**（`git rm -r --cached reference/` + `.gitignore`）。不建独立 repo（第三方源码再托管仍有许可证问题），不用 submodule（增加复杂度）。需要参考时本地磁盘仍在。

---

## Phase 1

### 1.1 配额
- [x] ✅ **免费档配额起步值**：**每日 5 次生成会话 + 每日 200K token（含输入输出）**。做成 `admin_config` 可调项，公测第 2 周按实际成本数据复核，不写死。
- [x] ✅ **全平台日成本熔断阈值**：起步 **$50/日**（按厂商成本价累计计算）。超阈值 → 平台 key 通道降级为 BYO-only + 管理员告警。同样进 `admin_config` 可调。

### 1.2 5 语言适配
- [x] ✅ **非英语 locale 推理质量**：采纳建议的 fallback 架构——**内部推理/工具调用固定英文，仅面向用户的输出文本（策略说明、澄清问题、回测解读）按 locale 生成**。即系统提示词加一条输出语言指令，而非整体切换提示词语言。这同时降低 5 语言提示词维护成本。
- [x] ✅ **MQL 注释语言**：tree-sitter 注释节点是原样字节流，不参与语义解析，理论上无影响；但为消除风险，**改为 MQL 代码内注释统一英文**，locale 化的说明放在代码块外的解释文本中。需为 `ja`/`vi`/`zh` 注释源码各加 1 条编译回归测试兜底（防用户粘贴带非 ASCII 注释的代码）。

---

## Phase 2

### 2.1 计费模型
- [x] ✅ **credit 锚定值**：**1 credit = $0.01**（美分级）。粒度足够细避免小数展示，充值面额整数化（如 500 credits = $5）。DB 存 `NUMERIC(20,8)`，展示层取整。
- [x] ✅ **预扣估算算法**：起步用**固定预扣**（= 该模型单轮会话 P90 历史成本，冷启动时用人工估的保守值），每轮工具调用后增量结算、会话结束多退少补。滚动平均优化留到有数据之后，不在首版实现。

---

## Phase 3

### 3.1 采集 schema
- [x] ✅ **Agent 全轨迹存储策略**：**proto bytes 存 `BYTEA`**。理由：符合平台 proto-only 规则本意；轨迹是只写不改的事件日志，无需 SQL 查询内部字段（分析时批量导出反序列化即可）；定义 `AgentTrajectoryEvent` proto（oneof: tool_call / tool_result / user_feedback / backtest_metrics）。可查询的聚合指标（成功率、token 数、locale）另拆**关系列**存同表，兼顾分析。不用 JSONB（对嵌套 proto 结构无查询优势且违反规则精神）。

---

## Phase 4

- [x] ✅ **credits 退款政策**：需提前声明。政策：**未消费的充值 credits 可在 7 日内人工退款（原路退回），已消费部分不退；赠送/免费额度不可退**。写入充值页面条款，Phase 2 人工充值阶段即生效（人工充值人工退，无需开发）。

---

## 时间线

- [x] ✅ **W3 非硬 deadline**：时间线是相对排序不是日历承诺。规则以计划正文为准：**Phase 0 全部验收通过是 Phase 1 的硬前置**，Phase 0 超期则 Phase 1 顺延，不允许并行抢跑。
- [x] ✅ **单人带宽冲突**：接受此风险并给出优先级规则——**公测期 P0/P1 bug > Phase 2 开发 > 一切其他**。Phase 2 上线时点允许从 W10 顺延（人工充值模式本身工作量小，是弹性最大的一段）；若持续被 bug 挤占，信号本身说明 Phase 0 质量不足，应回头补测试而非硬推计费。

---

## 附录 A

- [x] ✅ **数字验证**：已实测（git ls-files + wc）。非测试非生成 Go 代码 **153,730 行**；proto 文件 **142 个**（含 i18n），**排除 i18n 后业务 proto 140 个、7,840 行**；migrations 全量 **7,220 行**。业务 proto + schema ≈ **1.5 万行**，略超附录 A 的 "<1 万行" 表述，但 migrations 大半是历史增量（有效 schema 远小于 7,220 行），"人力可覆盖" 结论成立。附录 A 表述已按此口径理解，无需改动结论。

---

# 第二轮审阅（2026-08-03，基于修订版计划 + 第一轮澄清）

> 修订版计划新增了 Phase 0.5、盈利路径/止损线、商业模型调整等内容。
> 以下分两类：**A. 编辑性不一致**（计划正文未同步澄清决策，标注位置供编写者修正）、**B. 新问题**（需编写者决策）。

---

## A. 编辑性不一致（计划正文 vs 澄清决策）

以下 11 处计划正文仍为旧表述，需同步为第一轮澄清的决策：

| # | 计划位置 | 当前正文 | 应同步为 |
|---|---------|---------|---------|
| A1 | §0.1 L27 | "Backblaze B2 / Cloudflare R2" | **Cloudflare R2**（澄清 0.1） |
| A2 | §0.1 L29 | "热钱包助记词离线备份"（无条件 P0） | **条件性 P0**：先 solvency-check，余额为 0 则降级 Phase 4 前置（澄清 0.1） |
| A3 | §0.2 L37 | "VM 策略在断连期禁止开新仓" | **LiveRunner 层门禁**，非 VM 层（澄清 0.2） |
| A4 | §1.1 L71 | "N 次生成会话/日 + 总 token 上限" | **每日 5 次 + 200K token**（澄清 1.1） |
| A5 | §1.1 L74 | "$X/日" | **$50/日**（澄清 1.1） |
| A6 | §2.1 L103 | "固定法币锚定值" | **1 credit = $0.01**（澄清 2.1） |
| A7 | §2.2 L110 | "预扣估算 credits" | **固定预扣**（P90 历史成本，冷启动人工估值）（澄清 2.1） |
| A8 | §3.1 L126 | 未提及存储策略 | 补充 **proto bytes 存 BYTEA** + 关系列拆分（澄清 3.1） |
| A9 | §4 L158 | 未提及退款政策 | 补充 **7 日内未消费可退**（澄清 Phase 4） |
| A10 | §验收标准 L198 | "5 语言各 3 条 E2E 通过" | **P0 仅 zh-cn + en 各 3 条**（修订版 §1.2 L78 已将 P0 缩为 2 语言，验收标准未同步） |
| A11 | §附录A L220 | "proto + schema <1 万行" | **~1.5 万行**（澄清附录 A），结论不变 |

> ✅ **A1–A11 已全部同步进计划正文（2026-08-03 负责人修订）**。其中 A10 注意：验收标准改为“已启用语言各 3 条 E2E”，随语言启用进度动态适用。

---

## B. 新问题（需编写者决策）

### B1. `ai_token_usage` 表已存在 — 计划提议新建 `ai_usage_ledger` 冲突

**现状**：migration 015 已建 `ai_token_usage` 表，`AITokenUsageRepository`（`backend/internal/repository/ai_gateway_repository.go`）已在用。现有字段：`id, user_id, wallet_transaction_id, paid_by, provider_id, model_name, feature, input_tokens, output_tokens, cost, created_at`。

**计划 §1.1 提议新建**：`ai_usage_ledger(user_id, session_id, model, prompt_tokens, completion_tokens, cost_decimal, ts)` — 字段不同（多了 `session_id`，命名不同 `prompt_tokens` vs `input_tokens`）。

**需决策**：扩展现有 `ai_token_usage` 表（加 `session_id` 列 + 改名对齐），还是新建 `ai_usage_ledger` 并迁移？现有表已被 `SubscriptionService` 和 `AITokenUsageRepository` 引用，新建意味着两套并存或迁移成本。

- [x] ✅ **决策**：**REUSE：`ai_token_usage` @ migration 015 + `AITokenUsageRepository`。仅加 `session_id` 列（nullable，新迁移），不改现有列名**（`input_tokens`/`output_tokens` 保留，计划里的 `prompt_tokens` 命名仅为示意），不新建表。计划 §1.1/§2.2 正文已同步。这正是 Reuse Preflight 该抓的——干得好。

### B2. Credits 与订阅分层的关系未定义

**背景**：修订版 §0 商业模型将"订阅分层"定为**主线收入**（$50–200/月，Phase 4），Phase 2 引入 credits 作为 token 差价机制。但两者关系未说明。

**需决策**：Phase 4 订阅用户是否包含 credits 额度？还是订阅费 + credits 独立结算？这影响 Phase 2 的 `credit_accounts` schema 设计（是否需要 `subscription_plan_id` 关联列）。

- [x] ✅ **决策**：**订阅档位每月赠送 credits 额度（当月有效，不结转），超额从充值余额扣**。`credit_accounts` **不加** `subscription_plan_id` 列——赠送以 `credit_transactions.source='subscription_grant'` 记录，账户与订阅解耦（Phase 2 不依赖 Phase 4 的订阅表存在）。消耗顺序：赠送额度先扣、充值余额后扣。计划 §2.1 已同步。

### B3. Phase 0.5 内测失败后的用户 sunset 方案

**背景**：Phase 0.5 门槛"≥50% 完成生成→回测，≥3 人挂实盘"。若未达标 → "不进公测，先修产品方向"。但此时已有用户挂着实盘策略。

**需决策**：内测失败时，已挂实盘用户的策略如何处理？是否需要提前告知"内测可能终止"？是否需要 sunset 流程（通知 + 协助关闭 + 数据导出）？

- [x] ✅ **决策**：需要。① 内测协议提前声明“功能可能调整或终止”；② 若转向：**14 天通知 + 协助停止策略/平仓确认 + 数据导出**，期间实盘策略不强制立即停止但停止新挂载。注意：转向 ≠ 关站，大概率是产品形态调整，实盘基础设施（mthub/runner）在任何转向中都保留。计划 Phase 0.5 已同步。

### B4. Phase 0.5 "每周与至少 5 名用户直接对话"的时间预算

**背景**：单人全栈，Phase 0.5 持续 4 周，每周 5 名用户深度对话（按 30–60 分钟/人 = 2.5–5 小时/周）。同时需要修 bug、处理反馈。

**需决策**：这个时间是否纳入工时预算？还是 aspirational 目标？如果某周做不到 5 人，是否影响门槛判定？

- [x] ✅ **决策**：纳入工时预算（2.5–5h/周，内测期开发任务相应减量）。5 人/周是目标值，**最低 3 人**；访谈数**不影响门槛判定**——门槛只看用户行为指标（闭环完成率/实盘挂载数），访谈是归因手段不是考核指标。计划 Phase 0.5 已同步。

### B5. 澄清文档 §1.2 标题仍为"5 语言适配" — 与修订版计划 P0 缩为 2 语言不一致

**背景**：修订版计划 §1.2 L78 已将 P0 缩为 zh-cn + en，`zh-tw`/`ja`/`vi` 降 P1。但澄清文档 §1.2 的标题和回答仍按 5 语言表述（如"需为 `ja`/`vi`/`zh` 注释源码各加 1 条编译回归测试"）。

**需确认**：澄清文档 §1.2 的决策是否仍然有效？还是仅适用于 P1 阶段？P0 阶段是否仍需 ja/vi/zh 的编译回归测试？

- [x] ✅ **确认**：§1.2 的两条架构决策（英文推理 + locale 输出、MQL 注释英文）**仍然有效且 P0 就要实现**（架构一次做对，扩语言只加测试）。**ja/vi/zh 非 ASCII 注释编译回归测试保留在 P0**——它防的是用户粘贴带非 ASCII 注释的代码（与 UI 语言无关，zh-cn 用户粘中文注释 EA 是常态），成本 3 个小测试。仅“每语言 3 条 E2E”随语言启用进度动态适用（P0 只需 zh-cn+en）。

### B6. `strategy/runner` + `strategy/sdk` 零测试 — 1.5 周内四模块同时达标 70% 是否现实

**背景**：修订版计划 §0.3 L42 新增警告：`strategy/runner` 和 `strategy/sdk` 当前**零测试文件**（已验证：`find strategy/runner -name '*_test.go'` 和 `strategy/sdk` 均返回空）。而 `strategy/backtest` 有 4 个测试文件、`internal/oms` 有部分测试。

**问题**：四个模块中两个从零开始，1–1.5 周内全部达到 70% 覆盖率，时间是否够？是否需要调整优先级（如先 runner + oms 达标，sweep/hdwallet 延后）或拉长 Phase 0？

- [x] ✅ **决策**：不现实，已拆为两档（计划 §0.3 已改）：**0.3a（阻断 Phase 0.5）= `strategy/backtest` + `strategy/runner` + `internal/oms`，允许 Phase 0 延长至 2.5–3 周**；**0.3b = `sweep`/`hdwallet` 归入 Phase 4 前置（代码冻结期不阻断公测），计费覆盖归入 Phase 2 开发（新代码随写随测）**。理由：公测期碰钱的只有实盘执行路径；钱包代码冻结不运行，风险后置是合理的。

---

# 第三轮审阅（2026-08-03，基于第二轮澄清后的修订版）

> 第二轮 B1–B6 已全部答复 ✅，计划正文已同步。以下为同步后仍残留的问题。

---

## C. 实质问题（3 条）

### C1. Phase 0 时长三处矛盾

**现状**：计划正文三处对 Phase 0 时长表述不一致：
- L22 标题："预计 1.5–2 周"
- L44 正文："允许 Phase 0 为此延长至 2.5–3 周"（B6 决策）
- L172 时间线："W1–W2"（仅 2 周）

B6 决策明确允许延长到 2.5–3 周，但 L22 标题和 L172 时间线未同步。

- [x] ✅ **已修正**：Phase 0 标题改为“预计 2–3 周”，§0.3 标题改为“1.5–2.5 周，Phase 0 工期主变量”；时间线整体后移一周（Phase 0 = W1–W3，0.5 = W4–W7，公测 W8 起），止损线检查点同步顺延（W7/W11/W15/W19），并加注“周数为相对排序，以前一 Phase 验收为前提”。

### C2. L86 "澄清问答 5 语言适配"与 P0=2 语言矛盾

**现状**：L83 明确 P0 只做 zh-cn + en，但 L86 仍写"澄清问答（`internal/ai/clarification`）5 语言适配"。按 B5 决策（架构 P0 做对，内容随语言启用），P0 不应做满 5 语言内容。

- [x] ✅ **已修正**：改为“架构参数化支持 5 语言，P0 实现 zh-cn + en，其余随语言启用补充”。

### C3. `strategy/sdk` 零测试但不在任何覆盖率闸门中

**现状**：L44 警告 `strategy/sdk` 零测试，但 0.3a 只要求 `strategy/backtest` + `strategy/runner` + `internal/oms`，0.3b 只含 `sweep`/`hdwallet`。`strategy/sdk` 是 VM 与交易层的接口契约（`Broker`/`IndicatorSet`/`Strategy` interface），零测试且无覆盖率要求。

**需确认**：这是 intentional（通过 runner 集成测试间接覆盖 sdk 接口）还是遗漏？如果 intentional，建议在计划中注明"sdk 通过 runner 集成测试间接覆盖"；如果遗漏，是否加入 0.3a？

- [x] ✅ **确认**：部分遗漏，已修正。实测 `strategy/sdk` 共 790 行，大部分是纯接口定义（`Broker`/`IndicatorSet`/`Strategy`，无可测语句），但含 **~21 个具体函数**：`language.go`（语言检测启发式，路由入口，错了会把 MQL 当 Python 编译）、`series.go`（Bar 序列访问）、`SpreadDecimal`。**具体逻辑部分加入 0.3a ≥ 70%**（工作量小，半天内），纯接口经 runner 集成测试间接覆盖——计划 §0.3a 已加条目并注明。

---

## D. 编辑性小问题（3 条，可直接修正）

| # | 计划位置 | 当前正文 | 应改为 |
|---|---------|---------|--------|
| D1 | L4 | "16 条已全部答复 ✅" | **22 条**（16 第一轮 + 6 第二轮） |
| D2 | L200 | "1.1 的 ledger" | **`ai_token_usage`**（B1 决策：不新建 ledger，复用现有表） |
| D3 | L165 | 退款政策列在 Phase 4 启用清单中 | 移到 **Phase 2 §2.2 实现清单**（正文已注明"Phase 2 人工充值阶段即生效"，放在 Phase 4 清单中易被漏看） |

> ✅ **D1–D3 已全部修正**：L4 改为“三轮共 25 条”（含第三轮 C1–C3）；风险表改为“`ai_token_usage` 扩展”；退款政策移入 §2.2，Phase 4 处改为引用。
