// Package execalgo provides execution algorithms that slice a large parent order
// into smaller child orders to reduce market impact (M11-13).
//
// Four algorithms are implemented:
//   - TWAP: Time-Weighted Average Price (equal slices over time)
//   - VWAP: Volume-Weighted Average Price (slices proportional to historical volume)
//   - POV:  Percentage of Volume (targets a fixed participation rate)
//   - Shortfall: Implementation Shortfall (front-loaded to minimize arrival price drift)
//
// Each algo implements the Algo interface and returns an ExecutionSchedule.

package execalgo

import (
	"fmt"
	"math"
	"time"

	"github.com/shopspring/decimal"
)

// ParentOrder is the original large order to be sliced.
type ParentOrder struct {
	Symbol       string
	Side         string // "buy" or "sell"
	TotalVolume  decimal.Decimal
	StartTime    time.Time
	EndTime      time.Time
	LimitPrice   decimal.Decimal
	ArrivalPrice decimal.Decimal // price at decision time (used by Shortfall)
}

// ChildOrder is a single slice of a parent order.
type ChildOrder struct {
	Sequence   int
	Volume     decimal.Decimal
	TargetTime time.Time
	LimitPrice decimal.Decimal
}

// Schedule is the execution plan produced by an algo.
type Schedule struct {
	Parent  ParentOrder
	Slices  []ChildOrder
	Algo    string
}

// TotalScheduledVolume returns the sum of all child order volumes.
func (s *Schedule) TotalScheduledVolume() decimal.Decimal {
	total := decimal.Zero
	for _, c := range s.Slices {
		total = total.Add(c.Volume)
	}
	return total
}

// Validate checks that the schedule is consistent with the parent order.
func (s *Schedule) Validate() error {
	if len(s.Slices) == 0 {
		return nil // empty schedule is valid
	}
	total := s.TotalScheduledVolume()
	if total.GreaterThan(s.Parent.TotalVolume.Add(decimal.NewFromFloat(0.0001))) {
		return fmt.Errorf("scheduled volume %s exceeds parent volume %s", total.String(), s.Parent.TotalVolume.String())
	}
	for i, c := range s.Slices {
		if !c.Volume.GreaterThan(decimal.Zero) {
			return fmt.Errorf("slice %d has non-positive volume %s", i, c.Volume.String())
		}
		if c.TargetTime.Before(s.Parent.StartTime) {
			return fmt.Errorf("slice %d target %v before start %v", i, c.TargetTime, s.Parent.StartTime)
		}
		if c.TargetTime.After(s.Parent.EndTime) {
			return fmt.Errorf("slice %d target %v after end %v", i, c.TargetTime, s.Parent.EndTime)
		}
	}
	return nil
}

// Algo is the interface for all execution algorithms.
type Algo interface {
	Name() string
	Schedule(parent ParentOrder) (*Schedule, error)
}

// VolumeProfile provides historical volume distribution fractions for VWAP/POV.
type VolumeProfile interface {
	// Fraction returns the fraction (0-1) of daily volume expected in the given time bucket.
	Fraction(symbol string, bucketStart time.Time) float64
}

// FlatVolumeProfile returns a uniform volume profile — all buckets get equal weight.
type FlatVolumeProfile struct{}

func (FlatVolumeProfile) Fraction(_ string, _ time.Time) float64 { return 1.0 }

// Duration returns the time span of the parent order.
func (p ParentOrder) Duration() time.Duration {
	return p.EndTime.Sub(p.StartTime)
}

// numSlicesFromInterval returns the number of slices for a given interval within a duration.
func numSlicesFromInterval(dur, interval time.Duration) int {
	n := int(math.Ceil(float64(dur) / float64(interval)))
	if n < 1 {
		n = 1
	}
	return n
}

// spreadSlices evenly distributes total volume across n buckets starting from startTime.
// Returns slices spaced by interval.
func spreadSlices(totalVol decimal.Decimal, startTime time.Time, interval time.Duration, n int) []ChildOrder {
	volPerSlice := totalVol.Div(decimal.NewFromInt(int64(n)))
	slices := make([]ChildOrder, n)
	for i := 0; i < n; i++ {
		slices[i] = ChildOrder{
			Sequence:   i,
			Volume:     volPerSlice,
			TargetTime: startTime.Add(interval * time.Duration(i+1)),
		}
	}
	return slices
}
