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
// Think→Act→Observe→Repeat pattern using the LLM's native tool_use protocol
// (OpenAI function calling). The LLM receives tool definitions as JSON Schema,
// the LLM decides when to call tools, and we execute them and inject results
// as structured tool-result messages.

// AgentLoop orchestrates the LLM ↔ Tools conversation loop.
type AgentLoop struct {
	toolRegistry  *ToolRegistry
	llmStream     func(ctx context.Context, messages []systemai.ChatMessage, tools []systemai.ToolDefinition, onChunk func(systemai.ChatStreamChunk) error) error
	streamChunk   func(delta string) error                                 // forward delta to frontend
	toolStream    func(tc *antv1.ToolCall, tr *antv1.ToolResult) error     // forward tool events to frontend
	maxRounds     int
	currentCode   string // workspace code injected into ToolInput.Code
	toolDefs      []systemai.ToolDefinition // cached tool schemas built from registry
}

// NewAgentLoop creates an AgentLoop with the given tools and LLM streaming function.
// llmStream should be a pre-bound closure that already includes userID.
func NewAgentLoop(
	registry *ToolRegistry,
	llmStream func(ctx context.Context, messages []systemai.ChatMessage, tools []systemai.ToolDefinition, onChunk func(systemai.ChatStreamChunk) error) error,
	streamChunk func(delta string) error,
	toolStream func(tc *antv1.ToolCall, tr *antv1.ToolResult) error,
) *AgentLoop {
	return &AgentLoop{
		toolRegistry: registry,
		llmStream:    llmStream,
		streamChunk:  streamChunk,
		toolStream:   toolStream,
		maxRounds:    10,
		toolDefs:     registry.BuildToolSchemas(),
	}
}

// SetCurrentCode injects the workspace strategy code into tool inputs.
func (a *AgentLoop) SetCurrentCode(code string) {
	a.currentCode = code
}

// RunWithHistory executes the agent loop with pre-loaded conversation history.
func (a *AgentLoop) RunWithHistory(ctx context.Context, systemPrompt, userPrompt string, history []systemai.ChatMessage, userID uuid.UUID) (string, error) {
	messages := make([]systemai.ChatMessage, 0, 2+len(history))
	messages = append(messages, systemai.ChatMessage{Role: "system", Content: systemPrompt})
	for _, h := range history {
		messages = append(messages, h)
	}
	messages = append(messages, systemai.ChatMessage{Role: "user", Content: userPrompt})
	return a.run(ctx, messages, userID)
}

// Run executes the agent loop without history.
func (a *AgentLoop) Run(ctx context.Context, systemPrompt, userPrompt string, userID uuid.UUID) (string, error) {
	return a.RunWithHistory(ctx, systemPrompt, userPrompt, nil, userID)
}

func (a *AgentLoop) run(ctx context.Context, messages []systemai.ChatMessage, userID uuid.UUID) (string, error) {
	var fullBuf strings.Builder

	for round := 0; round < a.maxRounds; round++ {
		var roundBuf strings.Builder
		var toolCalls []systemai.ToolCall // accumulated from stream chunks

		err := a.llmStream(ctx, messages, a.toolDefs, func(chunk systemai.ChatStreamChunk) error {
			roundBuf.WriteString(chunk.Content)
			// Collect any tool calls from the final chunk.
			if len(chunk.ToolCalls) > 0 {
				for _, stc := range chunk.ToolCalls {
					toolCalls = append(toolCalls, systemai.ToolCall{
						ID:   stc.ID,
						Type: stc.Type,
						Function: systemai.ToolCallFunction{
							Name:      stc.Function.Name,
							Arguments: stc.Function.Arguments,
						},
					})
				}
			}
			return nil
		})
		if err != nil {
			return "", err
		}

		roundText := strings.TrimSpace(roundBuf.String())
		if roundText == "" && len(toolCalls) == 0 {
			return "", fmt.Errorf("agent: LLM returned empty response on round %d", round+1)
		}

		// Stream text content to frontend (non-tool-call text).
		if roundText != "" && a.streamChunk != nil {
			_ = a.streamChunk(roundText)
		}
		fullBuf.WriteString(roundText)

		// Update currentCode if this round contains Python code — so subsequent
		// tool calls (save_strategy, compile_python) see the latest code.
		if code := ExtractCode(roundText); code != "" {
			a.currentCode = code
		}

		// No native tool calls → check for text-based [TOOL: name args] fallback.
		if len(toolCalls) == 0 {
			textCalls := parseTextToolCalls(roundText)
			if len(textCalls) == 0 {
				return fullBuf.String(), nil
			}
			// Convert text-based calls to structured tool calls.
			for _, tc := range textCalls {
				toolCalls = append(toolCalls, systemai.ToolCall{
					ID:   "call_" + tc.Name,
					Type: "function",
					Function: systemai.ToolCallFunction{
						Name:      tc.Name,
						Arguments: tc.ArgsJSON,
					},
				})
			}
		}

		// Build the assistant message with tool calls.
		assistantMsg := systemai.ChatMessage{
			Role:      "assistant",
			Content:   roundText,
			ToolCalls: toolCalls,
		}
		messages = append(messages, assistantMsg)

		// Execute each tool call and collect results.
		for _, tc := range toolCalls {
			callID := tc.ID
			if callID == "" {
				callID = "call_" + tc.Function.Name
			}

			input := parseToolArguments(tc.Function.Name, tc.Function.Arguments)
			input.UserID = userID
			if input.Code == "" && a.currentCode != "" {
				input.Code = a.currentCode
			}

			tool := a.toolRegistry.FindPreTool(tc.Function.Name)
			var result ToolOutput
			if tool == nil {
				result = ToolOutput{Success: false, Error: fmt.Sprintf("unknown tool: %s", tc.Function.Name)}
			} else {
				result = tool.Run(ctx, input)
			}

			// Build tool result message (OpenAI protocol: role="tool").
			resultJSON, _ := json.Marshal(result.Output)
			resultContent := string(resultJSON)
			if !result.Success {
				resultContent = fmt.Sprintf(`{"error": %q}`, result.Error)
			}
			messages = append(messages, systemai.ChatMessage{
				Role:       "tool",
				ToolCallID: callID,
				Name:       tc.Function.Name,
				Content:    resultContent,
			})

			// Stream tool event to frontend.
			if a.toolStream != nil {
				_ = a.toolStream(
					&antv1.ToolCall{CallId: callID, Name: tc.Function.Name, ParamsJson: tc.Function.Arguments},
					&antv1.ToolResult{CallId: callID, Name: tc.Function.Name, Success: result.Success, OutputJson: resultContent, Error: result.Error},
				)
			}
		}
		// Loop continues — LLM sees tool results and decides next action.
	}

	return fullBuf.String(), fmt.Errorf("agent loop: max rounds (%d) exceeded", a.maxRounds)
}

// parseToolArguments parses the JSON arguments string from an LLM tool call
// and maps known fields into a ToolInput struct.
// Standard fields (symbol, timeframe, code) are mapped directly.
// Legacy memory tools use key/value which map to Symbol/Timeframe — these
// never overlap with standard fields in practice, but standard fields win on conflict.
func parseToolArguments(toolName, argsJSON string) ToolInput {
	in := ToolInput{}
	if argsJSON == "" {
		return in
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return in
	}
	if v, ok := args["symbol"].(string); ok {
		in.Symbol = v
	}
	if v, ok := args["timeframe"].(string); ok {
		in.Timeframe = v
	}
	if v, ok := args["code"].(string); ok {
		in.Code = v
	}
	// Memory tool fields (remember, recall, save_strategy, load_strategy)
	// map key/name → Symbol and value → Timeframe. Only applied when the
	// standard field wasn't already set (standard fields win on conflict).
	if in.Symbol == "" {
		if v, ok := args["key"].(string); ok {
			in.Symbol = v
		}
		if v, ok := args["name"].(string); ok {
			in.Symbol = v
		}
	}
	if in.Timeframe == "" {
		if v, ok := args["value"].(string); ok {
			in.Timeframe = v
		}
	}
	return in
}

// ── Text-based tool call fallback (for models without native tool_use) ──

type textToolCall struct {
	Name     string
	ArgsJSON string
}

// parseTextToolCalls extracts [TOOL: name key=val ...] patterns from text.
// Fallback for models (DeepSeek, GLM, Qwen) that don't support native function calling.
func parseTextToolCalls(text string) []textToolCall {
	var calls []textToolCall
	rest := text
	for {
		start := strings.Index(rest, "[TOOL:")
		if start < 0 {
			break
		}
		rest = rest[start+6:]
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
		tc := textToolCall{Name: parts[0]}

		// Build JSON args from positional or key=value arguments.
		args := make(map[string]string)
		for _, p := range parts[1:] {
			if kv := strings.SplitN(p, "=", 2); len(kv) == 2 {
				args[kv[0]] = kv[1]
			}
		}
		// If no key=value pairs, use positional: arg1=symbol, arg2=timeframe
		if len(args) == 0 && len(parts) >= 2 {
			args["symbol"] = parts[1]
		}
		if len(args) == 0 && len(parts) >= 3 {
			args["timeframe"] = parts[2]
		}

		jsonBytes, _ := json.Marshal(args)
		tc.ArgsJSON = string(jsonBytes)
		calls = append(calls, tc)
	}
	return calls
}
