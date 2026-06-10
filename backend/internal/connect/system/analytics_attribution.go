package system

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"

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
		SymbolPnls:       symbolPnLs,
		Direction:         dir,
		TradeDistribution: tradeDist,
		HourlyPnl:         hourlyPnl,
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
