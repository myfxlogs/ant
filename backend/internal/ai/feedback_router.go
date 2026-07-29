// feedback_router.go: classifies user NL feedback and routes to Phase 1 or Phase 2.
// "太激进了" → Phase 2 (re-optimize) vs "改成做空" → Phase 1 (regenerate code).

package ai

import "strings"

// FeedbackTarget indicates which pipeline phase should handle the user's feedback.
type FeedbackTarget int

const (
	FeedbackUnknown  FeedbackTarget = iota
	FeedbackPhase1                  // regenerate DSL code
	FeedbackPhase2                  // re-optimize parameters
)

// String returns a human-readable target name.
func (t FeedbackTarget) String() string {
	switch t {
	case FeedbackPhase1:
		return "Phase1_regenerate"
	case FeedbackPhase2:
		return "Phase2_reoptimize"
	default:
		return unknownStr
	}
}

// RoutingResult contains the classified target and any extracted parameters.
type RoutingResult struct {
	Target   FeedbackTarget
	Reason   string
	ParamMap map[string]string // extracted adjustment hints
}

// phase1Keywords trigger code regeneration (structural/behavioral changes).
var phase1Keywords = []string{
	"改成", "改为", "换成", "改为做", "多改成", "空改成",
	"加上", "添加", "增加", "加上止损", "加上止盈",
	"去掉", "删除", "移除", "不要",
	"换品种", "换一个", "改品种",
	"做空", "做多", "只做多", "只做空",
	"突破改成", "趋势改成", "马丁改成",
}

// phase2Keywords trigger parameter re-optimization (quantitative adjustments).
var phase2Keywords = []string{
	"太激进", "太保守", "风险太大", "太高了", "太低了",
	"稳健一点", "保守一点", "激进一点",
	"回撤太大", "回撤太高", "降低回撤", "控制回撤",
	"降低仓位", "减少仓位", "缩小仓位", "减小仓位",
	"提高仓位", "增加仓位", "加大仓位",
	"收紧止损", "放宽止损", "调整止损",
	"收益太低", "提高收益", "收益不高",
}

// RouteFeedback classifies user feedback and returns the target phase.
func RouteFeedback(message string, lastMetrics *FeedbackMetrics) *RoutingResult {
	msg := strings.ToLower(message)

	// Check Phase 2 keywords first (quantitative adjustments)
	if kw, ok := matchAny(msg, phase2Keywords); ok {
		result := &RoutingResult{
			Target: FeedbackPhase2,
			Reason: "matched keyword: " + kw,
		}
		if lastMetrics != nil {
			result.ParamMap = extractAdjustments(msg, lastMetrics)
		}
		return result
	}

	// Check Phase 1 keywords (structural/behavioral changes)
	if kw, ok := matchAny(msg, phase1Keywords); ok {
		return &RoutingResult{
			Target: FeedbackPhase1,
			Reason: "matched keyword: " + kw,
		}
	}

	// Default: Phase 1 (regenerate is safer than re-optimize)
	return &RoutingResult{
		Target: FeedbackPhase1,
		Reason: "no keyword matched, defaulting to Phase 1",
	}
}

// extractAdjustments maps user feedback to concrete parameter adjustments.
func extractAdjustments(msg string, m *FeedbackMetrics) map[string]string {
	pm := make(map[string]string)

	switch {
	case strings.Contains(msg, "降低仓位") || strings.Contains(msg, "减少仓位") || strings.Contains(msg, "缩小仓位"):
		pm["entry_pct"] = "0.5"
	case strings.Contains(msg, "提高仓位") || strings.Contains(msg, "增加仓位"):
		pm["entry_pct"] = "1.0"
	}

	if strings.Contains(msg, "收紧止损") {
		pm["stop_loss_pct"] = "0.01"
	} else if strings.Contains(msg, "放宽止损") {
		pm["stop_loss_pct"] = "0.05"
	}

	if strings.Contains(msg, "太激进") || strings.Contains(msg, "保守一点") || strings.Contains(msg, "稳健一点") {
		pm["max_drawdown"] = "0.10"
		pm["risk_level"] = "low"
	}
	if strings.Contains(msg, "太保守") || strings.Contains(msg, "激进一点") {
		pm["max_drawdown"] = "0.30"
		pm["risk_level"] = "high"
	}

	return pm
}

func matchAny(msg string, keywords []string) (string, bool) {
	for _, kw := range keywords {
		if strings.Contains(msg, kw) {
			return kw, true
		}
	}
	return "", false
}
