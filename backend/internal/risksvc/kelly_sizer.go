// Package risksvc provides the KellyFractionSizer (M10-BASE-C3).
//
// Kelly Criterion: f* = (p·b - q) / b
//
//	where:
//	  p = win probability
//	  q = loss probability (1 - p)
//	  b = win/loss ratio (avg_win / avg_loss)
//
// Default: half-Kelly (fraction = 0.5) for conservative sizing.
// The raw Kelly fraction is capped at KellyMax (default 0.25 = 25% of equity).

package risksvc

import (
	"context"
	"math"
)

// KellyFractionSizer sizes positions using the Kelly Criterion.
type KellyFractionSizer struct {
	// WinProb is the estimated win probability (0-1).
	WinProb float64

	// WinLossRatio is the average win divided by average loss (b in Kelly formula).
	WinLossRatio float64

	// Fraction is the fraction of full Kelly to use (default 0.5 = half-Kelly).
	Fraction float64

	// KellyMax caps the Kelly fraction as a percentage of equity (0-1).
	// Default 0.25 = max 25% of equity per trade.
	KellyMax float64

	// MaxLots caps absolute position size.
	MaxLots float64

	// MinLots floors position size (returns 0 if below).
	MinLots float64
}

func (s *KellyFractionSizer) Name() string { return "kelly_fraction" }

func (s *KellyFractionSizer) Size(_ context.Context, req *SizerRequest) (*SizerResult, error) {
	if s.Fraction <= 0 {
		s.Fraction = 0.5 // half-Kelly default
	}
	if s.KellyMax <= 0 {
		s.KellyMax = 0.25 // max 25% equity
	}

	p := s.WinProb
	if p <= 0 || p > 1 {
		return &SizerResult{Lots: 0, RiskUsed: 0, Method: s.Name()}, nil
	}
	q := 1.0 - p
	b := s.WinLossRatio
	if b <= 0 {
		return &SizerResult{Lots: 0, RiskUsed: 0, Method: s.Name()}, nil
	}

	// Kelly formula: f* = (p·b - q) / b
	fStar := (p*b - q) / b
	if fStar <= 0 {
		// Negative or zero edge — don't bet.
		return &SizerResult{Lots: 0, RiskUsed: 0, Method: s.Name()}, nil
	}

	// Apply half-Kelly fraction.
	fStar *= s.Fraction

	// Cap at KellyMax.
	if fStar > s.KellyMax {
		fStar = s.KellyMax
	}

	// Convert fraction of equity to lots.
	riskCapital := req.Equity * fStar
	if riskCapital <= 0 {
		return &SizerResult{Lots: 0, RiskUsed: 0, Method: s.Name()}, nil
	}

	price := req.Price
	if price <= 0 {
		price = 1.0
	}

	// Lots = risk_capital / (price × contract_size).
	contractSize := req.ContractSize
	if contractSize <= 0 {
		contractSize = 1
	}
	lots := riskCapital / (price * contractSize)

	if lots < s.MinLots {
		lots = 0
	}
	if s.MaxLots > 0 && lots > s.MaxLots {
		lots = s.MaxLots
	}

	return &SizerResult{
		Lots:     lots,
		RiskUsed: fStar,
		Method:   s.Name(),
	}, nil
}

// KellyMetrics computes derived Kelly statistics.
type KellyMetrics struct {
	FStar       float64 // raw Kelly fraction
	HalfKelly   float64 // half-Kelly fraction
	Edge        float64 // p*b - q = expected value
	IsPositive  bool    // true if edge > 0
}

// ComputeKellyMetrics derives Kelly statistics from win probability and W/L ratio.
func ComputeKellyMetrics(winProb, winLossRatio float64) KellyMetrics {
	p := winProb
	b := winLossRatio
	q := 1.0 - p
	edge := p*b - q
	fStar := math.Max(0, edge/b)
	return KellyMetrics{
		FStar:      fStar,
		HalfKelly:  fStar * 0.5,
		Edge:       edge,
		IsPositive: edge > 0,
	}
}
