package prompt

import (
	"fmt"
	"strings"
	"time"

	"anttrader/internal/model"
)

// ReportSystemPrompt 交易分析报告的系统提示词
const ReportSystemPrompt = `你是一个专业的交易分析师。根据提供的交易历史数据，生成详细的交易分析报告。

输出格式要求（必须严格遵循JSON格式）：
{
  "summary": "总体评价",
  "strengths": ["优点1", "优点2"],
  "weaknesses": ["不足1", "不足2"],
  "suggestions": ["建议1", "建议2"],
  "risk_assessment": "风险评估",
  "score": 0-100
}

分析要求：
1. 基于数据给出客观评价，不要凭空猜测
2. 优点和不足都要有数据支撑
3. 建议要具体可执行
4. 风险评估要考虑最大回撤、波动率等指标
5. 评分要综合考虑胜率、盈亏比、风险控制等因素
6. 只输出JSON，不要包含其他文字说明`

// BuildReportPrompt 构建交易分析报告提示词
func BuildReportPrompt(stats *model.TradeStats, riskMetrics *model.RiskMetrics, symbolStats []*model.SymbolStats, recentTrades []*model.TradeRecord) string {
	var sb strings.Builder

	sb.WriteString("请根据以下交易数据生成分析报告：\n\n")

	// 交易统计
	sb.WriteString("## 交易统计\n")
	sb.WriteString(formatTradeStats(stats))
	sb.WriteString("\n")

	// 风险指标
	if riskMetrics != nil {
		sb.WriteString("## 风险指标\n")
		sb.WriteString(formatRiskMetrics(riskMetrics))
		sb.WriteString("\n")
	}

	// 品种统计
	if len(symbolStats) > 0 {
		sb.WriteString("## 品种统计\n")
		sb.WriteString(formatSymbolStats(symbolStats))
		sb.WriteString("\n")
	}

	// 最近交易记录
	if len(recentTrades) > 0 {
		sb.WriteString("## 最近交易记录\n")
		sb.WriteString(formatRecentTrades(recentTrades))
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatTradeStats 格式化交易统计
func formatTradeStats(stats *model.TradeStats) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("- 总交易次数: %d\n", stats.TotalTrades))
	sb.WriteString(fmt.Sprintf("- 盈利交易: %d\n", stats.WinningTrades))
	sb.WriteString(fmt.Sprintf("- 亏损交易: %d\n", stats.LosingTrades))
	sb.WriteString(fmt.Sprintf("- 胜率: %.2f%%\n", stats.WinRate))
	sb.WriteString(fmt.Sprintf("- 总盈利: %.2f\n", stats.TotalProfit))
	sb.WriteString(fmt.Sprintf("- 总亏损: %.2f\n", stats.TotalLoss))
	sb.WriteString(fmt.Sprintf("- 净利润: %.2f\n", stats.NetProfit))
	sb.WriteString(fmt.Sprintf("- 盈亏比: %.2f\n", stats.ProfitFactor))
	sb.WriteString(fmt.Sprintf("- 平均盈利: %.2f\n", stats.AverageProfit))
	sb.WriteString(fmt.Sprintf("- 平均亏损: %.2f\n", stats.AverageLoss))
	sb.WriteString(fmt.Sprintf("- 平均交易: %.2f\n", stats.AverageTrade))
	sb.WriteString(fmt.Sprintf("- 最大单笔盈利: %.2f\n", stats.LargestWin))
	sb.WriteString(fmt.Sprintf("- 最大单笔亏损: %.2f\n", stats.LargestLoss))
	sb.WriteString(fmt.Sprintf("- 总交易量: %.2f\n", stats.TotalVolume))
	sb.WriteString(fmt.Sprintf("- 最大连续盈利次数: %d\n", stats.MaxConsecutiveWins))
	sb.WriteString(fmt.Sprintf("- 最大连续亏损次数: %d\n", stats.MaxConsecutiveLosses))
	sb.WriteString(fmt.Sprintf("- 平均持仓时间: %s\n", stats.AverageHoldingTime))

	return sb.String()
}

// formatRiskMetrics 格式化风险指标
func formatRiskMetrics(metrics *model.RiskMetrics) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("- 最大回撤: %.2f\n", metrics.MaxDrawdown))
	sb.WriteString(fmt.Sprintf("- 最大回撤百分比: %.2f%%\n", metrics.MaxDrawdownPercent))
	sb.WriteString(fmt.Sprintf("- 夏普比率: %.2f\n", metrics.SharpeRatio))
	sb.WriteString(fmt.Sprintf("- 索提诺比率: %.2f\n", metrics.SortinoRatio))
	sb.WriteString(fmt.Sprintf("- 卡玛比率: %.2f\n", metrics.CalmarRatio))
	sb.WriteString(fmt.Sprintf("- 波动率: %.2f\n", metrics.Volatility))
	sb.WriteString(fmt.Sprintf("- 95%% VaR: %.2f\n", metrics.ValueAtRisk95))
	sb.WriteString(fmt.Sprintf("- 预期亏损: %.2f\n", metrics.ExpectedShortfall))
	sb.WriteString(fmt.Sprintf("- 平均日收益: %.2f\n", metrics.AverageDailyReturn))
	sb.WriteString(fmt.Sprintf("- 收益标准差: %.2f\n", metrics.ReturnStdDev))

	return sb.String()
}

// formatSymbolStats 格式化品种统计
func formatSymbolStats(stats []*model.SymbolStats) string {
	var sb strings.Builder

	for i, stat := range stats {
		if i >= 10 { // 只显示前10个品种
			break
		}
		sb.WriteString(fmt.Sprintf("\n### %s\n", stat.Symbol))
		sb.WriteString(fmt.Sprintf("  - 交易次数: %d\n", stat.TotalTrades))
		sb.WriteString(fmt.Sprintf("  - 胜率: %.2f%%\n", stat.WinRate))
		sb.WriteString(fmt.Sprintf("  - 净利润: %.2f\n", stat.NetProfit))
		sb.WriteString(fmt.Sprintf("  - 盈亏比: %.2f\n", stat.ProfitFactor))
		sb.WriteString(fmt.Sprintf("  - 总交易量: %.2f\n", stat.TotalVolume))
	}

	return sb.String()
}

// formatRecentTrades 格式化最近交易记录
func formatRecentTrades(trades []*model.TradeRecord) string {
	var sb strings.Builder

	sb.WriteString("| 时间 | 品种 | 方向 | 手数 | 开仓价 | 平仓价 | 盈亏 |\n")
	sb.WriteString("|------|------|------|------|--------|--------|------|\n")

	for i, trade := range trades {
		if i >= 20 { // 只显示最近20笔
			break
		}
		direction := "买入"
		if trade.OrderType == "sell" {
			direction = "卖出"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %.2f | %.5f | %.5f | %.2f |\n",
			trade.CloseTime.Format("2006-01-02 15:04"),
			trade.Symbol,
			direction,
			trade.Volume,
			trade.OpenPrice,
			trade.ClosePrice,
			trade.Profit,
		))
	}

	return sb.String()
}

// AIReport AI生成的交易分析报告
type AIReport struct {
	Summary        string   `json:"summary"`
	Strengths      []string `json:"strengths"`
	Weaknesses     []string `json:"weaknesses"`
	Suggestions    []string `json:"suggestions"`
	RiskAssessment string   `json:"risk_assessment"`
	Score          int      `json:"score"`
	GeneratedAt    string   `json:"generated_at"`
}

// TradeRecordForPrompt 用于提示词的交易记录
type TradeRecordForPrompt struct {
	Symbol     string    `json:"symbol"`
	OrderType  string    `json:"order_type"`
	Volume     float64   `json:"volume"`
	OpenPrice  float64   `json:"open_price"`
	ClosePrice float64   `json:"close_price"`
	Profit     float64   `json:"profit"`
	OpenTime   time.Time `json:"open_time"`
	CloseTime  time.Time `json:"close_time"`
}
