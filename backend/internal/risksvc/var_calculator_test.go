package risksvc

import (
	"math"
	"testing"
)

func TestVaR(t *testing.T) {
	returns := []float64{100, -50, 75, -30, 20, -10, 15, -25, 50, -100}
	cfg := DefaultVaRConfig()
	result := ComputeVaR(returns, cfg)
	if result.VaR > 0 {
		t.Fatalf("VaR should be negative, got %.2f", result.VaR)
	}
	t.Logf("VaR=%.2f CVaR=%.2f AnnualVol=%.2f", result.VaR, result.CVaR, result.AnnualVol)
}

func TestStressTest(t *testing.T) {
	scenarios := PredefinedScenarios()
	if len(scenarios) != 4 {
		t.Fatalf("want 4 scenarios, got %d", len(scenarios))
	}
	results := RunStressTests(100000, 50000)
	for _, r := range results {
		if !r.Passed {
			t.Fatalf("%s: should pass with 100k equity", r.Scenario.Name)
		}
	}
}

func TestComputeVaR_NormalReturns(t *testing.T) {
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
	if result.VaR > 0 {
		t.Fatalf("VaR should be negative (loss), got %.2f", result.VaR)
	}
	// CVaR should be <= VaR (more extreme).
	if result.CVaR > result.VaR {
		t.Fatalf("CVaR %.2f should be <= VaR %.2f", result.CVaR, result.VaR)
	}
	if result.DailyVol <= 0 {
		t.Fatal("daily vol should be positive")
	}
	if result.AnnualVol <= 0 {
		t.Fatal("annual vol should be positive")
	}
	if result.MaxDrawdown < 0 {
		t.Fatal("max drawdown should be >= 0")
	}

	t.Logf("VaR=%.2f CVaR=%.2f DailyVol=%.2f AnnualVol=%.2f MaxDD=%.2f",
		result.VaR, result.CVaR, result.DailyVol, result.AnnualVol, result.MaxDrawdown)
}

func TestComputeVaR_Empty(t *testing.T) {
	cfg := DefaultVaRConfig()
	result := ComputeVaR(nil, cfg)

	if result.NumReturns != 0 {
		t.Fatalf("empty returns: num_returns should be 0, got %d", result.NumReturns)
	}
}

func TestComputeVaR_AllPositive(t *testing.T) {
	returns := []float64{10.0, 20.0, 30.0, 40.0, 50.0}
	cfg := DefaultVaRConfig()
	result := ComputeVaR(returns, cfg)

	// All returns positive, VaR should be the lowest value.
	if result.VaR <= 0 {
		t.Fatalf("all positive: VaR should be positive, got %.2f", result.VaR)
	}
}

func TestPredefinedScenarios_Count(t *testing.T) {
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
	results := RunStressTests(100000, 50000) // 100k equity, 50k minimum

	for _, r := range results {
		if r.StartingEquity != 100000 {
			t.Fatalf("%s: starting equity wrong", r.Scenario.Name)
		}
		if !r.Passed {
			t.Fatalf("%s: should pass with 100k equity vs 50k minimum, shocked=%.0f",
				r.Scenario.Name, r.ShockedEquity)
		}
		if r.LossAmount < 0 {
			t.Fatalf("%s: loss amount should be >= 0", r.Scenario.Name)
		}
	}
}

func TestRunStressTests_SomeFail(t *testing.T) {
	results := RunStressTests(100000, 95000) // tight margin

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
	custom := StressScenario{Name: "apocalypse", Description: "total collapse", Shock: -0.90}
	results := RunStressTests(100000, 50000, custom)

	last := results[len(results)-1]
	if last.Scenario.Name != "apocalypse" {
		t.Fatalf("last scenario should be custom: got %s", last.Scenario.Name)
	}
	if last.Passed {
		t.Fatal("apocalypse scenario should fail: 100k → 10k < 50k min")
	}
}

func TestMaxDrawdown(t *testing.T) {
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
