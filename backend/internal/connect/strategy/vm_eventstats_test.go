package strategy

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/runner"
	"alphaforge/tools/mql2go"
)

// TestQS3Baseline_EventStatsChain pins the stats passthrough:
// vm.Ticks() → VMRunner.LastEventStats() → Runner.EventStats(). The
// ant_strategy_vm_event_instructions metric reads this chain — it must
// report real instruction counts, not a constant. Mutation guard for
// QS-3-BASELINE (Ticks() returning a constant turns this RED).
func TestQS3Baseline_EventStatsChain(t *testing.T) {
	vmRunner, err := mql2go.CompileMQL(`int g = 0; void OnTick() { g = g + 1; }`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	r := runner.New(runner.Config{Mode: "live"})
	r.SetStrategy(vmRunner)

	if _, err := r.OnTick(context.Background(), decimal.NewFromFloat(1), decimal.NewFromFloat(1.1)); err != nil {
		t.Fatalf("OnTick: %v", err)
	}
	ticks, fatal, ok := r.EventStats()
	if !ok {
		t.Fatal("EventStats ok=false for VMRunner strategy, want true")
	}
	if ticks <= 0 {
		t.Fatalf("EventStats ticks = %d, want > 0 (real instruction count)", ticks)
	}
	if fatal != "" {
		t.Fatalf("EventStats fatal = %q, want empty", fatal)
	}
}

// TestQS3Baseline_EventStatsDegrade pins the degrade path: a runner with no
// strategy (or a non-VM strategy) reports ok=false so observeVMEvent falls
// back to duration-only.
func TestQS3Baseline_EventStatsDegrade(t *testing.T) {
	r := runner.New(runner.Config{Mode: "live"})
	if _, _, ok := r.EventStats(); ok {
		t.Fatal("EventStats ok=true without strategy, want false")
	}
}
