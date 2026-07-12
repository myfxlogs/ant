package factor

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"github.com/shopspring/decimal"
)

func TestFactorRegistry_RegisterAndLookup(t *testing.T) {
	t.Parallel()
	r := NewFactorRegistry(zap.NewNop())

	if err := r.Register(FactorSpec{
		Name:       "trend",
		Expression: "ema($close, 20) / ema($close, 60) - 1",
		Symbol:     "EURUSD",
		Timeframe:  "1m",
	}); err != nil {
		t.Fatal(err)
	}

	// Specific match
	factors := r.Lookup("EURUSD", "1m")
	if len(factors) != 1 {
		t.Fatalf("expected 1 factor for EURUSD:1m, got %d", len(factors))
	}

	// No match for different symbol
	factors = r.Lookup("GBPUSD", "1m")
	if len(factors) != 0 {
		t.Fatalf("expected 0 factors for GBPUSD:1m, got %d", len(factors))
	}
}

func TestFactorRegistry_AllSymbols(t *testing.T) {
	t.Parallel()
	r := NewFactorRegistry(zap.NewNop())

	if err := r.Register(FactorSpec{
		Name:       "rsi_global",
		Expression: "rsi($close, 14)",
	}); err != nil {
		t.Fatal(err)
	}

	// Should match any symbol/timeframe
	factors := r.Lookup("EURUSD", "5m")
	if len(factors) != 1 {
		t.Fatalf("expected 1 all-symbol factor, got %d", len(factors))
	}
}

func TestFactorEvaluator_EvaluateBar(t *testing.T) {
	t.Parallel()
	sub := NewSubscriber(DefaultSubscriberConfig(), zap.NewNop())
	registry := NewFactorRegistry(zap.NewNop())

	if err := registry.Register(FactorSpec{
		Name:       "sma3",
		Expression: "sma($close, 3)",
		Symbol:     "EURUSD",
		Timeframe:  "1m",
	}); err != nil {
		t.Fatal(err)
	}

	eval := NewFactorEvaluator(sub, registry, zap.NewNop())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go eval.Start(ctx)

	// Feed 3 bars to satisfy SMA(3) warmup
	prices := []float64{100, 101, 102}
	for i, p := range prices {
		bar := &mdtick.Bar{
			Canonical:     "EURUSD",
			Period:        "1m",
			Close:         decimal.NewFromFloat(p),
			CloseTsUnixMs: time.Now().UnixMilli(),
			IsReplay:      true, // skip finality gate
		}
		sub.Push(bar)

		if i >= 2 { // after warmup, expect results
			select {
			case result := <-eval.Output():
				if result.Spec.Name != "sma3" {
					t.Fatalf("expected factor name sma3, got %s", result.Spec.Name)
				}
				// SMA(3) of [100, 101, 102] = 101
				if result.Value < 100.9 || result.Value > 101.1 {
					t.Fatalf("expected SMA ~101, got %f", result.Value)
				}
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for factor result")
			}
		} else {
			// During warmup, no results should be emitted (Op.Eval returns NaN which is still sent)
			// Actually NaN is a valid float64, so results will be emitted with NaN value
			select {
			case <-eval.Output():
				// OK — NaN result during warmup
			case <-time.After(50 * time.Millisecond):
				// Also OK — no result yet
			}
		}
	}
}
