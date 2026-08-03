# 01 — 前端组件树与数据流

## 组件层级

```
RightPanel (workspace 右侧面板, width=380)
  ├─ Header: "🤖 AI Assistant" + Memory 按钮
  └─ StrategyChat (顶层容器)
       ├─ Toolbar: [symbol tag] [model selector] [🏠 History] [📄 Strategies] [⚙️ Settings]
       ├─ AgentGenChat (核心对话引擎)  ← key={activeConvId}
       │    ├─ ChatHistory (消息列表, 可滚动)
       │    │    ├─ User bubble (右对齐, pre-wrap)
       │    │    ├─ AI card:
       │    │    │    ├─ Phase badge (LoadingOutlined + phase label)
       │    │    │    ├─ Done/Failed status
       │    │    │    ├─ No-market-data Alert
       │    │    │    ├─ PlanCard (CollapsibleBlock, 📋)
       │    │    │    ├─ CompileError / BacktestError Alert
       │    │    │    ├─ Reasoning block (CollapsibleBlock, 💭, collapsed)
       │    │    │    ├─ StreamContent (纯文本, 不含代码块)
       │    │    │    ├─ GeneratedCode card (唯一 Apply 卡片, 含 Copy 按钮)
       │    │    │    ├─ Metrics (Row/Col/Statistic)
       │    │    │    ├─ Coverage score (Progress bar)
       │    │    │    ├─ Profile (CollapsibleBlock, 📊)
       │    │    │    └─ Analysis (CollapsibleBlock, 📝)
       │    │    └─ endRef (auto-scroll anchor)
       │    └─ ChatInput (TextArea, 始终可用)
       │
       ├─ History Drawer (StrategyChatHistory)
       │    ├─ [+ New Conversation] 按钮
       │    └─ 对话列表 (点击加载, 重命名, 删除)
       │
       ├─ Strategies Drawer (StrategyList)
       │    └─ 策略模板列表 (Load/Save/Rename/Delete/SendToAI)
       │
       └─ AI Settings Modal (lazy loaded)
```

## 数据流

### AgentGenChat 状态管理

```
AgentGenChat
  state:
    turns: ChatTurn[]          ← 所有对话轮次
    userInput: string          ← 输入框内容
    generating: boolean        ← 是否正在生成
    hasCode: boolean           ← 是否已有代码

  refs:
    conversationIdRef          ← crypto.randomUUID() 或 props.conversationId
    currentTurnIdRef           ← 当前 AI 回复的 turn ID
    streamTextRef              ← 累积 streamText (delta 拼接)
    reasoningRef               ← 累积 reasoning (reasoning delta 拼接)
    abortRef                   ← AbortController
    lastMsgRef                 ← 最后一条用户消息
    confirmedPlanRef           ← 已确认的 plan
```

### 流式回调链

```
agentGen.ts: agentGenerateStrategyStream(input, callbacks)
  │
  ├─ onPhase(phase)           → ChatTurn.phase (planning/generating/compiling/...)
  ├─ onDelta(delta)           → streamTextRef += delta → ChatTurn.streamText
  ├─ onReasoning(reasoning)   → reasoningRef += reasoning → ChatTurn.reasoning
  ├─ onPythonSource(code)     → ChatTurn.generatedCode (前端 Apply 卡片)
  ├─ onCompileError(err)      → ChatTurn.compileError
  ├─ onBacktestError(err)     → ChatTurn.backtestError
  ├─ onCoverageScore(score)   → ChatTurn.coverageScore
  ├─ onResult(result)         → ChatTurn.metrics + phase='done'
  ├─ onProfile(p)             → ChatTurn.profile
  ├─ onAnalysis(a)            → ChatTurn.analysis
  ├─ onAttempts(n)            → ChatTurn.attempts
  ├─ onError(e)               → ChatTurn.error
  └─ onPlan(plan)             → ChatTurn.plan
```

### ChatTurn 类型

```typescript
interface ChatTurn {
  id: string;
  role: 'user' | 'ai';
  message: string;           // user: 原始消息; ai: 空(内容在子字段)
  timestamp?: string;
  phase?: Phase;             // idle|planning|generating|compiling|backtesting|analyzing|done
  streamText?: string;       // 自由文本通道 (不含代码)
  reasoning?: string;        // 思考过程 (💭 折叠)
  generatedCode?: string;    // 唯一代码产物 (I1)
  compileError?: string;
  backtestError?: string;
  error?: string;
  coverageScore?: number;
  attempts?: number;
  metrics?: { label: string; value: string; positive?: boolean }[];
  plan?: StrategyPlan;
  profile?: StrategyProfile;
  analysis?: BacktestAnalysis;
  hasCode?: boolean;
}
```

### 对话 CRUD 数据流

```
StrategyChat.fetchConversations()
  → aiApi.listConversations()                  // ConnectRPC → backend ListConversations
  → conversations: Conversation[]

useConversationHandlers:
  handleNewConv()
    → aiApi.createConversation(title)
    → setActiveConvId(conv.id)                 // triggers AgentGenChat remount (key prop)
    → fetchConversations()                     // refresh list

  handleLoadConv(id)
    → aiApi.getConversation(id)
    → setActiveConvId(id)                      // triggers AgentGenChat remount
    → setMessages(messages)                    // restores old-style messages

  handleDeleteConv(convId)
    → aiApi.deleteConversation(convId)
    → setActiveConvId('') + setMessages([])
    → fetchConversations()
```

## 关键约定

1. **`key={activeConvId}`** — AgentGenChat 通过 key prop 在切换对话时完全 remount，清空所有内部状态
2. **conversationId 全链路传递** — StrategyChat → AgentGenChat → agentGen.ts → AgentGenerateStrategyRequest
3. **Apply 按钮禁用条件** — `turn.phase !== 'done' || !!(compileError || backtestError || error)`
4. **StreamContent 不含代码块** — 代码只经 write_strategy 工具进入系统，自由文本永不承载成品代码
5. **Copy 按钮** — 复制反馈用 `copiedId` state + 2s setTimeout
