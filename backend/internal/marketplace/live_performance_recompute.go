// live_performance_recompute.go — recomputePerformanceSummary and helpers extracted from live_performance.go.
package marketplace

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func (s *Service) recomputePerformanceSummary(ctx context.Context, tx pgx.Tx, sid, aid uuid.UUID, today time.Time, prevEquity decimal.Decimal) error {
	var totalReturn, maxDrawdown decimal.Decimal
	var allTrades, allWins int32
	var firstDate, lastDate time.Time
	err := tx.QueryRow(ctx,
		`SELECT
			   COALESCE(SUM(daily_pnl), 0),
			   COALESCE(MAX(drawdown), 0),
			   COALESCE(SUM(total_trades), 0)::int,
			   COALESCE(SUM(winning_trades), 0)::int,
			   MIN(date), MAX(date)
			 FROM marketplace_live_performance
			 WHERE strategy_id = $1 AND account_id = $2`,
		sid, aid,
	).Scan(&totalReturn, &maxDrawdown, &allTrades, &allWins, &firstDate, &lastDate)
	if err != nil {
		return fmt.Errorf("marketplace: recompute summary: %w", err)
	}

	var winRate *decimal.Decimal
	if allTrades > 0 {
		wr := decimal.NewFromInt(int64(allWins)).Div(decimal.NewFromInt(int64(allTrades)))
		winRate = &wr
	}

	var annualReturn, sharpeRatio *decimal.Decimal
	daysTracked := int64(lastDate.Sub(firstDate).Hours() / 24)
	if daysTracked > 30 && prevEquity.GreaterThan(decimal.Zero) {
		ar := totalReturn.Div(decimal.NewFromInt(daysTracked)).Mul(decimal.NewFromInt(365))
		annualReturn = &ar

		var meanRet, stdRet decimal.Decimal
		err = tx.QueryRow(ctx,
			`SELECT COALESCE(AVG(daily_return), 0), COALESCE(STDDEV(daily_return), 0)
				 FROM marketplace_live_performance
				 WHERE strategy_id = $1 AND account_id = $2 AND daily_return != 0`,
			sid, aid,
		).Scan(&meanRet, &stdRet)
		if err == nil {
			if stdRet.GreaterThan(decimal.Zero) {
				sr := meanRet.Div(stdRet).Mul(decimal.NewFromFloat(15.8745))
				sharpeRatio = &sr
			}
		}
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO marketplace_live_performance_summary
			   (strategy_id, account_id, total_return, annual_return, max_drawdown, sharpe_ratio, win_rate,
			    total_trades, tracking_since, last_updated, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now())
		 ON CONFLICT (strategy_id) DO UPDATE SET
		   total_return = $3, annual_return = $4, max_drawdown = $5, sharpe_ratio = $6, win_rate = $7,
		   total_trades = $8, tracking_since = $9, last_updated = $10, updated_at = now()`,
		sid, aid, totalReturn,
		nullDec(annualReturn), maxDrawdown,
		nullDec(sharpeRatio), nullDec(winRate),
		allTrades, firstDate, today)
	if err != nil {
		return fmt.Errorf("marketplace: upsert summary: %w", err)
	}
	return nil
}

func nullDec(d *decimal.Decimal) interface{} {
	if d == nil {
		return nil
	}
	return *d
}

// LivePerformanceCollector receives push-based profit updates for linked accounts
// and records daily performance. It is called from the OnAccountProfit callback.
// Uses an in-memory cache to avoid DB queries on every profit update for unlinked accounts.
type LivePerformanceCollector struct {
	svc   *Service
	log   *zap.Logger
	cache map[string]string // accountID → strategyID
	mu    sync.RWMutex
}
