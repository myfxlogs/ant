# Agent-Engine → Claude Code 水准整改方案

> 目标: 把 `GenerateStrategy` agent-engine 提升到 Claude Code 级别的自主编码 agent 水准
> 交付对象: DeepSeek 落地实现
> 基线代码: `internal/agent/generator_agent.go`, `internal/connect/ai/agent_loop.go`, `internal/connect/ai/tool_registry.go`
> 日期: 2026-07-07

---

## §0 核心差距总览

Claude Code 之所以强，不是模型强，是 **agent harness（工具环+反馈环）强**。当前我们的 harness 缺失关键能力：

| # | Claude Code 能力 | 我们的现状 | 差距等级 |
|---|-----------------|-----------|---------|
| G1 | **运行测试验证**（Bash 跑 test） | 只 `compile_python`，能编译不能验证行为 | 🔴 致命 |
| G2 | **多轮持久对话上下文** | `history=nil`，每次调用无状态 | 🔴 致命 |
| G3 | **surgical edit（Edit 工具）** | 每轮重生成整个文件 | 🔴 致命 |
| G4 | **读取现有代码/上下文**（Read/Grep） | 无 read_current_code，LLM 盲写 | 🟡 重要 |
| G5 | **计划管理（TodoWrite）** | plan_mode 设置存在但 loop 内无 todo 工具 | 🟡 重要 |
| G6 | **富工具参数（任意 typed args）** | `ToolInput` 只有 4 字段，靠 hack 复用 | 🟡 重要 |
| G7 | **context 压缩/管理** | 无，长 loop 撑爆窗口 | 🟡 重要 |
| G8 | **完整工具进度反馈** | 只 compile/read_kline 发 phase | 🟢 打磨 |

**优先级判断**: G1（run_backtest 工具）+ G3（edit 工具）+ G2（history）三者构成"生成→验证→修复"的闭环，是从"代码生成器"跃升到"自主策略工程师"的关键。先做这三个。

---

## §1 🔴 G1 — `run_backtest` 工具（生成→验证闭环）

### 问题根因
`Generator` 结构体已持有 `btRepo *repository.BacktestRunRepository`（`generator.go:19`），但 `buildPythonToolRegistry`（`agent_tools.go:51`）只挂了 `compile_python`。LLM 编译通过就"完工"，从不运行回测验证策略是否真的产生信号/盈亏。等价于 Claude Code 只做 `tsc --noEmit` 从不 `npm test`。

### 整改
新增 `run_backtest` 工具，LLM 在 compile 通过后可调用它跑一次真实回测，拿到 trades 数、胜率、PnL、最大回撤，据此判断策略是否合理，不合理则自我修正。

**REUSE 核对**（动工前执行 `bash scripts/cap.sh backtest`）:
- 回测执行: 复用现有 `SimBroker + Engine`（`backend/strategy/backtest/`）
- 编译到 VM: 复用 `mql2go.CompilePythonWithCoverage`（已用于 compile_python）
- 结果落库: 复用 `btRepo`（Generator 已持有）

**落地文件**: `internal/agent/agent_tools.go`（新增 `runBacktestTool`）

```go
// run_backtest tool — compiles current code to VM, runs a backtest over the
// workspace symbol/timeframe, returns trade count / win rate / PnL / max DD.
type runBacktestTool struct {
    result  *generateState
    btRepo  *repository.BacktestRunRepository
    mkt     repository.MarketDataStore
    symbol  string
    timeframe string
}

func (t *runBacktestTool) Name() string { return "run_backtest" }
func (t *runBacktestTool) Schema() systemai.ToolDefinition {
    return systemai.ToolDefinition{
        Type: "function",
        Function: systemai.ToolDefFunction{
            Name: "run_backtest",
            Description: "对当前 Python 策略代码运行真实回测，返回交易次数、胜率、总盈亏、最大回撤、夏普。用于验证策略是否真的产生交易信号且盈利合理。编译必须先通过。",
            Parameters: map[string]any{"type": "object", "properties": map[string]any{}},
        },
    }
}
// Run: CompilePython → backtest.Engine.Run over fetched bars → 摘要 metrics
```

**接线**: 修改 `buildPythonToolRegistry` 签名，接收 `btRepo`/`mkt`/`symbol`/`timeframe`，在 `runAgentLoop`（`generator_agent.go:28`）传入。

**⚠️ 精度红线**: 回测 metrics 涉及价格/盈亏，必须用 `decimal.Decimal`，摘要给 LLM 时才转字符串。不要 float64（参考审计 F6/F7 已知精度 bug，勿重蹈）。

**验收**: LLM 生成 EMA 交叉策略 → compile_python 通过 → run_backtest 返回 trades>0 → 若 trades=0 LLM 应自动放宽入场条件重试。

---

## §2 🔴 G3 — `edit_code` 工具（surgical 编辑）

### 问题根因
`AgentLoop.run`（`agent_loop.go:112`）每轮用 `ExtractCode(roundText)` 覆盖整个 `currentCode`。修一个 bug LLM 要重新输出整个策略文件 → 慢、贵、易引入新错。Claude Code 用 Edit 工具做 old_string→new_string 精确替换。

### 整改
新增 `edit_code` 工具，参数 `old_string` / `new_string`，在 `currentCode` 上做精确替换后重新 compile。

**REUSE 核对**（`bash scripts/cap.sh edit` / `replace`）:
- 字符串替换: Go 标准库 `strings.Replace`，无现成能力 → `NEW`
- 编译验证: 复用 `compile_python` 路径

**依赖 G6**（富参数）: 需要 `ToolInput` 支持 `OldString`/`NewString` 字段——见 §6。

**落地文件**: `internal/agent/agent_tools.go`

```go
type editCodeTool struct{ result *generateState }
// Schema: properties = {old_string, new_string}, required both
// Run:
//   1. 若 old_string 不在 currentCode → 返回 error "old_string not found"（强制 LLM 先读）
//   2. 若 old_string 出现多次 → 返回 error "not unique, add more context"
//   3. 替换 → 更新 result.PythonSource → 编译 → 返回 compile 结果
```

**关键设计**（照搬 Claude Code Edit 语义）:
- `old_string` 必须唯一，否则报错要求补充上下文
- `old_string` 未找到 → 报错（防止盲改）
- 替换后立即 compile，把错误反馈回 loop

**验收**: compile 报 "line 12 undefined var" → LLM 调 edit_code 只改那一行 → 重新 compile 通过，不重输出整个文件。

---

## §3 🔴 G2 — 多轮对话上下文（history 接线）

### 问题根因
`runAgentLoop`（`generator_agent.go:71`）硬编码 `history=nil`，注释 "until proto adds session_id"。而 conversation 持久层**已完整存在**（`ai_conversation.go`: Create/Get/GetMessages/List/Delete + `s.conversations` store）。用户第二次说"把止损改成 2%"时，agent 完全不记得上一轮生成了什么。

### 整改
**Step 1 — proto 加字段**: `AgentGenerateStrategyRequest` 增加 `string conversation_id`。

**Step 2 — 加载历史**: `runAgentLoop` 里，若 `conversation_id` 非空，调 `conversations.GetMessages` 加载历史，转成 `[]systemai.ChatMessage` 传给 `loop.RunWithHistory`（该方法**已支持 history 参数**，`agent_loop.go:56`）。

**Step 3 — 回写**: loop 完成后，把本轮 user message + assistant 最终代码写回 conversation（复用 `conversations` store 的写入方法）。

**REUSE 核对**（`bash scripts/cap.sh conversation`）:
- 会话 CRUD: `REUSE: s.conversations`（ConversationStore，`ai_conversation.go`）
- history 注入: `REUSE: AgentLoop.RunWithHistory @ agent_loop.go:56`

**⚠️ 依赖 G7**: 历史累积会撑爆 context。必须配合 §7 的 context 压缩，或至少限制加载最近 N 轮。

**落地文件**: proto + `generator_agent.go` + 前端 `agentGen.ts`（传 conversation_id）

**验收**: 第一轮生成 EMA 策略 → 第二轮"止损改 2%" → agent 基于上一轮代码 edit，而非从零重写。

---

## §4 🟡 G4 — `read_current_code` 工具

### 问题根因
LLM 看不到 workspace 里已有的代码（除非塞进 prompt）。做 edit 前必须先"读"，否则 old_string 全靠猜。

### 整改
新增 `read_current_code` 工具，返回当前 `result.PythonSource`（带行号，便于 LLM 定位 edit 位置）。无参数。

**落地文件**: `internal/agent/agent_tools.go`
**配合**: 是 G3 `edit_code` 的前置——system prompt 引导 "edit 前先 read_current_code"。
**验收**: LLM 修 bug 时先 read 拿到带行号代码，再精确 edit。

---

## §5 🟡 G5 — `update_plan` / todo 工具

### 问题根因
`agent.plan_mode = "plan"` 设置存在（`settings.go:67`）但 loop 内没有让 LLM 显式管理多步计划的工具。复杂策略（多指标+多空+过滤器）LLM 容易漏步骤。Claude Code 用 TodoWrite 强制拆解。

### 整改
新增 `update_plan` 工具，参数 `steps`（数组，每项 `{content, status}`）。工具本身只是把 plan 通过 `toolStream` 推到前端展示，并作为 tool result 回喂给 LLM 强化其"待办意识"。

**REUSE 核对**: 前端 plan 展示——检查 `AgentGenChat.tsx` 是否已有 plan 渲染（审计提到 onPlan 回调）。若有 `REUSE`，否则 `NEW`。

**依赖 G6**（数组参数）。

**验收**: "做一个带 ATR 止损、RSI 过滤、EMA 交叉入场的多空策略" → LLM 先 update_plan 列 4 步 → 逐步实现并更新 status。

---

## §6 🟡 G6 — `ToolInput` 富参数化（架构债）

### 问题根因
`ToolInput`（`tool_registry.go:18`）只有 `{Code, Symbol, Timeframe, UserID}` 4 字段。`parseToolArguments`（`agent_loop.go:196`）把 memory 工具的 `key`/`value` **hack 映射**到 `Symbol`/`Timeframe`（审计 B6）。G3 的 edit 需要 `old_string`/`new_string`，G5 需要 `steps` 数组——现有结构撑不住。

### 整改
把 `ToolInput` 从"固定字段结构体"改为**保留结构化通用字段 + 原始参数 map**:

```go
type ToolInput struct {
    Code      string
    Symbol    string
    Timeframe string
    UserID    uuid.UUID
    // 新增: 保留 LLM 传入的完整原始参数，工具自行解析所需字段
    RawArgs   map[string]any
}
```

`parseToolArguments` 保留标准字段映射，同时把完整 `args` 塞进 `RawArgs`。各工具从 `RawArgs["old_string"]` 等自取。移除 key/value→Symbol/Timeframe 的 hack。

**REUSE 核对**: `REUSE: parseToolArguments @ agent_loop.go:196`（改造而非新建）
**影响面**: 所有现有工具（compile/read_kline/memory）保持向后兼容——标准字段仍填充。
**验收**: memory 工具不再靠 Symbol 存 key；edit_code 能拿到 old_string/new_string。

---

## §7 🟡 G7 — Context 压缩

### 问题根因
`AgentLoop.run` 每轮 append assistant + 所有 tool result 到 `messages`，10 轮 + history 会超小模型（DeepSeek/GLM/Qwen）context。刚修完的 blank-response 问题就是 token 超限的表现。

### 整改
在 `run` 循环里加 context 预算检查:
- 估算 messages 总 token（简单按字符数/4 估）
- 超阈值时压缩：保留 system + 最近 K 轮 + 首条 user；中间旧的 tool result 用摘要替换（"[compiled OK]" / "[backtest: 42 trades]"）

**REUSE 核对**: token 估算——检查 systemai 是否已有估算函数；无则 `NEW` 简单实现。
**验收**: 20 轮长对话不触发 context 超限错误。

---

## §8 🟢 G8 — 完整工具进度反馈

### 问题根因
`toolStream`（`generator_agent.go:49`）只对 `compile_python` / `read_kline` 发 phase，新工具（run_backtest/edit_code）不发 → 前端卡在旧 phase。

### 整改
`toolStream` 的 switch 补齐所有工具 case，run_backtest 发 `phase: "backtesting"`，edit_code 发 `phase: "editing"`。前端 `AgentGenChat.tsx` 对应加 phase 文案。

**验收**: 前端实时显示"回测中…""编辑代码…"。

---

## §9 落地顺序（建议给 DeepSeek 的执行序）

```
第一批（闭环核心，必须一起做）:
  1. G6 ToolInput.RawArgs 富参数化   ← 其它工具的地基
  2. G1 run_backtest 工具
  3. G4 read_current_code 工具
  4. G3 edit_code 工具               ← 依赖 G6

第二批（记忆与规划）:
  5. G2 conversation history 接线    ← 依赖 proto 改动
  6. G7 context 压缩                 ← G2 的安全网，必须配套
  7. G5 update_plan 工具

第三批（打磨）:
  8. G8 工具进度反馈补齐
```

每个任务提交时按 AGENTS.md 规范标注 `REUSE: <symbol> @ <file:line>` 或 `NEW: 无现成能力（已搜: <关键词>）`。

---

## §10 硬约束提醒（DeepSeek 必读）

- ❌ 回测/盈亏 metrics 严禁 float64，用 `decimal.Decimal`，仅摘要给 LLM 时转字符串
- ❌ 工具间通信/持久化严禁 JSON（LLM tool_call args 的 json.Marshal 是 OpenAI 协议要求，已豁免；其余禁止）
- ❌ 文件行数: Go 软 300 硬 450。`agent_tools.go` 加 4 个工具会超标 → **按工具类型拆分**（如 `agent_tools_compile.go` / `agent_tools_backtest.go` / `agent_tools_edit.go`）
- ✅ 部署: `docker compose build backend && docker compose up -d backend`
- ✅ 提交前: `go build ./...` + `go run ./tools/check-file-lines --strict` + `bash scripts/gen_capability_map.sh`
