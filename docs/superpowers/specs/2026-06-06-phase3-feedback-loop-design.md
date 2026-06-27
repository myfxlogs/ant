# Phase 3: 对话式反馈闭环 — 设计文档

> **⚠️ 注意：** AI Prompt 模板当前面向 Python 策略代码生成。迁移至 Go SDK 后需更新系统 Prompt（见 ADR-0021）。

> **关联 spec**：`docs/spec/26-ai-strategy-generation.md` §5
> **前置依赖**：Phase 1 (GenerateStrategy, 已完成) + Phase 2 (Smart Tuning, 已完成)
> **创建日期**：2026-06-06

## 1. 使命

Phase 3 将 AI 从「代码生成器」升级为「策略顾问」——用户看到回测结果后，用自然语言给出反馈（"太激进了，加个止损"），AI 理解反馈、分析回测指标、生成优化后的代码并自动验证。形成 generate → backtest → feedback → iterate 的对话闭环。

目标体验：
```
User: "做一个 EURUSD 均线策略"
AI:   [生成代码 + 自动回测] → 聊天中展示回测指标卡片
      "Sharpe 0.82, 最大回撤 18% — 建议加入止损保护。需要调整吗？"

User: "加入止损，保守一点"
AI:   [分析回测 + 理解反馈] → 新代码 + 新回测
      "已加入 1% 止损并降低仓位。Sharpe 0.78, 回撤降至 9%。"
```

## 2. 架构：单 RPC 双模式

### 2.1 核心决策

**扩展现有 `GenerateStrategy` RPC，不新增 RPC。**

| 方案 | 前端复杂度 | Proto 变更 | 判定 |
|------|-----------|-----------|------|
| A. 新 `RefineStrategy` RPC | 需判断调用哪个 | 新 message + handler | ❌ 过度设计 |
| **B. 扩展 `GenerateStrategy`** | **始终调同一个** | **3 optional fields** | ✅ **最优** |
| C. 走 `codeAssist.revise` | 已有 revise 路径 | 无 | ❌ 无策略上下文 |

理由：
- 用户的一段对话是连续的——"做均线策略"和"加止损"是同一个 conversation
- 前端不需要知道当前是 fresh 还是 feedback 模式——后端根据 conversation 状态自动切换
- `codeAssist.revise` 是通用代码修改，感知不到 symbol/timeframe/backtestMetrics

### 2.2 双模式切换

```
GenerateStrategy handler:

  ┌─ req.backtestMetricsJson == ""?
  │    → FRESH MODE (现有流程)
  │       澄清 → 模板匹配 → LLM stream → 合规 → 自动回测 → done
  │
  └─ req.backtestMetricsJson != ""
       → FEEDBACK MODE (新增)
         1. LoadContext: 从 req 读取 previousCode + backtestMetrics
         2. BuildFeedbackPrompt: 注入指标 + 代码 + 用户反馈
         3. LLM Stream: analyzing → generating
         4. ParseSections: 提取 <section type="analysis|advice|code">
         5. Compliance check on extracted code
         6. Auto-backtest new code
         7. Emit "done" with new code + backtestRunId
```

### 2.3 反馈路由决策

**反馈始终走 LLM，不走 keyword → SmartTuning 的直接路径。**

`feedback_router.go` 的关键词→参数映射（"太激进"→`max_drawdown=0.10`）降级为 prompt 提示。

理由：用户反馈有 nuance——"感觉风险有点大但也不想太保守，止损调紧一点就好"。LLM 能同时理解"风险有点大"+"不想太保守"+"止损调紧"，而关键词只能匹配第一个。

`backtest_feedback.go` 的指标格式化完全复用。

## 3. Proto 变更

### 3.1 `GenerateStrategyRequest` 新增字段

```protobuf
message GenerateStrategyRequest {
  // ... existing fields 1-8 (unchanged) ...

  // Phase 3: feedback context (all optional — absent = fresh mode)
  string previous_code = 9;           // 上次生成的代码
  string backtest_metrics_json = 10;  // 回测指标 JSON (FeedbackMetrics 序列化)
  string feedback_message = 11;       // 用户反馈原文
}
```

三个字段全部 optional。现有调用方无需修改 —— fresh mode 行为完全不变。

### 3.2 `GenerateStrategyChunk` 新增字段

```protobuf
message GenerateStrategyChunk {
  // ... existing fields (unchanged) ...

  // Phase 3: structured feedback output
  string analysis = 10;   // AI 分析段落 (from <section type="analysis">)
  string advice = 11;     // AI 建议段落 (from <section type="advice">)
}
```

`analysis` 和 `advice` 在前端聊天中分开展示（分析→灰色引用块，建议→蓝色操作块）。

代码输出复用现有 `code` 字段（field 4），不新增冗余字段。

## 4. Prompt 设计

### 4.1 Feedback System Prompt

```go
func (b *StrategyPromptBuilder) BuildFeedbackPrompt(p *FeedbackPromptParams) (system, user string) {
    system = fmt.Sprintf(`你是量化策略迭代助手。用户已查看回测结果并给出反馈，你需要：

1. 分析回测结果的问题（1-2 句，中文）
2. 给出具体优化建议（1-2 条）
3. 生成优化后的完整 Python 策略代码

## 输出格式（严格遵守）
用 <section> 标签分隔三个部分：

<section type="analysis">
简要分析回测结果，指出问题。例如："Sharpe 0.45 偏低，最大回撤 28%% 超过风控线..."
</section>

<section type="advice">
具体优化建议。例如："建议将 fast_period 从 5 调整到 10 以减少过度交易"
</section>

<section type="code">
```python
# 完整的优化后策略代码
```
</section>

## 代码规范
%s

## 当前策略代码
%s

## 回测结果
%s

## 优化方向提示
%s`, CONTRACT_TEXT, p.PreviousCode, formatMetrics(p.Metrics), p.FeedbackHints)

    user = fmt.Sprintf("【用户反馈】%s\n\n请分析回测结果，给出建议，并生成优化后的代码。", p.FeedbackMessage)
    return
}
```

`FeedbackHints` 来自 `feedback_router.RouteFeedback()` 的输出，作为 prompt 中的软提示而非硬路由。

### 4.2 指标格式化

复用 `backtest_feedback.go` 的 `FormatPromptContext()`：

```
回测指标: Sharpe 0.82, 最大回撤 18.5%, 胜率 42%, 盈亏比 1.65, 总收益 12.3%, 交易次数 23
总结: 策略方向正确但回撤偏大，缺乏止损保护
```

### 4.3 Temperature 策略

| 模式 | Temperature | 原因 |
|------|-------------|------|
| Fresh (首次生成) | 0.3 | 代码需要确定性 |
| Feedback (迭代) | 0.5 | 需要一定创造性理解反馈，但仍然要保持代码质量 |

## 5. 后端实现

### 5.1 文件变更清单

| 文件 | 改动 | 行数 |
|------|------|------|
| `proto/strategy_generation.proto` | GenerateStrategyRequest +3 fields, GenerateStrategyChunk +3 fields | ~15 |
| `gen/proto/ant/v1/strategy_generation.pb.go` | 重新生成 | 自动 |
| `internal/ai/strategy_prompt.go` | `BuildFeedbackPrompt()` 方法 | ~40 |
| `internal/connect/ai/strategy_gen_handler.go` | feedback 分支: LoadContext → BuildFeedbackPrompt → ParseSections → emit analysis/advice | ~50 |
| `internal/connect/ai/strategy_gen_helpers.go` | `parseSections()` 辅助函数 | ~25 |
| `frontend/src/gen/ant/v1/strategy_generation_pb.ts` | 重新生成 | 自动 |

### 5.2 Handler: feedback 分支

```go
func (s *StrategyGenServer) GenerateStrategy(
    ctx context.Context,
    req *connect.Request[antv1.GenerateStrategyRequest],
    stream *connect.ServerStream[antv1.GenerateStrategyChunk],
) error {
    userID, err := userIDFromCtx(ctx)
    if err != nil { return err }
    m := req.Msg

    // ── FEEDBACK MODE ──
    if m.PreviousCode != "" && m.BacktestMetricsJson != "" {
        return s.handleFeedback(ctx, userID, m, stream)
    }

    // ── FRESH MODE (existing logic, unchanged) ──
    if result := s.runClarification(m); result != nil {
        return stream.Send(&antv1.GenerateStrategyChunk{
            Phase: "clarifying", Questions: result.Questions,
        })
    }
    // ... rest of existing flow ...
}

func (s *StrategyGenServer) handleFeedback(
    ctx context.Context, userID uuid.UUID,
    m *antv1.GenerateStrategyRequest,
    stream *connect.ServerStream[antv1.GenerateStrategyChunk],
) error {
    // 1. Parse backtest metrics
    var metrics ai.FeedbackMetrics
    if err := json.Unmarshal([]byte(m.BacktestMetricsJson), &metrics); err != nil {
        return stream.Send(&antv1.GenerateStrategyChunk{
            Phase: "done", Error: "failed to parse backtest metrics",
        })
    }

    // 2. Get feedback hints from router
    routing := ai.RouteFeedback(m.FeedbackMessage, &metrics)

    // 3. Build feedback prompt
    builder := ai.NewStrategyPromptBuilder()
    sysPrompt, userPrompt := builder.BuildFeedbackPrompt(&ai.FeedbackPromptParams{
        PreviousCode: m.PreviousCode,
        Metrics:      &metrics,
        FeedbackMessage: m.FeedbackMessage,
        FeedbackHints:   routing.Reason,
    })

    // 4. Stream LLM response, parse sections on the fly
    stream.Send(&antv1.GenerateStrategyChunk{Phase: "analyzing"})
    
    var fullBuf strings.Builder
    err := s.systemSvc.ChatCompletionStream(ctx, userID,
        []systemai.ChatMessage{{Role: "system", Content: sysPrompt}, {Role: "user", Content: userPrompt}},
        "", func(chunk systemai.ChatStreamChunk) error {
            fullBuf.WriteString(chunk.Content)
            sections := parseSections(fullBuf.String())
            return stream.Send(&antv1.GenerateStrategyChunk{
                Phase:    "generating",
                Delta:    chunk.Content,
                Analysis: sections.Analysis,
                Advice:   sections.Advice,
            })
        })
    if err != nil {
        return stream.Send(&antv1.GenerateStrategyChunk{
            Phase: "done", Error: systemai.FriendlyError(err),
        })
    }

    // 5. Final parse: extract code + run compliance
    raw := fullBuf.String()
    fullSections := parseSections(raw)
    code := s.extractCode(fullSections.Code)
    
    if issues := s.runComplianceCheck(code); len(issues) > 0 {
        return stream.Send(&antv1.GenerateStrategyChunk{
            Phase: "compliance", Code: code, ComplianceIssues: issues,
            Analysis: fullSections.Analysis, Advice: fullSections.Advice,
        })
    }

    // 6. Auto-backtest
    runID, btErr := s.finalizeWithBacktest(ctx, userID, code, m.Symbol, m.Timeframe)

    // 7. Persist exchange to conversation
    s.persistExchange(ctx, userID, m)

    return stream.Send(&antv1.GenerateStrategyChunk{
        Phase: "done", Code: code, BacktestRunId: runID,
        Analysis: fullSections.Analysis, Advice: fullSections.Advice,
        Error: btErr,
    })
}
```

### 5.3 辅助函数: parseSections

```go
// parseSections extracts <section type="..."> blocks from LLM output.
// Partial output is fine — missing sections are empty strings.
type parsedSections struct {
    Analysis string
    Advice   string
    Code     string
}

func parseSections(raw string) parsedSections {
    var s parsedSections
    s.Analysis = extractSection(raw, "analysis")
    s.Advice   = extractSection(raw, "advice")
    s.Code     = extractSection(raw, "code")
    return s
}

func extractSection(raw, sectionType string) string {
    // Match <section type="TYPE"> ... </section>
    re := regexp.MustCompile(
        `(?s)<section\s+type="` + regexp.QuoteMeta(sectionType) + `">(.*?)</section>`,
    )
    m := re.FindStringSubmatch(raw)
    if len(m) < 2 {
        return ""
    }
    return strings.TrimSpace(m[1])
}
```

### 5.4 Prompt Builder: BuildFeedbackPrompt

```go
// FeedbackPromptParams holds inputs for the feedback prompt builder.
type FeedbackPromptParams struct {
    PreviousCode    string
    Metrics         *FeedbackMetrics
    FeedbackMessage string
    FeedbackHints   string // from feedback_router
}

// BuildFeedbackPrompt returns system + user prompts for feedback mode.
func (b *StrategyPromptBuilder) BuildFeedbackPrompt(p *FeedbackPromptParams) (string, string) {
    system := fmt.Sprintf(
        feedbackSystemTemplate,
        strategyContractText(),
        p.PreviousCode,
        p.Metrics.FormatPromptContext(),
        p.FeedbackHints,
    )
    user := fmt.Sprintf("【用户反馈】%s\n\n请分析回测结果，给出建议，并生成优化后的代码。", p.FeedbackMessage)
    return system, user
}
```

常量 `feedbackSystemTemplate` 和 `strategyContractText()` 保持与现有 `BuildSystemPrompt` 一致。

## 6. 前端实现

### 6.1 文件变更清单

| 文件 | 改动 | 行数 |
|------|------|------|
| `client/strategyGen.ts` | Input 接口 +3 fields + handleChunk +3 fields | ~15 |
| `components/strategy/AIChatPanel.tsx` | BacktestResultCard + feedback 模式检测 + analysis/advice 展示 | ~50 |

### 6.2 Client: StrategyGenInput 扩展

```typescript
export interface StrategyGenInput {
  message: string;
  conversationId?: string;
  symbol?: string;
  timeframe?: string;
  templateId?: string;
  clarificationRound?: number;
  // Phase 3: feedback context
  previousCode?: string;
  backtestMetricsJson?: string;
  feedbackMessage?: string;
}

export interface StrategyGenCallbacks {
  onPhase: (phase: string) => void;
  onDelta: (delta: string) => void;
  onQuestions: (questions: string[]) => void;
  onCode: (code: string) => void;
  onBacktestId: (runId: string) => void;
  onTemplate: (name: string) => void;
  // Phase 3: structured feedback output
  onAnalysis: (text: string) => void;
  onAdvice: (text: string) => void;
  onError: (error: string) => void;
  onDone: () => void;
}
```

`handleChunk` 新增:
```typescript
if (chunk.analysis) cbs.onAnalysis(chunk.analysis);
if (chunk.advice) cbs.onAdvice(chunk.advice);
```

### 6.3 AIChatPanel: Backtest Context 感知

```typescript
// 新增 state
const [backtestMetrics, setBacktestMetrics] = useState<{
  sharpeRatio: number; maxDrawdown: number; winRate: number;
  profitFactor: number; totalReturn: number; totalTrades: number;
} | null>(null);
const [analysisText, setAnalysisText] = useState('');
const [adviceText, setAdviceText] = useState('');

// 修改 detectMode: 有回测 context 时强制走 generate (feedback mode)
function detectMode(msg: string, hasCode: boolean, hasBacktest: boolean): ... {
  if (hasBacktest) return 'generate';  // ← 新增：feedback mode
  // ... existing logic ...
}

// 修改 handleGenerate: 传入 feedback context
const handleGenerate = useCallback((msg: string, round: number, isFeedback = false) => {
  const input: StrategyGenInput = {
    message: msg, symbol, timeframe,
    clarificationRound: round,
    conversationId: sessionId || '',
  };
  if (isFeedback && backtestMetrics) {
    input.previousCode = code;
    input.backtestMetricsJson = JSON.stringify(backtestMetrics);
    input.feedbackMessage = msg;
  }
  // ... rest of stream call ...
}, [...]);
```

### 6.4 AIChatPanel: BacktestResultCard 组件

```tsx
function BacktestResultCard({ metrics, analysis, advice, onApply, code }: {
  metrics: BacktestMetrics | null;
  analysis: string;
  advice: string;
  onApply: (code: string) => void;
  code: string;
}) {
  if (!metrics) return null;
  return (
    <div style={{ padding: 10, margin: '8px 0', borderRadius: 6,
      background: '#f6ffed', border: '1px solid #b7eb8f' }}>
      {/* Metrics row */}
      <div style={{ display: 'flex', gap: 12, marginBottom: 6, flexWrap: 'wrap' }}>
        <MetricBadge label="Sharpe" value={metrics.sharpeRatio?.toFixed(2)} />
        <MetricBadge label="Max DD" value={`${(metrics.maxDrawdown * 100).toFixed(1)}%`}
          warn={metrics.maxDrawdown > 0.2} />
        <MetricBadge label="Win Rate" value={`${(metrics.winRate * 100).toFixed(0)}%`} />
        <MetricBadge label="Trades" value={String(metrics.totalTrades)} />
        <MetricBadge label="Return" value={`${(metrics.totalReturn * 100).toFixed(1)}%`}
          positive={metrics.totalReturn > 0} />
      </div>
      {/* Analysis */}
      {analysis && <div style={{ fontSize: 12, color: '#595959', marginBottom: 4 }}>🔍 {analysis}</div>}
      {/* Advice + Apply */}
      {advice && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontSize: 12, color: '#1677ff', flex: 1 }}>💡 {advice}</span>
          <Button size="small" type="primary" onClick={() => onApply(code)}>Apply Code</Button>
        </div>
      )}
    </div>
  );
}

function MetricBadge({ label, value, warn, positive }: {
  label: string; value: string; warn?: boolean; positive?: boolean;
}) {
  const color = warn ? '#cf1322' : positive === false ? '#cf1322' : positive ? '#389e0d' : '#595959';
  return (
    <span style={{ fontSize: 11, color, background: '#fff', padding: '2px 6px',
      borderRadius: 3, border: '1px solid #e8e8e8' }}>
      <b>{label}</b> {value}
    </span>
  );
}
```

### 6.5 AIChatPanel: 整合到聊天流

现有聊天流：
```
streamText (纯文本 delta)  → 直接展示在消息区
code → autoApply 到编辑器
```

增强后：
```
delta (streaming) → 实时展示
analysis/advice → 在流完成时替换为 BacktestResultCard
code → autoApply + 设置 backtestMetrics (为下一轮反馈准备)
```

回测结果获取：`AIChatPanel` 已有 `onBacktestId` 回调。监听 `WatchBacktestRun` SSE 获取指标：
```typescript
onBacktestId: (runId) => {
  setBacktestId(runId);
  // Watch for backtest completion to get metrics
  pythonStrategyApi.watchBacktestRun(runId, (update) => {
    if (update.status === 'SUCCEEDED' && update.metrics) {
      setBacktestMetrics(update.metrics);
    }
  });
},
```

### 6.6 状态流转

```
idle
  │ 用户输入 "做一个均线策略"
  ▼
streaming (fresh mode)
  │ delta → 实时展示
  │ code → apply + 触发 WatchBacktestRun
  ▼
done (backtestId 存在)
  │ backtest 完成 → backtestMetrics 填充
  │ BacktestResultCard 展示
  ▼
用户输入 "加入止损，保守一点"
  │ detectMode: hasBacktest=true → "generate" (feedback mode)
  ▼
streaming (feedback mode)
  │ analysis → 展示分析
  │ advice → 展示建议
  │ delta → 代码流
  │ code → apply + 触发 WatchBacktestRun
  ▼
done (新 backtestId + 新 backtestMetrics)
  │ 新 BacktestResultCard 展示
  │ 循环继续...
```

## 7. 错误处理

| 场景 | 处理 |
|------|------|
| LLM 未输出 `<section type="code">` | `extractCode()` fallback: 从原始输出提取代码块 |
| `backtestMetricsJson` 解析失败 | 返回 error chunk，降级到 fresh mode |
| 合规检查失败 | 返回 `compliance` phase + issues，不触发自动回测 |
| 自动回测失败 | 返回 code 但 `backtestRunId` 为空，显示 "回测触发失败" 警告 |
| 用户连续快速发送反馈 | `isBusy` 状态禁止发送（已有逻辑） |
| 反馈中无前次代码 | 前端不传 `previousCode` → handler 走 fresh mode |

## 8. 测试策略

```bash
# 1. Unit: parseSections 解析正确
go test ./internal/ai/ -run TestParseSections -v

# 2. Unit: BuildFeedbackPrompt 输出格式
go test ./internal/ai/ -run TestBuildFeedbackPrompt -v

# 3. Integration: handler feedback 分支
go test ./internal/connect/ai/ -run TestFeedbackLoop -v

# 4. E2E: 生成 → 反馈 → 迭代
go test -tags=e2e ./tests/e2e/ -run TestFeedbackIteration -v
```

测试用例：
- `TestParseSections`: 完整三 section / 缺 section / 乱序 / 空内容 / 嵌套 fenced code
- `TestBuildFeedbackPrompt`: 指标正确格式化 / hints 注入 / 多行代码保持缩进
- `TestFeedbackLoop`: fresh→metrics→feedback→code regenerated with stop-loss
- `TestFeedbackIteration`: 3 轮连续迭代不退化

## 9. 验收命令

```bash
# 1. Proto 编译
buf generate

# 2. Go build
go build ./...

# 3. TypeScript type check
cd frontend && npx tsc --noEmit

# 4. File size check
cd backend && python3 scripts/check-file-lines.py --strict

# 5. Unit tests
go test ./internal/ai/... -v -count=1
go test ./internal/connect/ai/... -v -count=1
```

## 10. 实施任务概要

| # | 任务 | 预估行数 | 依赖 |
|---|------|---------|------|
| 1 | Proto: `strategy_generation.proto` +5 fields (3 req + 2 chunk) | 10 | — |
| 2 | 重新生成 Go + TS proto | auto | 1 |
| 3 | `strategy_prompt.go`: `BuildFeedbackPrompt()` | 40 | — |
| 4 | `strategy_gen_helpers.go`: `parseSections()` | 25 | — |
| 5 | `strategy_gen_handler.go`: feedback 分支 | 50 | 2,3,4 |
| 6 | `client/strategyGen.ts`: Input + handleChunk 扩展 | 15 | 2 |
| 7 | `AIChatPanel.tsx`: BacktestResultCard + feedback 检测 + WatchBacktestRun | 50 | 2,6 |

## 11. 审计确认

| 决策 | 最优解 | 替代方案被拒原因 |
|------|--------|-----------------|
| 单 RPC 双模式 | ✅ | 新 RPC 增加前端路由复杂度 |
| 反馈全走 LLM | ✅ | 关键词路由丢失 nuance |
| `<section>` 结构化输出 | ✅ | 新增 proto response 类型需改接口契约 |
| `analysis`/`advice` 独立字段 | ✅ 与 `code` 语义分离 | 全塞 `delta` 中前端难解析 |
| Temperature 0.5 (feedback) | ✅ 平衡创造性与确定性 | 0.3 太僵, 0.7 太飘 |
| 向后兼容（`code` 字段保留） | ✅ 存量前端无影响 | 删除 `code` 字段是 breaking change |
