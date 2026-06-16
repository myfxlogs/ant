package system

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/repository"
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

	// Check cache — return immediately on hit.
	if s.cache != nil {
		if cached, _ := s.cache.GetAttribution(ctx, req.Msg.AccountId); cached != nil {
			return connect.NewResponse(cached), nil
		}
	}

	core := s.computeAttributionCore(ctx, accountID)

	// Sort symbol PnLs for display.
	sort.Slice(core.SymbolPnls, func(i, j int) bool {
		return core.SymbolPnls[i].NetProfit > core.SymbolPnls[j].NetProfit
	})

	// Add trade distribution + hourly PnL (not in core — not needed by AI report).
	now := time.Now()
	start := now.AddDate(-1, 0, 0)
	profits, err := s.repo.GetTradeProfitValues(ctx, accountID, start, now)
	if err != nil {
		s.log.Warn("analytics: get trade profit values failed", zap.Error(err))
	}
	core.TradeDistribution = buildTradeDistribution(profits)
	hourlyStats, err := s.repo.GetHourlyStats(ctx, accountID, start, now)
	if err != nil {
		s.log.Warn("analytics: get hourly stats for attribution failed", zap.Error(err))
	}
	hourlyPnl := make([]*antv1.HourlyPnL, 0, len(hourlyStats))
	for _, h := range hourlyStats {
		hourlyPnl = append(hourlyPnl, &antv1.HourlyPnL{
			Hour:   int32(h.HourStart),
			Profit: h.Profit.String(),
			Trades: int64(h.Trades),
			WinRate: math.Round(h.WinRate.InexactFloat64()*100) / 100,
		})
	}
	core.HourlyPnl = hourlyPnl

	if s.cache != nil {
		if err := s.cache.SetAttribution(ctx, req.Msg.AccountId, core); err != nil {
			s.log.Warn("analytics cache: set attribution failed", zap.Error(err))
		}
	}

	return connect.NewResponse(core), nil
}

// computeAttributionCore computes symbol PnLs and direction breakdown —
// the subset common to both GetAttributionAnalysis and AI report generation.
// Callers that need trade distribution or hourly PnL must add them after
// calling this method.
func (s *AnalyticsServer) computeAttributionCore(ctx context.Context, accountID uuid.UUID) *antv1.GetAttributionAnalysisResponse {
	now := time.Now()
	start := now.AddDate(-1, 0, 0)
	symbolStats, err := s.repo.GetSymbolStats(ctx, accountID, start, now)
	if err != nil {
		s.log.Warn("analytics: GetSymbolStats failed", zap.Error(err))
	}
	symbolPnLs := symbolStatsToSymbolPnLs(symbolStats)
	dirStats, err := s.repo.GetPnLByDirection(ctx, accountID, start, now)
	if err != nil {
		s.log.Warn("analytics: GetPnLByDirection failed", zap.Error(err))
	}
	return &antv1.GetAttributionAnalysisResponse{
		SymbolPnls: symbolPnLs,
		Direction:  buildDirectionBreakdown(dirStats),
	}
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
			dir.LongProfit = fmt.Sprintf("%.2f", s.Profit)
			dir.LongTrades = int64(s.Trades)
			dir.LongWinRate = math.Round(wr*100) / 100
		case "SELL":
			dir.ShortProfit = fmt.Sprintf("%.2f", s.Profit)
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
			b.MaxValue = fmt.Sprintf("%.2f", bounds[i])
		}
		if i > 0 {
			b.MinValue = fmt.Sprintf("%.2f", bounds[i-1])
		}
		dist = append(dist, b)
	}
	return &antv1.TradeDistribution{ProfitBuckets: dist}
}
