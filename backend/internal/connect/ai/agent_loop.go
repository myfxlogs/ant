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
		maxRounds:    5,
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

		// Stream text content to frontend (non-tool-call text).
		if roundText != "" && a.streamChunk != nil {
			_ = a.streamChunk(roundText)
		}
		fullBuf.WriteString(roundText)

		// No tool calls → LLM is done.
		if len(toolCalls) == 0 {
			return fullBuf.String(), nil
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
	// For tools like remember/recall that use "key"/"value" fields.
	if v, ok := args["key"].(string); ok {
		in.Symbol = v // reuse Symbol field for key
	}
	if v, ok := args["value"].(string); ok {
		in.Timeframe = v // reuse Timeframe field for value
	}
	// For save_strategy: "name" field → Symbol
	if v, ok := args["name"].(string); ok {
		in.Symbol = v
	}
	return in
}
