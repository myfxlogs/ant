# 04 — 不变量（I1-I4）与常见陷阱

## 四条不变量（来自 `docs/agent-first-principles-rebuild.md` §1）

这些是验收标准，不是建议。任何实现都必须恒真：

### I1 — 唯一确定的成品

> 任一轮对话结束，系统对"最终代码是哪份"有且仅有一个确定答案。

- **实现**: `generateState.PythonSource` 只由 `writeStrategyTool.Run()` 和 `editCodeTool.Run()` 写入
- **前端**: 只渲染一个 Apply 卡片 (`generatedCode`)
- **验证**: 删除 `compilePythonTool`（它也能写 PythonSource → 造成歧义）
- **禁止**: 正则 `ExtractCode` 作为代码获取主路径（只能作为 code-in-text guard 的检测手段）

### I2 — 交付前必经行为验证

> 编译通过 ≠ 满足需求，必须在真实 bars 上执行。

- **I2a (冒烟测试)**: 编译 + 真实 bars 执行不崩 + 产生交易。验证"逻辑会不会跑"。
- **I2b (绩效评估)**: 收益/回撤/夏普，只在 symbol/周期/资金为用户明确选定时才呈现。
- **实现**: `writeStrategyTool.Run()` 内部: compile → fetchBars → runVMBacktest
- **tier 判定**: symbol 已选 → `"performance"`; symbol 未选 → `"smoke"` (只显示冒烟测试结论)
- **禁止**: 只编译冒充回测（旧 `run_backtest` 的假验证）
- **注意**: `total_trades==0` 是信号不是 bug，别陷入无限修复循环

### I3 — 思考与成品永不混淆

> 模型的推理（scratchpad）绝不能冒充成品交付。

- **实现**: LLM stream → `reasoning_content` → `ChatStreamChunk.Reasoning` → `reasoningStream` callback → `AgentGenerateStrategyChunk.Reasoning` → 前端 💭 折叠
- **禁止**: reasoning 当 content 回退（已从 `agent_loop.go:154` 删除，`chat.go:297` 不存在此路径）
- **禁止**: reasoning 字段通过 `Delta` 发送（已修复: `generator_agent.go:71` 使用 `Reasoning: delta`）

### I4 — 回测输入必须透明

> 呈现给用户的回测结果必须同时展示输入参数。

- **实现**: `writeStrategyTool` 返回结果包含 `symbol`, `timeframe`, `date_range`, `initial_capital`, `commission`
- **禁止**: 绩效数字旁边找不到输入参数
- **注意**: symbol 未选时绝不瞎填，降级为冒烟测试并明确标注

## 常见陷阱

### 陷阱 1: 在 AgentGenChat 里生成新的 UUID 而不用父组件传入的

**症状**: 对话列表有历史记录，但 agent 不记得之前说了什么。
**原因**: `conversationIdRef = useRef(crypto.randomUUID())` 没有用 `props.conversationId`。
**修复**: `useRef(conversationId || crypto.randomUUID())` + `StrategyChat` 传 `conversationId={activeConvId}`。

### 陷阱 2: 忘记后端保存消息

**症状**: `ai_messages` 表为空，GetMessages 始终返回空。
**原因**: `generator_agent.go` 有 GetMessages 但没有 AddMessage。
**修复**: AgentLoop 完成后调用 `AddMessage("user")` + `AddMessage("assistant")` + `Touch()`。

### 陷阱 3: reasoning 通过 Delta 而非 Reasoning 字段发送

**症状**: 前端 streamText 里包含大段思考内容，刷屏。
**原因**: `generator_agent.go:71` 写了 `{Phase:"thinking", Delta:delta}` 应该是 `Reasoning:delta`。
**修复**: 改为 `{Phase:"thinking", Reasoning:delta}`。

### 陷阱 4: compile_python 与 write_strategy 并存导致 LLM 选错

**症状**: LLM 总是选 compile_python 而不是 write_strategy，代码通过自由文本而非工具产出。
**原因**: compile_python 在工具注册表中，LLM 认为它是更简单的选项。
**修复**: 从 `buildPythonToolRegistry` 中删除 compile_python（已做）。write_strategy 是唯一的代码提交路径。

### 陷阱 5: 前端 StreamContent 渲染代码块

**症状**: 用户看到代码块弹出 Apply 按钮，I1 被破坏。
**原因**: `chatUtils.tsx` 的 `parseCodeBlocks` + `CodeBlock` 允许自由文本渲染代码。
**修复**: `StreamContent` 改为纯文本渲染。代码只能通过 `generatedCode` (write_strategy 产出) 展示。

### 陷阱 6: Apply 按钮在迭代中仍可点击

**症状**: 用户 Apply 了一个还在修改中的半成品代码。
**原因**: `ChatHistory.tsx` 的 Apply 按钮没有禁用条件。
**修复**: `disabled={turn.phase !== 'done' || !!(compileError || backtestError || error)}`。

### 陷阱 7: conversationId 在请求中漏传

**症状**: 切换到旧对话后 agent 仍然失忆。
**原因**: `agentGen.ts:52` 构造 request 时没有 `conversationId` 字段。
**修复**: `conversationId: input.conversationId || ''`。

### 陷阱 8: AgentGenChat 不接收 conversationId prop

**症状**: StrategyChat 传了 activeConvId 但 AgentGenChat 没有声明这个 prop。
**原因**: Props 接口和函数签名都没有 `conversationId`。
**修复**: 添加到 Props + 函数解构参数。

## 管道调试方法

出问题时按数据流方向逐层排查：

```
前端检查:
  1. StrategyChat: activeConvId 是否有值? 点击历史对话后是否变化?
  2. AgentGenChat: conversationId prop 是否接收到? conversationIdRef 是否使用 prop?
  3. agentGen.ts: 请求中 conversationId 字段是否填充?
  4. Network tab: AgentGenerateStrategyRequest 体中有 conversation_id 吗?

后端检查:
  5. generator_agent.go: msg.ConversationId 是否非空? (加 log)
  6. conversationRepo.GetMessages: 返回什么? (查 ai_messages 表)
  7. conversationRepo.AddMessage: 是否被调用? (查 ai_messages 表有无新行)
  8. history 是否注入 messages? (agent_loop.go RunWithHistory)

流式检查:
  9. chunk.reasoning 是否非空? (generator_agent.go reasoningStream)
  10. chunk.delta 是否不含 reasoning 内容? (stripThinkBlocks 是否工作)
  11. 前端 onReasoning 是否被调用? (reasoningRef 是否累积)
  12. ChatHistory 是否渲染 💭 CollapsibleBlock?
```
