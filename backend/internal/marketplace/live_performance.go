package marketplace

import (
	"context"
	"fmt"
	"sync"
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
// Validates that the caller is the strategy owner and that neither the strategy
// nor the account is already linked to something else.
func (s *Service) LinkLiveAccount(ctx context.Context, strategyID, accountID, userID string) error {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	aid, err := uuid.Parse(accountID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid account_id: %w", err)
	}

	// Check ownership: the strategy must belong to the caller.
	var ownerID uuid.UUID
	err = s.pg.QueryRow(ctx,
		`SELECT publisher_id FROM marketplace_strategies WHERE strategy_id = $1`,
		sid,
	).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("marketplace: strategy not found: %w", err)
	}
	uid, err := uuid.Parse(userID)
	if err != nil || uid != ownerID {
		return fmt.Errorf("marketplace: only the strategy owner can link a live account")
	}

	// Check the account is not already linked to another strategy.
	var alreadyLinked bool
	err = s.pg.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM marketplace_strategies WHERE linked_account_id = $1 AND strategy_id != $2)`,
		aid, sid).Scan(&alreadyLinked)
	if err == nil && alreadyLinked {
		return fmt.Errorf("marketplace: account is already linked to another strategy")
	}

	_, err = s.pg.Exec(ctx,
		`UPDATE marketplace_strategies SET linked_account_id = $2, updated_at = now() WHERE strategy_id = $1`,
		sid, aid)
	if err != nil {
		return fmt.Errorf("marketplace: link live account: %w", err)
	}

	// Refresh the in-memory collector cache so OnProfitUpdate picks up the new linkage immediately.
	if s.livePerfCollector != nil {
		s.livePerfCollector.RefreshCache()
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
		`SELECT date, daily_pnl, daily_return, equity, drawdown,
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
		if err := rows.Scan(&p.Date, &p.DailyPnL, &p.DailyReturn, &p.Equity, &p.Drawdown, &p.TotalTrades, &p.WinningTrades); err != nil {
			return nil, nil, err
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var summary *LivePerformanceSummary
	var totalReturn, maxDrawdown decimal.Decimal
	var annualReturn, sharpeRatio, winRate, avgMonthlyReturn *decimal.Decimal
	var totalTrades int32
	var trackingSince, lastUpdated time.Time
	err = s.pg.QueryRow(ctx,
		`SELECT total_return, annual_return, max_drawdown,
		        sharpe_ratio, win_rate,
		        total_trades, avg_monthly_return,
		        tracking_since, last_updated
		 FROM marketplace_live_performance_summary WHERE strategy_id = $1`,
		strategyID,
	).Scan(&totalReturn, &annualReturn, &maxDrawdown,
		&sharpeRatio, &winRate,
		&totalTrades, &avgMonthlyReturn,
		&trackingSince, &lastUpdated)
	if err == nil {
		summary = &LivePerformanceSummary{
			TotalReturn:      totalReturn,
			MaxDrawdown:      maxDrawdown,
			TotalTrades:      totalTrades,
			TrackingSince:    trackingSince,
			LastUpdated:      lastUpdated,
			AnnualReturn:     annualReturn,
			SharpeRatio:      sharpeRatio,
			WinRate:          winRate,
			AvgMonthlyReturn: avgMonthlyReturn,
		}
	}

	return points, summary, nil
}

// UpsertDailyPerformance records or updates a day's live performance for a strategy.
// Called from the OnAccountProfit callback when a linked account has new equity data.
// Computes dailyPnL, dailyReturn, and drawdown from the previous day's closing equity.
func (s *Service) UpsertDailyPerformance(ctx context.Context, strategyID, accountID string, equity decimal.Decimal) error {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	aid, err := uuid.Parse(accountID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid account_id: %w", err)
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("marketplace: upsert daily perf begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Get yesterday's closing equity to compute daily PnL and return.
	var prevEquity decimal.Decimal
	err = tx.QueryRow(ctx,
		`SELECT equity FROM marketplace_live_performance
			WHERE strategy_id = $1 AND account_id = $2 AND date < $3
			ORDER BY date DESC LIMIT 1`,
		sid, aid, today,
	).Scan(&prevEquity)
	if err != nil {
		prevEquity = equity // first record: no PnL for day 0
	}

	dailyPnL := equity.Sub(prevEquity)
	dailyReturn := decimal.Zero
	if prevEquity.GreaterThan(decimal.Zero) {
		dailyReturn = dailyPnL.Div(prevEquity)
	}

	// Get running peak equity for drawdown calculation.
	var peak decimal.Decimal
	if err := tx.QueryRow(ctx,
		`SELECT MAX(equity) FROM marketplace_live_performance
			WHERE strategy_id = $1 AND account_id = $2`,
		sid, aid,
	).Scan(&peak); err != nil {
		s.log.Warn("live performance: load peak equity failed", zap.Error(err))
	}
	if equity.GreaterThan(peak) {
		peak = equity
	}
	drawdown := decimal.Zero
	if peak.GreaterThan(decimal.Zero) {
		drawdown = peak.Sub(equity).Div(peak)
	}

	// Query today's trade counts from trade_records.
	var totalTrades, winningTrades int32
	dayStart := today
	dayEnd := today.Add(24 * time.Hour)
	err = tx.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN profit > 0 THEN 1 ELSE 0 END), 0)
			 FROM trade_records
			 WHERE account_id = $1 AND close_time >= $2 AND close_time < $3
			   AND order_type NOT IN ('balance','credit','BALANCE','CREDIT','Balance','Credit')`,
		aid, dayStart, dayEnd,
	).Scan(&totalTrades, &winningTrades)
	if err != nil {
		totalTrades, winningTrades = 0, 0 // trade_records may not exist yet
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO marketplace_live_performance (strategy_id, account_id, date, daily_pnl, daily_return, equity, drawdown, total_trades, winning_trades)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (strategy_id, account_id, date)
		 DO UPDATE SET daily_pnl = $4, daily_return = $5, equity = $6, drawdown = $7, total_trades = $8, winning_trades = $9`,
		sid, aid, today, dailyPnL, dailyReturn, equity, drawdown, totalTrades, winningTrades)
	if err != nil {
		return fmt.Errorf("marketplace: upsert daily perf: %w", err)
	}

	// Recompute summary from all daily records.
	var totalReturn, maxDrawdown decimal.Decimal
	var allTrades, allWins int32
	var firstDate, lastDate time.Time
	err = tx.QueryRow(ctx,
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

	// Compute win rate.
	var winRate *decimal.Decimal
	if allTrades > 0 {
		wr := decimal.NewFromInt(int64(allWins)).Div(decimal.NewFromInt(int64(allTrades)))
		winRate = &wr
	}

	// Compute annualized return and Sharpe ratio if we have enough data.
	var annualReturn, sharpeRatio *decimal.Decimal
	daysTracked := int64(lastDate.Sub(firstDate).Hours() / 24)
	if daysTracked > 30 && prevEquity.GreaterThan(decimal.Zero) {
		ar := totalReturn.Div(decimal.NewFromInt(daysTracked)).Mul(decimal.NewFromInt(365))
		annualReturn = &ar

		// Sharpe: mean(dailyReturn) / stddev(dailyReturn) * sqrt(252)
		var meanRet, stdRet decimal.Decimal
		err = tx.QueryRow(ctx,
			`SELECT COALESCE(AVG(daily_return), 0), COALESCE(STDDEV(daily_return), 0)
				 FROM marketplace_live_performance
				 WHERE strategy_id = $1 AND account_id = $2 AND daily_return != 0`,
			sid, aid,
		).Scan(&meanRet, &stdRet)
		if err == nil {
			if stdRet.GreaterThan(decimal.Zero) {
				sr := meanRet.Div(stdRet).Mul(decimal.NewFromFloat(15.8745)) // sqrt(252)
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

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("marketplace: upsert daily perf commit: %w", err)
	}

	// Notify subscribers of performance anomalies (push-first, no cron).
	var title string
	_ = s.pg.QueryRow(context.Background(), `SELECT COALESCE(title,'') FROM marketplace_strategies WHERE strategy_id = $1`, sid).Scan(&title)
	go s.notifyPerformanceAnomaly(context.WithoutCancel(ctx), sid, title, dailyReturn, drawdown)

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

func NewLivePerformanceCollector(svc *Service, log *zap.Logger) *LivePerformanceCollector {
	c := &LivePerformanceCollector{svc: svc, log: log, cache: make(map[string]string)}
	c.loadCache()
	return c
}

// loadCache preloads all linked account→strategy mappings from DB at startup.
func (c *LivePerformanceCollector) loadCache() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rows, err := c.svc.pg.Query(ctx,
		`SELECT linked_account_id::text, strategy_id::text
			 FROM marketplace_strategies
			 WHERE linked_account_id IS NOT NULL AND status = 'published'`)
	if err != nil {
		c.log.Warn("live performance: failed to preload cache", zap.Error(err))
		return
	}
	defer rows.Close()
	c.mu.Lock()
	defer c.mu.Unlock()
	for rows.Next() {
		var aid, sid string
		if err := rows.Scan(&aid, &sid); err == nil {
			c.cache[aid] = sid
		}
	}
}

// OnProfitUpdate is called from the OnAccountProfit pipeline callback.
// Uses the in-memory cache to skip unlinked accounts without any DB query.
func (c *LivePerformanceCollector) OnProfitUpdate(accountID string, equity, balance decimal.Decimal) {
	c.mu.RLock()
	strategyID, ok := c.cache[accountID]
	c.mu.RUnlock()
	if !ok {
		return // not linked — skip without DB query
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.svc.UpsertDailyPerformance(ctx, strategyID, accountID, equity); err != nil {
		c.log.Warn("live performance: upsert failed",
			zap.String("account", accountID),
			zap.String("strategy", strategyID),
			zap.Error(err))
	}
}

// RefreshCache reloads the linked account→strategy mapping. Called after LinkLiveAccount.
func (c *LivePerformanceCollector) RefreshCache() {
	c.loadCache()
}
