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
//
// LIVE-ORDER-REENTRY-1 B6: financials and positions freshness are tracked
// independently with BOTH captured-at (broker event time) and received-at
// (local receipt time). A financial-only refresh must NOT make stale positions
// appear fresh, and a retained replay must NOT resurrect old positions.
type PositionCache struct {
	mu sync.RWMutex
	// snapshots holds the latest merged snapshot per account (B7: no
	// ephemeral UpdateTicket metadata in retained state).
	snapshots map[string]*mthub.PositionSnapshot
	// financialsReceivedAt is the local receipt time of the latest
	// FinancialsAuthoritative update.
	financialsReceivedAt map[string]time.Time
	// positionsReceivedAt is the local receipt time of the latest
	// PositionsAuthoritative update.
	positionsReceivedAt map[string]time.Time
	log                 *zap.Logger
}

// NewPositionCache creates a cache for push-based account snapshots.
func NewPositionCache(log *zap.Logger) *PositionCache {
	if log == nil {
		log = zap.NewNop()
	}
	return &PositionCache{
		snapshots:            make(map[string]*mthub.PositionSnapshot),
		financialsReceivedAt: make(map[string]time.Time),
		positionsReceivedAt:  make(map[string]time.Time),
		log:                  log,
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

	// Positions-only update (OnOrderUpdate without AccountSummary financials).
	if !snap.FinancialsAuthoritative {
		if current == nil || !current.FinancialsAuthoritative || !snap.PositionsAuthoritative {
			return
		}
		merged := *current
		merged.Positions = append([]mthub.PositionSnapshotItem(nil), snap.Positions...)
		merged.PendingOrders = append([]mthub.PositionSnapshotItem(nil), snap.PendingOrders...)
		merged.PositionsAuthoritative = true
		// B6: carry positions provenance from the incoming event.
		merged.PositionsCapturedAt = snap.PositionsCapturedAt
		merged.PositionsSource = snap.PositionsSource
		// R7a: do NOT carry ephemeral trigger metadata into retained cache.
		// The barrier confirmation listener reads these from the live
		// PositionSnapshotBroker channel, not from the retained cache.
		// Keeping them in retained state would let AccountSummary replays
		// resurrect stale one-shot events (B7 violation).
		merged.UpdateTicket = 0
		merged.UpdateType = ""
		merged.UpdateMagic = 0
		c.snapshots[snap.AccountID] = &merged
		c.positionsReceivedAt[snap.AccountID] = receivedAt
		return
	}

	// Financials-authoritative update (AccountSummary or full OnOrderUpdate).
	if snap.CapturedAt.IsZero() || snap.FinancialsSource == "" {
		c.log.Warn("PositionCache: authoritative snapshot missing provenance",
			zap.String("account", snap.AccountID), zap.String("source", snap.FinancialsSource))
		return
	}
	merged := *snap
	if !snap.PositionsAuthoritative && current != nil && current.PositionsAuthoritative {
		// Financial-only refresh: preserve existing authoritative positions
		// and pending orders + their provenance (B6: don't refresh positions captured/received).
		merged.Positions = append([]mthub.PositionSnapshotItem(nil), current.Positions...)
		merged.PendingOrders = append([]mthub.PositionSnapshotItem(nil), current.PendingOrders...)
		merged.PositionsAuthoritative = true
		merged.PositionsCapturedAt = current.PositionsCapturedAt
		merged.PositionsSource = current.PositionsSource
	}
	// B7: clear ephemeral trigger metadata from retained cache state.
	merged.UpdateTicket = 0
	merged.UpdateType = ""
	merged.UpdateMagic = 0
	c.snapshots[snap.AccountID] = &merged
	c.financialsReceivedAt[snap.AccountID] = receivedAt
	if snap.PositionsAuthoritative {
		c.positionsReceivedAt[snap.AccountID] = receivedAt
	}
}

// PutSnapshot stores a snapshot with its local receipt time. It is intended for wiring and tests.
func (c *PositionCache) PutSnapshot(snap *mthub.PositionSnapshot, receivedAt time.Time) {
	c.put(snap, receivedAt)
}

// Unsubscribe removes the cached snapshot for an account.
func (c *PositionCache) Unsubscribe(accountID string) {
	c.mu.Lock()
	delete(c.snapshots, accountID)
	delete(c.financialsReceivedAt, accountID)
	delete(c.positionsReceivedAt, accountID)
	c.mu.Unlock()
}

// GetSnapshot returns the latest accepted snapshot without a freshness check.
func (c *PositionCache) GetSnapshot(accountID string) *mthub.PositionSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.snapshots[accountID]
}

// GetFreshFinancialSnapshot returns a snapshot with fresh financials only.
// Positions may be stale or absent. Use for pure financial display.
func (c *PositionCache) GetFreshFinancialSnapshot(accountID string, now time.Time) (*mthub.PositionSnapshot, bool) {
	c.mu.RLock()
	snap := c.snapshots[accountID]
	finRec := c.financialsReceivedAt[accountID]
	c.mu.RUnlock()
	if snap == nil || !snap.FinancialsAuthoritative || snap.CapturedAt.IsZero() || finRec.IsZero() {
		return nil, false
	}
	// B6: check both captured-at (broker event time) and received-at (local time).
	// Future timestamps, zero timestamps, and over-age all fail-closed.
	if snap.CapturedAt.IsZero() || now.Before(snap.CapturedAt) || now.Before(finRec) ||
		now.Sub(snap.CapturedAt) > AccountSnapshotMaxAge || now.Sub(finRec) > AccountSnapshotMaxAge {
		return nil, false
	}
	return snap, true
}

// GetFreshPositionSnapshot returns a snapshot with fresh positions only.
// Financials may be stale. Use for schedule positions display.
func (c *PositionCache) GetFreshPositionSnapshot(accountID string, now time.Time) (*mthub.PositionSnapshot, bool) {
	c.mu.RLock()
	snap := c.snapshots[accountID]
	posRec := c.positionsReceivedAt[accountID]
	c.mu.RUnlock()
	if snap == nil || !snap.PositionsAuthoritative || posRec.IsZero() {
		return nil, false
	}
	// R2: zero PositionsCapturedAt = fail-closed (no provenance = not fresh).
	// Do NOT fall back to receivedAt — that would allow stale retained
	// snapshots to appear fresh when replayed.
	posCap := snap.PositionsCapturedAt
	if posCap.IsZero() {
		return nil, false
	}
	if now.Before(posCap) || now.Before(posRec) ||
		now.Sub(posCap) > AccountSnapshotMaxAge || now.Sub(posRec) > AccountSnapshotMaxAge {
		return nil, false
	}
	return snap, true
}

// GetFreshTradingSnapshot returns a snapshot where BOTH financials and positions
// are fresh. Use for live VM evaluation and Risk Gate — they need both.
func (c *PositionCache) GetFreshTradingSnapshot(accountID string, now time.Time) (*mthub.PositionSnapshot, bool) {
	c.mu.RLock()
	snap := c.snapshots[accountID]
	finRec := c.financialsReceivedAt[accountID]
	posRec := c.positionsReceivedAt[accountID]
	c.mu.RUnlock()
	if snap == nil || !snap.FinancialsAuthoritative || !snap.PositionsAuthoritative ||
		snap.CapturedAt.IsZero() || finRec.IsZero() || posRec.IsZero() {
		return nil, false
	}
	// B6: check financials captured-at + received-at.
	if now.Before(snap.CapturedAt) || now.Before(finRec) ||
		now.Sub(snap.CapturedAt) > AccountSnapshotMaxAge || now.Sub(finRec) > AccountSnapshotMaxAge {
		return nil, false
	}
	// R2: zero PositionsCapturedAt = fail-closed (no provenance = not fresh).
	posCap := snap.PositionsCapturedAt
	if posCap.IsZero() {
		return nil, false
	}
	if now.Before(posCap) || now.Before(posRec) ||
		now.Sub(posCap) > AccountSnapshotMaxAge || now.Sub(posRec) > AccountSnapshotMaxAge {
		return nil, false
	}
	return snap, true
}

// GetFreshSnapshot is a legacy alias for GetFreshTradingSnapshot.
// Deprecated: use GetFreshTradingSnapshot / GetFreshFinancialSnapshot /
// GetFreshPositionSnapshot for explicit freshness semantics.
func (c *PositionCache) GetFreshSnapshot(accountID string, now time.Time) (*mthub.PositionSnapshot, bool) {
	return c.GetFreshTradingSnapshot(accountID, now)
}
