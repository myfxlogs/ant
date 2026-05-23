package service

import (
	"math"
	"testing"
)

func TestAnalyticsService_Mean(t *testing.T) {
	svc := &AnalyticsService{}

	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"empty", []float64{}, 0},
		{"single", []float64{5.0}, 5.0},
		{"multiple", []float64{1.0, 2.0, 3.0, 4.0, 5.0}, 3.0},
		{"negative", []float64{-1.0, -2.0, -3.0}, -2.0},
		{"mixed", []float64{-1.0, 0.0, 1.0}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.mean(tt.values)
			if math.Abs(result-tt.expected) > 0.0001 {
				t.Errorf("mean(%v) = %f, expected %f", tt.values, result, tt.expected)
			}
		})
	}
}

func TestAnalyticsService_StdDev(t *testing.T) {
	svc := &AnalyticsService{}

	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{"empty", []float64{}, 0},
		{"single", []float64{5.0}, 0},
		{"two_values", []float64{1.0, 3.0}, math.Sqrt(2.0)},
		{"constant", []float64{5.0, 5.0, 5.0}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.stdDev(tt.values)
			if math.Abs(result-tt.expected) > 0.0001 {
				t.Errorf("stdDev(%v) = %f, expected %f", tt.values, result, tt.expected)
			}
		})
	}
}

func TestAnalyticsService_FilterNegative(t *testing.T) {
	svc := &AnalyticsService{}

	tests := []struct {
		name     string
		values   []float64
		expected int
	}{
		{"empty", []float64{}, 0},
		{"all_positive", []float64{1.0, 2.0, 3.0}, 0},
		{"all_negative", []float64{-1.0, -2.0, -3.0}, 3},
		{"mixed", []float64{-1.0, 2.0, -3.0, 4.0}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.filterNegative(tt.values)
			if len(result) != tt.expected {
				t.Errorf("filterNegative(%v) returned %d values, expected %d", tt.values, len(result), tt.expected)
			}
		})
	}
}

func TestAnalyticsService_Percentile(t *testing.T) {
	svc := &AnalyticsService{}

	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}

	tests := []struct {
		name     string
		p        float64
		expected float64
	}{
		{"p0", 0, 1.0},
		{"p50", 50, 5.5},
		{"p100", 100, 10.0},
		{"p25", 25, 3.25},
		{"p75", 75, 7.75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.percentile(values, tt.p)
			if math.Abs(result-tt.expected) > 0.1 {
				t.Errorf("percentile(%v, %f) = %f, expected %f", values, tt.p, result, tt.expected)
			}
		})
	}
}

func TestAnalyticsService_ExpectedShortfall(t *testing.T) {
	svc := &AnalyticsService{}

	values := []float64{-10.0, -8.0, -5.0, -3.0, -1.0, 1.0, 2.0, 3.0, 5.0, 10.0}

	result := svc.expectedShortfall(values, 5)
	if result >= 0 {
		t.Errorf("expectedShortfall should be negative for losses, got %f", result)
	}
}

func TestAnalyticsService_FormatDuration(t *testing.T) {
	svc := &AnalyticsService{}

	tests := []struct {
		name     string
		seconds  float64
		contains string
	}{
		{"less_than_minute", 30, "分钟"},
		{"minutes", 180, "分钟"},
		{"hours", 7200, "小时"},
		{"days", 172800, "天"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.formatDuration(tt.seconds)
			if result == "" {
				t.Errorf("formatDuration(%f) returned empty string", tt.seconds)
			}
		})
	}
}
