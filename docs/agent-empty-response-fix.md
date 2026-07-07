# 修复方案: Agent 空响应 —— 多内容通道解析（provider 兼容性）

> 交付对象: DeepSeek 落地实现
> 症状: 复杂/长 prompt 时 agent 报 `[deepseek] chat stream returned empty response`，`raw_len=0`
> 目标: 让 agent 像 Claude Code 一样兼容所有供应商（DeepSeek / Qwen / GLM / Opus / GPT）
> 日期: 2026-07-07

---

## §1 根因（已定位，非猜测）

流解析器只读单一 `delta.content` 字段：

**文件**: `backend/internal/service/systemai/chat_stream.go:130-139`
```go
var chunk struct {
    Choices []struct {
        Delta struct {
            Content   string           `json:"content"`        // ← 只认这一个通道
            ToolCalls []StreamToolCall `json:"tool_calls"`
        } `json:"delta"`
        FinishReason *string `json:"finish_reason"`
    } `json:"choices"`
    Usage *ChatUsage `json:"usage,omitempty"`
}
```

**问题**: 现代模型有多个输出通道：
- `content` —— 普通回答（deepseek-chat / GPT / Claude）
- `reasoning_content` —— 思维链通道（DeepSeek R1、Qwen QwQ、GLM-Zero，以及把模型重新包装的**聚合网关**，例如用户使用的 `deepseek-v4-flash`）

当模型/网关通过 `reasoning_content` 输出时，我们完全没解析它。于是 `chat_stream.go:210` 的判断：
```go
if totalContentLen == 0 && len(toolCallAcc) == 0 {
    return &failoverErr{msg: "chat stream returned empty response", transient: true}
}
```
误判为"空响应"→ 首轮就失败 → `raw_len=0`。

**结论**: 这是流解析层的结构性缺陷，与具体供应商无关。任何走 `reasoning_content` 的模型都会触发。这正是"agent 不兼容所有供应商"的根本原因。

**验证方式**: 全库 `grep reasoning_content` → 零处理。

---

## §2 修复方案（provider-agnostic，一次解决所有供应商）

核心原则（对齐 Claude Code）: **统一解析所有内容通道，永不因单通道为空就判定失败。**

### 改动 1 — 数据结构加 reasoning 通道

**文件**: `backend/internal/service/systemai/chat.go`

`ChatStreamChunk`（约 83-89 行）新增 `Reasoning` 字段：
```go
type ChatStreamChunk struct {
    Content      string
    Reasoning    string            // reasoning_content 通道
    Done         bool
    FinishReason string
    ToolCalls    []StreamToolCall
}
```

`ChatMessage`（约 19-25 行）新增 `ReasoningContent` 字段（用于非流式响应解析，`omitempty` 保证不会污染出站请求）：
```go
type ChatMessage struct {
    Role       string     `json:"role"`
    Content    string     `json:"content,omitempty"`
    ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
    ToolCallID string     `json:"tool_call_id,omitempty"`
    Name       string     `json:"name,omitempty"`
    ReasoningContent string `json:"reasoning_content,omitempty"` // 新增
}
```

### 改动 2 — 流解析器解析并计入 reasoning

**文件**: `backend/internal/service/systemai/chat_stream.go`

(a) delta struct 加字段（约 130-139 行）:
```go
Delta struct {
    Content          string           `json:"content"`
    ReasoningContent string           `json:"reasoning_content"` // 新增
    ToolCalls        []StreamToolCall `json:"tool_calls"`
} `json:"delta"`
```

(b) 新增累计变量（约 118 行 `totalContentLen := 0` 附近）:
```go
totalReasoningLen := 0
lastFinishReason := ""   // 用于改进错误诊断
```

(c) 累计 reasoning 长度（约 151-153 行 content 累计处）:
```go
if c.Delta.Content != "" {
    totalContentLen += len(c.Delta.Content)
}
if c.Delta.ReasoningContent != "" {
    totalReasoningLen += len(c.Delta.ReasoningContent)
}
```

(d) 记录 finishReason 并在 emit 时带上 Reasoning（约 174-200 行）:
```go
if finishReason != "" {
    lastFinishReason = finishReason
}
// ...
if err := onChunk(ChatStreamChunk{
    Content:      c.Delta.Content,
    Reasoning:    c.Delta.ReasoningContent, // 新增
    Done:         finishReason != "",
    FinishReason: finishReason,
    ToolCalls:    finalToolCalls,
}); err != nil {
    return err
}
```

(e) 修复空判断 + 改进错误诊断（约 209-211 行）:
```go
if totalContentLen == 0 && totalReasoningLen == 0 && len(toolCallAcc) == 0 {
    return &failoverErr{
        msg: fmt.Sprintf("[%s] chat stream returned empty response (finish_reason=%q)", p.providerID, lastFinishReason),
        transient: true,
    }
}
```
> 注意: 加入 `finish_reason` 到错误信息，未来若真遇到 `length`（max_tokens 截断）等其它空响应原因，日志能直接区分，不用再盲猜。

### 改动 3 — AgentLoop 消费 reasoning，content 空时回退

**文件**: `backend/internal/connect/ai/agent_loop.go`

在 `run` 的流回调（约 78-94 行）里，除了 `roundBuf.WriteString(chunk.Content)`，新增一个 `reasoningBuf`：
```go
var reasoningBuf strings.Builder
// onChunk 回调内:
roundBuf.WriteString(chunk.Content)
reasoningBuf.WriteString(chunk.Reasoning)
```

round 结束取 `roundText`（约 99 行）后，加回退逻辑：
```go
roundText := strings.TrimSpace(roundBuf.String())
reasoningText := strings.TrimSpace(reasoningBuf.String())
// content 为空但 reasoning 有内容 → 用 reasoning 兜底（聚合网关把答案放这里，
// 或 R1 类模型把代码写在思维链里），保证代码能被 ExtractCode 提取。
if roundText == "" && reasoningText != "" {
    roundText = reasoningText
}
```
> 这样 `roundText == "" && len(toolCalls) == 0` 的空错误（约 100-102 行）不再误触发，且 `ExtractCode(roundText)` 能从 reasoning 里提取 ```python 代码块。

### 改动 4 — 非流式路径同样回退

**文件**: `backend/internal/service/systemai/chat.go` → `tryChatCompletion`（约 282 行 return 处）

```go
msg := cr.Choices[0].Message
content := strings.TrimSpace(msg.Content)
if content == "" && msg.ReasoningContent != "" {
    content = strings.TrimSpace(msg.ReasoningContent) // 兜底
}
return content, msg.ToolCalls, cr.Usage, nil
```

---

## §3 REUSE / NEW 核对（提交时填）

- 流解析: `REUSE: tryChatCompletionStream @ chat_stream.go:74`（改造，非新建）
- 非流式: `REUSE: tryChatCompletion @ chat.go:232`（改造）
- Loop 消费: `REUSE: AgentLoop.run @ agent_loop.go:71`（改造）
- reasoning 通道解析: `NEW: 无现成能力（已搜: reasoning_content，全库零处理）`

---

## §4 验收标准

1. 用户原始复杂 prompt（"四小时放量大跌…加仓…出场 200 倍"）→ agent 不再报 empty response，能产出策略代码。
2. `docker logs ant-backend | grep "loop done"` → `raw_len > 0`。
3. 回归: 短 prompt（deepseek-chat 走 content 通道）行为不变。
4. 若仍空 → 新错误信息带 `finish_reason=...`，据此判断是否 max_tokens 截断（改调 §5 备选）。

---

## §5 备选/后续（若 §2 修复后仍偶发空响应）

若错误信息显示 `finish_reason="length"` → 是 max_tokens 输出截断（复杂策略 + 长思维链吃满 8192）。此时：
- 让用户在 AI Settings 调高该 provider 的 `max_tokens`（已可配置，见 `SystemAIConfigRow.MaxTokens`）
- 或对推理模型把 `defaultMaxTokens`（`chat.go:131`）调高到 16384

若显示 `finish_reason="content_filter"` → 供应商内容审查拦截，需换 provider（failover 已支持）。

---

## §6 硬约束提醒（DeepSeek 必读）

- ❌ 不引入 JSON 持久化/交换（LLM tool_call args 的 json 是 OpenAI 协议，已豁免；本次改动只加 struct tag，不新增 json.Marshal 业务逻辑）
- ✅ 提交前: `go build ./...` + `cd backend && go run ./tools/check-file-lines --strict`
- ✅ 部署: `docker compose build backend && docker compose up -d backend`
- ✅ 改动集中在 3 个文件，均为改造现有函数，无新文件，不触发行数红线
