// Package mdgateway provides SessionClock — a time abstraction that understands
// trading sessions, bar boundaries, swap rollover windows, and holidays (M10-BASE-F2).
//
// Aligned with Lean Engine/DataFeeds/Enumerators/RealTimeProviderConsolidator.cs
// (bar boundary handling + timezone + session phase).
package mdgateway

import (
	"sync"
	"time"
)

// SessionClock provides session-aware time operations.
type SessionClock struct {
	mu sync.RWMutex

	// Broker clock offset from local system time.
	brokerOffsetMs int64

	// Holiday calendar (date strings in YYYY-MM-DD format).
	holidays map[string]bool

	// Session schedule (NY close 17:00 EST = 22:00 UTC).
	sessionCloseUTC int // hour of day in UTC when NY session closes (default 22)
	sessionOpenUTC  int // hour of day in UTC when Sydney session opens (default 22)

	// Swap rollover window.
	swapWindowStartUTC int // hour in UTC (default 21, = 17:00 EST)
	swapWindowEndUTC   int // hour in UTC (default 23, = 19:00 EST)
}

// DefaultSessionClock creates a SessionClock with standard forex session defaults.
func DefaultSessionClock() *SessionClock {
	return &SessionClock{
		holidays:           make(map[string]bool),
		sessionCloseUTC:    22, // 17:00 EST
		sessionOpenUTC:     22, // Sydney open = same as NY close in UTC
		swapWindowStartUTC: 21, // 17:00 EST
		swapWindowEndUTC:   23, // 19:00 EST
	}
}

// SetBrokerOffset records the broker clock offset from local time.
func (sc *SessionClock) SetBrokerOffset(offsetMs int64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.brokerOffsetMs = offsetMs
}

// BrokerOffsetMs returns the current broker clock offset.
func (sc *SessionClock) BrokerOffsetMs() int64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.brokerOffsetMs
}

// AddHoliday registers a holiday date.
func (sc *SessionClock) AddHoliday(date string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.holidays[date] = true
}

// RemoveHoliday unregisters a holiday date.
func (sc *SessionClock) RemoveHoliday(date string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.holidays, date)
}

// IsHoliday checks if the given time falls on a registered holiday.
func (sc *SessionClock) IsHoliday(t time.Time) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	dateStr := t.UTC().Format("2006-01-02")
	return sc.holidays[dateStr]
}

// IsWeekend checks if the given time falls on a weekend.
// Forex market closes Friday 17:00 EST (22:00 UTC) and reopens Sunday 17:00 EST (22:00 UTC).
func (sc *SessionClock) IsWeekend(t time.Time) bool {
	utc := t.UTC()
	wd := utc.Weekday()
	hour := utc.Hour()
	if wd == time.Sunday && hour < sc.sessionOpenUTC {
		return true
	}
	if wd == time.Saturday {
		return true
	}
	if wd == time.Friday && hour >= sc.sessionCloseUTC {
		return true
	}
	return false
}

// SessionPhase returns the current trading session phase.
func (sc *SessionClock) SessionPhase(t time.Time) SessionPhase {
	if sc.IsHoliday(t) {
		return PhaseHoliday
	}
	if sc.IsWeekend(t) {
		return PhaseWeekend
	}
	return PhaseOpen
}

// InSwapWindow checks if the given time falls within the swap rollover window.
// Swap is charged at 17:00 EST (22:00 UTC) with ±1h window for broker variation.
func (sc *SessionClock) InSwapWindow(t time.Time) bool {
	utc := t.UTC()
	hour := utc.Hour()
	return hour >= sc.swapWindowStartUTC && hour < sc.swapWindowEndUTC
}

// BarBoundary returns the bar boundary close timestamp for the given time and period.
// Aligned with Lean Consolidator bar boundary logic.
func (sc *SessionClock) BarBoundary(t time.Time, periodMs int64) int64 {
	ts := t.UnixMilli()
	bucket := ts / periodMs
	return (bucket + 1) * periodMs
}

// BrokerTime returns the broker's current time estimate.
func (sc *SessionClock) BrokerTime() time.Time {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return Clk.Now().Add(time.Duration(sc.brokerOffsetMs) * time.Millisecond)
}

// ClockSkewMs calculates the absolute clock skew between broker timestamp and local time.
func (sc *SessionClock) ClockSkewMs(brokerTsUnixMs int64) int64 {
	now := Clk.Now().UnixMilli()
	skew := now - brokerTsUnixMs
	if skew < 0 {
		skew = -skew
	}
	return skew
}
