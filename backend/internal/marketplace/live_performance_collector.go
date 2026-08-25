// live_performance_collector.go — LivePerformanceCollector extracted from live_performance.go.
package marketplace

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

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
