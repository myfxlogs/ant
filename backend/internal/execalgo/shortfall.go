package execalgo

import (
	"errors"
	"math"
	"time"
)

// ShortfallAlgo implements Implementation Shortfall execution.
// It front-loads the schedule to minimize the expected slippage from arrival price.
// Higher urgency produces more aggressive front-loading.
//
// The schedule uses exponential decay weights: early slices are larger and later
// slices taper off. This balances market impact (concentrated trading) against
// timing risk (delay between decision and execution).
type ShortfallAlgo struct {
	// Urgency controls front-loading aggressiveness (0-1).
	// 0 = near-uniform (like TWAP), 1 = all volume in first slice.
	Urgency float64
	// NumSlices is the number of child orders. Defaults to 10.
	NumSlices int
}

// NewShortfall creates an Implementation Shortfall algo.
func NewShortfall(urgency float64, numSlices int) *ShortfallAlgo {
	if urgency < 0 {
		urgency = 0
	}
	if urgency > 1 {
		urgency = 1
	}
	if numSlices <= 0 {
		numSlices = 10
	}
	return &ShortfallAlgo{Urgency: urgency, NumSlices: numSlices}
}

func (a *ShortfallAlgo) Name() string { return "ImplementationShortfall" }

func (a *ShortfallAlgo) Schedule(parent ParentOrder) (*Schedule, error) {
	if parent.TotalVolume <= 0 {
		return nil, errors.New("shortfall: total volume must be positive")
	}
	dur := parent.Duration()
	if dur <= 0 {
		return nil, errors.New("shortfall: duration must be positive")
	}

	n := a.NumSlices
	if n <= 0 {
		n = 10
	}

	sliceDur := dur / time.Duration(n)
	weights := make([]float64, n)
	totalWeight := 0.0

	for i := 0; i < n; i++ {
		// Exponential decay: w(i) = exp(-urgency * i / (n-1))
		// urgency=0: all weights ~1.0 (uniform)
		// urgency=1: steep decay (most weight on early slices)
		var w float64
		if n == 1 {
			w = 1.0
		} else {
			w = math.Exp(-a.Urgency * float64(i) / float64(n-1))
		}
		weights[i] = w
		totalWeight += w
	}

	remaining := parent.TotalVolume
	slices := make([]ChildOrder, 0, n)
	for i := 0; i < n; i++ {
		var vol float64
		if i == n-1 {
			// Last slice takes the remainder to avoid float accumulation gaps.
			vol = remaining
		} else {
			vol = math.Round(parent.TotalVolume*weights[i]/totalWeight*10000) / 10000
		}
		if vol <= 0 {
			continue
		}
		vol = math.Min(vol, remaining)
		slices = append(slices, ChildOrder{
			Sequence:   i,
			Volume:     vol,
			TargetTime: parent.StartTime.Add(sliceDur * time.Duration(i+1)),
		})
		remaining -= vol
	}

	return &Schedule{
		Parent: parent,
		Slices: slices,
		Algo:   a.Name(),
	}, nil
}
