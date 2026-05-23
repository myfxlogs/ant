package service

import "strings"

// Code-gen prompts embed large static rules plus every agent summary; cap
// upstream text server-side so users never need to "slim prompts" manually.
const (
	codeGenMaxIntentRunes = 12000
	codeGenMaxAgentRunes = 3500
	codeGenMaxAgentBlocks = 16
)

func shrinkCodeGenInputs(intent string, agents []UpstreamSummary, locale string) (string, []UpstreamSummary) {
	intent = strings.TrimSpace(intent)
	if intent != "" {
		intent = truncateRunesWithBudgetNote(intent, codeGenMaxIntentRunes, locale)
	}
	if len(agents) == 0 {
		return intent, agents
	}
	if len(agents) > codeGenMaxAgentBlocks {
		agents = agents[:codeGenMaxAgentBlocks]
	}
	out := make([]UpstreamSummary, 0, len(agents))
	for _, a := range agents {
		t := strings.TrimSpace(a.Text)
		t = truncateRunesWithBudgetNote(t, codeGenMaxAgentRunes, locale)
		out = append(out, UpstreamSummary{Name: a.Name, Text: t})
	}
	return intent, out
}

func truncateRunesWithBudgetNote(s string, max int, locale string) string {
	if max <= 0 || s == "" {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	note := budgetTruncationNote(locale)
	nr := []rune(note)
	keep := max - len(nr)
	if keep < 64 {
		keep = 64
	}
	if keep >= len(r) {
		return s
	}
	return string(r[:keep]) + note
}

func budgetTruncationNote(locale string) string {
	switch normalizeV2Locale(locale) {
	case "zh-cn":
		return " …[已由 AntTrader 服务端截断以控制提示词长度]… "
	case "zh-tw":
		return " …[已由 AntTrader 服務端截斷以控制提示詞長度]… "
	case "ja":
		return " …[AntTrader サーバー側でプロンプト長のため省略]… "
	case "vi":
		return " …[AntTrader đã cắt bớt phía máy chủ để giới hạn độ dài prompt]… "
	default:
		return " …[truncated server-side by AntTrader for prompt budget]… "
	}
}
