package backtest

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

func TestDegradationRatio(t *testing.T) {
	tests := []struct {
		name   string
		is     string
		oos    string
		expect string
	}{
		{"no degradation", "2.0", "2.0", "1"},
		{"50% degradation", "2.0", "1.0", "0.5"},
		{"zero IS", "0", "1.0", "0"},
		{"negative IS", "-1.0", "1.0", "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := decimal.RequireFromString(tt.is)
			oos := decimal.RequireFromString(tt.oos)
			ratio := DegradationRatio(is, oos)
			if ratio.String() != tt.expect {
				t.Errorf("DegradationRatio(%s, %s) = %s, want %s", tt.is, tt.oos, ratio.String(), tt.expect)
			}
		})
	}
}

func TestRunWalkForward_InsufficientBars(t *testing.T) {
	bars := make([]sdk.Bar, 5)
	cfg := Config{InitialCapital: decimal.NewFromInt(10000)}
	strategy := &noopStrategy{}
	_, err := RunWalkForward(context.Background(), cfg, strategy, bars)
	if err == nil {
		t.Fatal("expected error for insufficient bars")
	}
}

type noopStrategy struct{}

func (s *noopStrategy) OnInit(ctx sdk.Context) error { return nil }
func (s *noopStrategy) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {
	return nil, nil
}
func (s *noopStrategy) OnDeinit(ctx sdk.Context, reason string) error { return nil }
