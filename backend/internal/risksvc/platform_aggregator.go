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
)

// PlatformExposure holds the aggregated platform-wide risk metrics.
type PlatformExposure struct {
	NetExposureBySymbol map[string]float64 // canonical -> net volume (long+ short-)
	TotalGrossExposure  float64             // sum of absolute exposure
	TotalNetExposure    float64             // sum of signed exposure
	TotalMarginUsed     float64
	BrokerLimitUsage    map[string]float64 // broker -> margin_used / limit
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

	brokerLimits map[string]float64
	stopCh       chan struct{}
}

// AggregatorPosition is the position data needed for aggregation.
type AggregatorPosition struct {
	Canonical string
	NetVolume float64
	Notional  float64
	Margin    float64
}

// NewPlatformAggregator creates an aggregator.
func NewPlatformAggregator() *PlatformAggregator {
	initial := &PlatformExposure{
		NetExposureBySymbol: map[string]float64{},
		BrokerLimitUsage:    map[string]float64{},
	}
	a := &PlatformAggregator{
		exposure:     initial,
		positions:    map[string]map[string]*AggregatorPosition{},
		brokerLimits: map[string]float64{},
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
func (a *PlatformAggregator) SetBrokerLimits(limits map[string]float64) {
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
		NetExposureBySymbol: map[string]float64{},
		BrokerLimitUsage:    map[string]float64{},
		AccountCount:        len(a.positions),
		UpdatedAt:           Clk.Now(),
	}

	brokerMargins := map[string]float64{}

	for _, positions := range a.positions {
		for _, pos := range positions {
			exposure.NetExposureBySymbol[pos.Canonical] += pos.NetVolume
			exposure.TotalGrossExposure += abs(pos.Notional)
			exposure.TotalNetExposure += pos.Notional
			exposure.TotalMarginUsed += pos.Margin
		}
	}

	for broker, limit := range limits {
		if limit > 0 {
			exposure.BrokerLimitUsage[broker] = brokerMargins[broker] / limit
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
func (a *PlatformAggregator) NetExposureForSymbol(canonical string) float64 {
	snap := a.GetSnapshot()
	if snap == nil {
		return 0
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
// call UpdatePosition or ClearAccount.
func (a *PlatformAggregator) Shutdown() {
	close(a.stopCh)
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
