package agent

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/tools/mql2go"
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
