package strategy

import (
	"context"
	"sync"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/mthub"
)

// PositionCache maintains a per-account position snapshot by subscribing
// to PositionSnapshotBroker (push-first).  Eliminates per-bar OpenedOrders
// polling from backfillContextStrings and MTAccountStateProvider.
type PositionCache struct {
	mu       sync.RWMutex
	snapshots map[string]*mthub.PositionSnapshot // accountID → latest snapshot
	log      *zap.Logger
}

// NewPositionCache creates a PositionCache.  Call Start to begin subscribing.
func NewPositionCache(log *zap.Logger) *PositionCache {
	if log == nil {
		log = zap.NewNop()
	}
	return &PositionCache{
		snapshots: make(map[string]*mthub.PositionSnapshot),
		log:      log,
	}
}

// Subscribe starts listening for position snapshots for the given account.
// The subscription runs in a goroutine until ctx is cancelled or unsub is called.
func (c *PositionCache) Subscribe(ctx context.Context, hub *mthub.MtHubService, accountID string) {
	if hub == nil {
		return
	}
	ch, unsub := hub.SubscribePositionSnapshots(ctx, accountID)
	go func() {
		defer unsub()
		for {
			select {
			case <-ctx.Done():
				return
			case snap, ok := <-ch:
				if !ok {
					return
				}
				c.mu.Lock()
				c.snapshots[accountID] = snap
				c.mu.Unlock()
			}
		}
	}()
}

// Unsubscribe removes the cached snapshot for an account (on strategy stop / disconnect).
func (c *PositionCache) Unsubscribe(accountID string) {
	c.mu.Lock()
	delete(c.snapshots, accountID)
	c.mu.Unlock()
}

// GetSnapshot returns the latest cached position snapshot for an account.
// Returns nil if no snapshot has been received yet.
func (c *PositionCache) GetSnapshot(accountID string) *mthub.PositionSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.snapshots[accountID]
}

// GetBalance returns the cached balance for an account, or decimal.Zero.
func (c *PositionCache) GetBalance(accountID string) decimal.Decimal {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if s, ok := c.snapshots[accountID]; ok {
		return s.Balance
	}
	return decimal.Zero
}

// GetEquity returns the cached equity for an account, or decimal.Zero.
func (c *PositionCache) GetEquity(accountID string) decimal.Decimal {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if s, ok := c.snapshots[accountID]; ok {
		return s.Equity
	}
	return decimal.Zero
}
