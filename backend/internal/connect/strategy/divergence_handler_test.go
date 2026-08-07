package strategy

import (
	"math"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
	"alphaforge/internal/repository"
)

func mustTime(ms int64) time.Time {
	return time.UnixMilli(ms)
}

func almostEqual(a, b, tol float64) bool {
	return math.Abs(a-b) < tol
}

func TestComputeBacktestMetrics(t *testing.T) {
	trades := []*repository.BacktestRunTrade{
		{PnL: decimal.NewFromFloat(100), OpenTs: 1000, CloseTs: 2000},
		{PnL: decimal.NewFromFloat(-50), OpenTs: 2000, CloseTs: 3000},
		{PnL: decimal.NewFromFloat(200), OpenTs: 3000, CloseTs: 4000},
	}
	m := computeBacktestMetrics(trades)

	if m.TradeCount != 3 {
		t.Errorf("TradeCount = %d, want 3", m.TradeCount)
	}
	if m.Wins != 2 {
		t.Errorf("Wins = %d, want 2", m.Wins)
	}
	if m.Losses != 1 {
		t.Errorf("Losses = %d, want 1", m.Losses)
	}
	if !almostEqual(m.WinRate, 200.0/3.0, 0.001) {
		t.Errorf("WinRate = %f, want %f", m.WinRate, 200.0/3.0)
	}
	netPnL, _ := decimal.NewFromString(m.NetPnl)
	if !netPnL.Equal(decimal.NewFromFloat(250)) {
		t.Errorf("NetPnl = %s, want 250", m.NetPnl)
	}
	if m.PeriodStartMs != 1000 {
		t.Errorf("PeriodStartMs = %d, want 1000", m.PeriodStartMs)
	}
	if m.PeriodEndMs != 4000 {
		t.Errorf("PeriodEndMs = %d, want 4000", m.PeriodEndMs)
	}
}

func TestComputeBacktestMetrics_Empty(t *testing.T) {
	m := computeBacktestMetrics(nil)
	if m.TradeCount != 0 {
		t.Errorf("TradeCount = %d, want 0", m.TradeCount)
	}
	if m.NetPnl != "" {
		t.Errorf("NetPnl = %s, want empty", m.NetPnl)
	}
}

func TestComputeLiveMetrics(t *testing.T) {
	profit1 := decimal.NewFromFloat(100)
	profit2 := decimal.NewFromFloat(-30)
	records := []*model.TradeRecord{
		{Profit: profit1, OpenTime: mustTime(1000), CloseTime: mustTime(2000)},
		{Profit: profit2, OpenTime: mustTime(2000), CloseTime: mustTime(3000)},
	}
	m := computeLiveMetrics(records)

	if m.TradeCount != 2 {
		t.Errorf("TradeCount = %d, want 2", m.TradeCount)
	}
	if m.Wins != 1 {
		t.Errorf("Wins = %d, want 1", m.Wins)
	}
	if m.Losses != 1 {
		t.Errorf("Losses = %d, want 1", m.Losses)
	}
	if m.WinRate != 50.0 {
		t.Errorf("WinRate = %f, want 50", m.WinRate)
	}
	netPnL, _ := decimal.NewFromString(m.NetPnl)
	if !netPnL.Equal(decimal.NewFromFloat(70)) {
		t.Errorf("NetPnl = %s, want 70", m.NetPnl)
	}
}

func TestComputeLiveMetrics_Empty(t *testing.T) {
	m := computeLiveMetrics(nil)
	if m.TradeCount != 0 {
		t.Errorf("TradeCount = %d, want 0", m.TradeCount)
	}
}

func TestComputeSharpe(t *testing.T) {
	tests := []struct {
		name   string
		pnls   []float64
		wantGt bool
	}{
		{"nil", nil, false},
		{"single", []float64{100}, false},
		{"all_same", []float64{50, 50, 50}, false},
		{"positive", []float64{100, -50, 200, -30, 80}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := computeSharpe(tt.pnls)
			if tt.wantGt && s <= 0 {
				t.Errorf("computeSharpe(%v) = %f, want > 0", tt.pnls, s)
			}
			if !tt.wantGt && s != 0 {
				t.Errorf("computeSharpe(%v) = %f, want 0", tt.pnls, s)
			}
		})
	}
}

func TestPctDivergence(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want float64
	}{
		{"both_zero", "0", "0", 0},
		{"a_zero_b_nonzero", "0", "100", 100},
		{"identical", "100", "100", 0},
		{"half_divergence", "100", "50", 50},
		{"full_divergence", "100", "0", 100},
		{"invalid_a", "abc", "100", 0},
		{"invalid_b", "100", "xyz", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pctDivergence(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("pctDivergence(%s, %s) = %f, want %f", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestCountDivergencePct(t *testing.T) {
	tests := []struct {
		name     string
		bt, live int32
		want     float64
	}{
		{"both_zero", 0, 0, 0},
		{"bt_zero_live_nonzero", 0, 10, 100},
		{"identical", 50, 50, 0},
		{"half", 100, 50, 50},
		{"double", 50, 100, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countDivergencePct(tt.bt, tt.live)
			if got != tt.want {
				t.Errorf("countDivergencePct(%d, %d) = %f, want %f", tt.bt, tt.live, got, tt.want)
			}
		})
	}
}

func TestAssessDivergence(t *testing.T) {
	bt := &antv1.DivergenceMetrics{TradeCount: 10, NetPnl: "100"}
	live := &antv1.DivergenceMetrics{TradeCount: 8, NetPnl: "90"}

	// Consistent: < 10% divergence
	report := &antv1.DivergenceReport{
		PnlDivergencePct:        5,
		TradeCountDivergencePct: 20,
		WinRateDivergencePct:    3,
	}
	status, detail := assessDivergence(report, bt, live)
	if status != antv1.DivergenceStatus_DIVERGENCE_STATUS_MINOR_DIVERGENCE {
		t.Errorf("status = %v, want MINOR_DIVERGENCE", status)
	}
	if detail == "" {
		t.Error("detail should not be empty")
	}

	// Major: > 30% divergence
	report.PnlDivergencePct = 50
	status, _ = assessDivergence(report, bt, live)
	if status != antv1.DivergenceStatus_DIVERGENCE_STATUS_MAJOR_DIVERGENCE {
		t.Errorf("status = %v, want MAJOR_DIVERGENCE", status)
	}

	// Insufficient data
	emptyBt := &antv1.DivergenceMetrics{TradeCount: 0}
	emptyLive := &antv1.DivergenceMetrics{TradeCount: 5}
	status, _ = assessDivergence(report, emptyBt, emptyLive)
	if status != antv1.DivergenceStatus_DIVERGENCE_STATUS_INSUFFICIENT_DATA {
		t.Errorf("status = %v, want INSUFFICIENT_DATA", status)
	}

	// Consistent: all < 10%
	report2 := &antv1.DivergenceReport{
		PnlDivergencePct:        5,
		TradeCountDivergencePct: 8,
		WinRateDivergencePct:    3,
	}
	status, _ = assessDivergence(report2, bt, live)
	if status != antv1.DivergenceStatus_DIVERGENCE_STATUS_CONSISTENT {
		t.Errorf("status = %v, want CONSISTENT", status)
	}
}
