# 02 — 后端管道：工具系统 + AgentLoop + 双通道流式

## AgentLoop 架构（Think→Act→Observe→Repeat）

```
用户消息
  │
  ▼
runAgentLoop (generator_agent.go)
  │
  ├─ 1. 构建 ToolRegistry
  │    └─ buildPythonToolRegistry(result, g.mkt, msg.BacktestConfig)
  │         ├─ writeStrategyTool   ← PRIMARY: code 必填 → compile → real backtest
  │         ├─ editCodeTool        ← old_string → new_string 精确替换
  │         ├─ readCurrentCodeTool ← 读取当前代码(带行号)
  │         └─ updatePlanTool      ← JSON [{step, status}]
  │
  ├─ 2. 加载对话历史
  │    └─ conversationRepo.GetMessages(ctx, userID, convID)
  │         → []systemai.ChatMessage → 注入 messages[1:] (system+first_user+history+current_user)
  │
  ├─ 3. 流式回调 (三个独立通道)
  │    ├─ streamChunk(delta)
  │    │    └─ stripThinkBlocks → {Phase:"generating", Delta:cleaned}
  │    ├─ reasoningStream(delta)
  │    │    └─ {Phase:"thinking", Reasoning:delta}       ← I3: 永不混入 content
  │    └─ toolStream(tc, tr)
  │         ├─ "write_strategy"  → {Phase:"generating", PythonSource:code}
  │         ├─ "edit_code"       → {Phase:"editing", PythonSource:code}
  │         ├─ "read_current_code" → {Phase:"reading"}
  │         └─ "update_plan"     → {Phase:"planning", Delta:json}
  │
  ├─ 4. NewAgentLoop(registry, llmStream, streamChunk, toolStream, reasoningStream)
  │    └─ RunWithHistory(ctx, sysPrompt, userPrompt, history, userID)
  │         │
  │         ▼ AgentLoop.run() 内部循环:
  │         ┌──────────────────────────────────────────┐
  │         │ while true:                               │
  │         │   llmStream(messages, toolDefs, onChunk)  │
  │         │     ├─ Content  → roundBuf + streamChunk  │
  │         │     ├─ Reasoning → reasoningBuf + reasoningStream │
  │         │     └─ ToolCalls → toolCalls[]            │
  │         │                                          │
  │         │   I3 guard: 无 content + 无 toolCall     │
  │         │     → error (reasoning 永不当答案)        │
  │         │                                          │
  │         │   收敛守卫: finish_reason=length          │
  │         │     OR (roundBuf empty + reasoning>500)  │
  │         │     → inject correction + retry (≤2次)   │
  │         │                                          │
  │         │   §3.1b guard: code-in-text without      │
  │         │     write_strategy call                  │
  │         │     → inject correction + retry (≤2次)   │
  │         │                                          │
  │         │   无 tool call → 检查 [TOOL: name args]  │
  │         │     → 有 → parse + execute               │
  │         │     → 无 → return (对话结束)              │
  │         │                                          │
  │         │   执行 tool calls → 注入 tool results    │
  │         │     → 继续循环 (LLM 看到结果后决策)       │
  │         └──────────────────────────────────────────┘
  │
  ├─ 5. 持久化对话 (generator_agent.go after loop)
  │    ├─ AddMessage(userID, convID, "user", userPrompt)
  │    ├─ AddMessage(userID, convID, "assistant", raw)
  │    └─ Touch(convID)
  │
  └─ 6. 最终流
       ├─ 成功: {Phase:"done", PythonSource:result.PythonSource}
       └─ 失败: {Phase:"done", PythonSource:result.PythonSource, Error:err}
```

## writeStrategyTool 内部流程

```
write_strategy(code: string)
  │
  ├─ 1. I1: PythonSource = code (唯一真源)
  │
  ├─ 2. Compile: mql2go.CompilePythonWithCoverage(code)
  │    └─ 失败 → {Success:false, compiled:"false", error:...}
  │
  ├─ 3. Backtest (条件性):
  │    ├─ 前提: mkt != nil && cfg != nil && cfg.Symbol != ""
  │    │
  │    ├─ 3a. fetchBarsForBacktest(ctx, mkt, cfg)
  │    │    └─ mkt.GetKlines(symbol, timeframe, from, to)
  │    │    └─ 失败 → {tier:"compile_only", backtest_note:"no symbol"}
  │    │
  │    ├─ 3b. runVMBacktest(ctx, runner, cfg, bars, params)
  │    │    └─ 30s timeout → real VM backtest via SimBroker
  │    │    └─ 失败 → {backtest_error:..., tier:"compile_only"}
  │    │
  │    └─ 3c. 判定 tier:
  │         ├─ symbol+capital 齐全 → tier:"performance" (I2b)
  │         └─ 参数不全 → tier:"smoke" (I2a, 不展示绩效数字)
  │
  └─ 4. 返回结构化结果 (I4: transparent inputs)
       { compiled, coverage, tier, total_trades, win_rate, total_return,
         max_drawdown, sharpe, symbol, timeframe, date_range,
         initial_capital, commission }
```

## REUSE 锚点

写任何 agent 工具前，先确认这些现成能力是否已经覆盖：

| 能力 | 位置 | REUSE 方式 |
|------|------|-----------|
| Python 编译+覆盖度 | `tools/mql2go/CompilePythonWithCoverage` | 直接调用 |
| VM 回测 | `backtest_helpers.go:18 runVMBacktest` | 传入 runner+cfg+bars |
| 回测结果转 proto | `backtest_helpers.go:60 buildBacktestResultProto` | 直接调用 |
| Bars 获取 | `generator_agent.go → fetchBarsForBacktest` 或 `gateway.go:298-327` | 同模式 |
| 完整 compile→backtest 流程 | `gateway.go:114-156` | 参照实现 |
| 工具注册表 | `agent_tools.go buildPythonToolRegistry` | AddPreTool() |
| 工具三件套 | `agent_tools_write.go` — Name()/Schema()/Run() | 照抄结构 |

## 工具开发模板

```go
// 新工具: myNewTool
type myNewTool struct {
    result *generateState  // 需要写入 PythonSource 时必需
    // 其他依赖...
}

func (t *myNewTool) Name() string { return "my_new_tool" }

func (t *myNewTool) Schema() systemai.ToolDefinition {
    return systemai.ToolDefinition{
        Type: "function",
        Function: systemai.ToolDefFunction{
            Name:        "my_new_tool",
            Description: "工具描述——写清楚什么时候用、参数含义",
            Parameters: map[string]any{
                "type":     "object",
                "required": []string{"param1"},
                "properties": map[string]any{
                    "param1": map[string]any{
                        "type":        "string",
                        "description": "参数1的描述",
                    },
                },
            },
        },
    }
}

func (t *myNewTool) Run(_ context.Context, in connectai.ToolInput) connectai.ToolOutput {
    // 1. 从 in.RawArgs 读取参数 (parseToolArguments @ agent_loop.go:228 已填充)
    val, _ := in.RawArgs["param1"].(string)
    // 2. 执行业务逻辑
    // 3. 返回结构化结果
    return connectai.ToolOutput{Success: true, Output: map[string]string{"key": "value"}}
}
```

注册: `buildPythonToolRegistry` 中添加 `reg.AddPreTool(&myNewTool{result: result})`

## LLM 流式管道 (reasoning/content 分离)

```
DeepSeek/OpenAI SSE stream
  │
  ▼
chat_stream.go: doChatStreamRequest()
  │ 解析 SSE 事件: {"choices":[{"delta":{"content":"...","reasoning_content":"..."}}]}
  │
  ├─ c.Delta.Content          → ChatStreamChunk.Content
  └─ c.Delta.ReasoningContent → ChatStreamChunk.Reasoning
  │
  ▼
agent_loop.go: llmStream callback
  ├─ chunk.Content   → roundBuf + streamChunk(delta)    ← 前端可见文本
  └─ chunk.Reasoning → reasoningBuf + reasoningStream(delta) ← 前端折叠块
  │
  ▼
generator_agent.go: stream callbacks
  ├─ streamChunk → AgentGenerateStrategyChunk{Phase:"generating", Delta:cleaned}
  └─ reasoningStream → AgentGenerateStrategyChunk{Phase:"thinking", Reasoning:delta}
  │
  ▼
前端 agentGen.ts: handleAgentChunk
  ├─ chunk.delta     → onDelta → streamTextRef
  └─ chunk.reasoning → onReasoning → reasoningRef
  │
  ▼
ChatHistory.tsx
  ├─ streamText → StreamContent (纯文本)
  └─ reasoning → CollapsibleBlock(💭, collapsed)
```

**I3 铁律**: reasoning_content 从不出现在 content 通道。content 空 + 无 tool call = 错误（不是回退到 reasoning）。
