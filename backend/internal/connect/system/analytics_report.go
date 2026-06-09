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
	"anttrader/internal/model"
	"anttrader/internal/repository"
	systemai "anttrader/internal/service/systemai"
)

// ── GetAttributionAnalysis ──

func (s *AnalyticsServer) GetAttributionAnalysis(ctx context.Context, req *connect.Request[antv1.GetAttributionAnalysisRequest]) (*connect.Response[antv1.GetAttributionAnalysisResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", err))
	}
	if err := s.verifyAccountOwnership(ctx, req.Msg.AccountId); err != nil {
		return nil, err
	}

	now := time.Now()
	start := now.AddDate(-1, 0, 0)

	// Symbol P&L — existing query already has profit per symbol.
	symbolStats, _ := s.repo.GetSymbolStats(ctx, accountID, start, now)
	symbolPnLs := make([]*antv1.SymbolPnL, 0, len(symbolStats))
	for _, ss := range symbolStats {
		wr := 0.0
		if ss.TotalTrades > 0 {
			wr = float64(ss.WinningTrades) / float64(ss.TotalTrades) * 100
		}
		pf := 0.0
		if !ss.TotalLoss.IsZero() {
			pf = math.Abs(ss.TotalProfit.InexactFloat64() / ss.TotalLoss.InexactFloat64())
		}
		symbolPnLs = append(symbolPnLs, &antv1.SymbolPnL{
			Symbol:           ss.Symbol,
			NetProfit:        math.Round(ss.NetProfit.InexactFloat64()*100) / 100,
			TotalTrades:      int64(ss.TotalTrades),
			WinRate:          math.Round(wr*100) / 100,
			ProfitFactor:     math.Round(pf*100) / 100,
			TradeSharePercent: 0, // computed client-side
		})
	}
	sort.Slice(symbolPnLs, func(i, j int) bool {
		return symbolPnLs[i].NetProfit > symbolPnLs[j].NetProfit
	})

	// Direction breakdown.
	dirStats, _ := s.repo.GetPnLByDirection(ctx, accountID, start, now)
	dir := buildDirectionBreakdown(dirStats)

	// Trade profit distribution histogram.
	profits, _ := s.repo.GetTradeProfitValues(ctx, accountID, start, now)
	tradeDist := buildTradeDistribution(profits)

	// Hourly P&L — reuse existing data.
	hourlyStats, _ := s.repo.GetHourlyStats(ctx, accountID, start, now)
	hourlyPnl := make([]*antv1.HourlyPnL, 0, len(hourlyStats))
	for _, h := range hourlyStats {
		hourlyPnl = append(hourlyPnl, &antv1.HourlyPnL{
			Hour:   int32(h.HourStart),
			Profit: math.Round(h.Profit.InexactFloat64()*100) / 100,
			Trades: int64(h.Trades),
			WinRate: math.Round(h.WinRate.InexactFloat64()*100) / 100,
		})
	}

	return connect.NewResponse(&antv1.GetAttributionAnalysisResponse{
		SymbolPnls:        symbolPnLs,
		Direction:          dir,
		TradeDistribution:  tradeDist,
		HourlyPnl:          hourlyPnl,
	}), nil
}

func buildDirectionBreakdown(stats []*repository.DirectionStat) *antv1.DirectionBreakdown {
	dir := &antv1.DirectionBreakdown{}
	for _, s := range stats {
		wr := 0.0
		if s.Trades > 0 {
			wr = float64(s.WinTrades) / float64(s.Trades) * 100
		}
		switch s.Direction {
		case "BUY":
			dir.LongProfit = math.Round(s.Profit*100) / 100
			dir.LongTrades = int64(s.Trades)
			dir.LongWinRate = math.Round(wr*100) / 100
		case "SELL":
			dir.ShortProfit = math.Round(s.Profit*100) / 100
			dir.ShortTrades = int64(s.Trades)
			dir.ShortWinRate = math.Round(wr*100) / 100
		}
	}
	return dir
}

func buildTradeDistribution(profits []float64) *antv1.TradeDistribution {
	if len(profits) == 0 {
		return &antv1.TradeDistribution{}
	}
	// Histogram buckets for profit distribution.
	bounds := []float64{-500, -200, -100, -50, -20, 0, 20, 50, 100, 200, 500}
	labels := []string{"<-500", "-500~-200", "-200~-100", "-100~-50", "-50~-20", "-20~0",
		"0~20", "20~50", "50~100", "100~200", "200~500", ">500"}
	buckets := make([]int64, len(labels))
	for _, p := range profits {
		idx := sort.SearchFloat64s(bounds, p)
		buckets[idx]++
	}
	dist := make([]*antv1.TradeBucket, 0, len(labels))
	for i, label := range labels {
		if buckets[i] == 0 {
			continue
		}
		b := &antv1.TradeBucket{Label: label, Count: buckets[i]}
		if i < len(bounds) {
			b.MaxValue = bounds[i]
		}
		if i > 0 {
			b.MinValue = bounds[i-1]
		}
		dist = append(dist, b)
	}
	return &antv1.TradeDistribution{ProfitBuckets: dist}
}

// ── GetRollingMetrics ──

func (s *AnalyticsServer) GetRollingMetrics(ctx context.Context, req *connect.Request[antv1.GetRollingMetricsRequest]) (*connect.Response[antv1.GetRollingMetricsResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", err))
	}
	if err := s.verifyAccountOwnership(ctx, req.Msg.AccountId); err != nil {
		return nil, err
	}

	now := time.Now()
	start := now.AddDate(-1, 0, 0)

	// Equity curve — base for rolling metrics and drawdown.
	equityCurve, _ := s.repo.GetEquityCurve(ctx, accountID, start, now)
	equityCurve = appendLiveEquity(ctx, s.repo, accountID, equityCurve)

	// Rolling Sharpe — 20-trade window from daily returns.
	dailyReturns := dailyReturnsToPercent(equityCurve)
	rollingSharpe := computeRollingSharpe(dailyReturns, equityCurve)

	// Drawdown events.
	ddEvents := computeDrawdownEvents(equityCurve)

	// Equity curve with drawdown overlay.
	eqProto, ddProto := buildEquityDrawdown(equityCurve)

	// Monthly win rates — from actual trade records.
	monthlyWinRates := computeMonthlyWinRates(ctx, s.repo, accountID, start, now)

	return connect.NewResponse(&antv1.GetRollingMetricsResponse{
		RollingSharpe:    rollingSharpe,
		DrawdownEvents:   ddEvents,
		MonthlyWinRates:  monthlyWinRates,
		EquityCurve:      eqProto,
		DrawdownCurve:    ddProto,
	}), nil
}

func computeRollingSharpe(dailyReturns []float64, curve []*model.EquityPoint) []*antv1.RollingPoint {
	const window = 20
	if len(dailyReturns) < window || len(curve) < window {
		return nil
	}
	points := make([]*antv1.RollingPoint, 0, len(dailyReturns)-window+1)
	for i := window - 1; i < len(dailyReturns); i++ {
		slice := dailyReturns[i-window+1 : i+1]
		avg := mean(slice)
		std := stdev(slice, avg)
		sharpe := 0.0
		if std > 0 {
			sharpe = avg / std * math.Sqrt(252)
		}
		dateIdx := i
		if dateIdx >= len(curve) {
			dateIdx = len(curve) - 1
		}
		points = append(points, &antv1.RollingPoint{
			Date:  curve[dateIdx].Date,
			Value: math.Round(sharpe*100) / 100,
		})
	}
	return points
}

func computeDrawdownEvents(curve []*model.EquityPoint) []*antv1.DrawdownEvent {
	if len(curve) < 2 {
		return nil
	}
	var events []*antv1.DrawdownEvent
	var inDD bool
	var ddStart string
	var ddDepth float64
	runningMax := 0.0

	for _, p := range curve {
		eq := p.Equity.InexactFloat64()
		if eq > runningMax {
			runningMax = eq
			if inDD {
				// Drawdown ended.
				events[len(events)-1].EndDate = p.Date
				events[len(events)-1].RecoveryDate = p.Date
				inDD = false
			}
		}
		if runningMax > 0 {
			dd := (runningMax - eq) / runningMax * 100
			if dd > 5 && !inDD { // 5% threshold to register a drawdown event.
				inDD = true
				ddStart = p.Date
				ddDepth = dd
				events = append(events, &antv1.DrawdownEvent{
					StartDate:    ddStart,
					DepthPercent: math.Round(dd*100) / 100,
				})
			}
			if inDD {
				if dd > ddDepth {
					ddDepth = dd
					if len(events) > 0 {
						events[len(events)-1].DepthPercent = math.Round(dd*100) / 100
					}
				}
			}
		}
	}
	return events
}

func computeMonthlyWinRates(ctx context.Context, repo *repository.AnalyticsRepository, accountID uuid.UUID, start, end time.Time) []*antv1.MonthlyWinRate {
	// Use the existing monthly PnL query which aggregates win/loss counts per month.
	year := end.Year()
	if start.Year() < year {
		year = start.Year()
	}
	monthlyPnL, err := repo.GetMonthlyPnL(ctx, accountID, year)
	if err != nil || len(monthlyPnL) == 0 {
		return nil
	}
	out := make([]*antv1.MonthlyWinRate, 0, len(monthlyPnL))
	for _, m := range monthlyPnL {
		wr := 0.0
		if m.Trades > 0 {
			wr = float64(m.WinTrades) / float64(m.Trades) * 100
		}
		out = append(out, &antv1.MonthlyWinRate{
			Month:       m.Month,
			WinRate:     math.Round(wr*100) / 100,
			TotalTrades: int64(m.Trades),
		})
	}
	return out
}

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func stdev(vals []float64, avg float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	sumSq := 0.0
	for _, v := range vals {
		d := v - avg
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(vals)-1))
}

func buildEquityDrawdown(curve []*model.EquityPoint) ([]*antv1.EquityPoint, []*antv1.DrawdownPoint) {
	if len(curve) == 0 {
		return nil, nil
	}
	eqProto := make([]*antv1.EquityPoint, len(curve))
	ddProto := make([]*antv1.DrawdownPoint, 0, len(curve))
	runningMax := 0.0
	for i, p := range curve {
		eq := p.Equity.InexactFloat64()
		eqProto[i] = &antv1.EquityPoint{
			Date:    p.Date,
			Equity:  math.Round(eq*100) / 100,
			Balance: math.Round(p.Balance.InexactFloat64()*100) / 100,
		}
		if eq > runningMax {
			runningMax = eq
		}
		dd := 0.0
		if runningMax > 0 {
			dd = (runningMax - eq) / runningMax * 100
		}
		ddProto = append(ddProto, &antv1.DrawdownPoint{
			Date:             p.Date,
			DrawdownPercent:  math.Round(dd*100) / 100,
		})
	}
	return eqProto, ddProto
}

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

	// Build prompt.
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
	NetProfit       float64 `json:"net_profit"`
	TotalTrades     int64   `json:"total_trades"`
	WinRate         float64 `json:"win_rate"`
	ProfitFactor    float64 `json:"profit_factor"`
	MaxDrawdown     float64 `json:"max_drawdown_percent"`
	SharpeRatio     float64 `json:"sharpe_ratio"`
	TopSymbols      []symbolSummary `json:"top_symbols"`
	DirectionBreakdown string   `json:"direction_breakdown"`
}

type symbolSummary struct {
	Symbol    string  `json:"symbol"`
	Profit    float64 `json:"profit"`
	Trades    int64   `json:"trades"`
	WinRate   float64 `json:"win_rate"`
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
	closeTag := fmt.Sprintf("</section>")
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
