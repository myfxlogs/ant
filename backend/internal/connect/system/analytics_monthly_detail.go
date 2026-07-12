package system

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	antv1 "alphaforge/gen/proto/ant/v1"
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

	// Run all 4 repo calls concurrently — single DB round trip.
	var (
		metrics    *antv1.MonthlyDetailMetrics
		symbolPnLs []*antv1.SymbolMonthlyPnL
		holding    *antv1.HoldingTimeStats
		bonus      *antv1.MonthlyBonus
	)
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		m, err := s.repo.GetMonthlyDetailMetrics(ctx, accountID, yy, ym)
		if err != nil {
			s.log.Warn("analytics: GetMonthlyDetailMetrics failed", zap.Error(err))
			return nil // warn-no-block: let other calls succeed
		}
		metrics = &antv1.MonthlyDetailMetrics{
			NetReturn:     m.NetReturn.StringFixed(2),
			ReturnPercent: m.ReturnPercent,
			TotalTrades:   int64(m.TotalTrades),
			WinRate:       m.WinRate,
			ProfitFactor:  m.ProfitFactor,
			BestTrade:     m.BestTrade.StringFixed(2),
			WorstTrade:    m.WorstTrade.StringFixed(2),
		}
		return nil
	})

	eg.Go(func() error {
		rows, err := s.repo.GetMonthlySymbolPnL(ctx, accountID, yy, ym)
		if err != nil {
			s.log.Warn("analytics: GetMonthlySymbolPnL failed", zap.Error(err))
			return nil
		}
		// SQL already ORDER BY net_profit DESC — no client-side sort needed.
		for _, r := range rows {
			symbolPnLs = append(symbolPnLs, &antv1.SymbolMonthlyPnL{
				Symbol:    r.Symbol,
				NetProfit: r.NetProfit.StringFixed(2), // already rounded in repo
				Trades:    int64(r.Trades),
				WinRate:   r.WinRate, // already rounded in repo
			})
		}
		return nil
	})

	eg.Go(func() error {
		h, err := s.repo.GetMonthlyHoldingStats(ctx, accountID, yy, ym)
		if err != nil {
			s.log.Warn("analytics: GetMonthlyHoldingStats failed", zap.Error(err))
			return nil
		}
		holding = &antv1.HoldingTimeStats{
			AverageHours: h.AverageHours,
			MedianHours:  h.MedianHours,
			MaxHours:     h.MaxHours,
			MinHours:     h.MinHours,
		}
		return nil
	})

	eg.Go(func() error {
		b, err := s.repo.GetMonthlyBonus(ctx, accountID, yy, ym)
		if err != nil {
			s.log.Warn("analytics: GetMonthlyBonus failed", zap.Error(err))
			return nil
		}
		bonus = &antv1.MonthlyBonus{
			// aggregate risk_ratio is derived from metrics.ProfitFactor below
			// (same SQL formula — avoids a redundant 4th query)
		}
		for _, s := range b.SymbolPopularity {
			bonus.SymbolPopularity = append(bonus.SymbolPopularity, &antv1.SymbolPopularityItem{
				Symbol:       s.Symbol,
				Trades:       int64(s.Trades),
				SharePercent: s.SharePercent,
			})
		}
		for _, r := range b.SymbolRisks {
			bonus.SymbolRisks = append(bonus.SymbolRisks, &antv1.SymbolRiskItem{
				Symbol:    r.Symbol,
				RiskRatio: r.RiskRatio,
			})
		}
		for _, h := range b.SymbolHoldings {
			bonus.SymbolHoldingSplit = append(bonus.SymbolHoldingSplit, &antv1.SymbolHoldingSplit{
				Symbol:           h.Symbol,
				BullsSeconds:     h.BullsSeconds,
				ShortTermSeconds: h.ShortTermSeconds,
			})
		}
		return nil
	})

	_ = eg.Wait() // each goroutine already logs its own warnings

	// Derive aggregate risk_ratio from metrics.ProfitFactor (same SQL formula).
	if bonus != nil && metrics != nil {
		bonus.RiskRatio = metrics.ProfitFactor
	}

	resp := &antv1.GetMonthlyDetailResponse{
		Metrics:      metrics,
		SymbolPnls:   symbolPnLs,
		HoldingStats: holding,
		Bonus:        bonus,
	}
	return resp
}
