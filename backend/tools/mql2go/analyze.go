package mql2go

import (
	"strings"
)

// Analyze parses MQL source and extracts a StrategyIntent.
// This is the Go-native entry point for the full pipeline.
func Analyze(source string) (*StrategyIntent, error) {
	// TODO: tree-sitter CST → AST → recognizers → intent
	// For now, provide a minimal implementation that extracts extern params
	// and detects basic entry patterns.

	intent := &StrategyIntent{
		Meta: StrategyMeta{
			MQLVersion: detectMQLVersion(source),
		},
		Execution: ExecutionModel{
			Kind: ExecOnBar,
		},
	}

	// Extract extern/input params
	intent.Params = extractParams(source)

	// Detect basic entries
	intent.Entry = detectEntries(source)

	// Detect basic exits
	intent.Exit = detectExits(source)

	return intent, nil
}

func detectMQLVersion(source string) string {
	if strings.Contains(source, "class ") {
		return "mql5"
	}
	return "mql4"
}

func extractParams(source string) []ParamSpec {
	var params []ParamSpec
	lines := strings.Split(source, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Match: extern type name = value; or input type name = value;
		if !strings.HasPrefix(line, "extern ") && !strings.HasPrefix(line, "input ") {
			continue
		}
		// Skip noise params (Chinese display text)
		if strings.Contains(line, "说明") || strings.Contains(line, "选择") ||
			strings.Contains(line, "提示") {
			continue
		}
		// Simple extraction: extern type name = value;
		fields := strings.Fields(line) // ["extern", "double", "Name", "=", "0.10;"]
		if len(fields) < 4 {
			continue
		}
		name := fields[2]
		// Skip section headers
		if strings.Contains(line, "====") {
			continue
		}
		// Skip long string values (>20 chars with non-ASCII)
		valIdx := 4
		if fields[3] != "=" {
			valIdx = 3
		}
		if valIdx < len(fields) {
			val := strings.TrimRight(fields[valIdx], ";")
			if len(val) > 20 && containsNonASCII(val) {
				continue
			}
		}
		pt := paramType(fields[1])
		params = append(params, ParamSpec{
			Name:  name,
			Label: name,
			Type:  pt,
			Group: guessGroup(name),
		})
	}
	return params
}

func detectEntries(source string) []EntryRule {
	var entries []EntryRule
	lines := strings.Split(source, "\n")
	inOnTick := false
	braceLevel := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "void OnTick()") || strings.Contains(trimmed, "OnTick()") {
			inOnTick = true
			continue
		}
		if !inOnTick {
			continue
		}
		braceLevel += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
		if braceLevel <= 0 && inOnTick && trimmed != "" {
			inOnTick = false
			continue
		}

		// Detect: OrderSend(Symbol(), OP_BUY, ...)
		if strings.Contains(trimmed, "OrderSend(") && strings.Contains(trimmed, "OP_") {
			entry := parseOrderSend(trimmed)
			if entry.Action != "" {
				entries = append(entries, entry)
			}
		}
	}
	return entries
}

func detectExits(source string) []ExitRule {
	var exits []ExitRule
	// Detect OrderClose in for-loops
	if strings.Contains(source, "OrderClose(") && strings.Contains(source, "OrderMagicNumber()") {
		exits = append(exits, ExitRule{
			Trigger:  TriggerMagic,
			Action:   "position_close",
			MagicVal: "s.magic",
		})
	}
	// Detect OrderDelete in for-loops
	if strings.Contains(source, "OrderDelete(") {
		exits = append(exits, ExitRule{
			Trigger:  TriggerDelete,
			Action:   "order_delete",
			MagicVal: "s.magic",
		})
	}
	return exits
}

// ── Helpers ──────────────────────────────────────────────────────

func paramType(mqlType string) ParamType {
	switch mqlType {
	case "int", "long", "uint", "ulong":
		return ParamInt
	case "double", "float":
		return ParamDouble
	case "string":
		return ParamString
	case "bool":
		return ParamBool
	}
	return ParamString
}

func guessGroup(name string) ParamGroup {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "lot") || strings.Contains(lower, "volume") {
		return GroupSizing
	}
	if strings.Contains(lower, "magic") || strings.Contains(lower, "comment") {
		return GroupSystem
	}
	if strings.Contains(lower, "sl") || strings.Contains(lower, "tp") ||
		strings.Contains(lower, "stop") || strings.Contains(lower, "take") {
		return GroupExit
	}
	if strings.Contains(lower, "risk") || strings.Contains(lower, "margin") {
		return GroupRisk
	}
	return GroupEntry
}

func containsNonASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}

func parseOrderSend(line string) EntryRule {
	// Simple extraction: find OP_xxx and determine direction
	entry := EntryRule{}
	if strings.Contains(line, "OP_BUYSTOP") || strings.Contains(line, "OP_SELLSTOP") {
		if strings.Contains(line, "OP_BUYSTOP") {
			entry.Action = ActionBuyStop
		} else {
			entry.Action = ActionSellStop
		}
	} else if strings.Contains(line, "OP_BUYLIMIT") || strings.Contains(line, "OP_SELLLIMIT") {
		if strings.Contains(line, "OP_BUYLIMIT") {
			entry.Action = ActionBuyLimit
		} else {
			entry.Action = ActionSellLimit
		}
	} else if strings.Contains(line, "OP_BUY") {
		entry.Action = ActionMarketBuy
	} else if strings.Contains(line, "OP_SELL") {
		entry.Action = ActionMarketSell
	}
	// Extract volume from OrderSend args
	// OrderSend(symbol, cmd, volume, price, ...)
	parts := strings.SplitN(line, "OrderSend(", 2)
	if len(parts) == 2 {
		args := strings.Split(parts[1], ",")
		if len(args) > 2 {
			entry.Volume = strings.TrimSpace(args[2])
		}
		if len(args) > 3 {
			entry.Price = strings.TrimSpace(args[3])
		}
		if len(args) > 7 {
			entry.Comment = strings.TrimSpace(args[7])
		}
		if len(args) > 8 {
			entry.Magic = strings.TrimSpace(args[8])
		}
	}
	return entry
}
