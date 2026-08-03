---
name: agent-engine
description: >
  AlphaForge 策略生成 Agent 全栈实现指南。涵盖 I1-I4 四不变量、
  write_strategy 工具系统、reasoning/content 双通道流式管道、
  多轮对话持久化、前端组件树。当需要实现、修改、调试或审查
  Agent 策略生成功能时使用此 skill。
  Triggers on: "agent", "策略生成", "write_strategy", "reasoning",
  "AgentGenChat", "agent_loop", "generator_agent", "对话上下文丢失",
  "思考过程", "回测假验证", "代码即工具产出", "I1", "I2", "I3", "I4",
  "agent-engine", "generateState", "ChatHistory", "工具注册表".
---

# Agent 引擎 — 策略生成全栈指南

> 最后验证: 2026-07-08，对标当前代码库。以 `docs/agent-first-principles-rebuild.md` 为上位设计文档。

## 参考文档

| 编号 | 主题 | 文件 | 说明 |
|------|------|------|------|
| 01 | 前端组件树 | [01-frontend-components.md](references/01-frontend-components.md) | 组件层级、数据流、props 传递 |
| 02 | 后端管道 | [02-backend-pipeline.md](references/02-backend-pipeline.md) | 工具系统、AgentLoop、双通道流式 |
| 03 | 对话持久化 | [03-conversation-persistence.md](references/03-conversation-persistence.md) | 多轮上下文保存/加载全链路 |
| 04 | 不变量与陷阱 | [04-invariants-and-pitfalls.md](references/04-invariants-and-pitfalls.md) | I1-I4 验收标准，常见 bug 及调试方法 |
| 05 | 教训与反模式 | [05-lessons-learned.md](references/05-lessons-learned.md) | 反模式检查清单、正确模式、今天踩过的坑 |

## 架构总览

```
前端 (React)
  RightPanel → StrategyChat → AgentGenChat → ChatHistory
                  │                  │
                  │ conversationId   │ streamText / reasoning / generatedCode
                  ▼                  ▼
              agentGen.ts ──AgentGenerateStrategyRequest──►
              (ConnectRPC streaming client)

后端 (Go)
  gateway.go (ConnectRPC handler)
    └─ generator.go (Generator.Generate)
         └─ generator_agent.go (runAgentLoop)
              ├─ buildPythonToolRegistry(mkt, cfg)
              │    ├─ writeStrategyTool  ← PRIMARY (I1: 唯一代码入口)
              │    ├─ editCodeTool       ← 精确编辑
              │    ├─ readCurrentCodeTool← 读取当前代码
              │    └─ updatePlanTool     ← 分步计划
              │
              ├─ AgentLoop (agent_loop.go)
              │    ├─ streamChunk     → content 通道 (Delta)
              │    ├─ reasoningStream → thinking 通道 (Reasoning)
              │    └─ toolStream      → 工具事件
              │
              └─ 对话持久化 (ai_conversation.go + repository)
                   写入: AddMessage(user/assistant) after each turn
                   读取: GetMessages → history → LLM context
```

## 关键文件索引

### 前端

| 文件 | 职责 |
|------|------|
| `frontend/src/components/strategy/StrategyChat.tsx` | 顶层容器: 工具栏(History/Strategies/Settings) + AgentGenChat |
| `frontend/src/components/strategy/AgentGenChat.tsx` | 对话状态管理: turns[], streamText, reasoning, conversationId |
| `frontend/src/components/strategy/ChatHistory.tsx` | 消息渲染: user气泡、AI卡片(thinking/plan/code/metrics) |
| `frontend/src/components/strategy/chatUtils.tsx` | StreamContent(纯文本)、phaseLabels |
| `frontend/src/components/strategy/CollapsibleBlock.tsx` | 可折叠块(thinking/plan/profile/analysis 共用) |
| `frontend/src/components/strategy/ChatInput.tsx` | 输入框 |
| `frontend/src/components/strategy/StrategyChatHistory.tsx` | 历史对话列表 Drawer |
| `frontend/src/components/strategy/useConversationHandlers.ts` | 对话 CRUD hooks |
| `frontend/src/client/agentGen.ts` | ConnectRPC 流式客户端: agentGatewayClient.generateStrategy() |
| `frontend/src/client/ai.ts` | aiApi (listConversations/createConversation/...) |

### 后端

| 文件 | 职责 |
|------|------|
| `backend/internal/agent/generator.go` | Generator 结构体 + Generate 入口 |
| `backend/internal/agent/generator_agent.go` | runAgentLoop: 工具注册 + stream callbacks + AgentLoop |
| `backend/internal/agent/agent_tools.go` | buildPythonToolRegistry: 依赖注入 mkt+cfg |
| `backend/internal/agent/agent_tools_write.go` | writeStrategyTool: 编译→真回测→写入 PythonSource |
| `backend/internal/agent/agent_tools_edit.go` | readCurrentCodeTool + editCodeTool |
| `backend/internal/agent/agent_tools_plan.go` | updatePlanTool: JSON 数组 [{step, status}] |
| `backend/internal/agent/backtest_helpers.go` | runVMBacktest + buildBacktestResultProto |
| `backend/internal/agent/gateway.go` | ConnectRPC handler: SubmitStrategy + GenerateStrategy |
| `backend/internal/connect/ai/agent_loop.go` | AgentLoop: Think→Act→Observe + code-in-text guard |
| `backend/internal/connect/ai/tool_registry.go` | ToolRegistry: AddPreTool / FindPreTool / BuildToolSchemas |
| `backend/internal/connect/ai/ai_conversation.go` | 对话 CRUD handlers (List/Get/Create/Delete/UpdateTitle) |
| `backend/internal/repository/ai_conversation_repository.go` | PG 持久化: Create/AddMessage/GetMessages/ListByUser |
| `backend/internal/service/systemai/chat_stream.go` | SSE 解析: reasoning_content → ChatStreamChunk.Reasoning |
| `backend/internal/service/systemai/chat.go` | ChatStreamChunk 结构体 (Content/Reasoning separation) |
| `proto/ant/v1/agent_gateway.proto` | Proto 定义: AgentGenerateStrategyChunk (含 reasoning field 15) |

## 核心设计决策

1. **代码是工具产出，不是聊天文本** — `write_strategy(code)` 是代码进入系统的唯一入口
2. **推理与成品永不混淆** — `reasoning_content` → Reasoning field → 💭 折叠块。永不回退当答案。
3. **交付前必经真实回测** — write_strategy 内部: compile → fetchBars → runVMBacktest
4. **单例终稿** — `generateState.PythonSource` 只有 write_strategy/edit_code 写入，前端只渲染一个 Apply 卡片
5. **多轮对话持久化** — conversationId 贯穿全链路，每轮结束自动保存 user+assistant 消息
