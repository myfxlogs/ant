# 03 — 多轮对话持久化全链路

## 问题根因

之前的实现中，`AgentGenChat` 每次 mount 都生成新的 `crypto.randomUUID()`，从不复用。
`StrategyChat` 虽然已有完整的对话 CRUD UI 和 handlers，但 **从未把 `activeConvId` 传给 `AgentGenChat`**。
后端 `generator_agent.go` 只读取历史（`GetMessages`）但从不写入（无 `AddMessage` 调用）。

结果：agent 每次都说"这是我们的第一次交互"——上下文完全丢失。

## 修复后的全链路

```
┌─ 前端 ────────────────────────────────────────────────────────┐
│                                                                 │
│  StrategyChat (顶层容器)                                        │
│    state: activeConvId, conversations[]                         │
│                                                                 │
│    mount → fetchConversations()                                 │
│      → aiApi.listConversations()                                │
│      → ConnectRPC: ListConversationsRequest                     │
│                                                                 │
│    用户点击 [+ New]                                              │
│      → aiApi.createConversation(title)                          │
│      → setActiveConvId(newId)                                   │
│      → AgentGenChat remount (key={newId})                       │
│                                                                 │
│    用户点击历史对话                                              │
│      → aiApi.getConversation(id)                                │
│      → setActiveConvId(id)                                      │
│      → AgentGenChat remount (key={id})                          │
│                                                                 │
│  AgentGenChat (key={activeConvId})                              │
│    conversationIdRef = props.conversationId || crypto.randomUUID()│
│                                                                 │
│    handleSend() → agentGenerateStrategyStream({                 │
│      message, symbol, timeframe,                                │
│      conversationId: conversationIdRef.current  ← 关键!         │
│    })                                                           │
│                                                                 │
└────────────────────┬────────────────────────────────────────────┘
                     │
    AgentGenerateStrategyRequest { conversation_id: "uuid..." }
                     │
┌─ 后端 ────────────▼────────────────────────────────────────────┐
│                                                                 │
│  generator_agent.go: runAgentLoop()                             │
│                                                                 │
│  1. 解析 conversationId                                         │
│     convID = uuid.Parse(msg.ConversationId)                     │
│                                                                 │
│  2. 加载历史 (已存在)                                            │
│     msgs = conversationRepo.GetMessages(ctx, userID, convID)    │
│     → history = [{role, content}, ...]                          │
│                                                                 │
│  3. 自动创建 (首次对话)                                          │
│     if convID == uuid.Nil:                                      │
│       conversationRepo.Create(ctx, userID, title)               │
│                                                                 │
│  4. AgentLoop.RunWithHistory(sysPrompt, userPrompt, history)    │
│     (history 注入 LLM context, LLM 能看到之前的对话)              │
│                                                                 │
│  5. 持久化本轮                                                   │
│     conversationRepo.AddMessage(userID, convID, "user", prompt) │
│     conversationRepo.AddMessage(userID, convID, "assistant", raw)│
│     conversationRepo.Touch(convID)                              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 后端 API

全部在 `backend/internal/connect/ai/ai_conversation.go`:

| RPC | 入参 | 出参 | 说明 |
|-----|------|------|------|
| `ListConversations` | - | `[{id, title, message_count, created_at}]` | 按 updated_at DESC, LIMIT 100 |
| `GetConversation` | id | `{conversation, messages[{id,role,content}]}` | JOIN 验证所有权 |
| `CreateConversation` | title | `{id, title, created_at}` | title 最长 200 字符 |
| `DeleteConversation` | id | - | 级联删 messages → 删 conversation |
| `UpdateConversationTitle` | id, title | - | title 最长 200 字符 |

## 数据库

表: `ai_conversations` (id, user_id, title, created_at, updated_at)
表: `ai_messages` (id, conversation_id, role, content, created_at)

查询安全性: 所有操作通过 `user_id` 过滤，防止越权访问。

## 前端 API 客户端

`frontend/src/client/ai.ts`:
```typescript
aiApi.listConversations()          → ListConversationsResponse
aiApi.getConversation(id)          → GetConversationResponse (含 messages)
aiApi.createConversation(title)    → CreateConversationResponse
aiApi.deleteConversation(id)       → {}
aiApi.updateConversationTitle(id, title) → {}
```

## 调试检查清单

对话上下文丢失时，按以下顺序排查:

1. **前端是否发送了 conversationId?**
   - 检查 `agentGen.ts:52-72` — `conversationId: input.conversationId || ''` 是否在 request 中
   - 检查 `AgentGenChat.tsx` — `conversationIdRef.current` 是否被正确传入
   - 检查 `StrategyChat.tsx:115` — `conversationId={activeConvId}` 是否传给 AgentGenChat

2. **后端是否接收到 conversationId?**
   - 日志: `generator_agent.go` 中 `msg.ConversationId` 是否非空
   - 若为空 → 前端没发或 AgentGenChat 没收到 prop

3. **后端是否加载了历史?**
   - `GetMessages` 返回什么? → 检查 `ai_messages` 表是否有数据
   - 若表为空 → 上一轮没有保存消息

4. **后端是否保存了消息?**
   - `generator_agent.go` 中 `AddMessage` 调用是否执行?
   - 检查 `convID != uuid.Nil && g.conversationRepo != nil` 条件

5. **对话是否被正确路由?**
   - 新对话 → `CreateConversation` → convID 写入 DB
   - 加载旧对话 → `GetMessages` → history → LLM
   - 同一 `activeConvId` → 同一 `convID` → 上下文正确累积
