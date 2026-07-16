// Package risksvc provides the cross-account net exposure aggregator (M10-BASE-C4).
//
// Aggregates positions across all accounts to produce a platform-wide view:
//   - NetExposureBySymbol: net long/short by canonical symbol
//   - TotalMarginUsed: sum of margin across all accounts
//   - BrokerLimitUsage: margin usage as fraction of broker limit
//
// Refresh loop (B-1.4): UpdatePosition / ClearAccount mark dirty=true;
// a 5s ticker goroutine checks dirty and runs Recalculate, then atomically
// swaps the snapshot.  This avoids O(N*M) recalculation on every position
// change.

package risksvc

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/shopspring/decimal"
)

// PlatformExposure holds the aggregated platform-wide risk metrics.
type PlatformExposure struct {
	NetExposureBySymbol map[string]decimal.Decimal // canonical -> net volume (long+ short-)
	TotalGrossExposure  decimal.Decimal             // sum of absolute exposure
	TotalNetExposure    decimal.Decimal             // sum of signed exposure
	TotalMarginUsed     decimal.Decimal
	BrokerLimitUsage    map[string]float64 // broker -> margin_used / limit (ratio, stays float64)
	AccountCount        int
	UpdatedAt           time.Time
}

// PlatformAggregator computes platform-wide risk from per-account positions.
type PlatformAggregator struct {
	mu        sync.RWMutex
	dirty     bool
	exposure  *PlatformExposure
	positions map[string]map[string]*AggregatorPosition // accountID -> canonical -> position

	snapshot unsafe.Pointer // *PlatformExposure — atomically swapped by refresh loop

	brokerLimits map[string]decimal.Decimal
	stopCh       chan struct{}
}

// AggregatorPosition is the position data needed for aggregation.
type AggregatorPosition struct {
	Canonical string
	NetVolume decimal.Decimal
	Notional  decimal.Decimal
	Margin    decimal.Decimal
	Broker    string // broker name for per-broker limit usage tracking
}

// NewPlatformAggregator creates an aggregator.
func NewPlatformAggregator() *PlatformAggregator {
	initial := &PlatformExposure{
		NetExposureBySymbol: map[string]decimal.Decimal{},
		BrokerLimitUsage:    map[string]float64{},
	}
	a := &PlatformAggregator{
		exposure:     initial,
		positions:    map[string]map[string]*AggregatorPosition{},
		brokerLimits: map[string]decimal.Decimal{},
		stopCh:       make(chan struct{}),
	}
	atomic.StorePointer(&a.snapshot, unsafe.Pointer(initial))
	return a
}

// UpdatePosition sets the position for an account+symbol.
// Marks dirty; the next refresh tick will recalculate.
func (a *PlatformAggregator) UpdatePosition(accountID string, pos *AggregatorPosition) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.positions[accountID]; !ok {
		a.positions[accountID] = map[string]*AggregatorPosition{}
	}
	a.positions[accountID][pos.Canonical] = pos
	a.dirty = true
}

// ClearAccount removes all positions for an account (disconnect/close).
// Marks dirty; the next refresh tick will recalculate.
func (a *PlatformAggregator) ClearAccount(accountID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.positions, accountID)
	a.dirty = true
}

// SetBrokerLimits replaces the broker limit map used by the refresh loop.
func (a *PlatformAggregator) SetBrokerLimits(limits map[string]decimal.Decimal) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.brokerLimits = limits
	a.dirty = true
}

// Recalculate rebuilds the platform-wide exposure snapshot.
// Caller must hold a.mu (at least RLock, but Lock is fine).
func (a *PlatformAggregator) Recalculate() *PlatformExposure {
	limits := a.brokerLimits
	exposure := &PlatformExposure{
		NetExposureBySymbol: map[string]decimal.Decimal{},
		BrokerLimitUsage:    map[string]float64{},
		AccountCount:        len(a.positions),
		UpdatedAt:           Clk.Now(),
	}

	brokerMargins := map[string]decimal.Decimal{}

	for _, positions := range a.positions {
		for _, pos := range positions {
			exposure.NetExposureBySymbol[pos.Canonical] = exposure.NetExposureBySymbol[pos.Canonical].Add(pos.NetVolume)
			exposure.TotalGrossExposure = exposure.TotalGrossExposure.Add(pos.Notional.Abs())
			exposure.TotalNetExposure = exposure.TotalNetExposure.Add(pos.Notional)
			exposure.TotalMarginUsed = exposure.TotalMarginUsed.Add(pos.Margin)
			if pos.Broker != "" {
				brokerMargins[pos.Broker] = brokerMargins[pos.Broker].Add(pos.Margin)
			}
		}
	}

	for broker, limit := range limits {
		if limit.GreaterThan(decimal.Zero) {
			exposure.BrokerLimitUsage[broker] = brokerMargins[broker].Div(limit).InexactFloat64()
		}
	}

	a.exposure = exposure
	atomic.StorePointer(&a.snapshot, unsafe.Pointer(exposure))
	a.dirty = false
	return exposure
}

// GetSnapshot returns the last computed platform exposure (lock-free).
func (a *PlatformAggregator) GetSnapshot() *PlatformExposure {
	return (*PlatformExposure)(atomic.LoadPointer(&a.snapshot))
}

// NetExposureForSymbol returns the net exposure for a given symbol (lock-free).
func (a *PlatformAggregator) NetExposureForSymbol(canonical string) decimal.Decimal {
	snap := a.GetSnapshot()
	if snap == nil {
		return decimal.Zero
	}
	return snap.NetExposureBySymbol[canonical]
}

// StartRefreshLoop begins a background goroutine that checks dirty every
// interval and runs Recalculate when needed.  Call Shutdown to stop.
func (a *PlatformAggregator) StartRefreshLoop(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				a.mu.Lock()
				if a.dirty {
					a.Recalculate()
				}
				a.mu.Unlock()
			case <-a.stopCh:
				return
			}
		}
	}()
}

// Shutdown stops the refresh loop. After Shutdown, callers should not
// call UpdatePosition or ClearAccount. Safe to call multiple times.
func (a *PlatformAggregator) Shutdown() {
	a.mu.Lock()
	defer a.mu.Unlock()
	select {
	case <-a.stopCh:
		// Already closed.
	default:
		close(a.stopCh)
	}
}

func abs(d decimal.Decimal) decimal.Decimal {
	return d.Abs()
}
