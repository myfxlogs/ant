package service

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/proto"
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
	if err := proto.Unmarshal(data, &resp); err != nil {
		c.log.Warn("analytics cache: unmarshal failed",
			zap.String("account", accountID), zap.Error(err))
		return nil, nil
	}
	return &resp, nil
}

// Set stores an analytics response in Redis with a 30‑minute TTL.
func (c *AnalyticsCache) Set(ctx context.Context, accountID string, resp *antv1.AccountAnalyticsResponse) error {
	data, err := proto.Marshal(resp)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, "analytics:"+accountID, data, 30*time.Minute).Err()
}

// GetAttribution returns the cached attribution analysis for an account, or nil on cache miss.
func (c *AnalyticsCache) GetAttribution(ctx context.Context, accountID string) (*antv1.GetAttributionAnalysisResponse, error) {
	data, err := c.redis.Get(ctx, "analytics:attribution:"+accountID).Bytes()
	if err != nil {
		return nil, nil // cache miss
	}
	var resp antv1.GetAttributionAnalysisResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		c.log.Warn("analytics cache: attribution unmarshal failed",
			zap.String("account", accountID), zap.Error(err))
		return nil, nil
	}
	return &resp, nil
}

// SetAttribution stores an attribution analysis response in Redis with a 30‑minute TTL.
func (c *AnalyticsCache) SetAttribution(ctx context.Context, accountID string, resp *antv1.GetAttributionAnalysisResponse) error {
	data, err := proto.Marshal(resp)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, "analytics:attribution:"+accountID, data, 30*time.Minute).Err()
}

// GetRolling returns the cached rolling metrics for an account, or nil on cache miss.
func (c *AnalyticsCache) GetRolling(ctx context.Context, accountID string) (*antv1.GetRollingMetricsResponse, error) {
	data, err := c.redis.Get(ctx, "analytics:rolling:"+accountID).Bytes()
	if err != nil {
		return nil, nil // cache miss
	}
	var resp antv1.GetRollingMetricsResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		c.log.Warn("analytics cache: rolling unmarshal failed",
			zap.String("account", accountID), zap.Error(err))
		return nil, nil
	}
	return &resp, nil
}

// SetRolling stores a rolling metrics response in Redis with a 30‑minute TTL.
func (c *AnalyticsCache) SetRolling(ctx context.Context, accountID string, resp *antv1.GetRollingMetricsResponse) error {
	data, err := proto.Marshal(resp)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, "analytics:rolling:"+accountID, data, 30*time.Minute).Err()
}

// GetMonthlyDetail retrieves a cached monthly detail response for a given year/month.
func (c *AnalyticsCache) GetMonthlyDetail(ctx context.Context, accountID string, year, month int32) (*antv1.GetMonthlyDetailResponse, error) {
	key := fmt.Sprintf("analytics:monthly_detail:%s:%d:%d", accountID, year, month)
	data, err := c.redis.Get(ctx, key).Bytes()
	if err != nil {
		return nil, nil // cache miss
	}
	var resp antv1.GetMonthlyDetailResponse
	if err := proto.Unmarshal(data, &resp); err != nil {
		c.log.Warn("analytics cache: monthly_detail unmarshal failed",
			zap.String("account", accountID), zap.Error(err))
		return nil, nil
	}
	return &resp, nil
}

// SetMonthlyDetail stores a monthly detail response in Redis with a 30‑minute TTL.
func (c *AnalyticsCache) SetMonthlyDetail(ctx context.Context, accountID string, year, month int32, resp *antv1.GetMonthlyDetailResponse) error {
	key := fmt.Sprintf("analytics:monthly_detail:%s:%d:%d", accountID, year, month)
	data, err := proto.Marshal(resp)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, key, data, 30*time.Minute).Err()
}

// Invalidate clears all analytics caches for an account after trade data changes.
func (c *AnalyticsCache) Invalidate(ctx context.Context, accountID string) {
	keys := []string{
		"analytics:" + accountID,
		"analytics:attribution:" + accountID,
		"analytics:rolling:" + accountID,
	}
	// Also clear all monthly_detail keys for this account (wildcard pattern).
	if detailKeys, err := c.redis.Keys(ctx, "analytics:monthly_detail:"+accountID+":*").Result(); err == nil {
		keys = append(keys, detailKeys...)
	}
	if err := c.redis.Del(ctx, keys...).Err(); err != nil {
		c.log.Warn("analytics cache: del failed",
			zap.String("account", accountID), zap.Error(err))
	}
}
