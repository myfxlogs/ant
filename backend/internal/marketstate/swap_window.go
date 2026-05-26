package marketstate

import "time"

// SwapWindowDetector identifies triple-swap days and rollover danger windows.
//
// Forex brokers apply triple swap (3x daily rate) on Wednesdays to account for
// weekend settlement. The rollover typically occurs at 5 PM EST (21:00–22:00 UTC).
// Trading during rollover can result in widened spreads and slippage.
type SwapWindowDetector struct {
	// RolloverHourUTC is the hour (0-23) when daily rollover occurs (default 21 = 5 PM EST).
	RolloverHourUTC int
	// DangerWindowMinutes is the minutes before and after rollover considered risky.
	DangerWindowMinutes int
}

// NewSwapWindowDetector creates a detector with sensible defaults.
func NewSwapWindowDetector() *SwapWindowDetector {
	return &SwapWindowDetector{
		RolloverHourUTC:     21,
		DangerWindowMinutes: 30,
	}
}

// IsTripleSwapDay returns true if today is a triple-swap settlement day.
// For forex: Wednesday (day 3) is the standard triple-swap day.
// For some brokers, it may be Tuesday or Thursday depending on the currency pair.
func (d *SwapWindowDetector) IsTripleSwapDay(t time.Time) bool {
	weekday := t.Weekday()
	// Standard: Wednesday UTC = triple swap for most pairs
	if weekday == time.Wednesday {
		return true
	}
	// Edge case: if broker settles on Tuesday (some exotic pairs)
	return false
}

// InRolloverWindow returns true if the given time is within the danger window
// around the daily rollover.
func (d *SwapWindowDetector) InRolloverWindow(t time.Time) bool {
	rollover := d.rolloverTime(t)
	diff := t.Sub(rollover)
	if diff < 0 {
		diff = -diff
	}
	return diff < time.Duration(d.DangerWindowMinutes)*time.Minute
}

// MinutesToRollover returns minutes until the next rollover (positive = future, negative = past).
func (d *SwapWindowDetector) MinutesToRollover(t time.Time) float64 {
	rollover := d.rolloverTime(t)
	// If we're past today's rollover, use tomorrow's
	if t.After(rollover) {
		rollover = rollover.Add(24 * time.Hour)
	}
	return rollover.Sub(t).Minutes()
}

func (d *SwapWindowDetector) rolloverTime(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), d.RolloverHourUTC, 0, 0, 0, time.UTC)
}
