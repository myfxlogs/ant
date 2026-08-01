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

// parseTextToolCalls extracts [TOOL: name key="val" ...] patterns from text.
// Fallback for models (DeepSeek, GLM, Qwen) that don't support native function calling.
//
// Format: [TOOL: tool_name key="value with spaces" key2="v2"]
// Quoted values may contain spaces, newlines, and escaped quotes (\").
// Unquoted bare values are also accepted: [TOOL: read_kline EURUSD H1]
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

		if content == "" {
			continue
		}

		name, argsStr := splitToolName(content)
		tc := textToolCall{Name: name}

		args := parseToolArgs(argsStr)
		if len(args) == 0 {
			// Fallback: treat remaining words as positional (symbol, timeframe).
			positional := strings.Fields(argsStr)
			if len(positional) >= 1 {
				args["symbol"] = positional[0]
			}
			if len(positional) >= 2 {
				args["timeframe"] = positional[1]
			}
		}

		jsonBytes, _ := json.Marshal(args)
		tc.ArgsJSON = string(jsonBytes)
		calls = append(calls, tc)
	}
	return calls
}

// splitToolName returns the tool name and the remaining argument string.
func splitToolName(content string) (name, args string) {
	// Tool name is the first word before any space.
	idx := strings.IndexAny(content, " \t")
	if idx < 0 {
		return content, ""
	}
	return content[:idx], strings.TrimSpace(content[idx+1:])
}

// parseToolArgs parses key="value" pairs with quote-aware scanning.
// Values in double quotes may contain spaces, newlines, and escaped quotes.
func parseToolArgs(argsStr string) map[string]string {
	args := make(map[string]string)
	if argsStr == "" {
		return args
	}

	i := 0
	n := len(argsStr)
	for i < n {
		// Skip whitespace.
		for i < n && (argsStr[i] == ' ' || argsStr[i] == '\t') {
			i++
		}
		if i >= n {
			break
		}

		// Read key until '='.
		eq := strings.IndexByte(argsStr[i:], '=')
		if eq < 0 {
			// No '=' means this is not a key=val pair — stop parsing.
			break
		}
		key := strings.TrimSpace(argsStr[i : i+eq])
		i += eq + 1 // skip '='

		if i >= n {
			args[key] = ""
			break
		}

		if argsStr[i] == '"' {
			// Quoted value — scan until closing unescaped quote.
			i++ // skip opening quote
			var buf strings.Builder
			for i < n {
				if argsStr[i] == '\\' && i+1 < n {
					// Escape sequence.
					i++
					switch argsStr[i] {
					case '"':
						buf.WriteByte('"')
					case 'n':
						buf.WriteByte('\n')
					case '\\':
						buf.WriteByte('\\')
					default:
						buf.WriteByte('\\')
						buf.WriteByte(argsStr[i])
					}
					i++
				} else if argsStr[i] == '"' {
					i++ // skip closing quote
					break
				} else {
					buf.WriteByte(argsStr[i])
					i++
				}
			}
			args[key] = buf.String()
		} else {
			// Unquoted value — read until next space.
			end := strings.IndexAny(argsStr[i:], " \t")
			if end < 0 {
				args[key] = argsStr[i:]
				break
			}
			args[key] = argsStr[i : i+end]
			i += end
		}
	}
	return args
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
