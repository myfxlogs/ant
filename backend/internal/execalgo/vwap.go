package execalgo

import (
	"errors"
	"math"
	"time"
)

// VwapAlgo implements Volume-Weighted Average Price execution.
// It distributes the parent volume across time buckets in proportion to
// the historical volume profile.
type VwapAlgo struct {
	// Profile provides historical volume fractions.
	// If nil, a FlatVolumeProfile is used (equivalent to TWAP).
	Profile VolumeProfile
	// NumBuckets is the number of time buckets. Defaults to 12 (5-min buckets per hour).
	NumBuckets int
}

// NewVwap creates a VWAP algo with the given volume profile and bucket count.
func NewVwap(profile VolumeProfile, numBuckets int) *VwapAlgo {
	if numBuckets <= 0 {
		numBuckets = 12
	}
	if profile == nil {
		profile = FlatVolumeProfile{}
	}
	return &VwapAlgo{Profile: profile, NumBuckets: numBuckets}
}

func (a *VwapAlgo) Name() string { return "VWAP" }

func (a *VwapAlgo) Schedule(parent ParentOrder) (*Schedule, error) {
	if parent.TotalVolume <= 0 {
		return nil, errors.New("vwap: total volume must be positive")
	}
	dur := parent.Duration()
	if dur <= 0 {
		return nil, errors.New("vwap: duration must be positive")
	}

	n := a.NumBuckets
	if n <= 0 {
		n = 12
	}
	bucketDur := dur / time.Duration(n)
	profile := a.Profile
	if profile == nil {
		profile = FlatVolumeProfile{}
	}

	// Collect fractions
	fracs := make([]float64, n)
	totalFrac := 0.0
	for i := 0; i < n; i++ {
		bucketStart := parent.StartTime.Add(bucketDur * time.Duration(i))
		f := profile.Fraction(parent.Symbol, bucketStart)
		if f < 0 {
			f = 0
		}
		fracs[i] = f
		totalFrac += f
	}

	// Normalize
	slices := make([]ChildOrder, n)
	for i := 0; i < n; i++ {
		vol := 0.0
		if totalFrac > 0 {
			vol = parent.TotalVolume * fracs[i] / totalFrac
		} else {
			vol = parent.TotalVolume / float64(n)
		}
		vol = math.Round(vol*10000) / 10000 // avoid float noise
		slices[i] = ChildOrder{
			Sequence:   i,
			Volume:     vol,
			TargetTime: parent.StartTime.Add(bucketDur * time.Duration(i+1)),
		}
	}

	return &Schedule{
		Parent: parent,
		Slices: slices,
		Algo:   a.Name(),
	}, nil
}
