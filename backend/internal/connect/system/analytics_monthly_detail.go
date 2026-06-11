package system

import (
	"context"
	"fmt"
	"math"
	"sort"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
)

// ── GetMonthlyDetail (drill-down) ──

// GetMonthlyDetail returns detailed metrics, per-symbol P&L, and holding time
// stats for a specific year/month. Used by the monthly analysis drill-down.
func (s *AnalyticsServer) GetMonthlyDetail(ctx context.Context, req *connect.Request[antv1.GetMonthlyDetailRequest]) (*connect.Response[antv1.GetMonthlyDetailResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", err))
	}
	if err := s.verifyAccountOwnership(ctx, req.Msg.AccountId); err != nil {
		return nil, err
	}

	year := req.Msg.Year
	month := req.Msg.Month
	if month < 1 || month > 12 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("month must be 1-12, got %d", month))
	}

	// Check cache — return immediately on hit.
	if s.cache != nil {
		if cached, _ := s.cache.GetMonthlyDetail(ctx, req.Msg.AccountId, year, month); cached != nil {
			return connect.NewResponse(cached), nil
		}
	}

	resp := s.computeMonthlyDetailCore(ctx, accountID, year, month)

	if s.cache != nil {
		if err := s.cache.SetMonthlyDetail(ctx, req.Msg.AccountId, year, month, resp); err != nil {
			s.log.Warn("analytics cache: set monthly_detail failed", zap.Error(err))
		}
	}

	return connect.NewResponse(resp), nil
}

// computeMonthlyDetailCore computes the monthly detail data — separated so
// it can be reused if needed (e.g. report generation).
func (s *AnalyticsServer) computeMonthlyDetailCore(ctx context.Context, accountID uuid.UUID, year, month int32) *antv1.GetMonthlyDetailResponse {
	ym := int(month)
	yy := int(year)

	metrics, err := s.repo.GetMonthlyDetailMetrics(ctx, accountID, yy, ym)
	if err != nil {
		s.log.Warn("analytics: GetMonthlyDetailMetrics failed", zap.Error(err))
		metrics = nil
	}

	symbolPnLs, err := s.repo.GetMonthlySymbolPnL(ctx, accountID, yy, ym)
	if err != nil {
		s.log.Warn("analytics: GetMonthlySymbolPnL failed", zap.Error(err))
	}

	holding, err := s.repo.GetMonthlyHoldingStats(ctx, accountID, yy, ym)
	if err != nil {
		s.log.Warn("analytics: GetMonthlyHoldingStats failed", zap.Error(err))
		holding = nil
	}

	resp := &antv1.GetMonthlyDetailResponse{}

	if metrics != nil {
		resp.Metrics = &antv1.MonthlyDetailMetrics{
			NetReturn:    metrics.NetReturn,
			ReturnPercent: metrics.ReturnPercent,
			TotalTrades:  int64(metrics.TotalTrades),
			WinRate:      metrics.WinRate,
			ProfitFactor: metrics.ProfitFactor,
			BestTrade:    metrics.BestTrade,
			WorstTrade:   metrics.WorstTrade,
		}
	}

	if len(symbolPnLs) > 0 {
		// Sort by net profit descending.
		sort.Slice(symbolPnLs, func(i, j int) bool {
			return symbolPnLs[i].NetProfit > symbolPnLs[j].NetProfit
		})
		for _, s := range symbolPnLs {
			resp.SymbolPnls = append(resp.SymbolPnls, &antv1.SymbolMonthlyPnL{
				Symbol:    s.Symbol,
				NetProfit: math.Round(s.NetProfit*100) / 100,
				Trades:    int64(s.Trades),
				WinRate:   math.Round(s.WinRate*100) / 100,
			})
		}
	}

	if holding != nil {
		resp.HoldingStats = &antv1.HoldingTimeStats{
			AverageHours: holding.AverageHours,
			MedianHours:  holding.MedianHours,
			MaxHours:     holding.MaxHours,
			MinHours:     holding.MinHours,
		}
	}

	return resp
}
