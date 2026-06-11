package system

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

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

	// Compute metrics — prefer cache, fall back to DB queries.
	_ = stream.Send(&antv1.GenerateReportChunk{Phase: "computing"})

	analyticsResp := s.getOrComputeAnalyticsForReport(ctx, accountID)
	attrResp := s.getOrComputeAttributionForReport(ctx, accountID)
	if analyticsResp == nil || attrResp == nil {
		_ = stream.Send(&antv1.GenerateReportChunk{
			Phase: "done",
			Error: "数据加载失败，请重试",
			Done:  true,
		})
		return nil
	}
	metrics := buildReportMetrics(analyticsResp, attrResp)

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

// getOrComputeAnalyticsForReport returns analytics data from cache, or delegates
// to computeAnalyticsCore on miss. Returns nil if both cache and compute fail.
// Deliberately omits cache write — the partial response (no equity/daily/hourly)
// must not pollute the shared cache key.
func (s *AnalyticsServer) getOrComputeAnalyticsForReport(ctx context.Context, accountID uuid.UUID) *antv1.AccountAnalyticsResponse {
	if s.cache != nil {
		if cached, _ := s.cache.Get(ctx, accountID.String()); cached != nil {
			return cached
		}
	}
	core, err := s.computeAnalyticsCore(ctx, accountID)
	if err != nil {
		s.log.Error("report: compute analytics core failed", zap.Error(err))
		return nil
	}
	return core
}

// getOrComputeAttributionForReport returns attribution data from cache, or
// delegates to computeAttributionCore on miss. Deliberately omits cache write —
// the partial response (no trade distribution / hourly PnL) must not pollute
// the shared cache key.
func (s *AnalyticsServer) getOrComputeAttributionForReport(ctx context.Context, accountID uuid.UUID) *antv1.GetAttributionAnalysisResponse {
	if s.cache != nil {
		if cached, _ := s.cache.GetAttribution(ctx, accountID.String()); cached != nil {
			return cached
		}
	}
	return s.computeAttributionCore(ctx, accountID)
}

// buildReportMetrics maps analytics + attribution proto responses into the
// AI prompt JSON struct. Pure function — no DB access.
func buildReportMetrics(analytics *antv1.AccountAnalyticsResponse, attribution *antv1.GetAttributionAnalysisResponse) *reportMetrics {
	m := &reportMetrics{
		NetProfit:    math.Round(analytics.TradeStats.NetProfit*100) / 100,
		TotalTrades:  analytics.TradeStats.TotalTrades,
		WinRate:      analytics.TradeStats.WinRate,
		ProfitFactor: analytics.TradeStats.ProfitFactor,
		MaxDrawdown:  math.Round(analytics.RiskMetrics.MaxDrawdownPercent*100) / 100,
		SharpeRatio:  math.Round(analytics.RiskMetrics.SharpeRatio*100) / 100,
	}

	// Top 5 symbols by profit — use attribution SymbolPnLs which have
	// TotalTrades and WinRate (unlike analytics SymbolStats).
	sorted := make([]*antv1.SymbolPnL, len(attribution.SymbolPnls))
	copy(sorted, attribution.SymbolPnls)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].NetProfit > sorted[j].NetProfit
	})
	for i, s := range sorted {
		if i >= 5 {
			break
		}
		m.TopSymbols = append(m.TopSymbols, symbolSummary{
			Symbol:  s.Symbol,
			Profit:  math.Round(s.NetProfit*100) / 100,
			Trades:  s.TotalTrades,
			WinRate: s.WinRate,
		})
	}

	// Direction breakdown.
	if attribution.Direction != nil {
		parts := make([]string, 0, 2)
		if attribution.Direction.LongTrades > 0 {
			parts = append(parts, fmt.Sprintf("BUY P&L=%.0f (%d trades)",
				attribution.Direction.LongProfit, attribution.Direction.LongTrades))
		}
		if attribution.Direction.ShortTrades > 0 {
			parts = append(parts, fmt.Sprintf("SELL P&L=%.0f (%d trades)",
				attribution.Direction.ShortProfit, attribution.Direction.ShortTrades))
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
