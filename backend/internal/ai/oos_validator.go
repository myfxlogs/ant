// oos_validator.go: out-of-sample validation for parameter optimization.
// Splits time window 70/30, detects overfitting via degradation.

package ai

import (
	"math"
	"time"
)

// OOSResult contains the out-of-sample validation output.
type OOSResult struct {
	InSampleScore  float64
	OOSScore       float64
	Degradation    float64 // (IS - OOS) / IS, 0 = no degradation
	IsOverfit      bool
	InSampleStart  time.Time
	InSampleEnd    time.Time
	OOSStart       time.Time
	OOSEnd         time.Time
}

// OOSValidator handles train/test splits for overfit detection.
type OOSValidator struct {
	SplitRatio      float64 // default 0.7
	MaxDegradation  float64 // default 0.4 (40%)
	MinTrainDays    int     // minimum days needed for split
	MinOOSDays      int     // minimum OOS days
}

// DefaultOOSValidator returns a validator with standard settings.
func DefaultOOSValidator() *OOSValidator {
	return &OOSValidator{
		SplitRatio:     0.7,
		MaxDegradation: 0.4,
		MinTrainDays:   30,
		MinOOSDays:     7,
	}
}

// ComputeWindows splits the time range into in-sample (70%) and out-of-sample (30%).
// Returns zero OOSResult if the window is too short for a split.
func (v *OOSValidator) ComputeWindows(start, end time.Time) *OOSResult {
	totalDays := end.Sub(start).Hours() / 24
	if totalDays < float64(v.MinTrainDays+v.MinOOSDays) {
		return nil // not enough data for split
	}
	splitSeconds := int64(float64(end.Unix()-start.Unix()) * v.SplitRatio)
	isEnd := start.Add(time.Duration(splitSeconds) * time.Second)
	return &OOSResult{
		InSampleStart: start,
		InSampleEnd:   isEnd,
		OOSStart:      isEnd,
		OOSEnd:        end,
	}
}

// Validate compares in-sample and out-of-sample scores to detect overfitting.
func (v *OOSValidator) Validate(isScore, oosScore float64) *OOSResult {
	if isScore <= 0 {
		return &OOSResult{InSampleScore: isScore, OOSScore: oosScore, IsOverfit: true}
	}
	deg := (isScore - oosScore) / isScore
	return &OOSResult{
		InSampleScore: math.Round(isScore*10) / 10,
		OOSScore:      math.Round(oosScore*10) / 10,
		Degradation:   math.Round(deg*1000) / 1000,
		IsOverfit:     deg > v.MaxDegradation,
	}
}
