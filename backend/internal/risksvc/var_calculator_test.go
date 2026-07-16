package risksvc

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
)

func TestVaR(t *testing.T) {
	t.Parallel()
	returns := []float64{100, -50, 75, -30, 20, -10, 15, -25, 50, -100}
	cfg := DefaultVaRConfig()
	result := ComputeVaR(returns, cfg)
	if result.VaR.GreaterThan(decimal.Zero) {
		t.Fatalf("VaR should be negative, got %s", result.VaR.String())
	}
	t.Logf("VaR result: daily_vol=%s annual_vol=%s var_95=%s max_drawdown=%s", result.DailyVol.String(), result.AnnualVol.String(), result.VaR.String(), result.MaxDrawdown.String())
}

func TestStressTest(t *testing.T) {
	t.Parallel()
	scenarios := PredefinedScenarios()
	if len(scenarios) != 4 {
		t.Fatalf("want 4 scenarios, got %d", len(scenarios))
	}
	results := RunStressTests(decF(100000), decF(50000))
	for _, r := range results {
		if !r.Passed {
			t.Fatalf("%s: should pass with 100k equity", r.Scenario.Name)
		}
	}
}

func TestComputeVaR_NormalReturns(t *testing.T) {
	t.Parallel()
	// Simulated daily returns: mostly small changes with one large loss.
	returns := []float64{
		100, -50, 75, -30, 20, -10, 15, -25, 50, -100,
		200, -75, 125, -40, 30, -60, 90, -20, 10, -5,
	}
	cfg := DefaultVaRConfig()
	result := ComputeVaR(returns, cfg)

	if result.NumReturns != 20 {
		t.Fatalf("num_returns: want 20, got %d", result.NumReturns)
	}
	if result.Confidence != 0.95 {
		t.Fatalf("confidence: want 0.95, got %.2f", result.Confidence)
	}
	// VaR at 95%: 5% * 20 = 1 → index 0 of sorted = -100
	if result.VaR.GreaterThan(decimal.Zero) {
		t.Fatalf("VaR should be negative (loss), got %s", result.VaR.String())
	}
	// CVaR should be <= VaR (more extreme).
	if result.CVaR.GreaterThan(result.VaR) {
		t.Fatalf("CVaR %s should be <= VaR %s", result.CVaR.String(), result.VaR.String())
	}
	if result.DailyVol.LessThanOrEqual(decimal.Zero) {
		t.Fatal("daily vol should be positive")
	}
	if result.AnnualVol.LessThanOrEqual(decimal.Zero) {
		t.Fatal("annual vol should be positive")
	}
	if result.MaxDrawdown.LessThan(decimal.Zero) {
		t.Fatal("max drawdown should be >= 0")
	}

	t.Logf("VaR result: daily_vol=%s annual_vol=%s var_95=%s max_drawdown=%s", result.DailyVol.String(), result.AnnualVol.String(), result.VaR.String(), result.MaxDrawdown.String())
}

func TestComputeVaR_Empty(t *testing.T) {
	t.Parallel()
	cfg := DefaultVaRConfig()
	result := ComputeVaR(nil, cfg)

	if result.NumReturns != 0 {
		t.Fatalf("empty returns: num_returns should be 0, got %d", result.NumReturns)
	}
}

func TestComputeVaR_AllPositive(t *testing.T) {
	t.Parallel()
	returns := []float64{10.0, 20.0, 30.0, 40.0, 50.0}
	cfg := DefaultVaRConfig()
	result := ComputeVaR(returns, cfg)

	// All returns positive, VaR should be the lowest value.
	if result.VaR.LessThanOrEqual(decimal.Zero) {
		t.Fatalf("all positive: VaR should be positive, got %s", result.VaR.String())
	}
}

func TestPredefinedScenarios_Count(t *testing.T) {
	t.Parallel()
	scenarios := PredefinedScenarios()
	if len(scenarios) != 4 {
		t.Fatalf("want 4 predefined scenarios, got %d", len(scenarios))
	}
	names := map[string]bool{}
	for _, s := range scenarios {
		names[s.Name] = true
	}
	for _, name := range []string{"2008_crash", "2015_snb", "2020_covid", "fomc_flash"} {
		if !names[name] {
			t.Fatalf("missing scenario: %s", name)
		}
	}
}

func TestRunStressTests_AllPass(t *testing.T) {
	t.Parallel()
	results := RunStressTests(decF(100000), decF(50000)) // 100k equity, 50k minimum

	for _, r := range results {
		if !r.StartingEquity.Equal(decF(100000)) {
			t.Fatalf("%s: starting equity wrong", r.Scenario.Name)
		}
		if !r.Passed {
			t.Fatalf("%s: should pass with 100k equity vs 50k minimum, shocked=%s",
				r.Scenario.Name, r.ShockedEquity.String())
		}
		if r.LossAmount.LessThan(decimal.Zero) {
			t.Fatalf("%s: loss amount should be >= 0", r.Scenario.Name)
		}
	}
}

func TestRunStressTests_SomeFail(t *testing.T) {
	t.Parallel()
	results := RunStressTests(decF(100000), decF(95000)) // tight margin

	failed := 0
	for _, r := range results {
		if !r.Passed {
			failed++
		}
	}
	if failed == 0 {
		t.Fatal("some stress scenarios should fail with tight margin")
	}
	t.Logf("Failed %d/%d scenarios", failed, len(results))
}

func TestRunStressTests_CustomScenario(t *testing.T) {
	t.Parallel()
	custom := StressScenario{Name: "apocalypse", Description: "total collapse", Shock: -0.90}
	results := RunStressTests(decF(100000), decF(50000), custom)

	last := results[len(results)-1]
	if last.Scenario.Name != "apocalypse" {
		t.Fatalf("last scenario should be custom: got %s", last.Scenario.Name)
	}
	if last.Passed {
		t.Fatal("apocalypse scenario should fail: 100k → 10k < 50k min")
	}
}

func TestMaxDrawdown(t *testing.T) {
	t.Parallel()
	returns := []float64{100, -50, -30, 20, -40, 50, -20}
	dd := computeMaxDrawdown(returns)

	if dd < 0 {
		t.Fatal("max drawdown must be >= 0")
	}
	// Peak=100, then -50→50, -30→20, +20→40, -40→0 → max DD = 100.
	if math.Abs(dd-100) > 1 {
		t.Fatalf("max drawdown: want ~100, got %.2f", dd)
	}
}
