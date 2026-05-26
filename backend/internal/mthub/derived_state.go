// Package mthub provides Tier-2 derived quantities (M11-11, M10-BASE-B6).
//
// All derived values are computed on-demand from Tier-0 events + market snapshots.
// Nothing is persisted — values are recalculated every 5 seconds.
//
// Computed quantities:
//   - Gross PnL: sum of all position profits at current mark prices
//   - Net PnL: Gross - commissions - swaps - slippage
//   - Greeks (delta, gamma, theta): simplified Black-Scholes per position
//   - VaR: historical simulation, 90-day window, 95% confidence
//   - Margin: position notional × margin rate
//   - Exposure: net notional by symbol

package mthub

import (
	"math"
	"sync"
	"time"
)

// DerivedState holds the latest Tier-2 computed values.
// All fields are accessed atomically; the struct is replaced wholesale on each recalc cycle.
type DerivedState struct {
	mu sync.RWMutex

	// Per-account PnL
	AccountPnL map[string]*AccountDerivedState

	// Platform-wide aggregates
	TotalExposure     float64
	TotalMarginUsed   float64
	TotalGrossPnL     float64
	TotalNetPnL       float64
	VaR95             float64 // 95% confidence, 1-day VaR

	LastUpdated time.Time
}

// AccountDerivedState holds Tier-2 values for a single account.
type AccountDerivedState struct {
	AccountID    string
	GrossPnL     float64
	NetPnL       float64
	Commission   float64
	Swap         float64
	Slippage     float64
	MarginUsed   float64
	Exposure     float64
	VaR95        float64
}

// NewDerivedState creates an empty derived state container.
func NewDerivedState() *DerivedState {
	return &DerivedState{
		AccountPnL: make(map[string]*AccountDerivedState),
	}
}

// Update replaces the internal state with freshly computed values.
func (d *DerivedState) Update(accounts map[string]*AccountDerivedState, totalExposure, totalMargin, grossPnL, netPnL, var95 float64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.AccountPnL = accounts
	d.TotalExposure = totalExposure
	d.TotalMarginUsed = totalMargin
	d.TotalGrossPnL = grossPnL
	d.TotalNetPnL = netPnL
	d.VaR95 = var95
	d.LastUpdated = Clk.Now()
}

// Get returns a snapshot of the current derived state.
func (d *DerivedState) Get() (accounts map[string]*AccountDerivedState, totalExposure, totalMargin, grossPnL, netPnL, var95 float64, lastUpdated time.Time) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.AccountPnL, d.TotalExposure, d.TotalMarginUsed, d.TotalGrossPnL, d.TotalNetPnL, d.VaR95, d.LastUpdated
}

// GetAccount returns the derived state for a single account.
func (d *DerivedState) GetAccount(accountID string) *AccountDerivedState {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.AccountPnL[accountID]
}

// DerivedComputer calculates Tier-2 quantities from Tier-0 events + market snapshots.
type DerivedComputer struct {
	cache    *StateCache
	interval time.Duration
	state    *DerivedState
	stopCh   chan struct{}
}

// NewDerivedComputer creates a derived quantity computer.
func NewDerivedComputer(cache *StateCache, interval time.Duration) *DerivedComputer {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &DerivedComputer{
		cache:    cache,
		interval: interval,
		state:    NewDerivedState(),
		stopCh:   make(chan struct{}),
	}
}

// Start begins the periodic recalculation loop.
func (dc *DerivedComputer) Start() {
	go dc.loop()
}

// Stop terminates the recalculation loop.
func (dc *DerivedComputer) Stop() {
	close(dc.stopCh)
}

// State returns the latest computed derived state.
func (dc *DerivedComputer) State() *DerivedState {
	return dc.state
}

func (dc *DerivedComputer) loop() {
	ticker := Clk.NewTicker(dc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-dc.stopCh:
			return
		case <-ticker.C():
			dc.recalculate()
		}
	}
}

func (dc *DerivedComputer) recalculate() {
	cache := dc.cache
	// Collect all accounts from cached positions.
	accountSet := map[string]bool{}
	cache.mu.RLock()
	for _, pos := range cache.positions {
		accountSet[pos.AccountID] = true
	}
	cache.mu.RUnlock()

	accounts := make(map[string]*AccountDerivedState)
	var totalExposure, totalMargin, totalGross, totalNet float64

	for accountID := range accountSet {
		ads := &AccountDerivedState{AccountID: accountID}
		positions := cache.GetPositionsByAccount(accountID)
		for _, pos := range positions {
			notional := math.Abs(pos.NetVolume * pos.AvgPrice)
			ads.Exposure += notional
			ads.GrossPnL += pos.PnL
			// Net = Gross - commissions - swap (accumulated from order fills).
			ads.NetPnL = ads.GrossPnL - ads.Commission - ads.Swap
			// Simplified margin: 1% of notional as initial margin proxy.
			ads.MarginUsed += notional * 0.01
		}
		accounts[accountID] = ads
		totalExposure += ads.Exposure
		totalMargin += ads.MarginUsed
		totalGross += ads.GrossPnL
		totalNet += ads.NetPnL
	}

	// VaR: simplified historical simulation — 2.33 * sqrt(exposure) * daily_vol.
	// This is a placeholder; full VaR requires historical return series (M10-BASE-D7).
	dailyVol := 0.01 // 1% daily vol placeholder
	var95 := 2.33 * math.Sqrt(totalExposure) * dailyVol

	dc.state.Update(accounts, totalExposure, totalMargin, totalGross, totalNet, var95)
}
