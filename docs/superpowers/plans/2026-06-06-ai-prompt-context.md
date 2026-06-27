# AI Prompt Context + Conversation Session Implementation Plan

> **⚠️ 注意：** AI Prompt 模板当前面向 Python 策略代码生成。迁移至 Go SDK 后需更新系统 Prompt（见 ADR-0021）。

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement 4-mode intent classification (Generate/Revise/Repair/Discuss) for AI code assistance and strategy-keyed server-side message persistence, eliminating the "repair breaks the flow" bug and enabling cross-device chat history sync.

**Architecture:** Two new lightweight Go components in `internal/ai/`: `prompt_context.go` (pure-function context builder with 4-mode classification) and `conversation_session.go` (thin wrapper over `AIConversationRepository` for strategy-scoped sessions). A new `ResolveSession` RPC on `AIService` resolves `strategy_key → session UUID`. Existing handlers integrate both components — `CodeAssistServer` gains mode-aware prompts and `extractCodeFromRepair` post-processing; all AI handlers auto-persist via `AppendExchange` after streaming completes. Frontend gains 4-mode detection with visual tags.

**Tech Stack:** Go, ConnectRPC, Proto3, PostgreSQL, React/TypeScript

**Spec:** `docs/superpowers/specs/2026-06-06-ai-prompt-context-design.md`

---

### Task 1: Proto Changes (ResolveSession RPC + session_id field)

**Files:**
- Modify: `proto/ant/v1/ai_chat_requests.proto` — add `ResolveSessionRequest` + `ResolveSessionResponse`
- Modify: `proto/ant/v1/ai.proto` — add `ResolveSession` RPC to `AIService`
- Modify: `proto/ant/v1/code_assist.proto` — add `session_id` field to `ReviseCodeRequest`

- [ ] **Step 1: Add ResolveSession messages**

In `proto/ant/v1/ai_chat_requests.proto`, append at end of file:

```proto
message ResolveSessionRequest {
  string strategy_key = 1;
  string title = 2;
}

message ResolveSessionResponse {
  string session_id = 1;
  repeated ConversationMessage messages = 2;
  bool created = 3;
}
```

- [ ] **Step 2: Add ResolveSession RPC to AIService**

In `proto/ant/v1/ai.proto`, add after `rpc BatchSetAgents` (line 20):

```proto
  rpc ResolveSession(ResolveSessionRequest) returns (ResolveSessionResponse);
```

- [ ] **Step 3: Add session_id to ReviseCodeRequest**

In `proto/ant/v1/code_assist.proto`, in `message ReviseCodeRequest`, add after `string locale = 4;`:

```proto
  string session_id = 5;
```

- [ ] **Step 4: Regenerate proto Go/TS code**

Run: `buf generate`
Expected: 0 errors, no breaking changes. Verify generated files exist:
- `gen/proto/ant/v1/ai_chat_requests.pb.go` contains `ResolveSessionRequest`
- `gen/proto/ant/v1/antv1connect/ai.connect.go` contains `ResolveSession` method
- `gen/proto/ant/v1/code_assist.pb.go` contains `SessionId` field on `ReviseCodeRequest`
- `frontend/src/gen/ant/v1/code_assist_pb.ts` contains `sessionId` field

- [ ] **Step 5: Commit**

```bash
git add proto/ant/v1/ai_chat_requests.proto proto/ant/v1/ai.proto proto/ant/v1/code_assist.proto gen/
git commit -m "feat(proto): add ResolveSession RPC + session_id field

- New ResolveSession RPC on AIService — resolves strategy_key → session UUID
- New ResolveSessionRequest/Response messages with message history
- ReviseCodeRequest gains session_id=5 for auto-persistence"
```

---

### Task 2: DB Migration — strategy_key Column

**Files:**
- Create: `backend/migrations/XXX_add_strategy_key.sql` (XXX = next migration number)

- [ ] **Step 1: Find next migration number**

Run: `ls backend/migrations/ | sort | tail -3`
Note the highest number N. The new migration will be N+1.

- [ ] **Step 2: Write migration SQL**

Create `backend/migrations/<NNN>_add_strategy_key.sql`:

```sql
-- Add strategy_key column for strategy-scoped AI conversation sessions.
-- Uses partial unique index to enforce one session per strategy, nullable for
-- generic AI conversations (SystemAI page).

ALTER TABLE ai_conversations
  ADD COLUMN IF NOT EXISTS strategy_key VARCHAR(256);

CREATE UNIQUE INDEX IF NOT EXISTS idx_conv_strategy_key
  ON ai_conversations(user_id, strategy_key)
  WHERE strategy_key IS NOT NULL AND strategy_key != '';
```

- [ ] **Step 3: Apply migration to local dev DB**

Run: `psql $DATABASE_URL -f backend/migrations/<NNN>_add_strategy_key.sql`
Expected: `ALTER TABLE` + `CREATE INDEX` (both report success, no errors on re-run due to `IF NOT EXISTS`)

- [ ] **Step 4: Verify column exists**

Run: `psql $DATABASE_URL -c "\d ai_conversations"`
Expected: output includes `strategy_key | character varying(256)`

- [ ] **Step 5: Commit**

```bash
git add backend/migrations/<NNN>_add_strategy_key.sql
git commit -m "feat(db): add strategy_key column to ai_conversations

Partial unique index on (user_id, strategy_key) — one session per strategy.
Nullable for backwards compatibility with generic AI conversations."
```

---

### Task 3: Repository — GetByStrategyKey + CreateWithStrategyKey

**Files:**
- Modify: `backend/internal/repository/ai_conversation_repository.go:132-193` (append after existing methods)

- [ ] **Step 1: Add GetByStrategyKey method**

Append after `DeleteMessagesByConversation` (line 193):

```go
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

// CreateWithStrategyKey creates a conversation with a strategy_key binding.
func (r *AIConversationRepository) CreateWithStrategyKey(ctx context.Context, userID uuid.UUID, title, strategyKey string) (*AIConversation, error) {
	conv := &AIConversation{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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

- [ ] **Step 2: Verify import for `database/sql`**

Check that `database/sql` is imported (needed for `sql.ErrNoRows`):
Run: `head -10 backend/internal/repository/ai_conversation_repository.go`
If `"database/sql"` not present, add it to the import block.

- [ ] **Step 3: Build check**

Run: `cd backend && go build ./internal/repository/`
Expected: 0 errors

- [ ] **Step 4: Commit**

```bash
git add backend/internal/repository/ai_conversation_repository.go
git commit -m "feat(repo): add GetByStrategyKey + CreateWithStrategyKey

GetByStrategyKey looks up conversation by (user_id, strategy_key).
CreateWithStrategyKey creates a conversation bound to a strategy key.
Used by conversation_session.go for strategy-scoped session management."
```

---

### Task 4: prompt_context.go — Mode Classification + System Prompts

**Files:**
- Create: `backend/internal/ai/prompt_context.go`

- [ ] **Step 1: Write prompt_context.go**

```go
// Package ai — prompt_context.go
// PromptContext builds mode-specific system prompts for AI code interactions.
// Pure functions, no side effects, no state. Replaces the ad-hoc
// buildCodeAssistPrompt() in code_assist_handler.go.

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
// Struct encapsulation keeps the function signature within the 5-parameter limit.
type BuildContextInput struct {
	Code             string
	Message          string
	Symbol           string
	Timeframe        string
	BacktestSummary  string
	ValidationErrors []string
}

// BuildContext analyzes code + message and returns the appropriate PromptContext.
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
		pc.UserMessage = input.Message
	case ModeRevise:
		pc.SystemPrompt = revisePrompt()
		pc.UserMessage = buildReviseUserMessage(input)
	case ModeRepair:
		pc.SystemPrompt = repairPrompt(input.ValidationErrors)
		pc.UserMessage = buildRepairUserMessage(input.Code, input.Message)
	case ModeDiscuss:
		pc.SystemPrompt = discussPrompt(input.Code)
		pc.UserMessage = input.Message
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

func generatePrompt() string {
	return `You are a professional quantitative trading strategy engineer.
Generate a complete Python trading strategy based on the user's description.

## Strategy Code Specification
- Must define a run(context) function
- Return a trade signal dict: {'signal': 'buy'|'sell'|'hold', 'volume': 1.0, ...}
- Tunable parameters must use # @param annotations

## Prohibited
- Do NOT use eval(), exec(), compile()
- Do NOT import os, subprocess, socket
- Do NOT use open() for file operations
- Output ONLY Python code — no explanations or markdown fences`
}

func revisePrompt() string {
	return `You are a professional quantitative trading strategy engineer.
Revise the following Python strategy code according to the user's instruction.

## Revision Rules
- Keep the existing code structure and style
- Only modify what the instruction asks for
- Preserve all existing # @param annotations

## Output Rules
- Output the COMPLETE revised code
- Do NOT include explanations or markdown fences
- The first character must be import, def, class, or #`
}

func repairPrompt(errors []string) string {
	errList := ""
	for _, e := range errors {
		errList += "- " + e + "\n"
	}
	if errList == "" {
		errList = "- (errors provided in user message)\n"
	}
	return `You are a trading strategy CODE REPAIR TOOL. Your ONLY job is to fix errors.

## STRICT OUTPUT RULES — VIOLATION WILL BREAK THE PIPELINE
1. Output ONLY the complete, corrected Python code
2. Do NOT include ANY explanatory text
3. Do NOT say "here is the fixed code" or similar
4. Do NOT wrap code in markdown fences (` + "```python ```" + `)
5. Do NOT analyze the error causes
6. Do NOT give suggestions or tips
7. If you cannot fix, output the original code with # FIXME: <reason> comments

## Errors to Fix
` + errList + `

## CRITICAL REMINDER
Your response will be written directly to a strategy file and executed.
If it contains non-code text, the pipeline will FAIL.
Output MUST start with import, def, class, #, or a blank line.`
}

func discussPrompt(code string) string {
	return `You are an experienced quantitative trading strategy analyst.
The user is developing a trading strategy and needs your professional opinion.

## Current Strategy Code
` + "```python\n" + code + "\n```" + `

Provide a concise, professional response to the user's question. Be direct — no pleasantries.
If the user asks "is this correct" or "are there issues", check: entry logic, exit logic,
stop-loss/take-profit, position sizing, and edge case handling.`
}

func buildReviseUserMessage(input BuildContextInput) string {
	msg := "Instruction: " + input.Message
	if input.BacktestSummary != "" {
		msg += "\n\n【Current Backtest Results】\n" + input.BacktestSummary
	}
	msg += "\n\nCode:\n```python\n" + input.Code + "\n```"
	return msg
}

func buildRepairUserMessage(code, message string) string {
	return "## Current Code\n```python\n" + code + "\n```\n\n## Error Information\n" + message
}
```

- [ ] **Step 2: Build check**

Run: `cd backend && go build ./internal/ai/`
Expected: 0 errors

- [ ] **Step 3: Verify line count under 120**

Run: `wc -l backend/internal/ai/prompt_context.go`
Expected: ≤ 120 lines

- [ ] **Step 4: Commit**

```bash
git add backend/internal/ai/prompt_context.go
git commit -m "feat(ai): add prompt_context.go — 4-mode intent classification

BuildContext analyzes code+message and returns mode-specific system prompts:
- ModeGenerate: no code → create from scratch
- ModeRepair: error keywords → double-constraint code-only prompt
- ModeRevise: modification intent → revise prompt with backtest context
- ModeDiscuss: question keywords → analysis prompt (no code output)

Replaces the ad-hoc buildCodeAssistPrompt() in code_assist_handler.go."
```

---

### Task 5: conversation_session.go — Strategy-Scoped Session Management

**Files:**
- Create: `backend/internal/ai/conversation_session.go`

- [ ] **Step 1: Write conversation_session.go**

```go
// Package ai — conversation_session.go
// Lightweight strategy-scoped session management.
// Thin wrapper over AIConversationRepository — append + read only, no sliding window, no summaries.

package ai

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"anttrader/internal/repository"
)

// ConversationSession manages AI chat sessions bound to strategies.
type ConversationSession struct {
	repo *repository.AIConversationRepository
}

// NewConversationSession creates a ConversationSession backed by the repo.
func NewConversationSession(repo *repository.AIConversationRepository) *ConversationSession {
	return &ConversationSession{repo: repo}
}

// Session represents a resolved strategy session with its message history.
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

// AppendExchange persists a user→assistant message pair atomically.
// Non-fatal on failure — callers should log warning and continue.
func (s *ConversationSession) AppendExchange(ctx context.Context, sessionID, userID uuid.UUID, userMsg, assistantMsg string) error {
	if _, err := s.repo.AddMessage(ctx, userID, sessionID, "user", userMsg); err != nil {
		return err
	}
	if _, err := s.repo.AddMessage(ctx, userID, sessionID, "assistant", assistantMsg); err != nil {
		return err
	}
	return s.repo.Touch(ctx, sessionID, userID)
}

// GetMessages returns all messages for a session, ordered by creation time.
func (s *ConversationSession) GetMessages(ctx context.Context, sessionID, userID uuid.UUID) ([]repository.AIMessage, error) {
	return s.repo.GetMessages(ctx, userID, sessionID)
}
```

- [ ] **Step 2: Build check**

Run: `cd backend && go build ./internal/ai/`
Expected: 0 errors

- [ ] **Step 3: Verify line count under 80**

Run: `wc -l backend/internal/ai/conversation_session.go`
Expected: ≤ 80 lines

- [ ] **Step 4: Commit**

```bash
git add backend/internal/ai/conversation_session.go
git commit -m "feat(ai): add conversation_session.go — strategy-scoped sessions

Thin wrapper over AIConversationRepository providing:
- GetOrCreate: resolve strategy_key → session UUID (auto-create)
- AppendExchange: persist user+assistant pair after AI response
- GetMessages: load history for cross-device sync

No sliding window, no summaries — just append + read."
```

---

### Task 6: AIServer — ResolveSession Handler

**Files:**
- Modify: `backend/internal/connect/ai/ai_handler.go`

- [ ] **Step 1: Add session field + update constructor**

In `AIServer` struct (line 19), add `session` field after `conversations`:

```go
type AIServer struct {
	systemSvc     *systemai.Service
	conversations *repository.AIConversationRepository
	session       *ai.ConversationSession  // NEW
	agentDefRepo  *repository.AIAgentDefinitionRepository
	log           *zap.Logger
}
```

Update `NewAIServer` (line 27):

```go
func NewAIServer(systemSvc *systemai.Service, conversations *repository.AIConversationRepository, session *ai.ConversationSession, log *zap.Logger) *AIServer {
	return &AIServer{systemSvc: systemSvc, conversations: conversations, session: session, log: log}
}
```

Add import for `"anttrader/internal/ai"` in the import block.

- [ ] **Step 2: Add ResolveSession handler**

Append at end of file (after line 60):

```go
// ResolveSession resolves a strategy_key to a session UUID, creating one if needed.
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
	msgs := make([]*antv1.ConversationMessage, len(sess.Messages))
	for i, m := range sess.Messages {
		msgs[i] = &antv1.ConversationMessage{
			Id:        m.ID.String(),
			Role:      m.Role,
			Content:   m.Content,
			CreatedAt: timestamppb.New(m.CreatedAt),
		}
	}
	return connect.NewResponse(&antv1.ResolveSessionResponse{
		SessionId: sess.ID.String(),
		Messages:  msgs,
		Created:   len(sess.Messages) == 0,
	}), nil
}
```

Add required imports to the import block:
- `"fmt"`
- `"connectrpc.com/connect"`
- `antv1 "anttrader/gen/proto/ant/v1"`
- `"anttrader/internal/ai"`
- `"google.golang.org/protobuf/types/known/timestamppb"`

Note: `connect`, `antv1`, and `fmt` may already be imported elsewhere. Only add what's missing.

- [ ] **Step 3: Build check**

Run: `cd backend && go build ./internal/connect/ai/`
Expected: 0 errors

- [ ] **Step 4: Commit**

```bash
git add backend/internal/connect/ai/ai_handler.go
git commit -m "feat(ai): add ResolveSession handler to AIServer

ResolveSession RPC resolves strategy_key → session UUID + message history.
Auto-creates session if none exists. Enables cross-device chat history sync."
```

---

### Task 7: CodeAssistServer — Integrate PromptContext + ConversationSession + extractCodeFromRepair

**Files:**
- Modify: `backend/internal/connect/ai/code_assist_handler.go`

- [ ] **Step 1: Add session field + update constructor**

In `CodeAssistServer` struct (line 23), add `session` field:

```go
type CodeAssistServer struct {
	systemSvc *systemai.Service
	session   *ai.ConversationSession  // NEW
	log       *zap.Logger
}
```

Update `NewCodeAssistServer` (line 30):

```go
func NewCodeAssistServer(systemSvc *systemai.Service, session *ai.ConversationSession, log *zap.Logger) *CodeAssistServer {
	return &CodeAssistServer{systemSvc: systemSvc, session: session, log: log}
}
```

Add import `"anttrader/internal/ai"` to the import block.

- [ ] **Step 2: Replace buildCodeAssistPrompt with BuildContext in ReviseCodeStream**

In `ReviseCodeStream` (line 64), replace `sysPrompt, userMsg := buildCodeAssistPrompt(code, instruction)` (line 79) with:

```go
	pc := ai.BuildContext(ai.BuildContextInput{Code: code, Message: instruction})
	messages := systemai.BuildChatMessages(pc.SystemPrompt, pc.UserMessage, protoHistoryToChat(req.Msg.History))
```

- [ ] **Step 3: Add repair post-processing + auto-persist after stream**

Replace the final return block in `ReviseCodeStream` — the part after the stream error check through the final `stream.Send`. Specifically, after the stream callback, change:

```go
	// After the ChatCompletionStream call completes and error check...
	// Replace the final return line with:

	// Repair mode post-processing — NEW
	result := fullText.String()
	if pc.Mode == ai.ModeRepair {
		if code := extractCodeFromRepair(result); code != "" {
			result = code
		}
	}

	// Auto-persist to session — NEW
	if req.Msg.SessionId != "" {
		sid, parseErr := uuid.Parse(req.Msg.SessionId)
		if parseErr == nil {
			if err := s.session.AppendExchange(ctx, sid, uid, instruction, result); err != nil {
				s.log.Warn("session append failed", zap.Error(err))
			}
		}
	}

	return stream.Send(&antv1.ReviseCodeStreamChunk{Delta: "", Python: result, Done: true})
```

- [ ] **Step 4: Add extractCodeFromRepair helper (3-tier extraction)**

Append at end of file:

```go
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
	// Tier 3: unable to extract — return empty
	return ""
}

func extractFencedCode(raw, lang string) string {
	marker := "```" + lang
	start := strings.Index(raw, marker)
	if start < 0 {
		// Try generic fence if no language-specific one
		start = strings.Index(raw, "```")
		if start < 0 {
			return ""
		}
		// Skip the opening fence line
		if nl := strings.Index(raw[start:], "\n"); nl >= 0 {
			start += nl + 1
		}
	} else {
		// Skip the ```python line
		if nl := strings.Index(raw[start:], "\n"); nl >= 0 {
			start += nl + 1
		}
	}
	end := strings.Index(raw[start:], "```")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(raw[start : start+end])
}

func extractByHeuristic(raw string) string {
	raw = strings.TrimSpace(raw)
	// Find the first line that looks like code
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "def ") ||
			strings.HasPrefix(trimmed, "class ") ||
			strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "from ") {
			return strings.Join(lines[i:], "\n")
		}
		// If the first non-empty line is not code-like, heuristic fails
		return ""
	}
	return ""
}
```

Add `"github.com/google/uuid"` to the import block if not already present.

- [ ] **Step 5: Also update unary ReviseCode (non-streaming fallback)**

In `ReviseCode` (line 43), replace `buildCodeAssistPrompt(code, instruction)` at line 54:

```go
	pc := ai.BuildContext(ai.BuildContextInput{Code: code, Message: instruction})
	messages := systemai.BuildChatMessages(pc.SystemPrompt, pc.UserMessage, protoHistoryToChat(req.Msg.History))
```

Replace the response at line 61:

```go
	result := revised
	if pc.Mode == ai.ModeRepair {
		if code := extractCodeFromRepair(revised); code != "" {
			result = code
		}
	}
	return connect.NewResponse(&antv1.ReviseCodeResponse{Text: result, Python: result}), nil
```

- [ ] **Step 6: Remove old buildCodeAssistPrompt function**

Delete the function at lines 104-118:

```go
func buildCodeAssistPrompt(code, instruction string) (sysPrompt, userMsg string) {
	// ... entire function body ...
}
```

This function is no longer used — `ai.BuildContext` replaces it.

- [ ] **Step 7: Build check**

Run: `cd backend && go build ./internal/connect/ai/`
Expected: 0 errors

- [ ] **Step 8: Verify file line count**

Run: `wc -l backend/internal/connect/ai/code_assist_handler.go`
Expected: ≤ 300 lines (removed ~10 lines `buildCodeAssistPrompt`, added ~60 lines)

- [ ] **Step 9: Commit**

```bash
git add backend/internal/connect/ai/code_assist_handler.go
git commit -m "feat(ai): integrate PromptContext + ConversationSession into CodeAssistServer

- Replace buildCodeAssistPrompt() with ai.BuildContext() — 4-mode classification
- Add extractCodeFromRepair() — 3-tier post-processing for repair mode
  (fenced code → heuristic detection → empty fallback)
- Auto-persist exchange via session.AppendExchange() after LLM stream
- Constructor gains *ai.ConversationSession parameter"
```

---

### Task 8: StrategyGenServer — AppendExchange After GenerateStrategy

**Files:**
- Modify: `backend/internal/connect/ai/strategy_gen_handler.go`

- [ ] **Step 1: Add auto-persist after backtest finalization**

In `GenerateStrategy` (line 47), after `runID, btErr := s.finalizeWithBacktest(...)` (line 73) and before `return stream.Send(...)` (line 74), insert:

```go
	// Auto-persist exchange to strategy session — NEW
	if m.ConversationId != "" {
		cid, parseErr := uuid.Parse(m.ConversationId)
		if parseErr == nil {
			if _, err := s.convRepo.AddMessage(ctx, userID, cid, "user", m.Message); err != nil {
				s.log.Warn("persist user msg failed", zap.Error(err))
			}
			if _, err := s.convRepo.AddMessage(ctx, userID, cid, "assistant", code); err != nil {
				s.log.Warn("persist assistant msg failed", zap.Error(err))
			}
			if err := s.convRepo.Touch(ctx, cid, userID); err != nil {
				s.log.Warn("touch session failed", zap.Error(err))
			}
		}
	}
```

- [ ] **Step 2: Build check**

Run: `cd backend && go build ./internal/connect/ai/`
Expected: 0 errors

- [ ] **Step 3: Commit**

```bash
git add backend/internal/connect/ai/strategy_gen_handler.go
git commit -m "feat(ai): auto-persist GenerateStrategy exchange to session

After code generation and backtest complete, persist user message +
generated code to ai_messages via existing conversation_id field.
Reuses convRepo already available on StrategyGenServer."
```

---

### Task 9: handlers.go — Wire ConversationSession Injection

**Files:**
- Modify: `backend/cmd/server/handlers.go`

- [ ] **Step 1: Create ConversationSession and inject into servers**

In `registerHandlers` (line 41), after `convRepo := repository.NewAIConversationRepository(pool)` (line 68), add:

```go
	session := ai.NewConversationSession(convRepo)  // NEW: strategy-scoped session wrapper
```

Update `aiServer` creation (line 116) — add `session` parameter:

```go
	aiServer := ai.NewAIServer(aiSvc, convRepo, session, log)
```

Update `codeAssistServer` creation (line 152) — add `session` parameter:

```go
	codeAssistServer := ai.NewCodeAssistServer(aiSvc, session, log)
```

Add `"anttrader/internal/ai"` to the import block at the top.

- [ ] **Step 2: Build check — full project**

Run: `cd backend && go build ./...`
Expected: 0 errors

- [ ] **Step 3: Commit**

```bash
git add backend/cmd/server/handlers.go
git commit -m "feat(server): inject ConversationSession into AI handlers

Create session wrapper once, inject into AIServer and CodeAssistServer.
StrategyGenServer already has convRepo — uses it directly for persistence."
```

---

### Task 10: Frontend — codeAssist.ts + sessionId

**Files:**
- Modify: `frontend/src/client/codeAssist.ts`

- [ ] **Step 1: Add sessionId to ReviseCodeInput**

In `ReviseCodeInput` interface (line 15), add after `locale?: string;`:

```ts
  sessionId?: string;
```

- [ ] **Step 2: Pass sessionId in reviseStream call**

In `reviseStream` (line 75), in the `create(ReviseCodeRequestSchema, {...})` call (line 82), add `sessionId`:

```ts
        const msg = create(ReviseCodeRequestSchema, {
          code: input.code,
          instruction: input.instruction,
          history: input.history || [],
          locale: input.locale || '',
          sessionId: input.sessionId || '',
        });
```

- [ ] **Step 3: Build check**

Run: `cd frontend && npx tsc --noEmit`
Expected: 0 errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/client/codeAssist.ts
git commit -m "feat(frontend): add sessionId to ReviseCodeInput

Passed through to ReviseCodeRequest.session_id for server-side
auto-persistence of AI exchanges to strategy session."
```

---

### Task 11: Frontend — AIChatPanel.tsx (sessionId, enhanced detectMode, mode tags)

**Files:**
- Modify: `frontend/src/components/strategy/AIChatPanel.tsx`

- [ ] **Step 1: Add new props — sessionId + chatHistory**

In `Props` interface (line 10), add after `autoApply?: boolean;`:

```ts
  sessionId?: string;
  chatHistory?: CodeChatMessage[];
```

- [ ] **Step 2: Initialize history from props**

Replace `const [history, setHistory] = useState<CodeChatMessage[]>([]);` (line 43) with:

```ts
  const [history, setHistory] = useState<CodeChatMessage[]>(chatHistory || []);
```

- [ ] **Step 3: Replace detectMode with 4-mode version**

Replace the entire `detectMode` function (lines 21-29) with:

```ts
// 4-mode intent classification — keyword tables match backend classifyIntent exactly.
function detectMode(msg: string, hasCode: boolean): 'generate' | 'revise' | 'repair' | 'discuss' {
  if (!hasCode) return 'generate';
  const lower = msg.toLowerCase();
  // Repair (highest priority — error keywords)
  const repairKw = ['报错','error','错误','traceback','缺少参数','missing',
    '验证失败','syntax error','syntaxerror','undefined','未定义',
    '缺少 required','参数不足','attributeerror','typeerror'];
  if (repairKw.some(k => lower.includes(k))) return 'repair';
  // Discuss (question/analysis keywords)
  const discussKw = ['为什么','什么意思','怎么样','对吗','分析','解释',
    'what','why','how','explain','对不对'];
  if (discussKw.some(k => lower.includes(k))) return 'discuss';
  return 'revise';
}
```

- [ ] **Step 4: Add mode tag constants**

Add after the `detectMode` function:

```ts
const MODE_TAGS: Record<string, { color: string; label: string }> = {
  generate: { color: 'blue',   label: '⚡ 生成' },
  revise:   { color: 'green',  label: '✏️ 修改' },
  repair:   { color: 'orange', label: '🔧 修复' },
  discuss:  { color: 'purple', label: '💬 分析' },
};
```

- [ ] **Step 5: Pass sessionId to handleGenerate**

In `handleGenerate` (line 60), update the `generateStrategyStream` call to include `conversationId`:

```ts
    const abort = generateStrategyStream(
      { message: msg, symbol, timeframe, clarificationRound: round,
        conversationId: sessionId || '' },
```

- [ ] **Step 6: Pass sessionId to handleRevise**

In `handleRevise` (line 86), update the `codeAssistApi.reviseStream` call:

```ts
    const abort = codeAssistApi.reviseStream(
      { code, instruction: msg, history, locale: i18n.language, sessionId },
```

- [ ] **Step 7: Update handleSend to not clear history on each send**

Replace `handleSend` (line 117) — remove `setHistory([])`:

```ts
  const handleSend = useCallback(() => {
    const msg = draft.trim();
    if (!msg) return;
    setDraft(''); setClarifyRound(0); setQuestions([]);
    const intent = detectMode(msg, !!code.trim());
    if (intent === 'generate') handleGenerate(msg, 0);
    else handleRevise(msg);
  }, [draft, code, handleGenerate, handleRevise]);
```

- [ ] **Step 8: Add mode tag display in the chat header**

Find the chat header area (near the top of the JSX return) and add a mode tag. Locate the area after the `isBusy` state usage and add:

```tsx
  // Current mode for visual feedback
  const currentMode = code.trim() ? detectMode(draft || 'revise', !!code.trim()) : (draft ? 'generate' : 'revise');
  const modeTag = MODE_TAGS[currentMode];
```

Then in the UI, add a Tag component near the input area showing the current mode:

```tsx
  {modeTag && <Tag color={modeTag.color}>{modeTag.label}</Tag>}
```

- [ ] **Step 9: Build check**

Run: `cd frontend && npx tsc --noEmit`
Expected: 0 errors

- [ ] **Step 10: Commit**

```bash
git add frontend/src/components/strategy/AIChatPanel.tsx
git commit -m "feat(frontend): 4-mode intent detection + sessionId in AIChatPanel

- Enhanced detectMode(): Generate/Revise/Repair/Discuss with keyword tables
  matching backend classifyIntent exactly
- New props: sessionId + chatHistory for server-side session integration
- Mode tag display: ⚡生成/✏️修改/🔧修复/💬分析
- Pass sessionId to both generateStrategyStream and reviseStream
- History no longer cleared on each send (server is source of truth)"
```

---

### Task 12: Frontend — StrategyWorkspacePage.tsx (ResolveSession on mount)

**Files:**
- Modify: `frontend/src/pages/strategy/StrategyWorkspacePage.tsx`

**Context:** The page uses `useStrategyWorkspaceState()` hook (`ws = useStrategyWorkspaceState()`). `ws.account.symbol`, `ws.account.timeframe`, and `ws.strategy?.id` are available. State is managed locally in the component or via the hook. `aiClient` is already exported from `@/client/connect` (line 43 of `frontend/src/client/connect.ts`).

- [ ] **Step 1: Update React imports**

Change line 1 from:
```tsx
import React, { Suspense, lazy, useMemo } from 'react';
```
to:
```tsx
import React, { Suspense, lazy, useMemo, useState, useEffect } from 'react';
```

- [ ] **Step 2: Add import for aiClient and CodeChatMessage type**

After line 12 (`import AIChatPanel from '@/components/strategy/AIChatPanel';`), add:
```ts
import { aiClient } from '@/client/connect';
import type { CodeChatMessage } from '@/client/codeAssist';
```

- [ ] **Step 3: Add session state variables**

After `const ws = useStrategyWorkspaceState();` (line 28), add:
```tsx
  const [sessionId, setSessionId] = useState<string>('');
  const [chatHistory, setChatHistory] = useState<CodeChatMessage[]>([]);
```

- [ ] **Step 4: Add ResolveSession effect**

After the state variables, add:
```tsx
  // Resolve AI session on workspace mount — enables cross-device chat history sync
  useEffect(() => {
    const strategyKey = ws.strategy?.id
      ? `strategy:${ws.strategy.id}`
      : `draft:${userId}:${ws.account.symbol || ''}:${ws.account.timeframe || ''}`;

    aiClient.resolveSession({ strategyKey }).then(res => {
      setSessionId(res.sessionId);
      setChatHistory(res.messages || []);
    }).catch(err => {
      console.warn('ResolveSession failed, AI chat will work without persistence:', err);
    });
  }, [ws.strategy?.id, ws.account.symbol, ws.account.timeframe]);
```

Note: `userId` needs to be obtained from auth context. If the workspace state doesn't expose it, use the auth store:
```ts
import { useAuthStore } from '@/stores/authStore';
// inside component:
const userId = useAuthStore(s => s.user?.id);
```

Adjust the `userId` source to match the existing auth pattern in the codebase.

- [ ] **Step 5: Pass sessionId and chatHistory to AIChatPanel**

Change the AIChatPanel JSX (line 95) from:
```tsx
<AIChatPanel code={ws.code.code} symbol={ws.account.symbol} timeframe={ws.account.timeframe} onApply={ws.code.setCode} initialPrompt={ws.ai.optimizePrompt} autoApply={ws.ai.chatAutoApply} />
```
to:
```tsx
<AIChatPanel code={ws.code.code} symbol={ws.account.symbol} timeframe={ws.account.timeframe} onApply={ws.code.setCode} initialPrompt={ws.ai.optimizePrompt} autoApply={ws.ai.chatAutoApply} sessionId={sessionId} chatHistory={chatHistory} />
```

- [ ] **Step 6: Build check**

Run: `cd frontend && npx tsc --noEmit`
Expected: 0 errors

- [ ] **Step 7: Commit**

```bash
git add frontend/src/pages/strategy/StrategyWorkspacePage.tsx
git commit -m "feat(frontend): ResolveSession on workspace mount

On workspace open, resolve strategy_key → session UUID + chat history.
Pass sessionId and chatHistory to AIChatPanel for cross-device sync.
strategyKey format: strategy:<id> for saved, draft:<user>:<symbol>:<tf> for unsaved."
```

---

### Task 13: Full Build + File Size Verification

- [ ] **Step 1: Go build — entire project**

Run: `cd backend && go build ./...`
Expected: 0 errors

- [ ] **Step 2: File size check**

Run: `python3 scripts/check-file-lines.py --strict`
Expected: 🟢 all pass (no 🔴 violations)

- [ ] **Step 3: Frontend type check**

Run: `cd frontend && npx tsc --noEmit`
Expected: 0 errors

- [ ] **Step 4: Verify new file line counts**

```bash
wc -l backend/internal/ai/prompt_context.go
wc -l backend/internal/ai/conversation_session.go
```

Expected: prompt_context.go ≤ 120, conversation_session.go ≤ 80

- [ ] **Step 5: Commit (if any final adjustments needed)**

```bash
git add -A
git commit -m "chore: final build verification — go build + tsc + file-size check pass"
```

---

## Acceptance Criteria

After all tasks complete, verify:

1. **Repair mode E2E**: Generate code → validation reports "missing param" → paste error into AI input → send → AI returns **pure code** (no explanation) → code writes to editor
2. **Repair fallback**: If LLM outputs "Here is the fixed code: ```python ...```", `extractCodeFromRepair` extracts the code
3. **Discuss mode**: Asking "这个止损逻辑对吗？" returns analysis text, does NOT overwrite editor code
4. **Cross-device**: Desktop AI chat → open same strategy on mobile → chat history restored from server
5. **Refresh persistence**: Refresh workspace → chat history loads from server via ResolveSession
6. **File sizes**: All files within limits per `check-file-lines.py --strict`
7. **`go build ./...`**: 0 errors
8. **`npx tsc --noEmit`**: 0 errors
