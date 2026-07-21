# 策略生成 Agent — 第一性原则重构方案（上位设计）

> 性质: **地基文档**。它定义 agent 块"本质上该是什么"，是以下补丁文档的上位约束
> 被它取代/收编的补丁文档:
> - `agent-empty-response-fix.md`（reasoning 回退 — 本方案判定为反模式，废弃）
> - `agent-generation-ux-and-verification-fix.md`（UX 4 bug — 大部分随重构自然消失）
> - `agent-thinking-budget-convergence-fix.md`（token 预算/收敛守卫 — 降级为安全网）
> 交付对象: DeepSeek 按本方案分阶段实现
> 日期: 2026-07-08

---

## §0 为什么要写这份（地基不牢，一切徒劳）

这个 agent 块两周内叠了五层补丁，每层都在治上一层的症状。这是"解错了问题"的信号。本方案不打补丁——它回到第一性，重新定义这个块的形状，让那些 bug **在结构上不可能发生**，而不是逐个去堵。

同时对齐 `AGENTS.md` 铁律: **禁止用「回退代替重新生成」「沉默代替修复」**。当前代码恰好两条都犯了（见 §2）。

---

## §1 第一性：不可再分的本质与不变量

### 本质
> 用户用自然语言描述交易策略 → 得到一份**经过真实验证、确实按描述交易**的可用策略代码。

### 三条不可违背的不变量（Invariants）
任何实现都必须恒真，违反即视为架构缺陷：

- **I1 — 唯一确定的成品**: 任一轮对话结束，系统对"最终代码是哪份"有且仅有一个确定答案。用户永远不需要在多个候选里猜。
- **I2 — 交付前必经行为验证（两层，价值不同）**: 编译通过 ≠ 满足需求，必须在真实 bars 上执行。但"验证"分两层：
  - **I2a 执行冒烟测试（强制）**: 编译 + 真实 bars 上执行不崩 + 产生交易（`trades>0`，否则如实说明逻辑未触发）。只需"有效市场数据"，验证"逻辑会不会跑/开单"。
  - **I2b 绩效评估（条件性）**: 收益/回撤/夏普等数字，**只在 symbol/周期/资金为用户明确选定时才可作为绩效呈现**。参数不全时，只能作为 I2a 冒烟测试呈现，禁止把冒烟结果包装成绩效。
- **I3 — 思考与成品永不混淆**: 模型的推理（scratchpad）绝不能冒充成品交付。成品只能来自结构化产出。
- **I4 — 回测输入必须透明（新增）**: 任何呈现给用户的回测结果，**必须同时显式展示其输入参数**（symbol / timeframe / 日期区间 / 初始资金 / 手续费）。**用户看不到输入的回测数字 = 误导 = 违规**（与假回测同性质，只是更隐蔽）。回测的有效性 = 其输入被用户知晓的程度。

### 最小闭环（本质环节，缺一不可）
```
理解意图(消歧) → 产出代码 → 行为验证 → 失败则修 → 交付唯一成品
```

---

## §2 当前实现如何违背本质（根因，非症状）

只有三个根因；此前所有补丁都是它们的下游症状。

### 根因 A — 违反 I2：验证是假的
`@/opt/ant/backend/internal/agent/agent_tools_backtest.go:39` 的 `run_backtest` 只重新编译，不回测。
→ 症状：生成的策略可能编译通过却完全不做用户要的事，用户还以为对。这是"**沉默代替修复**"。

### 根因 B — 违反 I1：交付物是自由文本，不是结构化产出
模型吐自由文本 → 正则 `ExtractCode` 猜哪块是终稿（`@/opt/ant/backend/internal/agent/generator_agent.go:109`）→ 前端把每个草稿块都渲染成可 Apply（`chatUtils.tsx`）。
→ 症状：多代码块、不知哪个是最终、apply 混乱。**这些 bug 只存在于"允许 N 个候选块"的设计里。**

### 根因 C — 违反 I3：思考冒充成品
`@/opt/ant/backend/internal/service/systemai/chat.go:297-299` + agent loop：content 空就把 `reasoning_content` 当答案。
→ 症状：中→英、思维链刷屏、token 烧光无成品。这是"**回退代替重新生成**"。

> 结论：`max_tokens`↑、`reasoning_effort`、收敛守卫，都是在治 B/C 的症状。地基修好后，它们退化为可选安全网。

---

## §3 目标架构（从不变量推导）

### 3.1 核心转变：代码是「工具产出」，不是「聊天文本」

**这是整个重构的支点，直接兑现 I1。**

新增/改造工具 `write_strategy`（或改造 `compile_python` 使其 `code` 为**必填**参数）：

```
write_strategy(code: string)  ← 模型把完整代码作为工具参数提交
  内部执行: compile → 若过 → 自动 backtest → 回结构化结果
  产物写入 generateState.PythonSource（唯一真源）
```

- 模型**不再靠自由文本交付代码**。代码只能经此工具进入系统。
- 自由文本通道只承载：简短说明 / 追问 / 计划。**永不承载成品代码。**
- I1 由此结构性成立：`generateState.PythonSource` 是唯一终稿，前端只渲染它。

### 3.1b 意图路由 + 工具可靠性（这个支点的两个致命前提）

**前提 1 — 不是每轮都产代码。** 用户可能只是提问（"这策略在干嘛"）、闲聊、要求解释。**绝不能无脑强制 write_strategy**，否则会给出用户没要的代码。规则：
- 意图=产出/修改代码 → 必须走 write_strategy（代码不进自由文本）。
- 意图=讨论/答疑/追问 → 走自由文本通道，**不调** write_strategy。
- 判定交给模型（提示词写清），而不是硬编码 `tool_choice=required`。

**前提 2 — 模型必须真的会调工具。** 纯推理模型（如 R1 风格）tool-calling 不可靠，可能永远不调 write_strategy → 整个支点失效。护栏：
- 若某轮意图明显是产代码，却出现"代码写进自由文本、没有 tool call" → 视为**违规输出**：注入纠偏消息要求改用 write_strategy 重试（≤2 次），**而不是**回落到正则 `ExtractCode`（那等于把根因 B 又请回来）。
- agent/generator 默认优选 instruct/chat 模型（对齐 §5 阶段2 的模型选择）。tool-calling 能力是本架构的硬依赖，选型时须校验。

### 3.2 验证内建到产出（兑现 I2）

`write_strategy` 内部直接复用现成真回测闭环——**不是新造轮子**：

- **`REUSE: mql2go.CompilePythonWithCoverage @ tools/mql2go`** — 编译成 `*VMRunner`
- **`REUSE: runVMBacktest @ /opt/ant/backend/internal/agent/backtest_helpers.go:18`** — 真实 VM 回测（SubmitStrategy 已用）
- **`REUSE: buildBacktestResultProto @ backtest_helpers.go:60`** — 结果转 proto
- **参照现成完整流程**: `@/opt/ant/backend/internal/agent/gateway.go:117-156`（SubmitStrategy: compile→取bars→runVMBacktest）与 `bridge.go` 的 `TranslateWithRetry`（compile→backtest→retry）

`write_strategy` 返回给模型的结构化结果：
```
{ compiled: bool, compile_error?: string,
  backtest: { total_trades, win_rate, total_return, max_drawdown, sharpe } | null,
  backtest_error?: string }
```
模型据此自我修正。⚠️ **注意 `total_trades==0` 不等于 bug**：可能是入场条件在该数据窗口内本就罕见（用户这类"4H放量+吞没+回调+5M吞没"的多条件策略尤其如此），或数据窗口太短。它是**信号**，不是自动判失败——不要因此陷入无限"修复"循环。正确做法：如实告知"该窗口内未触发交易"，让模型/用户判断是逻辑过严还是窗口/品种不合适。这就是 I2 的闭环。

> Generator 已持有 `mkt`(bars) 和 `btRepo`（`generator.go:18-19`），依赖齐全，只需把它们透传进工具。

### 3.2b 实现模板指引（DeepSeek 照抄这些现成锚点，勿从零发明）

`write_strategy` 不需要任何新机制，把下面四块拼起来即可：

1. **工具结构照抄**: `compilePythonTool`（`@/opt/ant/backend/internal/agent/agent_tools.go:14-49`）—— `Name()/Schema()/Run()` 三件套 + `result *generateState` 就地写入。`write_strategy` 只是在它基础上，编译成功后接着跑回测。
2. **参数读取照抄**: 工具从 `in.RawArgs["code"]` 或 `in.Code` 取代码（`ToolInput.RawArgs` 由 `parseToolArguments @ agent_loop.go:228` 填充）。`code` 设为 Schema 必填。
3. **compile→backtest 全流程照抄**: `@/opt/ant/backend/internal/agent/gateway.go:114-156` 就是完整参照——
   `CompilePythonWithCoverage(code)` → `fetchBars(ctx, btCfg)`（取 bars 的现成 helper，用同样参数走 `mkt`）→ `runVMBacktest(ctx, runner, btCfg, bars, params)` → `buildBacktestResultProto(r)`。
4. **依赖注入**: `buildPythonToolRegistry`（`agent_tools.go:52`）签名加 `mkt` + `btCfg`，由 `runAgentLoop`（`generator_agent.go`）传入 `g.mkt` 和 `msg.BacktestConfig`。

> 一句判据：`write_strategy` 里出现的每个函数调用，都应在上面锚点里能找到原型。若你发现需要"从零写"某段回测/编译逻辑，说明你漏看了现成能力——先 `bash scripts/cap.sh backtest` 再动手。

### 3.2c 回测上下文的来源与透明（兑现 I2a/I2b/I4 —— 别让默认参数把回测变成误导）

**问题**：生成代码时 symbol/周期/日期/资金可能不全。系统瞎填默认值跑出的绩效，用户既看不懂也会被误导。**现状**：工作区有 `msg.Symbol`/`msg.Timeframe`（用户选的，他知道），但 `AgentGenChat` 目前不发 `backtestConfig`（无日期/资金）。

`write_strategy` 按参数来源分三种处理，**绝不瞎填 symbol**：

| 情况 | 处理 |
|------|------|
| symbol 已选（`msg.Symbol!=""`，正常） | 用工作区 symbol+timeframe + **透明默认**日期区间(**有界最近窗口**，如最近 N 根 bar，勿"全部可用 bar"——5M 会巨慢/长 TF 太短无交易)/资金($10k)/手续费 → 跑 I2b 绩效回测。**结果必须随附这些输入参数**（I4） |
| symbol 未选 | **禁止默认一个 symbol**（symbol 是语义参数，§3.5）。降级为 I2a：要么明确要求用户先选 symbol，要么只做冒烟测试并如实标注"样本数据、非绩效评估" |
| 日期/资金缺失 | 可用惯例默认，但**必须在结果里显示**。透明是有效的前提（I4） |

**返回给模型和前端的回测结果结构，必须内嵌输入参数**（不只是指标）：
```
{ inputs: { symbol, timeframe, date_range, initial_capital, commission },  // I4：缺此即违规
  tier: "smoke" | "performance",   // I2a vs I2b，前端据此决定是否展示绩效数字
  compiled, backtest: {...} | null, ... }
```
前端渲染绩效卡片时，`tier=="smoke"` 只显示"✓ 可运行/产生 N 笔交易"，**不显示收益率等绩效数字**；`tier=="performance"` 才显示绩效，且必须附带 `inputs`。

### 3.3 思考与成品彻底分离（兑现 I3）

- `reasoning_content` → **只**走 thinking 通道（前端折叠展示的"💭思考"），**永不**回退成 content 冒充答案。
- 删除 `chat.go:297-299` 和 agent loop 里的"reasoning 当答案"回退。
- content 通道空 + 无 tool call = 一次**失败**（触发 §3.5 的收敛处理），而不是拿草稿糊弄。
- I3 成立后，中→英、多草稿刷屏自然消失（因为思维链不再当答案渲染）。

### 3.4 分解驱动执行（对齐 Claude Code，限界推理）

- `update_plan` 从"装饰"升为"驱动"：复杂策略先产出 plan（入场/仓位/加仓/出场分步），逐步 `write_strategy` + 验证。
- 每步推理被任务边界限界 → 不再有"一个大 turn 想爆 token"。
- 单步足够简单时可跳过 plan，直接 write_strategy。

### 3.5 语义歧义必须追问（交易域安全，兑现 I1 的正确性）

区分两类歧义，写进系统提示的判定规则：

| 类型 | 例子 | 处理 |
|------|------|------|
| 装饰性 | period=14, 阈值 | 用专业默认值 + 一行注释 |
| **语义性**（改变盈亏行为） | 仓位计算基准、`200倍`单位、时间周期(1h vs 4H/5M) | **必须问一个聚焦问题**，禁止乱猜 |

模型在真金白银的量化域猜错语义参数 = 交付一个用户误信的错误策略。宁可一次聚焦追问，绝不静默默认。

### 3.6 能力真空：SDK 多时间框架（非 agent 问题，但卡死 agent）
用户那条需求要 4H+1H+5M，而 SDK 无干净多周期访问 → 模型耗大量推理硬凑。
- **本方案标注为独立前置任务**：给策略 SDK 增加多时间框架 bar 访问能力（如 `ctx.bars(timeframe)`）。
- 在此之前，`write_strategy` 编译层应对"多周期不支持"返回明确错误，让模型走 §3.5 追问或降级，而不是空转。

### 3.7 已知风险与边界（DeepSeek 施工前必读，避免踩坑）

- **诚实排序警告**：触发这一切的那条旗舰需求（4H+1H+5M 多周期）**依赖 §3.6 的 SDK 多周期能力**，而它排在阶段 3。**只做阶段 1 无法让那条策略正确落地**——别过度承诺。阶段 1 先用单周期策略验证 I1–I4 闭环，多周期是独立能力任务。
- **回测延迟/成本**：`runVMBacktest` 有 30s 超时（`backtest_helpers.go:49`），每次 `write_strategy` 都编译+取bars+回测。模型迭代 5 次 = 5 次回测，慢且耗算力。策略：**迭代中允许"仅编译"快速反馈，交付前必跑一次完整回测**（兑现 I2 的时机是"交付前"，不是"每次改都全量回测"）。给模型的工具可区分 `compile_only` vs `full`，或由收敛阶段控制。
- **不要把 §5 阶段2 的收敛守卫当主力**：它是安全网。若阶段 1（结构化产出+意图路由）做对，模型不该频繁撞 `finish_reason=length`。守卫频繁触发 = 阶段 1 没做对，回去查根因，别靠守卫硬扛。
- **history 一致性**：多轮对话里 `edit_code`/`write_strategy` 反复改，**最新一次 write_strategy 的产物**才是 I1 的唯一终稿；确保 `generateState.PythonSource` 始终指向最新，历史轮的旧代码不得被误当终稿。

---

## §4 目标数据流（重构后）

```
用户描述
  ↓
[可选] 语义歧义? → 追问一个问题（自由文本通道）
  ↓
[复杂?] update_plan 分解
  ↓
write_strategy(code)  ────────────┐
  ↓ (工具内部)                     │ 失败(compile/backtest) → 结构化错误回模型
  compile → backtest(真实)         │ → 模型修正 → 再次 write_strategy
  ↓ 成功                           │  (≤N 次)
generateState.PythonSource ← 唯一终稿
  ↓
交付: 前端渲染 ONE 代码卡片(Apply) + 折叠思考 + 回测指标
```

对照三不变量：I1(唯一终稿✓) I2(内建回测✓) I3(思考分离✓)。

---

## §5 分阶段落地（先地基，后细节）

### 阶段 1 — 地基（务必先做，其余依赖它）
1. **`write_strategy` 工具**：`code` 必填参数，内部 compile→backtest（复用 §3.2 锚点），返回结构化结果。`buildPythonToolRegistry` 透传 `mkt`+`cfg`。
2. **删除 reasoning 回退**（`chat.go:297-299` + agent loop），改为 thinking 通道分离。
3. **成品唯一化**：`generateState.PythonSource` 只由 `write_strategy` 写入；前端只渲染它为唯一 Apply 卡片。
4. **`run_backtest` 真实化 / 或并入 `write_strategy`**（消灭假验证）。

### 阶段 2 — 收敛与体验
5. plan 驱动分步执行（`update_plan` 升级）。
6. 逐块流式（`agent_loop.go` onChunk 内转发；已在 UX 文档 §4 描述）。
7. 语义歧义追问规则（系统提示 + 提示词定稿）。
8. Apply 自动切 code tab（UX 文档 §3）。
9. 收敛安全网（token 预算/`finish_reason=length` 重试）——降级为兜底，非主力。

### 阶段 3 — 能力升级
10. SDK 多时间框架访问（独立任务）。
11. 上下文语义压缩（替代 char/4 粗算）。

---

## §6 验收（以不变量为准，不以功能点为准）

- **I1**: 任意对话结束，前端有且仅有一份可 Apply 的最终代码；无多草稿可 Apply。
- **I2a**: 交付的每份代码，日志可见其经过 `runVMBacktest` 且 `total_trades>0`（或明确说明逻辑未触发/无数据）。
- **I2b**: symbol 未选时，前端**不出现**收益率等绩效数字，只出现冒烟测试结论；symbol 已选时才显示绩效。
- **I3**: `reasoning_content` 从不出现在成品代码里；日志无"reasoning 当答案"回退路径。
- **I4**: 前端每一处回测绩效数字，旁边都能看到对应的 symbol/timeframe/日期区间/资金。找不到输入参数的绩效数字 = 不合格。
- 端到端：用户那条复杂 prompt → 要么产出经回测的代码，要么就语义歧义追问一次——绝不"想到 token 耗尽无成品"。

---

## §7 硬约束

- ❌ 不新增 REST/WebSocket；proto 通信；价格/盈亏 `decimal.Decimal`（注意 `backtest_helpers.go:74` 已知 float64 边界勿扩散）。
- ✅ 每个新 file/function 标 `REUSE:`/`NEW:`（§3.2 已给全部 REUSE 锚点，`write_strategy` 若为新增标 `NEW: 无现成"代码即工具产出"能力`）。
- ✅ proto 改动跑 gen 脚本；前后端 `_pb` 同步。
- ✅ 提交前 `go build ./...` + `go run ./tools/check-file-lines --strict`。
- ✅ 部署 `docker compose build backend && docker compose up -d backend`。
- ✅ **禁止**为绕过困难而回退/沉默/标 legacy（`AGENTS.md`）。

---

## §8 一句话总结

当前 agent 在解一个被扭曲的问题——"生成一大段可能含多草稿、只编译不验证、思考与成品混杂的自由文本"。本方案把它拨回本质："**产出一份经真实回测验证的、唯一确定的策略代码**"。做法不是加检查，而是改形状：**代码变成工具产出**（I1）、**验证内建**（I2）、**思考与成品分离**（I3）。地基立住，之前那堆 bug 在结构上就不再可能发生。

---

## §9 给 DeepSeek 的施工提示语（直接复制这段发给它）

```
你是本次重构的施工方。请严格按 docs/agent-first-principles-rebuild.md 落地，规则如下：

【读法】
1. 先读完整文档，重点吃透 §1 的四条不变量 I1/I2/I3/I4 —— 它们是验收标准，不是建议。
2. §2 是根因（A/B/C），§3 是目标架构，§3.1b/§3.2c/§3.7 是最容易踩的坑，务必读。
3. 其它 agent-*.md 文档已被本文档收编：reasoning 回退文档作废，token预算/收敛守卫降级为安全网。有冲突以本文档为准。

【只做阶段 1，做完停下等验收】
本轮只实现 §5 阶段 1 四件事，不要碰阶段 2/3：
  1. write_strategy 工具（code 必填 → compile → 交付前回测），照抄 §3.2b 的四个锚点，禁止从零写回测/编译逻辑。
  2. 删除 chat.go:297-299 的 reasoning 回退，改 thinking 通道分离（§3.3）。
  3. 成品唯一化：generateState.PythonSource 只由 write_strategy 写入，前端只渲染它为唯一 Apply 卡片。
  4. run_backtest 真实化或并入 write_strategy，消灭假验证。

【铁律，违反即返工】
- 禁止"回退代替重新生成"（如 reasoning 当答案）、"沉默代替修复"（如只编译冒充回测）、标 legacy 绕过。遇阻回根因，不许退而求其次（AGENTS.md）。
- 代码永不进自由文本，只经 write_strategy（§3.1b 前提1）；但不是每轮都产代码，讨论/答疑走自由文本，别无脑强制 tool（§3.1b）。
- 回测参数不透明就是误导：任何绩效数字必须随附 symbol/timeframe/日期/资金；symbol 未选绝不瞎填，降级为冒烟测试（§3.2c/I4）。
- total_trades==0 是信号不是 bug，别陷入无限修复循环（§3.2 注）。
- 每个新 file/function 在 PR 里标 REUSE: 或 NEW:；动工前 bash scripts/cap.sh <动词>。

【自检 + 提交】
- 做完按 §6 用 I1/I2/I3/I4 逐条自检，在 PR 里逐条给证据。
- go build ./... + cd backend && go run ./tools/check-file-lines --strict 必须过。
- proto 改动跑 gen 脚本，前后端 _pb 同步。
- 部署：docker compose build backend && docker compose up -d backend。
- 有任何"文档没覆盖/看似矛盾"的点，先停下问，不要自行猜测扩大范围。
```
