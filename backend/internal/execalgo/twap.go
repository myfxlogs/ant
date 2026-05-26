package execalgo

import (
	"errors"
	"time"
)

// TwapAlgo implements Time-Weighted Average Price execution.
// It splits the parent order into equal-sized slices evenly spaced over the duration.
type TwapAlgo struct {
	// SliceInterval is the time between consecutive child orders.
	// If zero or negative, it defaults to 1 minute.
	SliceInterval time.Duration
}

// NewTwap creates a TWAP algo with the given slice interval.
func NewTwap(interval time.Duration) *TwapAlgo {
	if interval <= 0 {
		interval = time.Minute
	}
	return &TwapAlgo{SliceInterval: interval}
}

func (a *TwapAlgo) Name() string { return "TWAP" }

func (a *TwapAlgo) Schedule(parent ParentOrder) (*Schedule, error) {
	if parent.TotalVolume <= 0 {
		return nil, errors.New("twap: total volume must be positive")
	}
	dur := parent.Duration()
	if dur <= 0 {
		return nil, errors.New("twap: duration must be positive")
	}
	interval := a.SliceInterval
	if interval <= 0 {
		interval = time.Minute
	}

	n := numSlicesFromInterval(dur, interval)
	slices := spreadSlices(parent.TotalVolume, parent.StartTime, interval, n)

	return &Schedule{
		Parent: parent,
		Slices: slices,
		Algo:   a.Name(),
	}, nil
}
