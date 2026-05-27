package risksvc

import (
	"context"
	"testing"
	"time"
)

func TestPipeline_FullPass(t *testing.T) {
	t.Parallel()
	capStore := NewCapabilityStore()
	capStore.Set(&Capability{UserID: "u1", Tier: Tier2LiveLimited})

	hardLimit := NewHardLimitEvaluator(
		&MarginFloorRule{FloorRatio: 0.5},
		&ContractExpiryRule{CoolingOffHours: 1},
	)

	sizer := &VolTargetSizer{RiskBudgetPct: 0.01, MaxLots: 100}

	p := NewSignalPipeline(PipelineConfig{
		CapStore:  capStore,
		HardLimit: hardLimit,
		Sizer:     sizer,
	})

	sig := &SignalRequest{
		UserID: "u1", AccountID: "a1", Symbol: "EURUSD", Side: "buy",
		Price: 1.0850, ATR: 0.0035, ContractSize: 100000, HoldingDays: 5,
		Equity: 100000, FreeMargin: 50000,
	}
	result := p.Process(context.Background(), sig)
	if !result.Allowed {
		t.Fatalf("expected pass, got blocked at %s: %s", result.Stage, result.Reason)
	}
	if result.Lots <= 0 {
		t.Fatalf("expected non-zero lots, got %.4f", result.Lots)
	}
	if result.Stage != "complete" {
		t.Fatalf("expected stage complete, got %s", result.Stage)
	}
	t.Logf("Full pass: lots=%.4f risk=%.4f method=%s", result.Lots, result.RiskUsed, result.Method)
}

func TestPipeline_CapabilityBlocks(t *testing.T) {
	t.Parallel()
	capStore := NewCapabilityStore()
	// u1 defaults to Tier0

	p := NewSignalPipeline(PipelineConfig{CapStore: capStore})

	sig := &SignalRequest{UserID: "u1", AccountID: "a1", Symbol: "EURUSD"}
	result := p.Process(context.Background(), sig)
	if result.Allowed {
		t.Fatal("Tier0 should be blocked")
	}
	if result.Stage != "capability" {
		t.Fatalf("want stage capability, got %s", result.Stage)
	}
}

func TestPipeline_KillSwitchBlocks(t *testing.T) {
	t.Parallel()
	capStore := NewCapabilityStore()
	capStore.Set(&Capability{UserID: "u1", Tier: Tier3LiveFull, KillSwitchOn: true})

	p := NewSignalPipeline(PipelineConfig{CapStore: capStore})

	sig := &SignalRequest{UserID: "u1", AccountID: "a1", Symbol: "EURUSD"}
	result := p.Process(context.Background(), sig)
	if result.Allowed {
		t.Fatal("killswitch should block all orders")
	}
}

func TestPipeline_HardLimitBlocks_MarginFloor(t *testing.T) {
	t.Parallel()
	capStore := NewCapabilityStore()
	capStore.Set(&Capability{UserID: "u1", Tier: Tier2LiveLimited})

	hardLimit := NewHardLimitEvaluator(&MarginFloorRule{FloorRatio: 1.0})

	p := NewSignalPipeline(PipelineConfig{CapStore: capStore, HardLimit: hardLimit})

	sig := &SignalRequest{
		UserID: "u1", AccountID: "a1", Symbol: "EURUSD", Side: "buy",
		SignalStrength: 10.0, Price: 1.0850,
		FreeMargin: 5, // 5 < 10*1.085*1.0 = 10.85 → blocked
	}
	result := p.Process(context.Background(), sig)
	if result.Allowed {
		t.Fatal("margin floor should block")
	}
	if result.Stage != "hardlimit" {
		t.Fatalf("want stage hardlimit, got %s", result.Stage)
	}
}

func TestPipeline_HardLimitBlocks_ContractExpiry(t *testing.T) {
	t.Parallel()
	capStore := NewCapabilityStore()
	capStore.Set(&Capability{UserID: "u1", Tier: Tier2LiveLimited})

	hardLimit := NewHardLimitEvaluator(&ContractExpiryRule{CoolingOffHours: 48})

	p := NewSignalPipeline(PipelineConfig{CapStore: capStore, HardLimit: hardLimit})

	sig := &SignalRequest{
		UserID: "u1", AccountID: "a1", Symbol: "ESM6", Side: "buy",
		SignalStrength: 1.0, Price: 4500,
		ContractExpiry: time.Now().Add(1 * time.Hour), // expires in 1h, window is 48h
		FreeMargin: 100000,
	}
	result := p.Process(context.Background(), sig)
	if result.Allowed {
		t.Fatal("contract expiry should block")
	}
}

func TestPipeline_PlatformLimitsBlocks(t *testing.T) {
	t.Parallel()
	capStore := NewCapabilityStore()
	capStore.Set(&Capability{UserID: "u1", Tier: Tier2LiveLimited})

	agg := NewPlatformAggregator()
	agg.UpdatePosition("a1", &AggregatorPosition{Canonical: "EURUSD", NetVolume: 10, Notional: 15_000_000, Margin: 2_000_000})
	agg.Recalculate(nil)

	limits := &PlatformLimits{MaxTotalGrossExposure: 10_000_000}

	p := NewSignalPipeline(PipelineConfig{
		CapStore: capStore,
		Platform: agg,
		Limits:   limits,
	})

	sig := &SignalRequest{UserID: "u1", AccountID: "a1", Symbol: "EURUSD"}
	result := p.Process(context.Background(), sig)
	if result.Allowed {
		t.Fatal("platform limit should block")
	}
	if result.Stage != "platform_limits" {
		t.Fatalf("want stage platform_limits, got %s", result.Stage)
	}
}

func TestPipeline_RiskEngineBlocks(t *testing.T) {
	t.Parallel()
	capStore := NewCapabilityStore()
	capStore.Set(&Capability{UserID: "u1", Tier: Tier2LiveLimited})

	engine := NewEngine(maxPositionRule(1)) // max 1 position

	p := NewSignalPipeline(PipelineConfig{CapStore: capStore, Engine: engine})

	sig := &SignalRequest{
		UserID: "u1", AccountID: "a1", Symbol: "EURUSD", Side: "buy",
		Positions: 5, // exceeds limit of 1
	}
	result := p.Process(context.Background(), sig)
	if result.Allowed {
		t.Fatal("risk engine should block")
	}
	if result.Stage != "risk_engine" {
		t.Fatalf("want stage risk_engine, got %s", result.Stage)
	}
}

func TestPipeline_NoSizer(t *testing.T) {
	t.Parallel()
	capStore := NewCapabilityStore()
	capStore.Set(&Capability{UserID: "u1", Tier: Tier2LiveLimited})

	p := NewSignalPipeline(PipelineConfig{CapStore: capStore})

	sig := &SignalRequest{UserID: "u1", AccountID: "a1", Symbol: "EURUSD"}
	result := p.Process(context.Background(), sig)
	if result.Allowed {
		t.Fatal("no sizer should block")
	}
	if result.Stage != "sizer" {
		t.Fatalf("want stage sizer, got %s", result.Stage)
	}
}

func TestPipeline_ZeroLotsFromSizer(t *testing.T) {
	t.Parallel()
	capStore := NewCapabilityStore()
	capStore.Set(&Capability{UserID: "u1", Tier: Tier2LiveLimited})

	sizer := &VolTargetSizer{RiskBudgetPct: 0.01, MinLots: 100} // min lots > what we can afford

	p := NewSignalPipeline(PipelineConfig{CapStore: capStore, Sizer: sizer})

	sig := &SignalRequest{
		UserID: "u1", AccountID: "a1", Symbol: "EURUSD",
		Price: 1.0850, ATR: 0.0035, ContractSize: 100000, HoldingDays: 5,
		Equity: 1000, FreeMargin: 500,
	}
	result := p.Process(context.Background(), sig)
	if result.Allowed {
		t.Fatal("zero lots should be rejected")
	}
	if result.Stage != "sizer" {
		t.Fatalf("want stage sizer, got %s", result.Stage)
	}
}

func TestPipeline_BlockAllocation_MultiAccount(t *testing.T) {
	t.Parallel()
	capStore := NewCapabilityStore()
	capStore.Set(&Capability{UserID: "u1", Tier: Tier2LiveLimited})

	sizer := &VolTargetSizer{RiskBudgetPct: 0.01, MaxLots: 100}
	alloc := &ProRataAllocator{}

	p := NewSignalPipeline(PipelineConfig{
		CapStore:  capStore,
		Sizer:     sizer,
		Allocator: alloc,
	})

	sig := &SignalRequest{
		UserID: "u1", AccountID: "a1", Symbol: "EURUSD",
		Price: 1.0850, ATR: 0.0035, ContractSize: 100000, HoldingDays: 5,
		Equity: 100000, FreeMargin: 50000,
		TargetAccounts: []AllocAccount{
			{AccountID: "a1", Equity: 60_000, FreeMargin: 30_000},
			{AccountID: "a2", Equity: 40_000, FreeMargin: 20_000},
		},
	}
	result := p.Process(context.Background(), sig)
	if !result.Allowed {
		t.Fatalf("expected pass, got %s: %s", result.Stage, result.Reason)
	}
	if len(result.Allocations) == 0 {
		t.Fatal("expected multi-account allocations")
	}
	if len(result.Allocations) != 2 {
		t.Fatalf("want 2 allocations, got %d", len(result.Allocations))
	}
	t.Logf("Block alloc: lots=%.4f allocs=%v", result.Lots, result.Allocations)
}

func TestPipeline_MinimalConfig(t *testing.T) {
	t.Parallel()
	sizer := &VolTargetSizer{RiskBudgetPct: 0.01}

	p := NewSignalPipeline(PipelineConfig{Sizer: sizer})

	sig := &SignalRequest{
		AccountID: "a1", Symbol: "EURUSD",
		Price: 1.0850, ATR: 0.0035, ContractSize: 100000, HoldingDays: 5,
		Equity: 100000,
	}
	result := p.Process(context.Background(), sig)
	if !result.Allowed {
		t.Fatalf("minimal config should pass, got %s: %s", result.Stage, result.Reason)
	}
	if result.Lots <= 0 {
		t.Fatal("expected non-zero lots")
	}
}

func TestPipeline_KellySizerIntegration(t *testing.T) {
	t.Parallel()
	capStore := NewCapabilityStore()
	capStore.Set(&Capability{UserID: "u1", Tier: Tier2LiveLimited})

	sizer := &KellyFractionSizer{
		WinProb:      0.6,
		WinLossRatio: 2.0,
		Fraction:     0.5,
		KellyMax:     0.25,
	}

	p := NewSignalPipeline(PipelineConfig{CapStore: capStore, Sizer: sizer})

	sig := &SignalRequest{
		UserID: "u1", AccountID: "a1", Symbol: "EURUSD",
		Price: 1.0850, ContractSize: 100000,
		Equity: 100000,
	}
	result := p.Process(context.Background(), sig)
	if !result.Allowed {
		t.Fatalf("Kelly sizer should pass, got %s: %s", result.Stage, result.Reason)
	}
	if result.Method != "kelly_fraction" {
		t.Fatalf("want kelly_fraction, got %s", result.Method)
	}
	t.Logf("Kelly pipeline: lots=%.4f risk=%.4f", result.Lots, result.RiskUsed)
}

// maxPositionRule is a test helper that blocks if positions >= limit.
type maxPositionRule int

func (r maxPositionRule) Name() string                 { return "max_position" }
func (r maxPositionRule) Check(_ context.Context, req *CheckRequest) *CheckResult {
	if req.Positions >= int(r) {
		return &CheckResult{Passed: false, Reason: "too many positions", Rule: "max_position"}
	}
	return &CheckResult{Passed: true, Rule: "max_position"}
}
