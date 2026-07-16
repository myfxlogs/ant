package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"

	"alphaforge/internal/model"
)

// monthlyDetailBaseFilter is the common WHERE clause excluding non-trade order types
// and balance/credit entries that lack a symbol.
// Used by every trade_records query in the monthly detail module.
const monthlyDetailBaseFilter = `
			AND order_type NOT IN ('balance', 'credit', 'BALANCE', 'CREDIT', 'Balance', 'Credit')
			AND symbol IS NOT NULL AND symbol != ''`

// ── Monthly Detail (drill-down) ──

// GetMonthlyDetailMetrics returns aggregated metrics for a single month.
func (r *AnalyticsRepository) GetMonthlyDetailMetrics(ctx context.Context, accountID uuid.UUID, year, month int) (*model.MonthlyDetailMetrics, error) {
	query := `
			SELECT
				COALESCE(SUM(profit), 0)                            AS net_return,
				COUNT(*)::int                                        AS total_trades,
				CASE WHEN COUNT(*) > 0
					THEN SUM(CASE WHEN profit > 0 THEN 1 ELSE 0 END)::float / COUNT(*) * 100
					ELSE 0
				END                                                  AS win_rate,
				CASE WHEN SUM(CASE WHEN profit < 0 THEN ABS(profit) ELSE 0 END) > 0
					THEN SUM(CASE WHEN profit > 0 THEN profit ELSE 0 END) /
					     SUM(CASE WHEN profit < 0 THEN ABS(profit) ELSE 0 END)
					ELSE 0
				END                                                  AS profit_factor,
				COALESCE(MAX(profit), 0)                             AS best_trade,
				COALESCE(MIN(profit), 0)                             AS worst_trade
			FROM trade_records
			WHERE account_id = $1
				AND EXTRACT(YEAR FROM close_time) = $2
				AND EXTRACT(MONTH FROM close_time) = $3`+monthlyDetailBaseFilter+`
		`
	m := &model.MonthlyDetailMetrics{}
	err := r.db.QueryRow(ctx, query, accountID, year, month).Scan(
		&m.NetReturn, &m.TotalTrades, &m.WinRate, &m.ProfitFactor, &m.BestTrade, &m.WorstTrade,
	)
	if err != nil {
		return nil, err
	}
	// Round monetary fields to 2 decimal places for display.
	m.NetReturn = m.NetReturn.Round(2)
	m.BestTrade = m.BestTrade.Round(2)
	m.WorstTrade = m.WorstTrade.Round(2)
	// Compute return percentage: NetReturn / starting_balance where available.
	// Fetch starting balance as the first equity value of the month.
	var startingBalance decimal.Decimal
	_ = r.db.QueryRow(ctx,
		`SELECT COALESCE(balance, equity) FROM account_balance_history
		 WHERE account_id = $1 AND date_trunc('month', ts) = make_date($2, $3, 1)
		 ORDER BY ts ASC LIMIT 1`, accountID, year, month).Scan(&startingBalance)
	if startingBalance.IsPositive() {
		m.ReturnPercent = m.NetReturn.Div(startingBalance).Mul(decimal.NewFromInt(100))
	}
	// Round for display consistency.
	m.WinRate = m.WinRate.Round(2)
	m.ProfitFactor = m.ProfitFactor.Round(2)
	return m, nil
}

// GetMonthlySymbolPnL returns per-symbol P&L for a single month, sorted by profit desc.
func (r *AnalyticsRepository) GetMonthlySymbolPnL(ctx context.Context, accountID uuid.UUID, year, month int) ([]*model.MonthlySymbolPnL, error) {
	query := `
			SELECT
				symbol,
				COALESCE(SUM(profit), 0) AS net_profit,
				COUNT(*)::int             AS trades,
				CASE WHEN COUNT(*) > 0
					THEN SUM(CASE WHEN profit > 0 THEN 1 ELSE 0 END)::float / COUNT(*) * 100
					ELSE 0
				END                        AS win_rate
			FROM trade_records
			WHERE account_id = $1
				AND EXTRACT(YEAR FROM close_time) = $2
				AND EXTRACT(MONTH FROM close_time) = $3`+monthlyDetailBaseFilter+`
			GROUP BY symbol
			ORDER BY net_profit DESC
		`
	rows, err := r.db.Query(ctx, query, accountID, year, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*model.MonthlySymbolPnL
	for rows.Next() {
		s := &model.MonthlySymbolPnL{}
		if err := rows.Scan(&s.Symbol, &s.NetProfit, &s.Trades, &s.WinRate); err != nil {
			return nil, err
		}
		s.NetProfit = s.NetProfit.Round(2)
		s.WinRate = s.WinRate.Round(2)
		results = append(results, s)
	}
	return results, rows.Err()
}

// GetMonthlyHoldingStats returns holding time stats for a single month (in hours).
func (r *AnalyticsRepository) GetMonthlyHoldingStats(ctx context.Context, accountID uuid.UUID, year, month int) (*model.MonthlyHoldingStats, error) {
	query := `
			SELECT
				COALESCE(AVG(EXTRACT(EPOCH FROM (close_time - open_time))) / 3600.0, 0)   AS avg_hours,
				COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (
					ORDER BY EXTRACT(EPOCH FROM (close_time - open_time))
				) / 3600.0, 0)                                                             AS median_hours,
				COALESCE(MAX(EXTRACT(EPOCH FROM (close_time - open_time))) / 3600.0, 0)   AS max_hours,
				COALESCE(MIN(EXTRACT(EPOCH FROM (close_time - open_time))) / 3600.0, 0)   AS min_hours
			FROM trade_records
			WHERE account_id = $1
				AND EXTRACT(YEAR FROM close_time) = $2
				AND EXTRACT(MONTH FROM close_time) = $3`+monthlyDetailBaseFilter+`
				AND open_time IS NOT NULL
				AND close_time IS NOT NULL
		`
	s := &model.MonthlyHoldingStats{}
	err := r.db.QueryRow(ctx, query, accountID, year, month).Scan(
		&s.AverageHours, &s.MedianHours, &s.MaxHours, &s.MinHours,
	)
	if err != nil {
		return nil, err
	}
	s.AverageHours = s.AverageHours.Round(2)
	s.MedianHours = s.MedianHours.Round(2)
	s.MaxHours = s.MaxHours.Round(2)
	s.MinHours = s.MinHours.Round(2)
	return s, nil
}

// ── Monthly Bonus (myfxbook-style panels) ──

// GetMonthlyBonus returns per-symbol risk ratios, symbol popularity, and
// holding-time split (long/short) for a single month — used by the
// myfxbook-style panels.  The three sub-queries run concurrently via errgroup.
// The aggregate risk_ratio is derived from metrics.ProfitFactor in the handler
// (same formula, avoids a redundant query).
func (r *AnalyticsRepository) GetMonthlyBonus(ctx context.Context, accountID uuid.UUID, year, month int) (*model.MonthlyAnalysisBonus, error) {
	bonus := &model.MonthlyAnalysisBonus{}
	eg, ctx := errgroup.WithContext(ctx)

	// 1. Per-symbol reward/risk ratios.
	eg.Go(func() error {
		rows, err := r.db.Query(ctx, `
				SELECT
					symbol,
					CASE WHEN SUM(CASE WHEN profit < 0 THEN ABS(profit) ELSE 0 END) > 0
						THEN SUM(CASE WHEN profit > 0 THEN profit ELSE 0 END) /
						     SUM(CASE WHEN profit < 0 THEN ABS(profit) ELSE 0 END)
						ELSE 0
					END AS risk_ratio
				FROM trade_records
				WHERE account_id = $1
					AND EXTRACT(YEAR FROM close_time) = $2
					AND EXTRACT(MONTH FROM close_time) = $3`+monthlyDetailBaseFilter+`
				GROUP BY symbol
				ORDER BY risk_ratio DESC
			`, accountID, year, month)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			row := &model.MonthlyBonusRiskRow{}
			if err := rows.Scan(&row.Symbol, &row.RiskRatio); err != nil {
				return err
			}
			row.RiskRatio = row.RiskRatio.Round(2)
			bonus.SymbolRisks = append(bonus.SymbolRisks, row)
		}
		return rows.Err()
	})

	// 2. Symbol popularity (trade share % via window function).
	eg.Go(func() error {
		rows, err := r.db.Query(ctx, `
				SELECT
					symbol,
					COUNT(*)::int                                                       AS trades,
					ROUND(COUNT(*)::numeric / SUM(COUNT(*)) OVER() * 100, 1)             AS share_percent
				FROM trade_records
				WHERE account_id = $1
					AND EXTRACT(YEAR FROM close_time) = $2
					AND EXTRACT(MONTH FROM close_time) = $3`+monthlyDetailBaseFilter+`
				GROUP BY symbol
				ORDER BY trades DESC
			`, accountID, year, month)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			row := &model.MonthlyBonusSymbol{}
			if err := rows.Scan(&row.Symbol, &row.Trades, &row.SharePercent); err != nil {
				return err
			}
			bonus.SymbolPopularity = append(bonus.SymbolPopularity, row)
		}
		return rows.Err()
	})

	// 3. Holding time split: BUY → bulls (long), SELL → short.
	eg.Go(func() error {
		rows, err := r.db.Query(ctx, `
				SELECT
					symbol,
					COALESCE(AVG(CASE WHEN order_type IN ('buy', 'BUY', 'Buy')
						THEN EXTRACT(EPOCH FROM (close_time - open_time)) END), 0)    AS bulls_seconds,
					COALESCE(AVG(CASE WHEN order_type IN ('sell', 'SELL', 'Sell')
						THEN EXTRACT(EPOCH FROM (close_time - open_time)) END), 0)    AS short_term_seconds
				FROM trade_records
				WHERE account_id = $1
					AND EXTRACT(YEAR FROM close_time) = $2
					AND EXTRACT(MONTH FROM close_time) = $3`+monthlyDetailBaseFilter+`
					AND open_time IS NOT NULL
					AND close_time IS NOT NULL
				GROUP BY symbol
				ORDER BY symbol
			`, accountID, year, month)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			row := &model.MonthlyBonusHoldingRow{}
			if err := rows.Scan(&row.Symbol, &row.BullsSeconds, &row.ShortTermSeconds); err != nil {
				return err
			}
			row.BullsSeconds = row.BullsSeconds.Round(2)
			row.ShortTermSeconds = row.ShortTermSeconds.Round(2)
			bonus.SymbolHoldings = append(bonus.SymbolHoldings, row)
		}
		return rows.Err()
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}
	// aggregate risk_ratio is derived from metrics.ProfitFactor in the handler
	// (same SQL formula — avoids a redundant 4th query)
	return bonus, nil
}
