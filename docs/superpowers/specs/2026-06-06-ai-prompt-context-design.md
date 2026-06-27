# AI 策略生成 — Prompt Context 架构设计

> **⚠️ 注意：** AI Prompt 模板当前面向 Python 策略代码生成。迁移至 Go SDK 后需更新系统 Prompt（见 ADR-0021）。

- **日期**：2026-06-06
- **状态**：Draft
- **关联**：spec/26、ADR-0017

## 1. 问题陈述

### 1.1 原设计假设已过时

spec/26 + ADR-0017 设计的 `conversation_store.go`（滑动窗口 + 策略上下文摘要）假设了一个**对话驱动**的交互模型：用户打开独立的 AI 策略对话，从模糊想法开始，AI 引导澄清→生成→迭代。

实际情况是**代码编辑器驱动**：workspace 页面里，用户看着代码、跑着回测、盯着图表，AI 是嵌入代码编辑流程的辅助工具。上下文来自 workspace 状态（code/symbol/timeframe/backtest 结果），不来自聊天历史。

**结论**：`conversation_store.go` 不建。用 `prompt_context.go`（轻量纯函数上下文构建器）替代。

### 1.2 修复场景断流

当前 `CodeAssistService.reviseCodeStream` 对修改和修复用同一套 system prompt。用户把验证报错信息发给 AI 要求修复时，LLM 倾向于"回答问题"而非"输出修正后的代码"——因为错误信息看起来像问题，LLM 的默认行为是解释。

**根因**：一行 `detectMode()` 只判断 `有code/无code`，未区分修改、修复、讨论三种意图。

## 2. 架构决策

### 2.1 conversation_store.go → prompt_context.go + conversation_session.go

原 ADR-0017 的 `conversation_store.go` 设计包含两个不同职责，需要拆开评估：

| 原设计功能 | 评估 | 决策 |
|-----------|------|------|
| 滑动窗口 20 条消息 | 对独立聊天应用有价值，对 workspace 过重 | **取消** |
| 消息自动摘要 | 摘要丢失精度，结构化数据直接注入更好 | **取消** |
| 策略上下文 JSONB 存储 | 代码在编辑器里，回测在页面上，不需要冗余存储 | **取消** |
| **服务端消息持久化** | 跨设备访问必需，刷新不丢必需 | **保留，简化** |
| **会话与策略关联** | 用户在同一策略上持续协作，需要恢复上下文 | **保留，新增** |

**结论**：拆成两个轻量组件，各司其职——

| 组件 | 职责 | 替代原设计的什么 |
|------|------|-----------------|
| `prompt_context.go`（新建） | 模式判定 + 专用 prompt 构建，纯函数 | 替代"策略上下文注入" |
| `conversation_session.go`（新建） | 服务端消息持久化 + 策略关联，薄封装 | 替代"滑动窗口存储"，但只做 append/read，不做窗口/摘要 |

### 2.2 为什么需要服务端持久化

策略开发不是一次性交互。真实场景：

```
桌面端: "做布林带策略" → 生成 → 跑回测 → "加止损" → 修改 → 再跑
  ↓ 出门
手机端: 打开同一策略 → 聊天记录没了 → 忘了之前 AI 改过什么 → 重来
  ↓ 第二天
桌面端: 刷新页面 → 聊天记录没了 → 上下文断裂
```

workspace 其他状态（代码、回测结果）都在服务端，唯独 AI 对话在浏览器内存里——这是断裂的。

### 2.3 设计原则

1. **用户无感** — 不要求用户手动"创建对话"或"切换对话"。打开 workspace，AI 聊天历史自动恢复
2. **策略绑定** — 会话自动绑定到策略。同一策略 = 同一会话 = 跨设备共享
3. **最小接口** — 只做 append（发消息时自动存）和 read（打开 workspace 时自动加载），不做滑动窗口、摘要、手动 CRUD
4. **复用已有基础设施** — 底层用 `AIConversationRepository` 的 PG 表，不新建表

### 2.4 prompt_context.go（纯函数上下文构建器）

```
输入：workspace 结构化状态
  ├── code (当前策略代码)
  ├── message (用户最新输入)
  ├── symbol / timeframe
  ├── backtestMetrics (可选，回测结果)
  └── validationErrors (可选，验证报错列表)

输出：模式判定 + 模式专用 system prompt + 用户消息
  ├── ModeGenerate  — 无 code，从零生成
  ├── ModeRevise    — 有 code，修改意图
  ├── ModeRepair    — 有 code，含错误关键词（关键修复）
  └── ModeDiscuss   — 有 code，提问/分析意图
```

纯函数，无状态，不持久化。被 `CodeAssistService` 和 `StrategyGenerationService` 共用。

### 2.5 conversation_session.go（服务端消息持久化）

```
职责：薄封装 AIConversationRepository，提供策略级会话管理

接口：
  GetOrCreateSession(ctx, userID, strategyKey) → sessionID
  AppendMessage(ctx, sessionID, role, content)
  GetMessages(ctx, sessionID) → []Message

策略绑定：
  strategyKey = strategy_id (已保存的策略) 
              ∨ user_id + symbol + timeframe (新建未保存的策略)

前端无感：
  打开 workspace → 前端请求 GetMessages(sessionID) → 恢复聊天历史
  发送消息 → 前端正常调 AI RPC → 后端 handler 自动调用 AppendMessage
  换设备 → 同一策略 → 同一 sessionID → 同一聊天历史
```

底层复用已有的 `ai_conversations` + `ai_messages` PG 表，不新建表。

### 2.6 不合并 AI 服务

保持三个独立服务，各有明确职责：

| 服务 | 职责 | 使用方 |
|------|------|--------|
| `StrategyGenerationService` | 从零生成（澄清→模板→生成→合规→回测） | workspace generate 模式 |
| `CodeAssistService` | 已有代码的修改/修复/解释/验证 | workspace 主力 |
| `AIService.Chat/ChatStream` | 通用对话 + 对话 CRUD | 将来独立 AI 页面 |

合并将产生 Go >450 行文件，违反项目约束。各服务职责边界清晰，不重叠。

## 3. 详细设计

### 3.1 意图判定算法

```
输入: code (string), message (string)
输出: InteractionMode (enum)

算法:
1. if code 为空 → ModeGenerate
2. 扫描 message 中的错误关键词
   - "报错", "error", "错误", "traceback", "缺少参数", "missing", 
     "验证失败", "syntax error", "undefined", "未定义", 
     "缺少 required", "参数不足"
   → 命中 + code 非空 → ModeRepair (优先级最高)
3. 扫描 message 中的讨论关键词
   - "为什么", "什么意思", "怎么样", "对吗", "对吗？", "分析",
     "解释", "what", "why", "how", "explain", "对不对"
   → 命中 + code 非空 → ModeDiscuss
4. 扫描 message 中的修改关键词
   - 复用 feedback_router.go 的 phase1Keywords + phase2Keywords
   → 命中 + code 非空 → ModeRevise
5. 默认: ModeRevise (有 code 时默认修改)
```

### 3.2 模式专用 System Prompt

#### ModeGenerate — 从零生成

```python
你是一位专业的量化交易策略工程师。
根据用户的自然语言描述，生成符合规范的 Python 策略代码。

## 策略代码规范
- 必须定义 run(context) 函数
- 返回交易信号字典: {'signal': 'buy'|'sell'|'hold', 'volume': 1.0, ...}
- 可调参数用 # @param 注释标注

## 禁止事项
- 不要使用 eval()、exec()、compile()
- 不要导入 os、subprocess、socket
- 不要使用 open() 文件操作
- 只输出 Python 代码，不要包含解释文字或 markdown 标记
```

#### ModeRevise — 修改已有代码（普通）

```python
你是一位专业的量化交易策略工程师。
根据用户的指令修改以下 Python 策略代码。

## 修改规则
- 保持代码结构和风格不变
- 只修改指令涉及的部分，不要改动其他逻辑
- 保留所有现有的 # @param 注释

## 输出规则
- 输出完整的修改后代码
- 不要输出解释、注释说明、或 markdown 标记
- 第一个字符必须是 import 或 def 或 class 或 #
```

#### ModeRepair — 修复错误（关键）

```python
你是一个交易策略代码修复工具。你的唯一任务是修复验证/运行时错误。

## 严格输出规则 — 违反将导致流水线失败
1. 只输出完整的修正后 Python 代码
2. 不要输出任何解释文字
3. 不要说"here is the fixed code"或"修复后的代码如下"
4. 不要加 markdown 代码块标记 (```python ```)
5. 不要分析错误原因
6. 不要给出建议或提示
7. 如果无法修复，输出原始代码并在代码中加 # FIXME: <原因> 注释

## 需要修复的错误
{error_list}

## 当前代码
```python
{code}
```

## 最终提醒
你的回复将直接写入策略文件并运行。如果回复包含非代码内容，流水线将报错。
输出内容必须以 import、def、class、# 或空行开头。
```

#### ModeDiscuss — 讨论/分析（非代码）

```python
你是一位经验丰富的量化交易策略分析师。
用户正在开发一个交易策略，需要你的专业意见。

## 当前策略代码
```python
{code}
```

请针对用户的问题给出简洁、专业的回答。直接回答问题，不需要客套话。
如果用户问"对吗"或"有没有问题"，请逐一检查：入场逻辑、出场逻辑、止损止盈、仓位管理、边界处理。
```

### 3.3 修复模式的后处理兜底

即使强约束 prompt，LLM 仍可能输出解释文字。`CodeAssistService` 增加后处理：

```go
func extractCodeFromRepair(raw string) string {
    // 1. 尝试提取 ```python ... ``` 代码块
    if code := extractFencedCode(raw); code != "" {
        return code
    }
    // 2. 尝试从 import/def/class/# 开头检测代码段
    if code := extractByHeuristic(raw); code != "" {
        return code
    }
    // 3. 无法提取 → 返回空，前端显示原始回复（不写入编辑器）
    return ""
}
```

### 3.4 上下文注入

`prompt_context.go` 的 `BuildContext()` 返回结构化上下文，各模式 prompt 注入以下信息：

```go
type PromptContext struct {
    Mode             InteractionMode
    SystemPrompt     string
    UserMessage      string
    Code             string
    Symbol           string
    Timeframe        string
    BacktestSummary  string  // 来自 backtest_feedback.go
    ValidationErrors []string
}
```

回测结果通过已有的 `backtest_feedback.go` 的 `FormatPromptContext()` 注入到 Revise/Repair 模式的 user message 中。用户说"太激进了"时，AI 能看到 `MaxDD 35%`。

## 4. conversation_session.go 详细设计

### 4.1 会话生命周期

```
用户在 workspace 打开策略 (已有 strategy_id="abc")
  │
  ▼
前端: GET /api/ai/session?strategy_id=abc
  │
  ▼
ConversationSession.GetOrCreate(ctx, userID, "strategy:abc")
  ├── 查询: SELECT id FROM ai_conversations
  │          WHERE user_id=$1 AND strategy_key='strategy:abc'
  ├── 存在 → 返回已有 sessionID + 历史消息
  └── 不存在 → 创建新 session, 返回空消息列表
  │
  ▼
前端: 恢复聊天历史到 AIChatPanel (本地 state + 服务端为真源)
  │
  ▼
用户输入消息 → 前端调 ReviseCodeStream / GenerateStrategy RPC
  │
  ▼
后端 handler:
  1. 执行业务逻辑 (prompt 构建 → LLM 调用 → 流式返回)
  2. 流结束后自动持久化 (见 §4.4)
  │
  ▼
用户换设备/刷新 → 重复上述流程 → 同一 strategy_key → 同一会话 → 聊天历史恢复
```

### 4.2 数据库（复用已有表）

不新建表。复用已存在的 `ai_conversations` 和 `ai_messages`，新增一个索引列：

```sql
-- 已有表，无需创建
-- ai_conversations (id, user_id, title, created_at, updated_at)
-- ai_messages (id, conversation_id, role, content, created_at)

-- 新增：策略关联字段（加一列，不建新表）
ALTER TABLE ai_conversations 
  ADD COLUMN IF NOT EXISTS strategy_key VARCHAR(256);

CREATE UNIQUE INDEX IF NOT EXISTS idx_conv_strategy_key 
  ON ai_conversations(user_id, strategy_key) 
  WHERE strategy_key IS NOT NULL AND strategy_key != '';
```

`strategy_key` 的取值规则：
- 已保存策略：`strategy:<strategy_id>`（如 `strategy:abc-123-def`）
- 新建未保存：`draft:<user_id>:<symbol>:<timeframe>`（如 `draft:user-456:EURUSD:1h`）
- 独立对话（SystemAI 页面）：`NULL`（兼容已有功能）

### 4.3 API

```go
// internal/ai/conversation_session.go

type ConversationSession struct {
    repo *repository.AIConversationRepository
}

func NewConversationSession(repo *repository.AIConversationRepository) *ConversationSession

// GetOrCreate returns the existing session for this strategy, or creates one.
// strategyKey examples: "strategy:abc-123", "draft:user-456:EURUSD:1h"
func (s *ConversationSession) GetOrCreate(ctx context.Context, userID uuid.UUID, strategyKey, title string) (*Session, error)

// AppendMessages atomically adds a user→assistant message pair to the session.
func (s *ConversationSession) AppendExchange(ctx context.Context, sessionID uuid.UUID, userID uuid.UUID, userMsg, assistantMsg string) error

// GetMessages returns all messages for a session, ordered by creation time.
func (s *ConversationSession) GetMessages(ctx context.Context, sessionID, userID uuid.UUID) ([]repository.AIMessage, error)

type Session struct {
    ID          uuid.UUID
    StrategyKey string
    Title       string
    Messages    []repository.AIMessage
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### 4.4 集成到现有 RPC Handler

#### ReviseCodeStream（CodeAssistService）

```go
// 在现有 ReviseCodeStream 末尾，LLM 流完成后：

// --- NEW: 自动持久化 ---
if req.Msg.SessionId != "" {
    sid, _ := uuid.Parse(req.Msg.SessionId)
    if err := s.session.AppendExchange(ctx, sid, uid, instruction, result); err != nil {
        s.log.Warn("session append failed", zap.Error(err))
        // non-fatal — streaming already succeeded
    }
}
```

#### GenerateStrategy（StrategyGenerationService）

```go
// 在 GenerateStrategy 末尾，code 生成完成后：

// --- NEW: 自动持久化 ---
if m.ConversationId != "" {
    cid, _ := uuid.Parse(m.ConversationId)
    s.session.AppendExchange(ctx, cid, uid, m.Message, code)
}
```

### 4.5 前端集成

```tsx
// StrategyWorkspacePage.tsx — onMount 解析 session，加载历史

useEffect(() => {
  const strategyKey = ws.strategy.id
    ? `strategy:${ws.strategy.id}`
    : `draft:${userId}:${ws.account.symbol}:${ws.account.timeframe}`;

  // 1. 解析 session（已存在则返回历史，不存在则创建）
  aiClient.resolveSession({ strategyKey }).then(res => {
    ws.ai.setSessionId(res.sessionId);       // 存入 workspace state
    ws.ai.setChatHistory(res.messages || []); // 恢复历史 → 传入 AIChatPanel
  });
}, [ws.strategy.id, ws.account.symbol, ws.account.timeframe]);

// AIChatPanel.tsx — 接收 sessionId + history prop
// 每次发送 AI 请求时，附带 sessionId
//   - generateStrategy({ conversationId: sessionId, ... })
//   - reviseCodeStream({ sessionId, ... })
// 服务端在 AI 响应完成后自动调用 AppendExchange 持久化
```

前端调用时序：
```
1. workspace onMount
   → ResolveSession(strategyKey)
   → { sessionId, messages, created }
   → 存入 ws state, 传入 AIChatPanel

2. 用户输入消息
   → detectMode(msg, code) → 模式标签
   → generateStrategy({ conversationId: sessionId, message, ... })
     或 reviseCodeStream({ sessionId, code, instruction, history })
   → 服务端处理后自动持久化

3. 换设备 / 刷新
   → 重复步骤 1
   → 同一 strategyKey → 同一 sessionId → 聊天历史完整恢复
```

### 4.6 Proto 变更（2 项最小改动）

#### 4.6.1 新 RPC：`ResolveSession`

挂在已有 `AIService` 上，不建新 service。前端 workspace 打开时调用，解析 `strategy_key` → `session_id` + 历史消息。

```proto
// ai_chat_requests.proto — 新增

message ResolveSessionRequest {
  string strategy_key = 1;  // "strategy:<uuid>" or "draft:<user>:<symbol>:<tf>"
  string title = 2;         // title for newly created sessions (optional)
}

message ResolveSessionResponse {
  string session_id = 1;                  // UUID of the resolved/created session
  repeated ConversationMessage messages = 2;  // message history (empty for new sessions)
  bool created = 3;                       // true if a new session was created
}

// ai.proto — AIService 新增 1 个 rpc
service AIService {
  // ... existing RPCs ...
  rpc ResolveSession(ResolveSessionRequest) returns (ResolveSessionResponse);
}
```

#### 4.6.2 ReviseCodeRequest 加 `session_id`

```proto
// code_assist.proto
message ReviseCodeRequest {
  string code = 1;
  string instruction = 2;
  repeated CodeChatMessage history = 3;
  string locale = 4;
  string session_id = 5;  // NEW: workspace session UUID (from ResolveSession)
}
```

#### 4.6.3 不需要改的 proto

- **`GenerateStrategyRequest`**：已有 `conversation_id` (field 1)，前端传 session UUID 即可，**0 改动**
- **`GenerateStrategyChunk`**：不需要加 session_id — 前端在调用前已通过 ResolveSession 拿到
- **`ReviseCodeStreamChunk`**：不需要加 session_id — 同上
- **`ai_conversation.proto`**：`ConversationMessage` 已存在，`ResolveSessionResponse` 直接复用

**Proto 净增**：2 个 message + 1 个 rpc + 1 个 field

## 5. 文件变更

### 5.1 新建文件

| 文件 | 行数 | 职责 |
|------|------|------|
| `backend/internal/ai/prompt_context.go` | ~100行 | 模式判定 + 4 种 prompt 模板 + 上下文构建 |
| `backend/internal/ai/conversation_session.go` | ~70行 | 策略级会话管理，薄封装 AIConversationRepository |

### 5.2 修改文件

| 文件 | 变更 | 行数 |
|------|------|------|
| `proto/ant/v1/ai_chat_requests.proto` | 新增 `ResolveSessionRequest` + `ResolveSessionResponse` | +14行 |
| `proto/ant/v1/ai.proto` | `AIService` 新增 `ResolveSession` rpc | +1行 |
| `proto/ant/v1/code_assist.proto` | `ReviseCodeRequest` 加 `session_id = 5` | +1行 |
| `backend/internal/connect/ai/code_assist_handler.go` | 接入 `PromptContext` + `ConversationSession`，`ReviseCodeStream` 增强 | +35行 |
| `backend/internal/connect/ai/strategy_gen_handler.go` | `GenerateStrategy` 末尾调用 `AppendExchange` 持久化 | +10行 |
| `backend/internal/connect/ai/ai_handler.go` | `AIServer` 加 `session` 字段 + `ResolveSession` handler | +20行 |
| `backend/internal/connect/ai/ai_conversation.go` | 无改动（ResolveSession 是新增 handler，不修改现有） | 0 |
| `backend/internal/repository/ai_conversation_repository.go` | 新增 `GetByStrategyKey`、`CreateWithStrategyKey` | +25行 |
| `backend/cmd/server/handlers.go` | 注入 `ConversationSession` 到 3 个 server | +5行 |
| `frontend/src/pages/strategy/StrategyWorkspacePage.tsx` | onMount 调 `resolveSession`，加载历史传入 AIChatPanel | +15行 |
| `frontend/src/components/strategy/AIChatPanel.tsx` | 接收 sessionId/history prop + 模式标签 | +15行 |
| `frontend/src/client/strategyGen.ts` | 无改动（已有 `conversationId` 字段） | 0 |
| `frontend/src/client/codeAssist.ts` | `ReviseCodeInput` 加 `sessionId` 字段 | +2行 |
| DB migration | `ai_conversations` 加 `strategy_key` 列 + partial unique index | 2行SQL |

### 5.3 不改的文件

- `ai_chat.go` — 通用对话不变（保留给独立 AI 页面）
- `backtest_feedback.go` — 已实现，直接调用
- `feedback_router.go` — 已实现，关键词被 prompt_context.go 复用
- `clarification.go` — 已实现，保持
- 已有 conversation CRUD handler（`ai_conversation.go`）— 保持，SystemAI 页面独立使用

### 5.4 净增：~280 行（含 proto 16 行 + SQL 2 行）

## 6. 实现步骤

### Step 1: `prompt_context.go`（新建）

```go
// Package ai — prompt_context.go
// PromptContext builds mode-specific system prompts for AI code interactions.
// Replaces the originally planned conversation_store.go with a stateless,
// workspace-aware context builder.

package ai

import "strings"

// InteractionMode classifies user intent for AI code assistance.
type InteractionMode int

const (
    ModeGenerate InteractionMode = iota // no code exists, create from scratch
    ModeRevise                          // modify existing code
    ModeRepair                          // fix validation/runtime errors
    ModeDiscuss                         // ask questions about the code
)

// PromptContext holds all context needed to build mode-specific prompts.
type PromptContext struct {
    Mode             InteractionMode
    SystemPrompt     string
    UserMessage      string
    Code             string
    Symbol           string
    Timeframe        string
    BacktestSummary  string
    ValidationErrors []string
}

// BuildContextInput is the parameter object for BuildContext.
// Struct encapsulation keeps the function signature within the 5‑parameter limit.
type BuildContextInput struct {
    Code             string
    Message          string
    Symbol           string
    Timeframe        string
    BacktestSummary  string
    ValidationErrors []string
}

// BuildContext analyzes code + message and returns the appropriate PromptContext.
// Pure function — no side effects, no state.
func BuildContext(input BuildContextInput) *PromptContext {
    mode := classifyIntent(input.Code, input.Message)

    pc := &PromptContext{
        Mode:             mode,
        Code:             input.Code,
        Symbol:           input.Symbol,
        Timeframe:        input.Timeframe,
        BacktestSummary:  input.BacktestSummary,
        ValidationErrors: input.ValidationErrors,
    }

    switch mode {
    case ModeGenerate:
        pc.SystemPrompt = generatePrompt()
        pc.UserMessage = message
    case ModeRevise:
        pc.SystemPrompt = revisePrompt()
        pc.UserMessage = buildReviseUserMessage(code, message, backtestSummary)
    case ModeRepair:
        pc.SystemPrompt = repairPrompt(validationErrors)
        pc.UserMessage = buildRepairUserMessage(code, message)
    case ModeDiscuss:
        pc.SystemPrompt = discussPrompt(code)
        pc.UserMessage = message
    }

    return pc
}

// classifyIntent determines the interaction mode from code + message.
func classifyIntent(code, message string) InteractionMode {
    if strings.TrimSpace(code) == "" {
        return ModeGenerate
    }
    lower := strings.ToLower(message)

    // Repair: error-related keywords (highest priority)
    repairKw := []string{
        "报错", "error", "错误", "traceback", "缺少参数", "missing",
        "验证失败", "syntax error", "syntaxerror", "undefined", "未定义",
        "缺少 required", "参数不足", "attributeerror", "typeerror",
    }
    for _, kw := range repairKw {
        if strings.Contains(lower, kw) {
            return ModeRepair
        }
    }

    // Discuss: question/analysis keywords
    discussKw := []string{
        "为什么", "什么意思", "怎么样", "对吗", "分析",
        "解释", "what", "why", "how", "explain", "对不对",
    }
    for _, kw := range discussKw {
        if strings.Contains(lower, kw) {
            return ModeDiscuss
        }
    }

    // Default: revise
    return ModeRevise
}
```

### Step 2: `code_assist_handler.go` — 集成 PromptContext + ConversationSession（修改）

```go
// struct 新增 session 字段
type CodeAssistServer struct {
    systemSvc *systemai.Service
    session   *ai.ConversationSession  // NEW
    log       *zap.Logger
}

func NewCodeAssistServer(systemSvc *systemai.Service, session *ai.ConversationSession, log *zap.Logger) *CodeAssistServer {
    return &CodeAssistServer{systemSvc: systemSvc, session: session, log: log}
}

// ReviseCodeStream 修改点：
// 1. 调用 BuildContext 获取模式和专用 prompt（替代 buildCodeAssistPrompt）
// 2. 修复模式走 post-processing（extractCodeFromRepair 兜底提取）
// 3. 流完成后自动持久化消息到 session

func (s *CodeAssistServer) ReviseCodeStream(...) error {
    // ... existing validation (code length, instruction length, auth) ...

    // REPLACE: old buildCodeAssistPrompt(code, instruction)
    // WITH:
    pc := ai.BuildContext(ai.BuildContextInput{Code: code, Message: instruction})
    messages := systemai.BuildChatMessages(pc.SystemPrompt, pc.UserMessage, protoHistoryToChat(req.Msg.History))

    var fullText strings.Builder
    err = s.systemSvc.ChatCompletionStream(ctx, uid, messages, codeAssistModel,
        func(chunk systemai.ChatStreamChunk) error {
            fullText.WriteString(chunk.Content)
            return stream.Send(&antv1.ReviseCodeStreamChunk{Delta: chunk.Content, Done: chunk.Done})
        })
    // ... existing error handling (unchanged) ...

    // Repair mode post-processing — NEW
    result := fullText.String()
    if pc.Mode == ai.ModeRepair {
        if code := extractCodeFromRepair(result); code != "" {
            result = code
        }
    }

    // Auto-persist to session — NEW
    if req.Msg.SessionId != "" {
        sid, _ := uuid.Parse(req.Msg.SessionId)
        if err := s.session.AppendExchange(ctx, sid, uid, instruction, result); err != nil {
            s.log.Warn("session append failed", zap.Error(err))
            // non-fatal — streaming already succeeded
        }
    }

    return stream.Send(&antv1.ReviseCodeStreamChunk{Delta: "", Python: result, Done: true})
}

// extractCodeFromRepair attempts to salvage Python code from an LLM response
// that may contain explanatory text (3-tier extraction).
func extractCodeFromRepair(raw string) string {
    // Tier 1: extract from ```python ... ``` fence
    if code := extractFencedCode(raw, "python"); code != "" {
        return code
    }
    // Tier 2: heuristic — find lines starting with import/def/class/#
    if code := extractByHeuristic(raw); code != "" {
        return code
    }
    // Tier 3: unable to extract — return empty, frontend shows raw text
    return ""
}
```

### Step 3: `strategy_gen_handler.go` — GenerateStrategy 末尾持久化（修改）

```go
// GenerateStrategy 末尾，在最终 chunk 发送前：
// 复用已有的 conversation_id 字段（field 1），前端传入 session UUID

runID, btErr := s.finalizeWithBacktest(ctx, userID, code, m.Symbol, m.Timeframe)

// --- NEW: auto-persist exchange ---
if m.ConversationId != "" {
    cid, _ := uuid.Parse(m.ConversationId)
    if err := s.convRepo.AddMessage(ctx, userID, cid, "user", m.Message); err != nil {
        s.log.Warn("persist user msg failed", zap.Error(err))
    }
    if err := s.convRepo.AddMessage(ctx, userID, cid, "assistant", code); err != nil {
        s.log.Warn("persist assistant msg failed", zap.Error(err))
    }
    s.convRepo.Touch(ctx, cid, userID)
}
// --- END NEW ---

return stream.Send(&antv1.GenerateStrategyChunk{Phase: "done", Code: code, BacktestRunId: runID, Error: btErr})
```

### Step 4: 前端 — session 解析 + mode tag（修改）

#### StrategyWorkspacePage.tsx

```tsx
// onMount: 调 ResolveSession RPC 解析 strategy_key → sessionId + 历史消息

import { aiClient } from '@/client/connect';

useEffect(() => {
  const strategyKey = ws.strategy.id
    ? `strategy:${ws.strategy.id}`
    : `draft:${userId}:${ws.account.symbol}:${ws.account.timeframe}`;

  aiClient.resolveSession({ strategyKey }).then(res => {
    ws.ai.setSessionId(res.sessionId);       // 存入 workspace state
    ws.ai.setChatHistory(res.messages || []); // 服务端历史 → AIChatPanel props
  });
}, [ws.strategy.id, ws.account.symbol, ws.account.timeframe]);
```

#### AIChatPanel.tsx

```tsx
// 修改点:
// 1. 新增 sessionId + history props（从服务端加载，替代 localStorage）
// 2. enhance detectMode() → 四种模式（与后端 classifyIntent 关键词表一致）
// 3. 模式标签显示
// 4. handleSend 传递 sessionId 到 RPC（GenerateStrategy 复用 conversationId 字段）

interface Props {
  // ... existing code, onApply, symbol, timeframe, initialPrompt, autoApply ...
  sessionId?: string;            // NEW: from ResolveSession
  history?: CodeChatMessage[];   // NEW: server-loaded history
}

// detectMode 增强版（关键词表与后端 classifyIntent 严格一致）
function detectMode(msg: string, hasCode: boolean): InteractionMode {
  if (!hasCode) return 'generate';
  const lower = msg.toLowerCase();
  // Repair (highest priority)
  const repairKw = ['报错','error','错误','traceback','缺少参数','missing',
    '验证失败','syntax error','undefined','未定义','缺少 required','参数不足'];
  if (repairKw.some(k => lower.includes(k))) return 'repair';
  // Discuss
  const discussKw = ['为什么','什么意思','怎么样','对吗','分析','解释',
    'what','why','how','explain','对不对'];
  if (discussKw.some(k => lower.includes(k))) return 'discuss';
  return 'revise';
}

const MODE_TAGS: Record<string, { color: string; label: string }> = {
  generate: { color: 'blue',   label: '⚡ 生成' },
  revise:   { color: 'green',  label: '✏️ 修改' },
  repair:   { color: 'orange', label: '🔧 修复' },
  discuss:  { color: 'purple', label: '💬 分析' },
};

// handleGenerate — sessionId → conversationId (GenerateStrategyRequest field 1)
const handleGenerate = (msg: string, round: number) => {
  generateStrategyStream(
    { message: msg, symbol, timeframe, clarificationRound: round,
      conversationId: sessionId || '' },  // session UUID as conversation_id
    // ...
  );
};

// handleRevise — sessionId → ReviseCodeRequest.session_id (field 5)
const handleRevise = (msg: string) => {
  codeAssistApi.reviseStream(
    { code, instruction: msg, history, sessionId },
    // ...
  );
};
```

### Step 5: `conversation_session.go`（新建）

```go
// internal/ai/conversation_session.go
// Lightweight strategy-scoped session management.
// Thin wrapper over AIConversationRepository — append + read only.

package ai

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "anttrader/internal/repository"
)

type ConversationSession struct {
    repo *repository.AIConversationRepository
}

func NewConversationSession(repo *repository.AIConversationRepository) *ConversationSession {
    return &ConversationSession{repo: repo}
}

type Session struct {
    ID          uuid.UUID
    StrategyKey string
    Title       string
    Messages    []repository.AIMessage
}

// GetOrCreate finds an existing session for strategyKey, or creates one.
func (s *ConversationSession) GetOrCreate(ctx context.Context, userID uuid.UUID, strategyKey, title string) (*Session, error) {
    conv, err := s.repo.GetByStrategyKey(ctx, userID, strategyKey)
    if err == nil {
        msgs, _ := s.repo.GetMessages(ctx, userID, conv.ID)
        return &Session{ID: conv.ID, StrategyKey: strategyKey, Title: conv.Title, Messages: msgs}, nil
    }
    // Create new
    if title == "" {
        title = "AI 策略协作"
    }
    conv, err = s.repo.CreateWithStrategyKey(ctx, userID, title, strategyKey)
    if err != nil {
        return nil, fmt.Errorf("create session: %w", err)
    }
    return &Session{ID: conv.ID, StrategyKey: strategyKey, Title: title}, nil
}

// AppendExchange persists a user→assistant message pair.
// Non-fatal on failure — callers log warning and continue.
func (s *ConversationSession) AppendExchange(ctx context.Context, sessionID, userID uuid.UUID, userMsg, assistantMsg string) error {
    if _, err := s.repo.AddMessage(ctx, userID, sessionID, "user", userMsg); err != nil {
        return err
    }
    if _, err := s.repo.AddMessage(ctx, userID, sessionID, "assistant", assistantMsg); err != nil {
        return err
    }
    return s.repo.Touch(ctx, sessionID, userID)
}
```

### Step 6: repository 层新增 `GetByStrategyKey` + `CreateWithStrategyKey`

```go
// ai_conversation_repository.go — 新增方法

// GetByStrategyKey finds a conversation by (user_id, strategy_key).
// Returns sql.ErrNoRows if not found.
func (r *AIConversationRepository) GetByStrategyKey(ctx context.Context, userID uuid.UUID, strategyKey string) (*AIConversation, error) {
    var conv AIConversation
    err := r.db.QueryRow(ctx,
        `SELECT id, user_id, title, created_at, updated_at FROM ai_conversations
         WHERE user_id = $1 AND strategy_key = $2`,
        userID, strategyKey,
    ).Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.CreatedAt, &conv.UpdatedAt)
    if err != nil {
        return nil, err
    }
    return &conv, nil
}

// CreateWithStrategyKey creates a conversation with a strategy_key.
func (r *AIConversationRepository) CreateWithStrategyKey(ctx context.Context, userID uuid.UUID, title, strategyKey string) (*AIConversation, error) {
    conv := &AIConversation{
        ID: uuid.New(), UserID: userID, Title: title,
        CreatedAt: time.Now(), UpdatedAt: time.Now(),
    }
    _, err := r.db.Exec(ctx,
        `INSERT INTO ai_conversations (id, user_id, title, strategy_key, created_at, updated_at)
         VALUES ($1, $2, $3, $4, NOW(), NOW())`,
        conv.ID, conv.UserID, conv.Title, strategyKey,
    )
    if err != nil {
        return nil, err
    }
    return conv, nil
}
```

### Step 7: `ResolveSession` handler（AIServer 新增）

```go
// internal/connect/ai/ai_handler.go — AIServer 新增字段 + handler

type AIServer struct {
    systemSvc     *systemai.Service
    conversations *repository.AIConversationRepository
    session       *ai.ConversationSession  // NEW
    agentDefRepo  *repository.AIAgentDefinitionRepository
    log           *zap.Logger
}

func NewAIServer(systemSvc *systemai.Service, conversations *repository.AIConversationRepository, session *ai.ConversationSession, log *zap.Logger) *AIServer {
    return &AIServer{systemSvc: systemSvc, conversations: conversations, session: session, log: log}
}

// ResolveSession resolves a strategy_key to a session, creating one if needed.
func (s *AIServer) ResolveSession(ctx context.Context, req *connect.Request[antv1.ResolveSessionRequest]) (*connect.Response[antv1.ResolveSessionResponse], error) {
    uid, err := userIDFromCtx(ctx)
    if err != nil {
        return nil, err
    }
    if req.Msg.StrategyKey == "" {
        return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("strategy_key is required"))
    }
    sess, err := s.session.GetOrCreate(ctx, uid, req.Msg.StrategyKey, req.Msg.Title)
    if err != nil {
        s.log.Error("ResolveSession", zap.Error(err))
        return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
    }
    var msgs []*antv1.ConversationMessage
    for _, m := range sess.Messages {
        msgs = append(msgs, &antv1.ConversationMessage{
            Id: m.ID.String(), Role: m.Role, Content: m.Content,
            CreatedAt: timestamppb.New(m.CreatedAt),
        })
    }
    return connect.NewResponse(&antv1.ResolveSessionResponse{
        SessionId: sess.ID.String(),
        Messages:  msgs,
        Created:   len(sess.Messages) == 0,
    }), nil
}
```

### Step 8: `handlers.go` 注入 ConversationSession

```go
// cmd/server/handlers.go — 修改

convRepo := repository.NewAIConversationRepository(pool)
session := ai.NewConversationSession(convRepo)  // NEW: create session wrapper

// ... existing ...

// CodeAssistServer — 新增 session 参数
codeAssistServer := ai.NewCodeAssistServer(aiSvc, session, log)

// AIServer — 新增 session 参数
aiServer := ai.NewAIServer(aiSvc, convRepo, session, log)

// StrategyGenServer — 已有 convRepo, 不需要新增 session 参数
// GenerateStrategy 直接通过 convRepo 持久化（已有 convRepo 字段）
```

## 7. 数据流总览

```
┌──────────────────────────────────────────────────────────────────┐
│ StrategyWorkspacePage.onMount()                                  │
│                                                                  │
│  1. ResolveSession(strategyKey) ──→ { sessionId, messages[] }    │
│     strategyKey = strategy:<id> | draft:<user>:<symbol>:<tf>     │
│     ├── sessionId → ws.ai.sessionId (workspace state)             │
│     └── messages → ws.ai.chatHistory → AIChatPanel props          │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│ AIChatPanel 用户交互                                              │
│                                                                  │
│  mode tag: [⚡ 生成] / [✏️ 修改] / [🔧 修复] / [💬 分析]         │
│  history: 服务端加载 (跨设备同步)                                  │
│                                                                  │
│  handleSend() → detectMode(msg, code) → RPC                      │
│    ├── generate → GenerateStrategy(conversationId=sessionId)     │
│    └── others   → ReviseCodeStream(sessionId, code, instruction)  │
└──────────────┬───────────────────────────────────────────────────┘
               │ ReviseCodeStream(已有, 增强) / GenerateStrategy(已有)
               ▼
┌──────────────────────────────────────────────────────────────────┐
│ Backend Handler                                                   │
│                                                                  │
│  1. ai.BuildContext(input) — 模式判定 + prompt 构建              │
│     ├── ModeRepair  → repairPrompt (double constraints)          │
│     ├── ModeRevise  → revisePrompt + backtestSummary             │
│     ├── ModeDiscuss → discussPrompt + code reference             │
│     └── ModeGenerate → (goes to StrategyGen pipeline)            │
│                                                                  │
│  2. LLM stream → client chunks (proto unchanged)                 │
│                                                                  │
│  3. Post-process (Repair → extractCodeFromRepair)                │
│                                                                  │
│  4. session.AppendExchange(sessionId, userMsg, assistantMsg)     │
│     └── ai_messages INSERT (user + assistant pair)               │
│     └── ai_conversations UPDATE updated_at                       │
└──────────────────────────────────────────────────────────────────┘
```

## 8. 验收标准

1. **修复模式 E2E**：生成代码 → 验证报错"missing param" → 复制错误到 AI 输入 → 发送 → LLM 返回**纯代码**（不含解释文字） → 代码写入编辑器
2. **修复兜底**：LLM 返回"Here is the fixed code: ```python ...```" → `extractCodeFromRepair` 提取出代码 → 写入编辑器
3. **讨论模式不影响代码**：用户问"这个止损逻辑对吗？" → AI 返回分析文字 → 不写入编辑器
4. **回测上下文注入**：用户输入附带回测指标 → AI prompt 包含指标数据
5. **跨设备同步**：桌面端 AI 聊天 → 换手机打开同一策略 → 聊天历史完整恢复
6. **刷新不丢**：刷新 workspace 页面 → 聊天历史从服务端加载恢复
7. **文件行数**：`prompt_context.go` ≤ 120 行，`conversation_session.go` ≤ 80 行

## 9. 设计自审

- **placeholder 扫描**：无 TBD/TODO
- **一致性**：
  - 模式判定关键词：前端 `detectMode` 与后端 `classifyIntent` 共用同一套关键词表
  - 策略绑定：`strategy:<id>` / `draft:<user>:<symbol>:<tf>` 格式统一
  - session 解析：前端通过 `ResolveSession` RPC 获取 sessionId，后作为 `conversation_id`/`session_id` 传入 AI RPC
- **Proto 最小化**：仅新增 1 个 RPC + 2 个 message + 1 个 field，复用已有 `ConversationMessage` 类型
- **范围**：Phase 1 的 prompt 上下文 + 会话持久化，不涉及 Phase 2/3/4
- **歧义检查**：
  - 修复 + 讨论同时命中 → 修复优先（`classifyIntent` 中修复关键词优先匹配）
  - 未保存策略 → `draft:` 前缀 key，保存后迁移为 `strategy:` 前缀（migration 在策略保存时触发，不在此设计范围）
  - AppendExchange 失败 → 非致命，warn 日志，不影响 AI 流式响应
  - ResolveSession 并发 → PG partial unique index 保证同一 strategy_key 最多一条记录
