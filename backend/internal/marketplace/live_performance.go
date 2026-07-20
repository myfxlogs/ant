package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// LivePerformancePoint is a single day's live performance data.
type LivePerformancePoint struct {
	Date          time.Time
	DailyPnL      decimal.Decimal
	DailyReturn   decimal.Decimal
	Equity        decimal.Decimal
	Drawdown      decimal.Decimal
	TotalTrades   int32
	WinningTrades int32
}

// LivePerformanceSummary is the aggregated live performance for a strategy.
type LivePerformanceSummary struct {
	TotalReturn     decimal.Decimal
	AnnualReturn    *decimal.Decimal
	MaxDrawdown     decimal.Decimal
	SharpeRatio     *decimal.Decimal
	WinRate         *decimal.Decimal
	TotalTrades     int32
	AvgMonthlyReturn *decimal.Decimal
	TrackingSince   time.Time
	LastUpdated     time.Time
}

// LinkLiveAccount links a trading account to a published strategy for live tracking.
func (s *Service) LinkLiveAccount(ctx context.Context, strategyID, accountID string) error {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	aid, err := uuid.Parse(accountID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid account_id: %w", err)
	}
	_, err = s.pg.Exec(ctx,
		`UPDATE marketplace_strategies SET linked_account_id = $2, updated_at = now() WHERE strategy_id = $1`,
		sid, aid)
	if err != nil {
		return fmt.Errorf("marketplace: link live account: %w", err)
	}
	return nil
}

// GetLinkedAccountID returns the linked account ID for a strategy, if any.
func (s *Service) GetLinkedAccountID(ctx context.Context, strategyID string) (string, error) {
	var accountID *uuid.UUID
	err := s.pg.QueryRow(ctx,
		`SELECT linked_account_id FROM marketplace_strategies WHERE strategy_id = $1`,
		strategyID,
	).Scan(&accountID)
	if err != nil {
		return "", err
	}
	if accountID == nil {
		return "", nil
	}
	return accountID.String(), nil
}

// GetLivePerformance returns daily performance points and summary for a strategy.
func (s *Service) GetLivePerformance(ctx context.Context, strategyID string, limit int) ([]LivePerformancePoint, *LivePerformanceSummary, error) {
	if limit <= 0 {
		limit = 90
	}

	rows, err := s.pg.Query(ctx,
		`SELECT date, daily_pnl::text, daily_return::text, equity::text, drawdown::text,
		        total_trades, winning_trades
		 FROM marketplace_live_performance
		 WHERE strategy_id = $1
		 ORDER BY date DESC LIMIT $2`,
		strategyID, limit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var points []LivePerformancePoint
	for rows.Next() {
		var p LivePerformancePoint
		var pnlStr, retStr, eqStr, ddStr string
		if err := rows.Scan(&p.Date, &pnlStr, &retStr, &eqStr, &ddStr, &p.TotalTrades, &p.WinningTrades); err != nil {
			return nil, nil, err
		}
		p.DailyPnL, _ = decimal.NewFromString(pnlStr)
		p.DailyReturn, _ = decimal.NewFromString(retStr)
		p.Equity, _ = decimal.NewFromString(eqStr)
		p.Drawdown, _ = decimal.NewFromString(ddStr)
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var summary *LivePerformanceSummary
	var trStr, arStr, mdStr, srStr, wrStr, amrStr string
	var totalTrades int32
	var trackingSince, lastUpdated time.Time
	err = s.pg.QueryRow(ctx,
		`SELECT total_return::text, COALESCE(annual_return::text, ''), max_drawdown::text,
		        COALESCE(sharpe_ratio::text, ''), COALESCE(win_rate::text, ''),
		        total_trades, COALESCE(avg_monthly_return::text, ''),
		        tracking_since, last_updated
		 FROM marketplace_live_performance_summary WHERE strategy_id = $1`,
		strategyID,
	).Scan(&trStr, &arStr, &mdStr, &srStr, &wrStr, &totalTrades, &amrStr, &trackingSince, &lastUpdated)
	if err == nil {
		summary = &LivePerformanceSummary{
			TotalReturn:   parseDecSafe(trStr),
			MaxDrawdown:   parseDecSafe(mdStr),
			TotalTrades:   totalTrades,
			TrackingSince: trackingSince,
			LastUpdated:   lastUpdated,
		}
		if arStr != "" {
			d := parseDecSafe(arStr)
			summary.AnnualReturn = &d
		}
		if srStr != "" {
			d := parseDecSafe(srStr)
			summary.SharpeRatio = &d
		}
		if wrStr != "" {
			d := parseDecSafe(wrStr)
			summary.WinRate = &d
		}
		if amrStr != "" {
			d := parseDecSafe(amrStr)
			summary.AvgMonthlyReturn = &d
		}
	}

	return points, summary, nil
}

func parseDecSafe(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

// UpsertDailyPerformance records or updates a day's live performance for a strategy.
// Called from the OnAccountProfit callback when a linked account has new equity data.
func (s *Service) UpsertDailyPerformance(ctx context.Context, strategyID, accountID string, equity decimal.Decimal, dailyPnL decimal.Decimal, totalTrades, winningTrades int32) error {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	aid, err := uuid.Parse(accountID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid account_id: %w", err)
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	dailyReturn := decimal.Zero
	drawdown := decimal.Zero

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("marketplace: upsert daily perf begin: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO marketplace_live_performance (strategy_id, account_id, date, daily_pnl, daily_return, equity, drawdown, total_trades, winning_trades)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (strategy_id, account_id, date)
		 DO UPDATE SET daily_pnl = $4, daily_return = $5, equity = $6, drawdown = $7, total_trades = $8, winning_trades = $9`,
		sid, aid, today, dailyPnL.StringFixed(8), dailyReturn.StringFixed(6), equity.StringFixed(8), drawdown.StringFixed(6), totalTrades, winningTrades)
	if err != nil {
		return fmt.Errorf("marketplace: upsert daily perf: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO marketplace_live_performance_summary (strategy_id, account_id, total_return, max_drawdown, total_trades, tracking_since, last_updated, updated_at)
		 VALUES ($1, $2, $4, $5, $6, $3, $3, now())
		 ON CONFLICT (strategy_id) DO UPDATE SET
		   total_return = $4, max_drawdown = $5, total_trades = $6,
		   last_updated = $3, updated_at = now()`,
		sid, aid, today, dailyPnL.StringFixed(6), drawdown.StringFixed(6), totalTrades)
	if err != nil {
		return fmt.Errorf("marketplace: upsert summary: %w", err)
	}

	return tx.Commit(ctx)
}

// LivePerformanceCollector receives push-based profit updates for linked accounts
// and records daily performance. It is called from the OnAccountProfit callback.
type LivePerformanceCollector struct {
	svc *Service
	log *zap.Logger
}

func NewLivePerformanceCollector(svc *Service, log *zap.Logger) *LivePerformanceCollector {
	return &LivePerformanceCollector{svc: svc, log: log}
}

// OnProfitUpdate is called from the OnAccountProfit pipeline callback.
// It checks if the account is linked to a marketplace strategy and, if so,
// upserts the daily performance record.
func (c *LivePerformanceCollector) OnProfitUpdate(accountID string, equity, balance decimal.Decimal) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var strategyID string
	err := c.svc.pg.QueryRow(ctx,
		`SELECT strategy_id::text FROM marketplace_strategies WHERE linked_account_id = $1 AND status = 'published'`,
		accountID,
	).Scan(&strategyID)
	if err != nil {
		return // not linked or not published — skip silently
	}

	dailyPnL := equity.Sub(balance)
	if dailyPnL.LessThan(decimal.Zero) {
		dailyPnL = decimal.Zero
	}

	if err := c.svc.UpsertDailyPerformance(ctx, strategyID, accountID, equity, dailyPnL, 0, 0); err != nil {
		c.log.Warn("live performance: upsert failed",
			zap.String("account", accountID),
			zap.String("strategy", strategyID),
			zap.Error(err))
	}
}
