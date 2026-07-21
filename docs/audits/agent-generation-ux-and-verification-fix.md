# 修复方案: Agent 生成体验 4 大 UX Bug + run_backtest 真实化

> 交付对象: DeepSeek 落地实现
> 背景: 空响应已修（reasoning_content 多通道解析），但修复暴露了下游 UX 问题
> 根因均已由 Cascade 核实到具体文件行号，请勿偏离
> 日期: 2026-07-08

---

## §0 一个关键背景（先理解，再动手）

用户用的 `deepseek-v4-flash`（聚合网关模型）把**整段英文思维链 + 多个代码草稿**全塞进 `reasoning_content` 通道，且经常在推理阶段就耗尽 `max_tokens`（`finish_reason=length`），从没产出最终 `content`。

我们之前的修复让 `reasoning_content` 在 content 为空时**兜底当答案**（`agent_loop.go:120-122`）。这解决了"空响应"，但把**英文思维链 + 多个草稿代码块**直接当答案显示了。这是 Bug 1、Bug 2 的共同根源。

**核心原则（对齐 Claude Code）**: `reasoning_content` 是"思考过程"，必须与"最终答案"分离 —— 思考折叠展示，最终代码单独作为唯一交付物。

---

## §1 Bug 1 — 中文输入，英文输出

### 根因（已核实）
两个叠加因素：
1. **英文思维链被当答案**: 网关模型的 `reasoning_content` 是英文，我们把它当答案流式出来了（§0）。
2. **系统提示没强制语言**: `@/opt/ant/backend/internal/ai/locale_agent_en.go:6` 的默认英文 prompt 没有"用用户的语言回复"指令。且当 `msg.Locale` 未正确传/归一化时会 fallback 到英文 prompt（`@/opt/ant/backend/internal/agent/generator_agent.go:36-37`）。

### 修复
1. **§2 的思考/答案分离修好后，Bug 1 大部分自动消失**（不再显示英文 CoT）。
2. 在**所有语言**的 agent system prompt 末尾加一句显式指令（`locale_agent_en.go` 及对应 ZH/ZHTW/JA/VI 文件）:
   ```
   ## Language
   Always respond in the SAME language as the user's latest message. If the user writes in Chinese, respond in Chinese.
   ```
3. 核对 `ai.NormalizeLocale`（`@/opt/ant/backend/internal/ai/locale.go`）是否正确把前端传来的 `i18n.language`（可能是 `zh-CN` / `zh-Hans`）映射到 `zh`。前端在 `@/opt/ant/frontend/src/client/agentGen.ts:59` 传 `locale: i18n.language || 'en'`。若映射缺失 → 补上。

**验收**: 中文 prompt → 思考区（若展示）和最终解释均为中文。

---

## §2 Bug 2 — 生成多个代码块，用户不知哪个是最终代码

### 根因（已核实）
1. **后端**: `reasoning_content`（含多个 ```python 草稿）被兜底当 `roundText`，通过 `streamChunk` 以 `Phase:"generating"` 的 `delta` 流给前端（`@/opt/ant/backend/internal/agent/generator_agent.go:61-63`）。
2. **前端**: `onDelta` 累积成 `streamText`（`@/opt/ant/frontend/src/components/strategy/AgentGenChat.tsx:53-56`）→ `StreamContent` 用 `parseCodeBlocks` 把**每个**代码块都渲染成带 "Apply" 按钮的卡片（`@/opt/ant/frontend/src/components/strategy/chatUtils.tsx:59-72`）。
3. **权威最终代码没被突出**: 后端 `ExtractCode` 的结果通过 `Phase:"done"` 的 `pythonSource` → `onPythonSource` → 存进 `turn.generatedCode`（`AgentGenChat.tsx:57-60`），但 `ChatHistory` **只渲染 `turn.streamText`，从不渲染 `turn.generatedCode`**（`@/opt/ant/frontend/src/components/strategy/ChatHistory.tsx:154-158`）。所以用户看到的是一堆草稿，唯独看不到那份"官方最终代码"。

### 修复（分离思考 / 答案 / 最终代码）

**后端**（关键——不要再把 reasoning 当 content 流给"生成"阶段）:
1. proto: 给 `AgentGenerateStrategyChunk` 增加一个 `string reasoning` 字段（或复用一个新 `Phase:"thinking"`）。
2. `AgentLoop.run`（`@/opt/ant/backend/internal/connect/ai/agent_loop.go`）: 把 reasoning 和 content 分开回调（见 §4，两者都要**逐块**流）。reasoning 走 thinking 通道，content 走 generating 通道。
3. **不要**再把 reasoning 兜底塞进 `roundText` 后当 content 流给前端展示。兜底逻辑只用于"内部代码提取"（`ExtractCode`），不用于展示。即：reasoning 仅用于（a）thinking 展示、（b）content 全空时的代码提取兜底，二者分离。

**前端**（`ChatHistory.tsx` + `chatUtils.tsx`）:
1. **最终代码作为唯一交付物**: 当 `turn.generatedCode` 存在时，渲染**一个**权威代码卡片（带 Apply），放在显眼位置。这是用户唯一需要 Apply 的东西。
2. **思考过程折叠**: reasoning 文本放进 `CollapsibleBlock`（已有组件，见 `ChatHistory.tsx:9`），标题"💭 思考过程"，默认折叠。里面的草稿代码块**不给 Apply 按钮**（`StreamContent` 传 `onApply={undefined}`）。
3. `streamText` 里若混着 content 文本（非 reasoning），保留渲染但同样不把草稿块当可 Apply 交付物。

**验收**: 复杂 prompt → 聊天区显示"💭 思考过程（折叠）" + **一个**最终代码卡片（带 Apply）。用户一眼看到该 Apply 哪个。

---

## §3 Bug 3 — 点击"应用代码到编辑器"没有自动打开 code tab

### 根因（已核实）
Apply 按钮调用链: `CodeBlock.onApply` → `ChatHistory.onApplyCode` → `AgentGenChat.onApply` → `StrategyChat` 的 `onApplyCode`（`@/opt/ant/frontend/src/components/strategy/StrategyChat.tsx:115`）。但 `useConversationHandlers` 里 `setTab: noop`（`@/opt/ant/frontend/src/components/strategy/StrategyChat.tsx:65`）—— **apply 只写入代码，从不切换 tab**。

### 修复
1. 找到 `onApplyCode` 的定义处（`StrategyChat` 的父组件，即 workspace 页面，管理 code/chat tab 的 `activeTab` state）。
2. 让 apply 动作在写入代码的同时 `setActiveTab('code')`（或对应的 code tab key）。
3. 把真实的 `setTab` 传进 `useConversationHandlers`（替换 `noop`），或在 `onApplyCode` 回调内直接切 tab。

**定位提示**: workspace tab 的容器很可能在 `pages/strategy/` 下，`grep "activeTab\|Tabs\|onApplyCode"` 找持有 tab state 的父组件。

**验收**: 点 Apply → 代码写入编辑器 **且** 自动跳到 code tab。

---

## §4 Bug 4 — 没有流式输出（前端一次性显示，长时间空等）

### 根因（已核实）
前端**支持**增量渲染（`onDelta` 累加 `streamText`）。但后端 `AgentLoop.run` 的 `onChunk` 回调**只累积不转发**：

```@/opt/ant/backend/internal/connect/ai/agent_loop.go:96-113
		err := a.llmStream(ctx, messages, a.toolDefs, func(chunk systemai.ChatStreamChunk) error {
			roundBuf.WriteString(chunk.Content)
			reasoningBuf.WriteString(chunk.Reasoning)
			// Collect any tool calls from the final chunk.
			...
			return nil
		})
```

真正的 `a.streamChunk(...)` 在整轮 LLM 响应**完全结束后**才调用一次（`@/opt/ant/backend/internal/connect/ai/agent_loop.go:128-130`）。所以对推理模型：用户空等整段推理完成 → 一次性 dump。

### 修复
在 `onChunk` 回调内**逐块**转发（同时保留累积用于最终提取）:
```go
err := a.llmStream(ctx, messages, a.toolDefs, func(chunk systemai.ChatStreamChunk) error {
    if chunk.Content != "" {
        roundBuf.WriteString(chunk.Content)
        if a.streamChunk != nil {
            _ = a.streamChunk(chunk.Content) // 逐块流 content
        }
    }
    if chunk.Reasoning != "" {
        reasoningBuf.WriteString(chunk.Reasoning)
        if a.reasoningStream != nil {
            _ = a.reasoningStream(chunk.Reasoning) // 逐块流 thinking（§2 新通道）
        }
    }
    // tool calls 累积不变
    return nil
})
```
然后**删除/调整**轮末那次一次性 `a.streamChunk(roundText)`（`agent_loop.go:128-130`），避免重复。注意：`fullBuf.WriteString(roundText)`（用于最终 `ExtractCode`）保留。

需要给 `AgentLoop` 加一个 `reasoningStream func(delta string) error` 字段，并在 `generator_agent.go` 里用 `Phase:"thinking"` 或新 `reasoning` 字段接线（配合 §2）。

**⚠️ SSE flush**: 确认 ConnectRPC 的 server stream 每次 `stream.Send` 会立即 flush（ConnectRPC 默认逐消息 flush，一般 OK）。若前端仍不增量，检查 nginx 是否缓冲 SSE（`proxy_buffering off`）。

**验收**: 复杂 prompt → 思考文字逐字/逐块出现，用户不再空等。

---

## §5 Part B — run_backtest 真实化（当前是假的）

### 根因（已核实）
`@/opt/ant/backend/internal/agent/agent_tools_backtest.go:31-50` 的 `run_backtest` **只是重新编译**，没跑回测。注释自己写着 "Phase 2: will run a real backtest"。它甚至和 `compile_python` 重复，还误导 LLM "编译通过=策略正确"。

### 修复（复用现成回测引擎）
**REUSE 现成能力**（`bash scripts/cap.sh backtest` 确认）:
- **`REUSE: runVMBacktest @ /opt/ant/backend/internal/agent/backtest_helpers.go:18`** —— 已被 SubmitStrategy 和 Generator 共用的 VM 回测执行器
- **`REUSE: buildBacktestResultProto @ backtest_helpers.go:60`** —— 回测结果转 proto
- 编译到 VMRunner + 取 bars: 参照 SubmitStrategy 现有流程（`grep runVMBacktest` 找调用方，照抄 VMRunner 构建 + `g.mkt.GetKlines` 取 bars）

**落地步骤**:
1. `runBacktestTool` 结构体增加依赖: `mkt repository.MarketDataStore`、`cfg *antv1.AgentBacktestConfig`（workspace 的 symbol/timeframe/资金等）。
2. `buildPythonToolRegistry`（`@/opt/ant/backend/internal/agent/agent_tools.go:52`）签名增加 `mkt` + `cfg`，从 `runAgentLoop`（`generator_agent.go`）传入 `g.mkt` 和 `msg.BacktestConfig`。
3. `runBacktestTool.Run`:
   - 用 `t.result.PythonSource` 编译成 VMRunner（照抄 SubmitStrategy 的编译步骤）
   - `t.mkt.GetKlines(ctx, cfg.Symbol, "", cfg.Timeframe, ...)` 取 bars
   - 调 `runVMBacktest(ctx, runner, cfg, bars, params)`
   - 返回摘要给 LLM: `total_trades / win_rate / total_return / max_drawdown / sharpe`
4. LLM 拿到真实 trades/PnL 后可自我修正（如 trades=0 → 放宽入场）。

**⚠️ 精度红线**: 返回给 LLM 的摘要用字符串（`r.Metrics` 里的 float64 是回测 VM 边界，勿在应用层再引入 `decimal.NewFromFloat`；注意 `backtest_helpers.go:74` 已有一个已知 float64 边界，别扩散）。

**⚠️ 别再让 run_backtest 退化成 compile**: 若 bars 不足/无数据，返回明确错误让 LLM 知道，而不是 fallback 到"只编译"。

**验收**: LLM 生成策略 → compile_python 通过 → run_backtest 返回真实 `total_trades>0` + 胜率/PnL → trades=0 时 LLM 自动调整重试。

---

## §6 落地顺序建议

```
第一批（UX 闭环，用户立刻有感）:
  1. Bug 4 逐块流式（agent_loop.go onChunk）      ← 最简单，收益最大
  2. Bug 2 思考/答案分离（proto reasoning 字段 + 前后端）
  3. Bug 1 语言指令（prompt + NormalizeLocale 核对）  ← 依赖 Bug 2
  4. Bug 3 Apply 切 tab（前端）

第二批（能力升级）:
  5. Part B run_backtest 真实化
```

---

## §7 硬约束提醒（DeepSeek 必读 —— 上次你漏了转发，这次逐条核对）

- ✅ **Bug 4 的核心是"逐块转发"**: onChunk 里必须调 `streamChunk`/`reasoningStream`，不是只累积。这正是上次同类修复漏掉的模式，务必核对。
- ✅ proto 改动后跑 `buf generate`（或项目的 gen 脚本），前后端 `_pb` 同步。
- ❌ 不新增 REST/WebSocket/JSON 持久化；价格/盈亏用 `decimal.Decimal`。
- ✅ 提交前: `go build ./...` + `cd backend && go run ./tools/check-file-lines --strict` + 前端 `pnpm build`（或项目对应命令）。
- ✅ 后端部署: `docker compose build backend && docker compose up -d backend`
- ✅ 前端部署: `docker cp frontend/dist/. alphaforge-frontend:/usr/share/nginx/html/ && docker exec alphaforge-frontend nginx -s reload`
- ✅ 每个新 file/function 标注 `REUSE:` 或 `NEW:`（Part B 已给出 REUSE 目标）。
