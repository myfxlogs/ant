package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	systemai "anttrader/internal/service/systemai"
)

// ── Agent Loop ──
// Formalized Think→Act→Observe→Repeat pattern (Claude Code / OpenAI Agents SDK style).
// The LLM can request tools via [TOOL: name args] markup and see results as [RESULT: name] json.
// The loop continues until the LLM stops requesting tools or maxRounds is reached.

// AgentLoop orchestrates the LLM ↔ Tools conversation loop.
type AgentLoop struct {
	toolRegistry *ToolRegistry
	llmStream    func(ctx context.Context, messages []systemai.ChatMessage, onChunk func(systemai.ChatStreamChunk) error) error
	streamChunk  func(delta string) error                           // forward delta to frontend
	toolStream   func(tc *antv1.ToolCall, tr *antv1.ToolResult) error // forward tool events to frontend
	maxRounds    int
	currentCode  string // workspace code injected into ToolInput.Code for tools like analyze_strategy
}

// NewAgentLoop creates an AgentLoop with the given tools and LLM streaming function.
// llmStream should be a pre-bound closure that already includes userID.
func NewAgentLoop(
	registry *ToolRegistry,
	llmStream func(ctx context.Context, messages []systemai.ChatMessage, onChunk func(systemai.ChatStreamChunk) error) error,
	streamChunk func(delta string) error,
	toolStream func(tc *antv1.ToolCall, tr *antv1.ToolResult) error,
) *AgentLoop {
	return &AgentLoop{
		toolRegistry: registry,
		llmStream:    llmStream,
		streamChunk:  streamChunk,
		toolStream:   toolStream,
		maxRounds:    5,
	}
}

// SetCurrentCode injects the workspace strategy code into tool inputs.
// Tools like analyze_strategy use this to access the code being discussed.
func (a *AgentLoop) SetCurrentCode(code string) {
	a.currentCode = code
}

// RunWithHistory executes the agent loop with pre-loaded conversation history.
// messages are initialized as: system prompt + history + user prompt.
func (a *AgentLoop) RunWithHistory(ctx context.Context, systemPrompt, userPrompt string, history []systemai.ChatMessage, userID uuid.UUID) (string, error) {
	messages := make([]systemai.ChatMessage, 0, 2+len(history))
	messages = append(messages, systemai.ChatMessage{Role: "system", Content: systemPrompt})
	for _, h := range history {
		messages = append(messages, h)
	}
	messages = append(messages, systemai.ChatMessage{Role: "user", Content: userPrompt})
	return a.run(ctx, messages, userID)
}

// Run executes the agent loop: systemPrompt + userPrompt → LLM → [TOOL:] → execute → RESULT → repeat.
func (a *AgentLoop) Run(ctx context.Context, systemPrompt, userPrompt string, userID uuid.UUID) (string, error) {
	return a.RunWithHistory(ctx, systemPrompt, userPrompt, nil, userID)
}

func (a *AgentLoop) run(ctx context.Context, messages []systemai.ChatMessage, userID uuid.UUID) (string, error) {
	var fullBuf strings.Builder

	for round := 0; round < a.maxRounds; round++ {
		var roundBuf strings.Builder
		err := a.llmStream(ctx, messages, func(chunk systemai.ChatStreamChunk) error {
			roundBuf.WriteString(chunk.Content)
			return nil
		})
		if err != nil {
			return "", err
		}

		roundText := roundBuf.String()

		// Parse tool calls and truncate at the first [TOOL: marker.
		// Everything after [TOOL: is speculative — the LLM hallucinated
		// a tool result before the tool actually ran. Truncate so the AI
		// doesn't see its own hallucination in the next round, and the
		// frontend doesn't display contradictory text.
		calls := parseToolCalls(roundText)
		hadPostToolText := false
		if len(calls) > 0 {
			if idx := strings.Index(roundText, "[TOOL:") ; idx >= 0 {
				postTool := strings.TrimSpace(roundText[idx:])
				// If there's substantial text beyond just the tool call syntax,
				// the LLM speculated about tool results — flag for correction.
				if len(postTool) > len("[TOOL: x y]")+20 {
					hadPostToolText = true
				}
				roundText = strings.TrimSpace(roundText[:idx])
			}
		}

		// Stream the sanitized (non-hallucinated) text to frontend.
		if roundText != "" && a.streamChunk != nil {
			_ = a.streamChunk(roundText)
		}
		fullBuf.WriteString(roundText)

		if len(calls) == 0 {
			return fullBuf.String(), nil
		}

		// Execute each tool call
		var toolResults []string
		for _, call := range calls {
			call.Input.UserID = userID // wire authenticated user for DB tools (remember, save, etc.)
		if call.Input.Code == "" && a.currentCode != "" {
			call.Input.Code = a.currentCode
		}
			tool := a.toolRegistry.FindPreTool(call.Name)
			var result ToolOutput
			if tool == nil {
				result = ToolOutput{Success: false, Error: fmt.Sprintf("unknown tool: %s", call.Name)}
			} else {
				result = tool.Run(ctx, call.Input)
			}

			resultText := formatToolResult(call.Name, result)
			toolResults = append(toolResults, resultText)
			fullBuf.WriteString("\n" + resultText + "\n")

			// Stream tool event to frontend
			if a.toolStream != nil {
				outJSON, _ := json.Marshal(result.Output)
				_ = a.toolStream(
					&antv1.ToolCall{CallId: "call_" + call.Name, Name: call.Name, ParamsJson: "{}"},
					&antv1.ToolResult{CallId: "call_" + call.Name, Name: call.Name, Success: result.Success, OutputJson: string(outJSON), Error: result.Error},
				)
			}
		}

		// Feed truncated assistant response + tool results to next round.
		// The AI never sees its hallucinated text — only the real tool results.
		sysMsg := strings.Join(toolResults, "\n")
		if hadPostToolText {
			// The LLM speculated about tool results before they ran.
			// Inject a strong correction reminder so the AI reconciles.
			sysMsg += "\n[SYSTEM] 以上是工具返回的真实数据。如果你在调用工具后曾猜测过结果，那些猜测是错的——忽略它们，以这里的数据为准。如果真实数据与你之前说的不一致，主动承认并更正。"
		}
		messages = append(messages,
			systemai.ChatMessage{Role: "assistant", Content: roundText},
			systemai.ChatMessage{Role: "system", Content: sysMsg},
		)
	}

	return fullBuf.String(), fmt.Errorf("agent loop: max rounds (%d) exceeded", a.maxRounds)
}

// ── Tool call parsing ──

type toolCall struct {
	Name  string
	Input ToolInput
}

// parseToolCalls extracts all [TOOL: name args] from text.
func parseToolCalls(text string) []toolCall {
	var calls []toolCall
	rest := text
	for {
		start := strings.Index(rest, "[TOOL:")
		if start < 0 {
			break
		}
		rest = rest[start+6:] // after "[TOOL:"
		end := strings.Index(rest, "]")
		if end < 0 {
			break
		}
		content := strings.TrimSpace(rest[:end])
		rest = rest[end+1:]

		parts := strings.Fields(content)
		if len(parts) == 0 {
			continue
		}
		tc := toolCall{Name: parts[0]}
		if len(parts) > 1 {
			tc.Input = ToolInput{Symbol: parts[1]}
		}
		if len(parts) > 2 {
			tc.Input.Timeframe = parts[2]
		}
		calls = append(calls, tc)
	}
	return calls
}

// formatToolResult formats a tool output as a system-visible result.
func formatToolResult(name string, out ToolOutput) string {
	if out.Success {
		jsonBytes, err := json.Marshal(out.Output)
		if err != nil {
			return fmt.Sprintf("[RESULT: %s] %v", name, out.Output)
		}
		return fmt.Sprintf("[RESULT: %s] %s", name, string(jsonBytes))
	}
	return fmt.Sprintf("[RESULT: %s] error: %s", name, out.Error)
}
