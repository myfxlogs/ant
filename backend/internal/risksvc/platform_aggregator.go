// Package risksvc provides the cross-account net exposure aggregator (M10-BASE-C4).
//
// Aggregates positions across all accounts to produce a platform-wide view:
//   - NetExposureBySymbol: net long/short by canonical symbol
//   - TotalMarginUsed: sum of margin across all accounts
//   - BrokerLimitUsage: margin usage as fraction of broker limit
//
// Refreshed every 5 seconds from the live state cache.

package risksvc

import (
	"sync"
	"time"
)

// PlatformExposure holds the aggregated platform-wide risk metrics.
type PlatformExposure struct {
	NetExposureBySymbol map[string]float64 // canonical → net volume (long+ short-)
	TotalGrossExposure  float64             // sum of absolute exposure
	TotalNetExposure    float64             // sum of signed exposure
	TotalMarginUsed     float64
	BrokerLimitUsage    map[string]float64 // broker → margin_used / limit
	AccountCount        int
	UpdatedAt           time.Time
}

// PlatformAggregator computes platform-wide risk from per-account positions.
type PlatformAggregator struct {
	mu        sync.RWMutex
	exposure  *PlatformExposure
	positions map[string]map[string]*AggregatorPosition // accountID → canonical → position
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
	return &PlatformAggregator{
		exposure:  &PlatformExposure{NetExposureBySymbol: map[string]float64{}, BrokerLimitUsage: map[string]float64{}},
		positions: map[string]map[string]*AggregatorPosition{},
	}
}

// UpdatePosition sets the position for an account+symbol.
func (a *PlatformAggregator) UpdatePosition(accountID string, pos *AggregatorPosition) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.positions[accountID]; !ok {
		a.positions[accountID] = map[string]*AggregatorPosition{}
	}
	a.positions[accountID][pos.Canonical] = pos
}

// ClearAccount removes all positions for an account (disconnect/close).
func (a *PlatformAggregator) ClearAccount(accountID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.positions, accountID)
}

// Recalculate rebuilds the platform-wide exposure snapshot.
func (a *PlatformAggregator) Recalculate(brokerLimits map[string]float64) *PlatformExposure {
	a.mu.Lock()
	defer a.mu.Unlock()

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

	// Broker limit usage.
	for broker, limit := range brokerLimits {
		if limit > 0 {
			exposure.BrokerLimitUsage[broker] = brokerMargins[broker] / limit
		}
	}

	a.exposure = exposure
	return exposure
}

// GetSnapshot returns the last computed platform exposure.
func (a *PlatformAggregator) GetSnapshot() *PlatformExposure {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.exposure
}

// NetExposureForSymbol returns the net exposure for a given symbol.
func (a *PlatformAggregator) NetExposureForSymbol(canonical string) float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.exposure.NetExposureBySymbol[canonical]
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
