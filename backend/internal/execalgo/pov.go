package execalgo

import (
	"errors"
	"math"
	"time"
)

// PovAlgo implements Percentage-of-Volume execution.
// It targets a fixed participation rate of expected market volume in each slice.
// Child volumes are capped so the total never exceeds the parent volume.
type PovAlgo struct {
	// ParticipationRate is the target fraction of market volume (0-1).
	// E.g., 0.05 means 5% participation.
	ParticipationRate float64
	// SliceInterval is the time between consecutive child orders.
	SliceInterval time.Duration
	// ExpectedVolumePerSlice is the forecast market volume in each time slice.
	ExpectedVolumePerSlice float64
}

// NewPov creates a POV algo with the given participation rate.
func NewPov(rate float64, interval time.Duration, expectedVolPerSlice float64) *PovAlgo {
	if rate <= 0 {
		rate = 0.05
	}
	if interval <= 0 {
		interval = time.Minute
	}
	return &PovAlgo{
		ParticipationRate:      rate,
		SliceInterval:          interval,
		ExpectedVolumePerSlice: expectedVolPerSlice,
	}
}

func (a *PovAlgo) Name() string { return "POV" }

func (a *PovAlgo) Schedule(parent ParentOrder) (*Schedule, error) {
	if parent.TotalVolume <= 0 {
		return nil, errors.New("pov: total volume must be positive")
	}
	dur := parent.Duration()
	if dur <= 0 {
		return nil, errors.New("pov: duration must be positive")
	}
	if a.ParticipationRate <= 0 {
		return nil, errors.New("pov: participation rate must be positive")
	}

	interval := a.SliceInterval
	if interval <= 0 {
		interval = time.Minute
	}
	n := numSlicesFromInterval(dur, interval)

	// Target volume per slice = participation rate × expected market volume
	targetPerSlice := a.ParticipationRate * a.ExpectedVolumePerSlice
	remaining := parent.TotalVolume

	slices := make([]ChildOrder, 0, n)
	for i := 0; i < n; i++ {
		if remaining <= 0 {
			break
		}
		vol := math.Min(targetPerSlice, remaining)
		vol = math.Round(vol*10000) / 10000
		if vol <= 0 {
			continue
		}
		slices = append(slices, ChildOrder{
			Sequence:   i,
			Volume:     vol,
			TargetTime: parent.StartTime.Add(interval * time.Duration(i+1)),
		})
		remaining -= vol
	}

	return &Schedule{
		Parent: parent,
		Slices: slices,
		Algo:   a.Name(),
	}, nil
}
