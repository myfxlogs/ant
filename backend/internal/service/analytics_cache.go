package service

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
)

// AnalyticsSnapshot holds a pre-computed analytics summary for caching.
type AnalyticsSnapshot struct {
	TotalTrades    int32   `json:"total_trades"`
	WinRate        float64 `json:"win_rate"`
	ProfitFactor   float64 `json:"profit_factor"`
	NetProfit      float64 `json:"net_profit"`
	MaxDrawdownPct float64 `json:"max_drawdown_pct"`
	SharpeRatio    float64 `json:"sharpe_ratio"`
	UpdatedAt      int64   `json:"updated_at"`
}

// AnalyticsCache caches full AccountAnalyticsResponse values keyed by account ID,
// backed by Redis with a 30‑minute TTL.
type AnalyticsCache struct {
	mu    sync.RWMutex
	redis *goredis.Client
	log   *zap.Logger
}

// NewAnalyticsCache creates a new AnalyticsCache.
func NewAnalyticsCache(redis *goredis.Client, log *zap.Logger) *AnalyticsCache {
	return &AnalyticsCache{redis: redis, log: log}
}

// Get returns the cached analytics response for an account, or nil on cache miss.
func (c *AnalyticsCache) Get(ctx context.Context, accountID string) (*antv1.AccountAnalyticsResponse, error) {
	data, err := c.redis.Get(ctx, "analytics:"+accountID).Bytes()
	if err != nil {
		return nil, nil // cache miss
	}
	var resp antv1.AccountAnalyticsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		c.log.Warn("analytics cache: unmarshal failed",
			zap.String("account", accountID), zap.Error(err))
		return nil, nil
	}
	return &resp, nil
}

// Set stores an analytics response in Redis with a 30‑minute TTL.
func (c *AnalyticsCache) Set(ctx context.Context, accountID string, resp *antv1.AccountAnalyticsResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, "analytics:"+accountID, data, 30*time.Minute).Err()
}

// Invalidate clears the cache for an account after trade data changes.
func (c *AnalyticsCache) Invalidate(ctx context.Context, accountID string) {
	if err := c.redis.Del(ctx, "analytics:"+accountID).Err(); err != nil {
		c.log.Warn("analytics cache: del failed",
			zap.String("account", accountID), zap.Error(err))
	}
}
