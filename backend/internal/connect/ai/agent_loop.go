package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"

	antv1 "alphaforge/gen/proto/ant/v1"
	systemai "alphaforge/internal/service/systemai"
)

const maxAgentRounds = 1000

// ── Agent Loop ──
// Think→Act→Observe→Repeat pattern using the LLM's native tool_use protocol
// (OpenAI function calling). The LLM receives tool definitions as JSON Schema,
// the LLM decides when to call tools, and we execute them and inject results
// as structured tool-result messages.

// AgentLoop orchestrates the LLM ↔ Tools conversation loop.
type AgentLoop struct {
	toolRegistry    *ToolRegistry
	llmStream       func(ctx context.Context, messages []systemai.ChatMessage, tools []systemai.ToolDefinition, onChunk func(systemai.ChatStreamChunk) error) error
	streamChunk     func(delta string) error                                 // forward content delta to frontend
	reasoningStream func(delta string) error                                 // forward reasoning delta to frontend
	toolStream      func(tc *antv1.ToolCall, tr *antv1.ToolResult) error     // forward tool events to frontend
	currentCode     string // workspace code injected into ToolInput.Code
	toolDefs        []systemai.ToolDefinition // cached tool schemas built from registry
	timeBudget      time.Duration // max wall-clock time for the loop
}

// NewAgentLoop creates an AgentLoop with the given tools and LLM streaming function.
// llmStream should be a pre-bound closure that already includes userID.
func NewAgentLoop(
	registry *ToolRegistry,
	llmStream func(ctx context.Context, messages []systemai.ChatMessage, tools []systemai.ToolDefinition, onChunk func(systemai.ChatStreamChunk) error) error,
	streamChunk func(delta string) error,
	toolStream func(tc *antv1.ToolCall, tr *antv1.ToolResult) error,
	reasoningStream func(delta string) error,
) *AgentLoop {
	return &AgentLoop{
		toolRegistry: registry,
		llmStream:    llmStream,
		streamChunk:     streamChunk,
		toolStream:      toolStream,
		reasoningStream: reasoningStream,
		toolDefs:     registry.BuildToolSchemas(),
		timeBudget:   10 * time.Minute,
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
	messages = append(messages, history...)
	messages = append(messages, systemai.ChatMessage{Role: "user", Content: userPrompt})
	return a.run(ctx, messages, userID)
}

// Run executes the agent loop without history.
func (a *AgentLoop) Run(ctx context.Context, systemPrompt, userPrompt string, userID uuid.UUID) (string, error) {
	return a.RunWithHistory(ctx, systemPrompt, userPrompt, nil, userID)
}

func (a *AgentLoop) run(ctx context.Context, messages []systemai.ChatMessage, userID uuid.UUID) (string, error) {
	var fullBuf strings.Builder
	deadline := time.Now().Add(a.timeBudget)
	codeConvergences := 0

	for round := 0; round < maxAgentRounds; round++ {
		if time.Now().After(deadline) {
			return fullBuf.String(), fmt.Errorf("agent: total time budget exceeded (%v)", a.timeBudget)
		}
		messages = compressContext(messages)

		var roundBuf strings.Builder
		var reasoningBuf strings.Builder
		var roundFinishReason string
		var toolCalls []systemai.ToolCall

		err := a.llmStream(ctx, messages, a.toolDefs, a.makeOnChunk(&roundBuf, &reasoningBuf, &roundFinishReason, &toolCalls))
		if err != nil {
			return "", err
		}

		reasoningText := strings.TrimSpace(reasoningBuf.String())
		messages, err = a.handleConvergenceRetry(ctx, messages, &roundBuf, &reasoningBuf, &roundFinishReason, &toolCalls, reasoningText)
		if err != nil {
			return "", err
		}

		roundText := strings.TrimSpace(roundBuf.String())
		if roundText == "" && len(toolCalls) == 0 {
			return "", fmt.Errorf("agent: LLM returned empty response on round %d", round+1)
		}

		fullBuf.WriteString(roundText)

		code := ExtractCode(roundText)
		if code != "" {
			a.currentCode = code
		}

		if code != "" && !hasWriteStrategyCall(toolCalls, roundText) && codeConvergences < 2 {
			codeConvergences++
			messages = append(messages, systemai.ChatMessage{
				Role:    "user",
				Content: "I notice you included code directly in your response instead of using the submission tool.\n\nPlease call the appropriate tool to submit your strategy code. The code in your chat text was extracted, but for proper compilation and backtesting, you must use the available tools.\n\nIf you're unsure which tool to use, review the tool descriptions in the system prompt. Do not paste code in text again.",
			})
			continue
		}

		if len(toolCalls) == 0 {
			textCalls := parseTextToolCalls(roundText)
			if len(textCalls) == 0 {
				return fullBuf.String(), nil
			}
			for _, tc := range textCalls {
				toolCalls = append(toolCalls, systemai.ToolCall{
					ID:   "call_" + tc.Name,
					Type: toolTypeFunction,
					Function: systemai.ToolCallFunction{
						Name:      tc.Name,
						Arguments: tc.ArgsJSON,
					},
				})
			}
		}

		assistantMsg := systemai.ChatMessage{
			Role:      "assistant",
			Content:   roundText,
			ToolCalls: toolCalls,
		}
		messages = append(messages, assistantMsg)

		messages = a.executeToolCalls(ctx, messages, toolCalls, userID)
	}
	return fullBuf.String(), fmt.Errorf("agent: round budget exhausted (%d rounds)", maxAgentRounds)
}

func compressContext(messages []systemai.ChatMessage) []systemai.ChatMessage {
	if len(messages) <= 24 {
		return messages
	}
	totalChars := 0
	for _, m := range messages {
		totalChars += len(m.Content)
	}
	if totalChars/4 <= 8000 {
		return messages
	}
	keep := make([]systemai.ChatMessage, 0, 22)
	keep = append(keep, messages[0])
	keep = append(keep, messages[1])
	if len(messages) > 20 {
		keep = append(keep, messages[len(messages)-20:]...)
	}
	return keep
}

func (a *AgentLoop) handleConvergenceRetry(
	ctx context.Context,
	messages []systemai.ChatMessage,
	roundBuf, reasoningBuf *strings.Builder,
	roundFinishReason *string,
	toolCalls *[]systemai.ToolCall,
	reasoningText string,
) ([]systemai.ChatMessage, error) {
	lengthConvergences := 0
	for (*roundFinishReason == "length" || (roundBuf.Len() == 0 && len(reasoningText) > 500 && len(*toolCalls) == 0)) && lengthConvergences < 2 {
		lengthConvergences++
		messages = append(messages, systemai.ChatMessage{Role: "user", Content: "You've spent enough time analyzing. Now is the time to produce the deliverable.\n\nOutput the complete strategy code. For any parameters you're uncertain about:\n- Use industry-standard defaults (e.g. RSI period=14, MA period=20, ATR multiplier=2)\n- Document your choices in brief comments\n- Focus on correctness over optimization\n\nDo not continue analyzing. Produce the code now."})
		roundBuf.Reset()
		reasoningBuf.Reset()
		*toolCalls = nil
		*roundFinishReason = ""
		if err := a.llmStream(ctx, messages, a.toolDefs, a.makeOnChunk(roundBuf, reasoningBuf, roundFinishReason, toolCalls)); err != nil {
			return nil, err
		}
	}
	return messages, nil
}

func (a *AgentLoop) executeToolCalls(ctx context.Context, messages []systemai.ChatMessage, toolCalls []systemai.ToolCall, userID uuid.UUID) []systemai.ChatMessage {
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

		var resultContent string
		if !result.Success {
			resultContent = fmt.Sprintf(`{"error": %q}`, result.Error)
		} else if result.Output != nil {
			if s, ok := result.Output.(string); ok {
				resultContent = fmt.Sprintf(`{"output": %q}`, s)
			} else {
				resultJSON, _ := json.Marshal(result.Output)
				resultContent = string(resultJSON)
			}
		}
		messages = append(messages, systemai.ChatMessage{
			Role:       "tool",
			ToolCallID: callID,
			Name:       tc.Function.Name,
			Content:    resultContent,
		})

		if a.toolStream != nil {
			var paramsStruct *structpb.Struct
			if s, err := structpb.NewStruct(parseJSONToMap(tc.Function.Arguments)); err == nil {
				paramsStruct = s
			}
			var outputStruct *structpb.Struct
			if s, err := structpb.NewStruct(parseJSONToMap(resultContent)); err == nil {
				outputStruct = s
			}
			_ = a.toolStream(
				&antv1.ToolCall{CallId: callID, Name: tc.Function.Name, Params: paramsStruct},
				&antv1.ToolResult{CallId: callID, Name: tc.Function.Name, Success: result.Success, Output: outputStruct, Error: result.Error},
			)
		}
	}
	return messages
}

// makeOnChunk creates a streaming callback that accumulates content, reasoning,
// finish reason, and tool calls into the provided pointers.
func (a *AgentLoop) makeOnChunk(roundBuf, reasoningBuf *strings.Builder, finishReason *string, toolCalls *[]systemai.ToolCall) func(systemai.ChatStreamChunk) error {
	return func(chunk systemai.ChatStreamChunk) error {
		if chunk.Content != "" {
			roundBuf.WriteString(chunk.Content)
			if a.streamChunk != nil { _ = a.streamChunk(chunk.Content) }
		}
		if chunk.Reasoning != "" {
			reasoningBuf.WriteString(chunk.Reasoning)
			if a.reasoningStream != nil { _ = a.reasoningStream(chunk.Reasoning) }
		}
		if chunk.FinishReason != "" {
			*finishReason = chunk.FinishReason
		}
		if len(chunk.ToolCalls) > 0 {
			for _, stc := range chunk.ToolCalls {
				*toolCalls = append(*toolCalls, systemai.ToolCall{
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
	}
}
