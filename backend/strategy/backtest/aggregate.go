package backtest

import (
	"time"

	"alphaforge/strategy/sdk"
)

// tfDuration converts a timeframe string to a duration.
func tfDuration(tf string) time.Duration {
	switch tf {
	case "M1":
		return time.Minute
	case "M5":
		return 5 * time.Minute
	case "M15":
		return 15 * time.Minute
	case "M30":
		return 30 * time.Minute
	case "H1":
		return time.Hour
	case "H2":
		return 2 * time.Hour
	case "H4":
		return 4 * time.Hour
	case "H6":
		return 6 * time.Hour
	case "H8":
		return 8 * time.Hour
	case "H12":
		return 12 * time.Hour
	case "D1":
		return 24 * time.Hour
	case "W1":
		return 7 * 24 * time.Hour
	default:
		return time.Hour
	}
}

// bucketStart calculates the timestamp of the bar bucket that contains the given
// timestamp for the target timeframe. For sub-daily timeframes, this is a simple
// epoch-aligned bucket. For D1, it aligns to UTC midnight. For W1, it aligns to
// the start of the week (Monday 00:00 UTC).
func bucketStart(ts int64, durMs int64, tf string) int64 {
	if tf == "W1" {
		// Align to Monday 00:00 UTC
		t := time.UnixMilli(ts).UTC()
		weekday := int(t.Weekday())
		if weekday == 0 {
			weekday = 6 // Sunday → last day of week
		} else {
			weekday-- // Monday=0, ..., Sunday=6
		}
		startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		weekStart := startOfDay.AddDate(0, 0, -weekday)
		return weekStart.UnixMilli()
	}
	// For all other timeframes, epoch-aligned bucket
	return (ts / durMs) * durMs
}

// aggregateBars aggregates primary timeframe bars into a higher timeframe.
// Bars are grouped by aligning their timestamps to the target timeframe's bucket boundary.
// Only bars in the input slice are used — the caller must ensure no future bars are included.
func aggregateBars(bars []sdk.Bar, tf string) []sdk.Bar {
	if len(bars) == 0 {
		return nil
	}
	dur := tfDuration(tf)
	durMs := dur.Milliseconds()
	var result []sdk.Bar
	var current *sdk.Bar
	var curBucket int64

	for i := range bars {
		bar := bars[i]
		barBucket := bucketStart(bar.Timestamp, durMs, tf)

		if current == nil || barBucket != curBucket {
			if current != nil {
				result = append(result, *current)
			}
			current = &sdk.Bar{
				Timestamp: barBucket,
				Open:      bar.Open,
				High:      bar.High,
				Low:       bar.Low,
				Close:     bar.Close,
				Volume:    bar.Volume,
			}
			curBucket = barBucket
		} else {
			if bar.High.GreaterThan(current.High) {
				current.High = bar.High
			}
			if bar.Low.LessThan(current.Low) {
				current.Low = bar.Low
			}
			current.Close = bar.Close
			current.Volume += bar.Volume
		}
	}
	if current != nil {
		result = append(result, *current)
	}
	return result
}
