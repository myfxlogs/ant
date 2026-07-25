package ai

import (
	"encoding/json"
	"strings"

	systemai "alphaforge/internal/service/systemai"
)

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
	in.RawArgs = args
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
	if v, ok := args["key"].(string); ok {
		in.Key = v
	}
	if v, ok := args["name"].(string); ok {
		if in.Key == "" {
			in.Key = v
		}
	}
	if v, ok := args["value"].(string); ok {
		in.Value = v
	}
	return in
}

// parseJSONToMap parses a JSON string into a map[string]any.
// Used to convert LLM JSON output to structpb.Struct for proto fields.
func parseJSONToMap(s string) map[string]any {
	if s == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	return m
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

		args := make(map[string]string)
		for _, p := range parts[1:] {
			if kv := strings.SplitN(p, "=", 2); len(kv) == 2 {
				args[kv[0]] = kv[1]
			}
		}
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

// hasWriteStrategyCall checks whether the LLM invoked write_strategy,
// either via native tool calls or text-based [TOOL: write_strategy ...].
func hasWriteStrategyCall(nativeCalls []systemai.ToolCall, roundText string) bool {
	for _, tc := range nativeCalls {
		if tc.Function.Name == "write_strategy" {
			return true
		}
	}
	textCalls := parseTextToolCalls(roundText)
	for _, tc := range textCalls {
		if tc.Name == "write_strategy" {
			return true
		}
	}
	return false
}
