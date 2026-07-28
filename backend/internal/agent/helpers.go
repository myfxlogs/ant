package agent

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/tools/mql2go"
)

func parseDecimalDefault(s, defaultVal string) decimal.Decimal {
	if s == "" {
		s = defaultVal
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		d, _ = decimal.NewFromString(defaultVal)
	}
	return d
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// stripMarkdownFences removes ```python ... ``` or ``` ... ``` fences if present.
func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		idx := strings.Index(s, "\n")
		if idx < 0 {
			return s
		}
		s = s[idx+1:]
	}
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

// stripThinkBlocks removes [THINK]...[/THINK] blocks from LLM output.
// These are reasoning traces that should never appear in extracted code.
func stripThinkBlocks(s string) string {
	for {
		start := strings.Index(s, "[THINK]")
		if start < 0 {
			break
		}
		end := strings.Index(s[start:], "[/THINK]")
		if end < 0 {
			s = s[:start]
			break
		}
		s = s[:start] + s[start+end+len("[/THINK]"):]
	}
	return strings.TrimSpace(s)
}

func buildBridgeChanges(orig, bridged *mql2go.CoverageResult) []*antv1.SemanticChange {
	var changes []*antv1.SemanticChange

	stillPresent := make(map[string]bool)
	for _, bs := range bridged.BlindSpots {
		stillPresent[bs.Builtin] = true
	}
	for _, bs := range orig.BlindSpots {
		if !stillPresent[bs.Builtin] {
			changes = append(changes, &antv1.SemanticChange{
				Kind:        "removed",
				Description: fmt.Sprintf("盲区 %s 已通过 Python 翻译消除", bs.Builtin),
			})
		}
	}

	for _, bs := range bridged.BlindSpots {
		changes = append(changes, &antv1.SemanticChange{
			Kind:        "remaining",
			Description: fmt.Sprintf("盲区 %s 仍存在 (severity: %s)", bs.Builtin, bs.Severity),
		})
	}

	return changes
}

// buildBridgeFailureReport generates a SemanticDiff for the bridge_failed degradation path.
// ADR §5.4: blind spot report + MT hosted suggestion.
func buildBridgeFailureReport(coverage *mql2go.CoverageResult) *antv1.SemanticDiff {
	var changes []*antv1.SemanticChange

	for _, bs := range coverage.BlindSpots {
		changes = append(changes, &antv1.SemanticChange{
			Kind:        "remaining",
			Description: fmt.Sprintf("无法桥接的函数: %s (severity: %s, count: %d)", bs.Builtin, bs.Severity, bs.Count),
		})
	}

	changes = append(changes, &antv1.SemanticChange{
		Kind:        "added",
		Description: "建议 1: 在 MetaTrader 客户端中直接运行此 EA (MT 托管模式)，平台仅做信号跟踪",
	})
	changes = append(changes, &antv1.SemanticChange{
		Kind:        "added",
		Description: "建议 2: 手动修改 EA 移除不支持函数后重新上传",
	})

	return &antv1.SemanticDiff{
		Changes:       changes,
		EffectSummary: "Agent 已尝试自动桥接但未成功。此 EA 包含平台不支持的函数，建议使用 MT 托管模式或手动修改。",
	}
}

// unquote removes surrounding quotes from a value.
func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// splitTrimmed splits a string by sep and trims each field, dropping empty entries.
func splitTrimmed(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseProfileLines parses KEY: "value" lines into a StrategyProfile.
// Shared by parseProfileResponse (source+coverage) and parseProfileResponseNL (NL only).
func parseProfileLines(raw string) *antv1.StrategyProfile {
	profile := &antv1.StrategyProfile{}
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "```") {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		val := strings.TrimSpace(line[colonIdx+1:])
		val = unquote(val)
		switch key {
		case "strategy_type":
			profile.StrategyType = val
		case "description":
			profile.Description = val
		case "indicators_used":
			if val != "" {
				profile.IndicatorsUsed = splitTrimmed(val, ",")
			}
		case "entry_logic":
			profile.EntryLogic = val
		case "exit_logic":
			profile.ExitLogic = val
		case "risk_management":
			profile.RiskManagement = val
		case "timeframe_preference":
			profile.TimeframePreference = val
		case "market_regime":
			profile.MarketRegime = val
		case "strengths":
			profile.Strengths = splitTrimmed(val, ",")
		case "weaknesses":
			profile.Weaknesses = splitTrimmed(val, ",")
		case "coverage_score":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				profile.CoverageScore = f
			}
		case "blind_spots":
			if val != "" {
				profile.BlindSpots = splitTrimmed(val, ",")
			}
		}
	}
	return profile
}
