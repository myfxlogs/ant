# ADR-0025 · Agent-Native 交互体验与自我进化设计

- **状态**：Accepted
- **日期**：2026-07-03
- **决策者**：人类负责人
- **关联 ADR**：ADR-0024（Agent 架构，本 ADR 扩展 Phase 4）
- **参考**：Claude Code（分层配置/三层记忆/Plan Mode/权限模型/Hooks）、Hermes Agent（Nudge Engine 自进化闭环）、OpenClaw（子 Agent 编排/模型降级/push-based 结果）

---

## 1. 背景

ADR-0024 解决了"Agent 怎么跑"——双编译前端、单 VM、Agent 循环。它定义了执行层面的架构，但没有覆盖：

- Agent 如何跟用户交互（非程序员用户如何参与策略设计）
- Agent 如何跨会话积累经验（不只是 pgvector 表结构，是用户体验层面的内存模型）
- Agent 如何被管理和约束（平台 Admin、租户 Admin、交易员的分层权限）
- Agent 如何自我进化（不是被动的"存一条经验"，是主动的"这件事值得学"）

Claude Code、Hermes Agent、OpenClaw 三个成熟 Agent 框架在各自的领域给出了经过验证的设计。本 ADR 从三者中提取适用的设计模式，聚焦到策略 Agent 的领域，定义我们的交互体验、自我进化和管理层架构。

---

## 2. 三层借鉴：取什么、砍什么

### 2.1 从 Claude Code 取

| 取 | 理由 |
|---|---|
| Plan Mode（先出方案，用户确认，再生成代码） | 非程序员用户参与策略设计的唯一入口。proto 已有 `planning` phase，只缺前端交互 |
| 三层记忆系统（全局领域知识 + 用户手写 + Agent 自动写） | "Agent 越来越懂我"的核心。平台运营提供领域知识地板，用户偏好 + Agent 经验提供个性化成长 |
| 分层设置 5-tier（Managed > Tenant > User > Session > Default） | SaaS 商业化的前提。管理员能设熔断，租户能设白名单，用户不能覆盖 |
| Capability 权限引擎（Allow/Deny + glob 匹配 + Deny 优先） | 替换硬编码的风控检查为声明式规则 |
| 领域知识注入（平台运营维护种子知识，Agent 自动按 scope 匹配注入） | 弱模型的策略基础知识补丁——GLM/GPT-4o-mini 等预训练语料不足的模型也能生成靠谱策略 |
| 5 个关键生命周期钩子（Pre* 可阻断） | 实盘安全网——上线前必须过风控门，失败就阻断 |
| 模型降级链 | 用户选的模型挂了自动切备选，不中断策略生成 |
| Fail-closed 安全 | Admin 配置解析失败 = 锁定，不是开放 |
| 热重载 + 记忆管理页面 | Admin 改配置零停机；用户可浏览 Agent 学到的东西 |

| 砍掉 | 理由 |
|---|---|
| 多通道适配器（WhatsApp/Slack/Discord） | 策略 Agent 只有一个入口（前端 SSE），不需要 |
| `.claude/rules/` 按文件路径 scope | 我们按品种/周期/策略类型 scope，领域知识注入（§4.5）按同一维度匹配 |
| `AGENTS.md` 兼容 | 我们没有历史包袱 |
| `claudeMdExcludes` monorepo 排除 | 不适用 |
| Skills 用户创建 + 社区市场 | 用户不会写也不会调用 Skills，不需要一个用户可见的技能系统。领域知识由平台运营一手维护，Agent 透明注入，无用户输入即无恶意注入风险 |

### 2.2 从 Hermes Agent 取

| 取 | 理由 |
|---|---|
| Nudge Engine 原理（事后复盘 → 抽取经验 → 写入记忆） | Agent 自我进化的核心机制 |
| 记忆压缩原则（索引 < 200 行，详细数据存主题文件） | 防止 prompt 膨胀 |

| 砍掉 | 理由 |
|---|---|
| 通用 Nudge Engine（每次对话后都复盘） | 成本高（每次额外 LLM 调用）、范围散（什么都记）。聚焦到策略领域：只在策略生成流程结束后触发 |
| DSPy prompt 自动优化 | 需要几百次会话数据才能生效，平台未上线没数据可喂 |
| 技能自动生成 | 风险高——Agent 自己判断"这个模式值得固化成 Skill"易产生垃圾，用户手动保存策略更可靠 |
| SQLite + FTS5 存储 | 我们已有 pgvector，不引入新存储 |

### 2.3 从 OpenClaw 取

| 取 | 理由 |
|---|---|
| 模型降级链（primary → fallback → fallback） | 生产可靠性——LLM API 挂了策略生成不能中断 |
| push-based 结果返回 | 多个分析（画像 + 解读 + 回溯）并行，谁先完谁先汇报 |

| 砍掉 | 理由 |
|---|---|
| 多通道适配器 | 同上 |
| 子 Agent `steer`（父 Agent 中途重定向子 Agent） | 实现成本高，Phase 1-2 的 3 次重试 + 编译错误反馈已够用。Phase 3 再评估 |
| 任务队列 + 优先级 | Phase 1 是无状态子进程，队列是 Phase 2 长驻服务的事 |
| 99.9% 可用性目标 | 不需要——策略生成是 on-demand 的，不是 7×24 |

---

## 3. Plan Mode 交互设计

### 3.1 流程

```
用户: "帮我做EURUSD H1趋势跟踪"
  │
  ▼
Agent: [planning phase] 不出代码, 先生成策略方案
  │
  ▼
  ┌─────────────────────────────┐
  │  策略方案卡片 (前端)          │
  │                             │
  │  类型: 双均线 Crossover      │
  │  入场: EMA(10)↑EMA(30)       │
  │       + ADX(14)>25          │
  │  出场: EMA(10)↓EMA(30)       │
  │       + ATR(14)×2 移动止损   │
  │  风控: 2%风险/笔, RR 1:2     │
  │  适用: 趋势市, H1             │
  │                             │
  │  [确认生成] [修改后再生成]     │
  │  [换一个方案]                 │
  └─────────────────────────────┘
  │
  ▼ (用户点"修改")
输入框激活: "把RSI过滤也加上, 入场要求RSI<70"
  │
  ▼
Agent 更新方案, 不出代码
  │
  ▼
用户确认 → Agent 生成 Python 子集 → 编译 → 回测 → 结果
```

### 3.2 实现要点

`GenerateStrategy` RPC 的 `planning` phase 已经存在。前端改动：

- `phase=planning` 时渲染策略方案卡片（非纯文本）
- 卡片结构化内容从 `AgentGenerateStrategyChunk.plan` 字段解析（LLM 输出轻量行格式）
- Plan 格式约定：

```
TYPE: dual_ema_crossover
ENTRY: EMA(10) cross above EMA(30) AND ADX(14) > 25
EXIT: EMA(10) cross below EMA(30) OR trailing_stop(ATR(14)*2)
RISK: 2%_per_trade, RR 1:2
MARKET: trending, ADX>25, H1
```

Go 端行解析器消费 → 前端渲染为卡片 UI。

### 3.3 Web 原生交互（非终端命令）

Claude Code 使用 `/design`、`/memory` 等斜杠命令——这是终端工具的交互惯例。我们的平台是网页前端，用户不会在聊天框里敲 `/`。

**交互原则：每个用户意图都由一个可见的 UI 元素承载。**

| 用户意图 | 终端 (Claude Code) | 网页 (Ant) |
|---|---|---|
| 设计策略 | `/design` | 聊天框旁的**"策略设计"按钮**，或直接自然语言输入 |
| 查看记忆 | `/memory` | 侧边栏/顶部导航的**"记忆管理"入口** |
| 调整偏好 | `/profile` | 设置页面的**"策略偏好"标签页** |
| 切换模型 | `/model` | 聊天框顶部的**模型选择下拉框** |
| 保存模板 | `/save` | 记忆管理页面中的 **[+ 新增偏好]** 按钮 |

**Plan Mode 不需要命令触发。** 用户说"帮我做趋势跟踪"→ Agent 自动进入 planning phase → 前端渲染方案卡片。触发方式是自然语言+上下文，不是用户主动输入 `/design`。

---

## 4. 三层记忆系统

### 4.1 三层记忆模型

| | 全局领域知识 | 用户策略模板 | Agent 自动记忆 |
|---|---|---|---|
| 谁写 | 平台运营（手工） | 用户（手动保存） | Agent（自动，策略流程结束时） |
| 存储 | `domain_knowledge` 表 | `user_strategy_templates` 表 | `agent_experience` 表（ADR-0024 §5.5） |
| Scope | 全局共享 | 按租户隔离 | 按租户隔离 |
| 注入时机 | 会话启动，按 scope 匹配 | 会话启动 | 会话启动，知识库检索 |
| 管理方式 | Admin 管理端编辑/上下架 | 用户可编辑、删除 | 用户可查看、删除（只读，Agent 写） |

### 4.2 存储格式

全局领域知识：

```sql
CREATE TABLE domain_knowledge (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content     TEXT NOT NULL,              -- 自然语言, 3-5 句
    scope       JSONB NOT NULL DEFAULT '{}', -- {symbols, timeframes, strategy_families}
    status      VARCHAR(16) NOT NULL DEFAULT 'active', -- active | archived
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

平台运营在 Admin 管理端编辑。Agent 启动时检索 `status='active'` 的条目，按 scope 匹配当前会话上下文，全量注入 LLM prompt（~10-20 条，每条 3-5 句，总量可控）。

用户策略模板：

```sql
CREATE TABLE user_strategy_templates (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL,
    name       VARCHAR(128) NOT NULL,      -- e.g. "我的EURUSD偏好"
    content    TEXT NOT NULL,              -- 自然语言描述
    scope      JSONB NOT NULL DEFAULT '{}', -- {symbols: ["EURUSD"], timeframes: ["H1"]}
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Agent 自动记忆：

沿用 ADR-0024 §5.5 的 `agent_experience` 表（含 `fingerprint` 列和双路径检索）。新增 MEMORY.md 模式：

```
~/.ant/projects/<tenant>/memory/
  MEMORY.md              ← 经验索引（每次会话加载, 前 200 行/25KB）
  eurusd-h1-patterns.md  ← EURUSD H1 策略模式
  risk-models.md          ← 风控模型经验
  optimization-log.md     ← 参数优化历史
```

`MEMORY.md` 是 `agent_experience` 的物化视图——Agent 写入 `agent_experience` 表后，异步更新 `MEMORY.md` 索引。会话启动时先加载 `MEMORY.md` 索引，语义检索按需加载 topic 文件。

### 4.3 会话启动注入

```
新会话启动
  │
  ▼
1. 加载全局领域知识（status=active, 按 scope 匹配当前品种/周期/策略类型）
  │
  ▼
2. 加载用户策略模板（按 tenant_id + scope 匹配当前品种/周期）
  │
  ▼
3. 加载 MEMORY.md 索引（前 200 行）
  │
  ▼
4. 三者合并注入 LLM system prompt:
  "领域知识: ...
   用户偏好: ...
   历史经验: ..."
  │
  ▼
Agent 带着"记忆"开始服务
```

### 4.4 记忆管理页面

前端新增"记忆管理"导航页——用户看到两栏：

```
┌─ 我的策略偏好 (可编辑) ────────┐  ┌─ Agent 学到的东西 (可删除) ──────┐
│                               │  │                                  │
│ EURUSD只做H1，不做M5           │  │ EMA(12,26)在EURUSD H1比EMA(10,30)│
│ 每笔风险上限2%                 │  │ 好，夏普1.6 vs 1.2               │
│ 不做亚盘时段 (0-8 UTC)         │  │                                  │
│ [+ 新增]                       │  │ ATR(14)×2止损在趋势市有效，      │
│                               │  │ 在震荡市导致过早出场              │
└───────────────────────────────┘  └──────────────────────────────────┘
```

### 4.5 领域知识注入

用户策略模板和 Agent 自动记忆覆盖了"用户偏好"和"租户经验"两层。还有一层两者都不覆盖：**弱模型的策略基础知识空缺**。Claude 和 DeepSeek 的预训练语料里有几千篇交易策略文章，GLM-4、GPT-4o-mini、本地小模型不一定有。领域知识注入是给所有模型的策略生成质量兜底。

**两层模型：**

```
┌─ 全局领域知识 (平台运营手工维护) ──────────────────┐
│                                                     │
│  来源: 平台团队手写（~10-20 条种子）                   │
│  内容: 通用策略模式、风控公式、品种特性、常见陷阱        │
│  管理: Admin 管理端 → 领域知识页 → 编辑、上下架、版本    │
│  注入: Agent 按 scope 匹配, 所有租户共享, 启动时自动注入  │
│                                                     │
│  示例:                                                │
│  "双均线 crossover: 快线周期 < 5 时噪音过大,           │
│   震荡市假突破 > 40%, 必须加 ADX>25 过滤器"            │
│  "EURUSD 亚盘(0-8 UTC)流动性低,                       │
│   滑点成本通常是伦敦盘的 2-3 倍"                       │
└─────────────────────────────────────────────────────┘

┌─ 租户经验池 (Agent 自动积累) ────────────────────────┐
│                                                     │
│  来源: 策略回溯 Agent (§6) 自动生成                    │
│  内容: 本租户从历史策略中学到的经验                     │
│  注入: 仅注入本租户的会话, 不跨租户共享                  │
│  详见 §4.1 Agent 自动记忆                              │
└─────────────────────────────────────────────────────┘
```

**为什么全局知识必须人手写：** 10-20 条种子，一条 3-5 句话。一个懂交易的平台运营花一个下午写完。这些不是策略逻辑（LLM 会生成），是 LLM 训练语料里可能缺失的领域常识和已知陷阱。写完就放那儿，Agent 每次启动按 scope 匹配注入。

**为什么 Agent 经验不进全局库：** Agent 说"EMA(12,26) 更好"——可能只是单次回测的过拟合噪音，不一定是普适规律。同一租户内引用没问题，不能作为全局真理灌给所有租户。人工 review 的成本不低——平台运营要先验证"EMA(12,26) 在 EURUSD H1 的多个时间段确实优于 EMA(10,30)"，才能升级为全局知识。这个流程在平台没有百万用户之前不值得做。

**为什么不用用户可见的 Skills 系统：** 用户不会写也不会调用结构化的技能文件。全局知识由平台一手维护，Agent 透明注入，用户不需要知道它的存在。没有用户输入路径就没有恶意 Skill 注入风险。

---

### 5.1 5-tier 设置优先级

```
Managed (平台运营)    最高, 低层不可覆盖
  ↓
Tenant (企业 Admin)   团队默认, 不可高于 Managed
  ↓
User (交易员)         个人偏好
  ↓
Session (单次会话)    临时覆盖
  ↓
Default              出厂默认
```

### 5.2 Managed 层关键字段

```json
{
  "allowed_models": ["claude-sonnet-5", "deepseek-v4"],
  "enforce_allowed_models": true,
  "max_cost_ceiling_usd": 0.50,
  "max_iterations_per_strategy": 50,
  "disable_live_trading": false,
  "required_risk_gates": ["lookahead", "walkforward", "paper"],
  "audit_retention_days": 365,
  "require_tenant_isolation": true
}
```

解析失败 = fail-closed：`disable_live_trading` 强制为 `true`，`allowed_models` 强制为空白名单。

### 5.3 Capability 权限引擎

规则格式：`Effect Action(Resource:Selector)`

```
ALLOW  live_trading(*:volume<=1.0)
ALLOW  backtest(*:*)
DENY   live_trading(*:leverage>100)
DENY   model(gpt-4:*)
```

- Deny 永远优先于 Allow
- 规则按 Managed → Tenant → User 层合并
- `allowManagedRulesOnly` = true → User/Tenant 层规则忽略

### 5.4 Admin 管理端 UI

- 模型白名单管理（哪些 LLM 可用，哪些被禁）
- 风控门配置（哪些门必须过才能上实盘）
- 全局熔断开关（`disable_live_trading`）
- 权限规则编辑器（`Allow/Deny` 规则列表）
- 审计日志查看
- 租户配额管理

---

## 6. 聚焦版策略回溯 Agent

### 6.1 与通用 Nudge Engine 的差异

| | Hermes 通用 Nudge Engine | 策略回溯 Agent |
|---|---|---|
| 触发时机 | 每次对话后 | 策略生成流程结束后 |
| 复盘范围 | 所有对话内容 | 本次策略生成的完整上下文 |
| 输出 | 任意值得记住的事实 | 1-2 条结构化策略经验 |
| LLM 成本 | 每次会话 1 次额外调用 | 每策略 1 次额外调用 |
| 噪音风险 | 中 | 低（输入聚焦策略） |

### 6.2 流程

```
策略生成流程结束 (GenerateStrategy done)
  │
  ▼
触发 PostGenerationHook → 收集上下文:
  - 初始需求 (用户自然语言)
  - 第一版策略是什么样的
  - 回测结果 (夏普/回撤/胜率/交易分布)
  - 做了哪些修改 (语义 diff 列表)
  - 最终策略是什么样的
  │
  ▼
PostGenerationHook 调用回溯 Agent (一次 LLM 调用, ~$0.0005):
  "基于这次策略生成的全过程，提取 1-2 条值得复用的经验"
  │
  ▼
LLM 输出结构化经验:
  - key_finding: "EMA(12,26) c/o on EURUSD H1: sharpe 1.6, maxDD 14%"
  - success_factor: "ADX>25 filter reduced whipsaws by 60%"
  - failure_avoid: "Avoid <8 UTC session, false breakout rate >40%"
  - indicators_used: ["EMA", "ADX", "ATR"]
  - condition_structure: "crossover_and_threshold"
  │
  ▼
调用 StoreExperience → 写入 agent_experience 表
  │
  ▼
异步更新 MEMORY.md 索引
```

### 6.3 为什么现在做

回溯 Agent 依赖 `GenerateStrategy` 流程完整跑通（已有）。每次额外 LLM 调用成本 ~$0.0005，在配额内。不需要新的基础设施——复用 `StoreExperience` RPC（proto 已有，Go 实现是 stub）。

---

## 7. 领域知识注入体系

> §4.5 已详述两层模型（全局领域知识 + 租户经验池）及其设计决策。此处不再重复。

本 ADR 不引入用户可见的 Skills 系统。理由：

- **用户不会写也不会调用。** 用户说的是"帮我做趋势跟踪"，不是"加载双均线 Skill"。领域知识注入对用户完全透明。
- **全局知识来源单一。** 平台运营一手维护 ~10-20 条种子知识，不开放用户创建。
- **无用户输入即无注入风险。** Claude Code 社区有 800+ 恶意 Skills 的先例。我们的设计不存在这个攻击面。

实施要点：

- Admin 管理端新增"领域知识"页：`domain_knowledge` 表，字段 `content` / `scope JSONB` / `status (active|archived)`
- Agent 启动时 Go 层检索 `status=active` 的知识条目，按 scope 匹配当前会话（品种/周期/策略类型），注入 LLM system prompt
- 平台运营可随时上下架、修改内容，热重载生效（PG NOTIFY → Agent 重新加载）

---

## 8. 生命周期钩子

ADR-0024 §5.3 Agent 层约束中已定义实盘修改需用户显式确认。本 ADR 在此基础上新增 5 个关键钩子事件：

| 事件 | 触发时机 | 类型 | 可阻断 |
|------|---------|------|--------|
| `PreStrategySubmit` | Agent 提交策略前 | webhook/command | ✅ 合规预检、阻断违规策略 |
| `PostBacktest` | 回测完成 | webhook/command | ❌ 触发分析、知识库存储 |
| `PreLiveDeploy` | 实盘上线前 | webhook/command | ✅ 强制风控门检查 |
| `DegradationAlert` | 实盘绩效退化 | SSE push | ❌ Agent 自动分析 |
| `PostStrategyGeneration` | 策略生成流程结束 | internal | ❌ 触发回溯 Agent |

钩子类型：
- `command` — shell 命令，stdin 接收 JSON，exit code 2 = 阻断
- `webhook` — HTTP POST，返回 JSON `{"allow": false, "reason": "..."}` 可阻断
- `internal` — Go 内部函数调用

---

## 9. 模型降级链

```json
{
  "model_failover": {
    "primary": "claude-sonnet-5",
    "fallback": ["claude-haiku-4-5", "deepseek-v4"],
    "strategy": "sequential",
    "health_check": {
      "interval_seconds": 30,
      "timeout_ms": 5000
    }
  }
}
```

Agent 启动时注入路由配置。`systemai.Service` 在调用 LLM 前先做 health check，主模型不可用 → 自动切备选。降级时通知前端：状态栏显示"当前使用备选模型 Claude Haiku，功能可能受限"。

---

## 10. 实施优先级与 ADR-0024 的关系

### 10.1 ADR-0024 Phase 4 的处理

ADR-0024 Phase 4 的计划是"知识库 + 策略进化 (4 周)"。**本 ADR 不取消 Phase 4——它重新定义 Phase 4 的设计。** GLM 继续施工 Phase 4，但实施细节按本 ADR 执行：

| ADR-0024 Phase 4 原计划 | ADR-0025 调整 |
|---|---|
| pgvector schema + 语义检索 + 结构指纹 (3d) | **保留**，schema 已在 §4.2 |
| 经验存储 + 双路径检索 + prompt 注入 (3d) | **增强**：增加 MEMORY.md 索引模式 + 会话启动注入 + 用户策略模板表 + 领域知识注入 |
| 策略进化 Agent（退化检测 + 改进循环）(5d) | **替换**为聚焦版策略回溯 Agent（§6），更聚焦、成本更低 |
| SSE 退化告警 + 前端通知 (3d) | **保留**，纳入 `DegradationAlert` 钩子（§8） |
| 跨策略学习 + 主动建议 (5d) | **替换**为领域知识注入（§7）+ 回溯 Agent 的跨策略经验积累。总工时 ~4d（节省 1d） |
| E2E 测试 (4d) | **保留** |

总工期缩短至 ~3.5 周（原 4 周）。

### 10.2 实施优先级

| 优先级 | 功能 | 来源 | 对应 ADR-0024 Phase | 依赖 |
|--------|------|------|-------------------|------|
| **P0** | Plan Mode 交互卡片 | Claude Code | Phase 3 前端增强 | `GenerateStrategy` 已有 `planning` phase |
| **P1** | 三层记忆系统 | Claude Code | Phase 4 知识库 | `SearchExperience`/`StoreExperience` stub |
| **P2** | 分层权限 + Admin 管理端 | Claude Code | 新增 | PG 建表 |
| **P3** | 策略回溯 Agent | Hermes Agent | Phase 4 策略进化 | P1 完成 |
| **P4** | 5 个生命周期钩子 | Claude Code | 插入 Phase 2/3/4 | Hook engine |
| **P5** | 模型降级链 | OpenClaw | 插入 Phase 0 | `systemai.Service` 已有适配层 |
| **P6** | 热重载 + Fail-closed | Claude Code | Admin 管理端 | PG NOTIFY |
| **P7** | 策略设计入口 + 记忆管理导航 | Claude Code | Phase 4 前端 | P1 完成 |

### 10.3 P0-P3 关键交付物

```
P0 (1 周):
  backend/  AgentGenerateStrategyChunk.plan 格式约定 + 行解析器
  frontend/ PlanCard.tsx 策略方案卡片组件 + 交互逻辑

P1 (2 周):
  backend/  agent_experience 表建表 migration
  backend/  domain_knowledge 表建表 migration + Admin 管理端领域知识页
  backend/  memory.go — 三重记忆系统 (StoreExperience 实现 + MEMORY.md 读写 + 领域知识 scope 匹配注入)
  backend/  Session startup: 加载用户模板 + 领域知识 + 经验索引 → 注入 prompt
  frontend/ MemoryPage.tsx — 记忆管理页面

P2 (1.5 周):
  proto/    admin_settings.proto (ManagedSettings + CapabilityRule)
  backend/  admin/settings.go — 5-tier 设置引擎
  backend/  admin/permissions.go — Capability Rule Engine
  frontend/ AdminSettingsPage.tsx — Admin 管理端

P3 (1 周):
  backend/  agent/retrospect.go — PostGenerationHook + 回溯 Agent
  backend/  MEMORY.md 异步更新
```

---

## 11. 验证方式

### 11.1 Plan Mode

```
E2E: 用户输入 "EURUSD H1 趋势跟踪"
  → 前端渲染策略方案卡片 (planning phase)
  → 用户点"修改" → 输入 "加 RSI 过滤"
  → Agent 更新方案，不出代码
  → 用户确认 → 生成代码 → 回测 → 显示结果
```

### 11.2 三层记忆

```
1. 用户保存策略模板 → 下次新会话 Agent 自动引用
2. 策略生成完成 → agent_experience 表有新增记录 → MEMORY.md 更新
3. 记忆管理页面显示用户模板 + Agent 经验，可编辑/删除
```

### 11.3 分层权限

```
1. Admin 设 allowed_models = ["claude-sonnet-5"] + enforce = true
   → 用户选 deepseek → Agent 拒绝："此模型不在平台白名单中"
2. Admin 设 disable_live_trading = true
   → 所有实盘请求被拒 → 前端显示全局熔断已开启
3. 权限规则 DENY live_trading(*:leverage>100)
   → 用户提交 leverage=200 → 实盘 PreLiveDeploy 阻断
```

### 11.4 策略回溯 Agent

```
策略生成完成 → PostGenerationHook 触发 → agent_experience 新增 1-2 条 → MEMORY.md 更新
  下次用户说 "EURUSD H1 趋势跟踪" → Agent 检索到上次的经验 → 注入 prompt
```

---

## 12. 与 ADR-0024 的关系

| ADR-0024 部分 | ADR-0025 的影响 |
|---|---|
| Phase 0（Agent Gateway + 画像 + 解读） | 不变。P6 模型降级链可插入 Phase 0 |
| Phase 1（compile_py.go） | 不变 |
| Phase 2（盲区桥接） | P5 的 PreLiveDeploy/DegradationAlert 钩子可在 Phase 2 预留接口 |
| Phase 3（策略生成 Agent） | P0 Plan Mode 增强前端交互，不改变后端 RPC 接口 |
| Phase 4（知识库 + 策略进化） | **本 ADR 替代 Phase 4 的实施方案。** 工期缩短至 ~3.5 周，交付物更聚焦 |
| §5.5 知识库设计 | 保留 schema + 结构指纹，增加 MEMORY.md 索引 + 用户策略模板 + 领域知识注入 + 会话启动注入 |
| §5.3 Agent 层约束 | 保留实盘显式确认，增加 5 个钩子事件 |
| §7 实施路线 | Phase 优先级参考本 ADR §10 |

---

## 13. 文件结构

```
backend/internal/
  agent/
    memory.go             (新建 — 三重记忆 CRUD + MEMORY.md 读写 + 领域知识注入)
    retrospect.go          (新建 — 策略回溯 Agent + PostGenerationHook)
    hooks.go               (新建 — 5 个钩子事件 + command/webhook/internal 三种类型)
  admin/
    settings.go            (新建 — 5-tier 设置引擎)
    permissions.go         (新建 — Capability Rule Engine)

proto/ant/v1/
  admin_settings.proto     (新建 — ManagedSettings + TenantSettings + CapabilityRule)
  agent_hooks.proto        (新建 — HookConfig + HookEvent)

frontend/src/pages/
  strategy/components/
    PlanCard.tsx            (新建 — 策略方案卡片)
  admin/
    AdminSettingsPage.tsx   (新建 — Admin 管理端)
  memory/
    MemoryPage.tsx          (新建 — 记忆管理页)
```

---

## 审核

| 角色 | 审核人 | 日期 | 结论 |
|------|--------|------|------|
| 作者 | Claude | 2026-07-03 | — |
| 审核人 | — | — | — |
