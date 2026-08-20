package strategy

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"alphaforge/internal/mthub"
)

const AccountSnapshotMaxAge = 90 * time.Second

// PositionCache stores the latest broker-authoritative account snapshot per account.
type PositionCache struct {
	mu         sync.RWMutex
	snapshots  map[string]*mthub.PositionSnapshot
	receivedAt map[string]time.Time
	log        *zap.Logger
}

// NewPositionCache creates a cache for push-based account snapshots.
func NewPositionCache(log *zap.Logger) *PositionCache {
	if log == nil {
		log = zap.NewNop()
	}
	return &PositionCache{
		snapshots:  make(map[string]*mthub.PositionSnapshot),
		receivedAt: make(map[string]time.Time),
		log:        log,
	}
}

// Subscribe starts listening for snapshots for one account until ctx is cancelled.
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
				c.put(snap, time.Now())
			}
		}
	}()
}

// put accepts authoritative financial snapshots and merges position-only updates into them.
func (c *PositionCache) put(snap *mthub.PositionSnapshot, receivedAt time.Time) {
	if snap == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	current := c.snapshots[snap.AccountID]
	if !snap.FinancialsAuthoritative {
		if current == nil || !current.FinancialsAuthoritative || !snap.PositionsAuthoritative {
			return
		}
		merged := *current
		merged.Positions = append([]mthub.PositionSnapshotItem(nil), snap.Positions...)
		merged.PositionsAuthoritative = true
		c.snapshots[snap.AccountID] = &merged
		return
	}
	if snap.CapturedAt.IsZero() || snap.FinancialsSource == "" {
		c.log.Warn("PositionCache: authoritative snapshot missing provenance",
			zap.String("account", snap.AccountID), zap.String("source", snap.FinancialsSource))
		return
	}
	merged := *snap
	if !snap.PositionsAuthoritative && current != nil && current.PositionsAuthoritative {
		merged.Positions = append([]mthub.PositionSnapshotItem(nil), current.Positions...)
		merged.PositionsAuthoritative = true
	}
	c.snapshots[snap.AccountID] = &merged
	c.receivedAt[snap.AccountID] = receivedAt
}

// PutSnapshot stores a snapshot with its local receipt time. It is intended for wiring and tests.
func (c *PositionCache) PutSnapshot(snap *mthub.PositionSnapshot, receivedAt time.Time) {
	c.put(snap, receivedAt)
}

// Unsubscribe removes the cached snapshot for an account.
func (c *PositionCache) Unsubscribe(accountID string) {
	c.mu.Lock()
	delete(c.snapshots, accountID)
	delete(c.receivedAt, accountID)
	c.mu.Unlock()
}

// GetSnapshot returns the latest accepted snapshot without a freshness check.
func (c *PositionCache) GetSnapshot(accountID string) *mthub.PositionSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.snapshots[accountID]
}

// GetFreshSnapshot returns a usable authoritative snapshot when both capture and receipt are fresh.
func (c *PositionCache) GetFreshSnapshot(accountID string, now time.Time) (*mthub.PositionSnapshot, bool) {
	c.mu.RLock()
	snap := c.snapshots[accountID]
	receivedAt := c.receivedAt[accountID]
	c.mu.RUnlock()
	if snap == nil || !snap.FinancialsAuthoritative || snap.CapturedAt.IsZero() || receivedAt.IsZero() {
		return nil, false
	}
	if now.Before(snap.CapturedAt) || now.Before(receivedAt) || now.Sub(snap.CapturedAt) > AccountSnapshotMaxAge || now.Sub(receivedAt) > AccountSnapshotMaxAge {
		return nil, false
	}
	return snap, true
}
