package system

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/interceptor"
	systemai "anttrader/internal/service/systemai"
)

// ── GenerateReport (SSE streaming) ──

func (s *AnalyticsServer) GenerateReport(ctx context.Context, req *connect.Request[antv1.GenerateReportRequest], stream *connect.ServerStream[antv1.GenerateReportChunk]) error {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", err))
	}
	if err := s.verifyAccountOwnership(ctx, req.Msg.AccountId); err != nil {
		return err
	}
	if s.aiSvc == nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI service not configured"))
	}

	userID := interceptor.GetUserID(ctx)

	// Compute metrics.
	_ = stream.Send(&antv1.GenerateReportChunk{Phase: "computing"})
	metrics := s.buildReportMetrics(ctx, accountID, req.Msg.Period)

	sysPrompt := `你是一位专业的量化交易分析师。用户提供了交易账户的历史数据，请分析并生成一份简洁的交易报告。
请使用以下结构输出报告：

<section type="summary">
总体评价——2-3句话概括账户表现。包含胜率、盈亏比、最大回撤等关键数据。
</section>

<section type="findings">
关键发现——列出2-4个具体发现。引用具体数据（品种、时段、胜率变化等）。每个发现使用一句话。
</section>

<section type="recommendations">
改进建议——基于发现给出2-3条可操作建议。
</section>

要求：简洁、数据驱动、避免泛泛而谈。使用中文输出。`

	metricsJSON, _ := json.Marshal(metrics)
	userMsg := fmt.Sprintf("请分析以下交易账户的%v周期表现：\n%s", req.Msg.Period, string(metricsJSON))

	messages := []systemai.ChatMessage{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: userMsg},
	}

	var fullText strings.Builder
	err = s.aiSvc.ChatCompletionStream(ctx, uuid.MustParse(userID), messages, "", func(chunk systemai.ChatStreamChunk) error {
		fullText.WriteString(chunk.Content)
		phase := "analyzing"
		if chunk.Done {
			phase = "done"
		}
		return stream.Send(&antv1.GenerateReportChunk{
			Phase: phase,
			Delta: chunk.Content,
			Done:  chunk.Done,
		})
	})
	if err != nil {
		s.log.Error("report generation stream failed", zap.Error(err))
		_ = stream.Send(&antv1.GenerateReportChunk{
			Phase: "done",
			Error: "报告生成失败，请重试",
			Done:  true,
		})
		return nil
	}

	// Parse sections and send final chunk.
	sections := parseReportSections(fullText.String())
	_ = stream.Send(&antv1.GenerateReportChunk{
		Phase:           "done",
		Done:            true,
		Summary:         sections.summary,
		Findings:        sections.findings,
		Recommendations: sections.recommendations,
	})
	return nil
}

type reportMetrics struct {
	NetProfit          float64         `json:"net_profit"`
	TotalTrades        int64           `json:"total_trades"`
	WinRate            float64         `json:"win_rate"`
	ProfitFactor       float64         `json:"profit_factor"`
	MaxDrawdown        float64         `json:"max_drawdown_percent"`
	SharpeRatio        float64         `json:"sharpe_ratio"`
	TopSymbols         []symbolSummary `json:"top_symbols"`
	DirectionBreakdown string          `json:"direction_breakdown"`
}

type symbolSummary struct {
	Symbol  string  `json:"symbol"`
	Profit  float64 `json:"profit"`
	Trades  int64   `json:"trades"`
	WinRate float64 `json:"win_rate"`
}

func (s *AnalyticsServer) buildReportMetrics(ctx context.Context, accountID uuid.UUID, period string) *reportMetrics {
	now := time.Now()
	start := now.AddDate(-1, 0, 0)
	switch period {
	case "week":
		start = now.AddDate(0, 0, -7)
	case "month":
		start = now.AddDate(0, -1, 0)
	case "quarter":
		start = now.AddDate(0, -3, 0)
	}

	trades, _ := s.repo.GetTradeRecords(ctx, accountID, start, now)
	stats := computeTradeStats(trades)

	m := &reportMetrics{
		NetProfit:    math.Round(stats.NetProfit.InexactFloat64()*100) / 100,
		TotalTrades:  int64(stats.TotalTrades),
		WinRate:      stats.WinRate.InexactFloat64(),
		ProfitFactor: stats.ProfitFactor.InexactFloat64(),
	}

	// Max drawdown.
	_, m.MaxDrawdown, _ = s.repo.GetMaxDrawdown(ctx, accountID, start, now)
	m.MaxDrawdown = math.Round(m.MaxDrawdown*100) / 100

	// Sharpe.
	eq, _ := s.repo.GetEquityCurve(ctx, accountID, start, now)
	if dailies := dailyReturnsToPercent(eq); len(dailies) > 0 {
		m.SharpeRatio, _, _, _, _ = computeRiskMetrics(dailies, m.MaxDrawdown)
		m.SharpeRatio = math.Round(m.SharpeRatio*100) / 100
	}

	// Top symbols.
	symbolStats, _ := s.repo.GetSymbolStats(ctx, accountID, start, now)
	sort.Slice(symbolStats, func(i, j int) bool {
		return symbolStats[i].NetProfit.GreaterThan(symbolStats[j].NetProfit)
	})
	for i, ss := range symbolStats {
		if i >= 5 {
			break
		}
		wr := 0.0
		if ss.TotalTrades > 0 {
			wr = float64(ss.WinningTrades) / float64(ss.TotalTrades) * 100
		}
		m.TopSymbols = append(m.TopSymbols, symbolSummary{
			Symbol:  ss.Symbol,
			Profit:  math.Round(ss.NetProfit.InexactFloat64()*100) / 100,
			Trades:  int64(ss.TotalTrades),
			WinRate: math.Round(wr*100) / 100,
		})
	}

	// Direction.
	dirStats, _ := s.repo.GetPnLByDirection(ctx, accountID, start, now)
	if len(dirStats) > 0 {
		parts := make([]string, len(dirStats))
		for i, d := range dirStats {
			parts[i] = fmt.Sprintf("%s P&L=%.0f (%d trades)", d.Direction, d.Profit, d.Trades)
		}
		m.DirectionBreakdown = strings.Join(parts, ", ")
	}

	return m
}

type reportSections struct {
	summary         string
	findings        string
	recommendations string
}

func parseReportSections(raw string) reportSections {
	var s reportSections
	s.summary = extractSection(raw, "summary")
	s.findings = extractSection(raw, "findings")
	s.recommendations = extractSection(raw, "recommendations")
	return s
}

func extractSection(raw, sectionType string) string {
	openTag := fmt.Sprintf("<section type=\"%s\">", sectionType)
	closeTag := "</section>"
	start := strings.Index(raw, openTag)
	if start < 0 {
		return ""
	}
	start += len(openTag)
	end := strings.Index(raw[start:], closeTag)
	if end < 0 {
		return strings.TrimSpace(raw[start:])
	}
	return strings.TrimSpace(raw[start : start+end])
}
