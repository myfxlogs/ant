package agent

import (
	"fmt"

	"github.com/google/uuid"
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

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func buildBridgeChanges(orig, bridged *mql2go.CoverageResult) []*antv1.SemanticChange {
	var changes []*antv1.SemanticChange

	resolved := make(map[string]bool)
	for _, bs := range bridged.BlindSpots {
		resolved[bs.Builtin] = true
	}
	for _, bs := range orig.BlindSpots {
		if !resolved[bs.Builtin] {
			changes = append(changes, &antv1.SemanticChange{
				Kind:        "removed",
				Description: fmt.Sprintf("盲区 %s 已通过 Python 翻译消除", bs.Builtin),
			})
		}
	}

	for _, bs := range bridged.BlindSpots {
		changes = append(changes, &antv1.SemanticChange{
			Kind:        "modified",
			Description: fmt.Sprintf("盲区 %s 仍存在 (severity: %s)", bs.Builtin, bs.Severity),
		})
	}

	return changes
}
