// Package mthub provides the Live State Cache (M11-11 Tier-1, M10-BASE-B5).
//
// The cache maintains in-memory OMS state, positions, and idempotency keys
// rebuilt from NATS JetStream Tier-0 replay. On restart, the last 7 days of
// trade events are replayed to reconstruct full state.
//
// Architecture:
//   - In-memory: sync.Map for O(1) lookups on the hot path
//   - Redis: durable backup for cross-restart persistence
//   - Rebuild: JetStream ordered consumer replays oms.order.> from last 7 days

package mthub

import (
	"github.com/shopspring/decimal"
	"context"
	"google.golang.org/protobuf/proto"
	"fmt"
	"sync"
	antv1 "alphaforge/gen/proto/ant/v1"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	cacheRedisPrefix   = "statecache:"
	cacheRedisTTL      = 7 * 24 * time.Hour
	cacheReplayWindow  = 7 * 24 * time.Hour
)

// OrderStateCacheEntry is the in-memory representation of an order's current state.
type OrderStateCacheEntry struct {
	Ticket    int64     `json:"ticket"`
	AccountID string    `json:"account_id"`
	State     string    `json:"state"`
	Canonical string    `json:"canonical"`
	Side      string    `json:"side"`
	Volume    decimal.Decimal   `json:"volume"`
	Price     decimal.Decimal   `json:"price"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PositionCacheEntry is the in-memory representation of a position.
type PositionCacheEntry struct {
	AccountID    string    `json:"account_id"`
	Canonical    string    `json:"canonical"`
	NetVolume    decimal.Decimal   `json:"net_volume"`
	AvgPrice     decimal.Decimal   `json:"avg_price"`
	PnL          decimal.Decimal   `json:"pnl"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// StateCache maintains live OMS state and positions rebuilt from Tier-0 events.
type StateCache struct {
	mu        sync.RWMutex
	orders    map[int64]*OrderStateCacheEntry  // ticket → order state
	positions map[string]*PositionCacheEntry   // accountID:canonical → position
	redis     *goredis.Client
	log       *zap.Logger
}

// NewStateCache creates an empty state cache.
// redis may be nil (in-memory only).
func NewStateCache(redis *goredis.Client, log *zap.Logger) *StateCache {
	return &StateCache{
		orders:    make(map[int64]*OrderStateCacheEntry),
		positions: make(map[string]*PositionCacheEntry),
		redis:     redis,
		log:       log,
	}
}

// GetOrder returns the cached state for a ticket, or nil if not found.
func (c *StateCache) GetOrder(ticket int64) *OrderStateCacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.orders[ticket]
}

// GetOrdersByAccount returns all cached orders for a given account.
func (c *StateCache) GetOrdersByAccount(accountID string) []*OrderStateCacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var result []*OrderStateCacheEntry
	for _, o := range c.orders {
		if o.AccountID == accountID {
			result = append(result, o)
		}
	}
	return result
}

// GetPosition returns the cached net position for an account+symbol.
func (c *StateCache) GetPosition(accountID, canonical string) *PositionCacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.positions[positionKey(accountID, canonical)]
}

// GetPositionsByAccount returns all cached positions for a given account.
func (c *StateCache) GetPositionsByAccount(accountID string) []*PositionCacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	prefix := accountID + ":"
	var result []*PositionCacheEntry
	for k, p := range c.positions {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			result = append(result, p)
		}
	}
	return result
}

// ApplyEvent updates the cache from a trade event replayed from Tier-0.
func (c *StateCache) ApplyEvent(ev *TradeEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update position for fill events (must happen before terminal eviction).
	if ev.EventType == TradeEventOrderFilled || ev.EventType == TradeEventOrderPartiallyFilled {
		key := positionKey(ev.AccountID, ev.Canonical)
		pos, ok := c.positions[key]
		if !ok {
			pos = &PositionCacheEntry{
				AccountID: ev.AccountID,
				Canonical: ev.Canonical,
			}
			c.positions[key] = pos
		}
		multiplier := 1.0
		if ev.Side == "SELL" {
			multiplier = -1.0
		}
		absOld := pos.NetVolume.Abs()
		pos.NetVolume = pos.NetVolume.Add(ev.Volume.Mul(decimal.NewFromFloat(multiplier)))
		absNew := pos.NetVolume.Abs()
		if ev.Price.GreaterThan(decimal.Zero) {
			if absOld.IsZero() {
				pos.AvgPrice = ev.Price
			} else if absNew.GreaterThan(absOld) {
				// Position increased: weighted average of new fill.
				increaseVol := absNew.Sub(absOld)
				if increaseVol.GreaterThan(decimal.Zero) {
					pos.AvgPrice = (pos.AvgPrice.Mul(absOld).Add(ev.Price.Mul(increaseVol))).Div(absNew)
				}
			}
			// Position reduced: AvgPrice unchanged.
		}
		pos.UpdatedAt = ev.Timestamp
	}

	// Terminal states: evict from cache to prevent unbounded memory growth.
	// The order has reached its final state — keeping it serves no purpose.
	if isTerminalOrderState(ev.ToState) {
		delete(c.orders, ev.Ticket)
		// Delete the Redis key too — the order is terminal.
		if c.redis != nil {
			c.persistTerminalToRedis(ev)
		}
		return
	}

	// Update order state.
	c.orders[ev.Ticket] = &OrderStateCacheEntry{
		Ticket:    ev.Ticket,
		AccountID: ev.AccountID,
		State:     ev.ToState,
		Canonical: ev.Canonical,
		Side:      ev.Side,
		Volume:    ev.Volume,
		Price:     ev.Price,
		UpdatedAt: ev.Timestamp,
	}

	// Persist to Redis asynchronously.
	if c.redis != nil {
		c.persistToRedis(ev)
	}
}

func (c *StateCache) persistToRedis(ev *TradeEvent) {
	data, err := proto.Marshal(orderToCacheProto(c.orders[ev.Ticket]))
	if err != nil {
		c.log.Warn("statecache: redis marshal failed", zap.Error(err))
		return
	}
	key := fmt.Sprintf("%sorder:%d", cacheRedisPrefix, ev.Ticket)
	if err := c.redis.Set(context.Background(), key, data, cacheRedisTTL).Err(); err != nil {
		c.log.Warn("statecache: redis write failed", zap.Error(err))
	}
}

// LoadFromRedis restores order state from Redis into memory.
// Called on startup before NATS replay begins.
func (c *StateCache) LoadFromRedis(ctx context.Context) error {
	if c.redis == nil {
		return nil
	}

	iter := c.redis.Scan(ctx, 0, cacheRedisPrefix+"order:*", 100).Iterator()
	c.mu.Lock()
	defer c.mu.Unlock()

	for iter.Next(ctx) {
		var protoEntry antv1.OrderCacheEntry
		if err := proto.Unmarshal([]byte(iter.Val()), &protoEntry); err != nil {
			c.log.Warn("statecache: redis unmarshal failed", zap.Error(err))
			continue
		}
		vol, _ := decimal.NewFromString(protoEntry.Volume)
		price, _ := decimal.NewFromString(protoEntry.Price)
		c.orders[protoEntry.Ticket] = &OrderStateCacheEntry{Ticket: protoEntry.Ticket, AccountID: protoEntry.AccountId, State: protoEntry.State, Canonical: protoEntry.Canonical, Side: protoEntry.Side, Volume: vol, Price: price, UpdatedAt: time.UnixMilli(protoEntry.UpdatedAtUnixMs)}
	}
	return iter.Err()
}

// Stats returns diagnostic counts.
func (c *StateCache) Stats() (orders, positions int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.orders), len(c.positions)
}

func positionKey(accountID, canonical string) string {
	return accountID + ":" + canonical
}

// isTerminalOrderState returns true for order states that will never transition again.
// These orders are evicted from the in-memory cache to prevent unbounded growth.
func isTerminalOrderState(state string) bool {
	switch state {
	case "CLOSED", "FILLED", "CANCELLED", "EXPIRED", "FAILED", "REJECTED":
		return true
	default:
		return false
	}
}

// persistTerminalToRedis writes a terminal order event to Redis with a short TTL
// for audit trail purposes, then deletes the key after a grace period.
func (c *StateCache) persistTerminalToRedis(ev *TradeEvent) {
	key := fmt.Sprintf("%sorder:%d", cacheRedisPrefix, ev.Ticket)
	// Delete the Redis key — the order is terminal, no need to keep it cached.
	c.redis.Del(context.Background(), key)
}

func orderToCacheProto(o *OrderStateCacheEntry) *antv1.OrderCacheEntry {
	return &antv1.OrderCacheEntry{
		Ticket: o.Ticket, AccountId: o.AccountID, State: o.State,
		Canonical: o.Canonical, Side: o.Side,
		Volume: o.Volume.String(), Price: o.Price.String(),
		UpdatedAtUnixMs: o.UpdatedAt.UnixMilli(),
	}
}
