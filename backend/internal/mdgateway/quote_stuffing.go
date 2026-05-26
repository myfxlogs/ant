// Package mdgateway provides quote stuffing detection (M10-BASE-F4).
//
// Detects abnormal tick rate spikes (Z-score > 4σ) and auto-pauses
// the affected symbol for 30 seconds to prevent downstream overload.
package mdgateway

import (
	"sync"
	"time"
)

// StuffingDetectorConfig configures quote stuffing detection.
type StuffingDetectorConfig struct {
	ZscoreThreshold float64       // tick rate Z-score above which stuffing is detected (default 4.0)
	PauseDuration   time.Duration // how long to pause a stuffed symbol (default 30s)
	WindowSize      int           // tick rate observation window (default 50)
}

// DefaultStuffingDetectorConfig returns standard parameters.
func DefaultStuffingDetectorConfig() StuffingDetectorConfig {
	return StuffingDetectorConfig{
		ZscoreThreshold: 4.0,
		PauseDuration:   30 * time.Second,
		WindowSize:      50,
	}
}

// StuffingDetector tracks per-symbol tick rates and detects quote stuffing.
type StuffingDetector struct {
	cfg StuffingDetectorConfig

	mu          sync.Mutex
	tickCounts  map[string]int       // key -> count since last tick
	tickRates   map[string][]float64 // key -> recent rates
	lastTickAt  map[string]time.Time // key -> last tick time
	pausedUntil map[string]time.Time // key -> pause expiration
}

// NewStuffingDetector creates a new quote stuffing detector.
func NewStuffingDetector(cfg StuffingDetectorConfig) *StuffingDetector {
	return &StuffingDetector{
		cfg:         cfg,
		tickCounts:  make(map[string]int),
		tickRates:   make(map[string][]float64),
		lastTickAt:  make(map[string]time.Time),
		pausedUntil: make(map[string]time.Time),
	}
}

// IsPaused returns true if the symbol is currently paused due to quote stuffing.
func (d *StuffingDetector) IsPaused(broker, canonical string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	key := broker + ":" + canonical
	if until, ok := d.pausedUntil[key]; ok {
		if Clk.Now().Before(until) {
			return true
		}
		delete(d.pausedUntil, key)
	}
	return false
}

// PausedSymbols returns all currently paused symbols.
func (d *StuffingDetector) PausedSymbols() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	now := Clk.Now()
	var result []string
	for key, until := range d.pausedUntil {
		if now.Before(until) {
			result = append(result, key)
		}
	}
	return result
}

// Observe records a tick for the given symbol and returns true if quote stuffing is detected.
func (d *StuffingDetector) Observe(broker, canonical string) (stuffed bool, zscore float64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := broker + ":" + canonical
	now := Clk.Now()

	// Check if currently paused.
	if until, ok := d.pausedUntil[key]; ok && now.Before(until) {
		return true, 0
	}

	// Count ticks; compute rate when we have accumulated data.
	d.tickCounts[key]++

	if lastAt, ok := d.lastTickAt[key]; ok {
		elapsed := now.Sub(lastAt).Seconds()
		if elapsed >= 1.0 {
			ratePerSec := float64(d.tickCounts[key]) / elapsed
			d.tickCounts[key] = 0
			d.lastTickAt[key] = now

			// Track rate history.
			d.tickRates[key] = append(d.tickRates[key], ratePerSec)
			if len(d.tickRates[key]) > d.cfg.WindowSize {
				d.tickRates[key] = d.tickRates[key][1:]
			}

			// Compute Z-score.
			if len(d.tickRates[key]) >= 10 {
				zscore = computeRateZscore(d.tickRates[key], ratePerSec)
				if zscore > d.cfg.ZscoreThreshold {
					d.pausedUntil[key] = now.Add(d.cfg.PauseDuration)
					recordStuffingDetected()
					return true, zscore
				}
			}
		}
	} else {
		d.lastTickAt[key] = now
		d.tickCounts[key] = 0
	}

	return false, 0
}

func computeRateZscore(history []float64, rate float64) float64 {
	var sum, sumSq float64
	for _, v := range history {
		sum += v
		sumSq += v * v
	}
	n := float64(len(history))
	mean := sum / n
	variance := sumSq/n - mean*mean
	if variance <= 0 {
		return 0
	}
	stdDev := 0.0
	// Simple sqrt via Newton.
	if variance > 0 {
		stdDev = variance
		for i := 0; i < 10; i++ {
			stdDev = (stdDev + variance/stdDev) / 2
		}
	}
	if stdDev == 0 {
		return 0
	}
	return (rate - mean) / stdDev
}
