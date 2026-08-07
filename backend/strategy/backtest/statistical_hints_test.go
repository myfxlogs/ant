package backtest

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

func TestCheckZeroEquityVariance_FlatEquity(t *testing.T) {
	t.Parallel()
	result := &Result{
		Equity: []EquityPoint{
			{Equity: decimal.NewFromFloat(10000)},
			{Equity: decimal.NewFromFloat(10000)},
			{Equity: decimal.NewFromFloat(10000)},
		},
	}
	bs := CheckZeroEquityVariance(result)
	if bs == nil {
		t.Fatal("flat equity should trigger zero_equity_variance hint")
	}
	if bs.Id != "zero_equity_variance" {
		t.Fatalf("hint ID: want zero_equity_variance, got %s", bs.Id)
	}
	if bs.Severity != severityHint {
		t.Fatalf("severity: want %s, got %s", severityHint, bs.Severity)
	}
}

func TestCheckZeroEquityVariance_VaryingEquity(t *testing.T) {
	t.Parallel()
	result := &Result{
		Equity: []EquityPoint{
			{Equity: decimal.NewFromFloat(10000)},
			{Equity: decimal.NewFromFloat(10100)},
			{Equity: decimal.NewFromFloat(9900)},
		},
	}
	if bs := CheckZeroEquityVariance(result); bs != nil {
		t.Fatal("varying equity should not trigger hint")
	}
}

func TestCheckZeroEquityVariance_TooFewPoints(t *testing.T) {
	t.Parallel()
	result := &Result{
		Equity: []EquityPoint{{Equity: decimal.NewFromFloat(10000)}},
	}
	if bs := CheckZeroEquityVariance(result); bs != nil {
		t.Fatal("single equity point should not trigger hint")
	}
}

func TestCheckMonotonicPositionGrowth_Increasing(t *testing.T) {
	t.Parallel()
	trades := []Trade{
		{Volume: decimal.NewFromFloat(0.1)},
		{Volume: decimal.NewFromFloat(0.2)},
		{Volume: decimal.NewFromFloat(0.4)},
		{Volume: decimal.NewFromFloat(0.8)},
	}
	bs := CheckMonotonicPositionGrowth(trades)
	if bs == nil {
		t.Fatal("monotonically increasing volume should trigger hint")
	}
	if bs.Id != "monotonic_position_growth" {
		t.Fatalf("hint ID: want monotonic_position_growth, got %s", bs.Id)
	}
}

func TestCheckMonotonicPositionGrowth_NotMonotonic(t *testing.T) {
	t.Parallel()
	trades := []Trade{
		{Volume: decimal.NewFromFloat(0.1)},
		{Volume: decimal.NewFromFloat(0.3)},
		{Volume: decimal.NewFromFloat(0.2)},
	}
	if bs := CheckMonotonicPositionGrowth(trades); bs != nil {
		t.Fatal("non-monotonic volume should not trigger hint")
	}
}

func TestCheckMonotonicPositionGrowth_TooFewTrades(t *testing.T) {
	t.Parallel()
	trades := []Trade{
		{Volume: decimal.NewFromFloat(0.1)},
		{Volume: decimal.NewFromFloat(0.2)},
	}
	if bs := CheckMonotonicPositionGrowth(trades); bs != nil {
		t.Fatal("two trades should not trigger hint")
	}
}

func TestCheckAllSameDirectionSameVolume_AllSame(t *testing.T) {
	t.Parallel()
	vol := decimal.NewFromFloat(0.1)
	trades := []Trade{
		{Side: sdk.SideBuy, Volume: vol},
		{Side: sdk.SideBuy, Volume: vol},
		{Side: sdk.SideBuy, Volume: vol},
	}
	bs := CheckAllSameDirectionSameVolume(trades)
	if bs == nil {
		t.Fatal("all same direction+volume should trigger hint")
	}
	if bs.Id != "all_same_direction_same_volume" {
		t.Fatalf("hint ID: want all_same_direction_same_volume, got %s", bs.Id)
	}
}

func TestCheckAllSameDirectionSameVolume_DifferentVolume(t *testing.T) {
	t.Parallel()
	trades := []Trade{
		{Side: sdk.SideBuy, Volume: decimal.NewFromFloat(0.1)},
		{Side: sdk.SideBuy, Volume: decimal.NewFromFloat(0.2)},
		{Side: sdk.SideBuy, Volume: decimal.NewFromFloat(0.1)},
	}
	if bs := CheckAllSameDirectionSameVolume(trades); bs != nil {
		t.Fatal("different volumes should not trigger hint")
	}
}

func TestCheckAllSameDirectionSameVolume_MixedDirection(t *testing.T) {
	t.Parallel()
	vol := decimal.NewFromFloat(0.1)
	trades := []Trade{
		{Side: sdk.SideBuy, Volume: vol},
		{Side: sdk.SideSell, Volume: vol},
		{Side: sdk.SideBuy, Volume: vol},
	}
	if bs := CheckAllSameDirectionSameVolume(trades); bs != nil {
		t.Fatal("mixed directions should not trigger hint")
	}
}

func TestCheckAbnormalTradeFrequency_HighFrequency(t *testing.T) {
	t.Parallel()
	trades := make([]Trade, 2000)
	result := &Result{
		Config: Config{
			StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:        time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			InitialCapital: decimal.NewFromFloat(10000),
		},
		Trades: trades,
	}
	bs := CheckAbnormalTradeFrequency(result)
	if bs == nil {
		t.Fatal("2000 trades in 1 day should trigger high frequency hint")
	}
	if bs.Id != "abnormal_trade_frequency_high" {
		t.Fatalf("hint ID: want abnormal_trade_frequency_high, got %s", bs.Id)
	}
}

func TestCheckAbnormalTradeFrequency_NormalFrequency(t *testing.T) {
	t.Parallel()
	trades := make([]Trade, 10)
	result := &Result{
		Config: Config{
			StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:        time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			InitialCapital: decimal.NewFromFloat(10000),
		},
		Trades: trades,
	}
	if bs := CheckAbnormalTradeFrequency(result); bs != nil {
		t.Fatal("10 trades in 30 days should not trigger hint")
	}
}

func TestCheckAbnormalTradeFrequency_NoTrades(t *testing.T) {
	t.Parallel()
	result := &Result{
		Config: Config{
			StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		},
	}
	if bs := CheckAbnormalTradeFrequency(result); bs != nil {
		t.Fatal("no trades should not trigger hint")
	}
}

func TestCheckStatisticalHints_AllHints(t *testing.T) {
	t.Parallel()
	vol := decimal.NewFromFloat(0.1)
	trades := []Trade{
		{Side: sdk.SideBuy, Volume: vol},
		{Side: sdk.SideBuy, Volume: vol},
		{Side: sdk.SideBuy, Volume: vol},
	}
	result := &Result{
		Config: Config{
			StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:        time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			InitialCapital: decimal.NewFromFloat(10000),
		},
		Equity: []EquityPoint{
			{Equity: decimal.NewFromFloat(10000)},
			{Equity: decimal.NewFromFloat(10000)},
		},
		Trades: trades,
	}
	hints := CheckStatisticalHints(result)
	if len(hints) < 2 {
		t.Fatalf("expected at least 2 hints (zero_equity + all_same), got %d", len(hints))
	}
	for _, h := range hints {
		if h.Category != "statistical" {
			t.Errorf("hint category: want statistical, got %s", h.Category)
		}
		if h.Severity != severityHint {
			t.Errorf("hint severity: want %s, got %s", severityHint, h.Severity)
		}
	}
}

func TestCheckStatisticalHints_NoHints(t *testing.T) {
	t.Parallel()
	result := &Result{
		Config: Config{
			StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:        time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			InitialCapital: decimal.NewFromFloat(10000),
		},
		Equity: []EquityPoint{
			{Equity: decimal.NewFromFloat(10000)},
			{Equity: decimal.NewFromFloat(10500)},
			{Equity: decimal.NewFromFloat(9800)},
		},
		Trades: []Trade{
			{Side: sdk.SideBuy, Volume: decimal.NewFromFloat(0.1)},
			{Side: sdk.SideSell, Volume: decimal.NewFromFloat(0.2)},
			{Side: sdk.SideBuy, Volume: decimal.NewFromFloat(0.15)},
		},
	}
	hints := CheckStatisticalHints(result)
	if len(hints) != 0 {
		t.Fatalf("normal result should have 0 hints, got %d: %+v", len(hints), hints)
	}
}
