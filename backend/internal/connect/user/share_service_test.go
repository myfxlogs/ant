package user

import (
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/internal/model"
)

func TestComputeMaxDrawdownPct(t *testing.T) {
	tests := []struct {
		name   string
		points []*model.EquityPoint
		want   string
	}{
		{
			name: "spec case: [100,120,90,110] → 25%",
			points: []*model.EquityPoint{
				{Equity: decimal.NewFromInt(100)},
				{Equity: decimal.NewFromInt(120)},
				{Equity: decimal.NewFromInt(90)},
				{Equity: decimal.NewFromInt(110)},
			},
			want: "25",
		},
		{
			name: "monotonic increase → 0%",
			points: []*model.EquityPoint{
				{Equity: decimal.NewFromInt(100)},
				{Equity: decimal.NewFromInt(110)},
				{Equity: decimal.NewFromInt(120)},
			},
			want: "0",
		},
		{
			name: "single point → 0%",
			points: []*model.EquityPoint{
				{Equity: decimal.NewFromInt(100)},
			},
			want: "0",
		},
		{
			name:   "empty → 0%",
			points: nil,
			want:   "0",
		},
		{
			name: "full drawdown: [100,0] → 100%",
			points: []*model.EquityPoint{
				{Equity: decimal.NewFromInt(100)},
				{Equity: decimal.NewFromInt(0)},
			},
			want: "100",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeMaxDrawdownPct(tt.points)
			if got.String() != tt.want {
				t.Errorf("computeMaxDrawdownPct() = %s, want %s", got.String(), tt.want)
			}
		})
	}
}

// TestSummarizeTrades_NoMaxDD verifies that summarizeTrades no longer
// computes maxDD (single worst trade). The old bug stored t.Profit min
// as maxDrawdown; now it must not exist.
func TestSummarizeTrades_NoMaxDD(t *testing.T) {
	trades := []*model.TradeRecord{
		{Profit: decimal.NewFromInt(50)},
		{Profit: decimal.NewFromInt(-30)},
		{Profit: decimal.NewFromInt(20)},
	}
	s := summarizeTrades(trades)
	// bestTrade should be 50, worstTrade should be -30
	if !s.bestTrade.Equal(decimal.NewFromInt(50)) {
		t.Errorf("bestTrade = %s, want 50", s.bestTrade.String())
	}
	if !s.worstTrade.Equal(decimal.NewFromInt(-30)) {
		t.Errorf("worstTrade = %s, want -30", s.worstTrade.String())
	}
	// wins=2, losses=1
	if s.wins != 2 || s.losses != 1 {
		t.Errorf("wins=%d losses=%d, want 2/1", s.wins, s.losses)
	}
}

func TestAggregateSymbolStats(t *testing.T) {
	trades := []*model.TradeRecord{
		{Symbol: "EURUSD", Profit: decimal.NewFromInt(50)},
		{Symbol: "EURUSD", Profit: decimal.NewFromInt(-20)},
		{Symbol: "GBPUSD", Profit: decimal.NewFromInt(30)},
	}
	stats := aggregateSymbolStats(trades)
	if len(stats) != 2 {
		t.Fatalf("len = %d, want 2", len(stats))
	}
	// First symbol should be EURUSD (insertion order preserved)
	if stats[0].Symbol != "EURUSD" {
		t.Errorf("stats[0].Symbol = %s, want EURUSD", stats[0].Symbol)
	}
	if stats[0].Count != 2 {
		t.Errorf("stats[0].Count = %d, want 2", stats[0].Count)
	}
	if stats[0].Net != "30" {
		t.Errorf("stats[0].Net = %s, want 30", stats[0].Net)
	}
}
