// backtest_feedback.go: injects backtest results into AI conversation context.
// Formats metrics as Chinese-language prompt snippets for Phase 3 feedback loop.

package ai

import "fmt"

// FeedbackMetrics holds the key backtest metrics for prompt injection.
type FeedbackMetrics struct {
	SharpeRatio  float64
	MaxDrawdown  float64
	WinRate      float64
	ProfitFactor float64
	TotalReturn  float64
	TotalTrades  int
	DSLCode      string
	Summary      string // one-sentence summary in Chinese
}

// FormatPromptContext converts backtest metrics to a prompt-ready Chinese string.
// Used in Phase 3 to inject "【上一轮策略回测结果】" into the LLM conversation.
func (m *FeedbackMetrics) FormatPromptContext() string {
	return fmt.Sprintf(
		"【上一轮策略回测结果】\n"+
			"回测指标: Sharpe %.2f, 最大回撤 %.1f%%, 胜率 %.0f%%, "+
			"盈亏比 %.2f, 总收益 %.1f%%, 交易次数 %d\n"+
			"总结: %s",
		m.SharpeRatio, m.MaxDrawdown*100,
		m.WinRate*100, m.ProfitFactor, m.TotalReturn*100,
		m.TotalTrades, m.Summary,
	)
}

// Summarize generates a one-line Chinese summary of the key metrics.
// For use when the full prompt context is not needed.
func (m *FeedbackMetrics) Summarize() string {
	switch {
	case m.MaxDrawdown > 0.25:
		return fmt.Sprintf("回撤偏大(%.0f%%)，建议降低仓位或收紧止损", m.MaxDrawdown*100)
	case m.SharpeRatio < 0.5:
		return fmt.Sprintf("Sharpe偏低(%.2f)，策略方向可能需要调整", m.SharpeRatio)
	case m.TotalTrades < 10:
		return fmt.Sprintf("交易次数少(%d次)，回测统计意义有限", m.TotalTrades)
	case m.TotalReturn < 0:
		return fmt.Sprintf("总收益为负(%.1f%%)，策略逻辑需要修正", m.TotalReturn*100)
	default:
		return fmt.Sprintf("Sharpe %.2f, 回撤 %.0f%%, 胜率 %.0f%%, 交易%d次",
			m.SharpeRatio, m.MaxDrawdown*100, m.WinRate*100, m.TotalTrades)
	}
}

// FromBacktestResult converts a BacktestMetrics + code into a FeedbackMetrics.
func FromBacktestResult(metrics *BacktestMetrics, code string) *FeedbackMetrics {
	if metrics == nil {
		return &FeedbackMetrics{Summary: "回测未产生结果", DSLCode: code}
	}
	return &FeedbackMetrics{
		SharpeRatio:  metrics.SharpeRatio,
		MaxDrawdown:  metrics.MaxDrawdown,
		WinRate:      metrics.WinRate,
		ProfitFactor: metrics.ProfitFactor,
		TotalReturn:  metrics.TotalReturn,
		TotalTrades:  metrics.TotalTrades,
		DSLCode:      code,
		Summary:      (&FeedbackMetrics{MaxDrawdown: metrics.MaxDrawdown, SharpeRatio: metrics.SharpeRatio, TotalTrades: metrics.TotalTrades, TotalReturn: metrics.TotalReturn}).Summarize(),
	}
}
