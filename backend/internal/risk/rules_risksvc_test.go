package risk

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
)

func TestContractExpiryRule_PassesForSpot(t *testing.T) {
	rule := &ContractExpiryRule{
		CoolingOffHours: 2,
		ExpiryProvider:  func(_ string) int64 { return 0 }, // no expiry
	}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{Symbol: "EURUSD"}, nil)
	if !result.Allowed {
		t.Errorf("spot FX should always pass, got: %s", result.Reason)
	}
}

func TestContractExpiryRule_PassesBeforeCoolingOff(t *testing.T) {
	future := time.Now().Add(5 * time.Hour).UnixMilli()
	rule := &ContractExpiryRule{
		CoolingOffHours: 2,
		ExpiryProvider:  func(_ string) int64 { return future },
	}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{Symbol: "CL"}, nil)
	if !result.Allowed {
		t.Errorf("5h before expiry should pass, got: %s", result.Reason)
	}
}

func TestContractExpiryRule_BlocksInsideCoolingOff(t *testing.T) {
	nearFuture := time.Now().Add(1 * time.Hour).UnixMilli()
	rule := &ContractExpiryRule{
		CoolingOffHours: 2,
		ExpiryProvider:  func(_ string) int64 { return nearFuture },
	}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{Symbol: "CL"}, nil)
	if result.Allowed {
		t.Error("1h before expiry with 2h cooling-off should block")
	}
}

func TestContractExpiryRule_NilProviderPasses(t *testing.T) {
	rule := &ContractExpiryRule{CoolingOffHours: 2}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{Symbol: "CL"}, nil)
	if !result.Allowed {
		t.Error("nil expiry provider should pass (no data = allow)")
	}
}

func TestMarginFloorRule_PassesWhenMarginOk(t *testing.T) {
	rule := &MarginFloorRule{FloorRatio: 1.0}
	state := &AccountState{
		FreeMargin: newDec("50000"),
	}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{Volume: "1.0", Price: "1000"}, state)
	if !result.Allowed {
		t.Errorf("free margin 50k >> 1k required, should pass, got: %s", result.Reason)
	}
}

func TestMarginFloorRule_BlocksWhenMarginLow(t *testing.T) {
	rule := &MarginFloorRule{FloorRatio: 1.0}
	state := &AccountState{
		FreeMargin: newDec("500"),
	}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{Volume: "1.0", Price: "1000"}, state)
	if result.Allowed {
		t.Error("free margin 500 < 1000 required, should block")
	}
}

func TestMarginFloorRule_SkipsMarketOrder(t *testing.T) {
	rule := &MarginFloorRule{FloorRatio: 1.0}
	state := &AccountState{
		FreeMargin: newDec("1"),
	}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{Volume: "0", Price: "0"}, state) // market order
	if !result.Allowed {
		t.Error("market orders (price=0) should skip margin floor check")
	}
}

func TestMarginFloorRule_NilStatePasses(t *testing.T) {
	rule := &MarginFloorRule{FloorRatio: 1.0}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{Volume: "1.0", Price: "100"}, nil)
	if !result.Allowed {
		t.Error("nil state should pass (can't check without data)")
	}
}

func TestKycJurisdictionRule_NilGatePasses(t *testing.T) {
	rule := &KycJurisdictionGateRule{Gate: nil}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{UserId: "u1"}, nil)
	if !result.Allowed {
		t.Error("nil gate should pass (not yet wired)")
	}
}

func TestCapabilityTierRule_NilStorePasses(t *testing.T) {
	rule := &CapabilityTierRule{Store: nil}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{UserId: "u1", Volume: "100"}, nil)
	if !result.Allowed {
		t.Error("nil store should pass (not yet wired)")
	}
}

func TestCapabilityTierRule_NoTierUserPasses(t *testing.T) {
	store := &testCapStore{tier: CapabilityTier{}} // empty tier = no restriction
	rule := &CapabilityTierRule{Store: store}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{UserId: "u1", Volume: "5.0", Symbol: "XAUUSD"}, nil)
	if !result.Allowed {
		t.Errorf("empty tier should pass, got: %s", result.Reason)
	}
}

func TestCapabilityTierRule_BlocksOverVolume(t *testing.T) {
	store := &testCapStore{tier: CapabilityTier{
		Name:      "basic",
		MaxVolume: 1.0,
	}}
	rule := &CapabilityTierRule{Store: store}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{UserId: "u1", Volume: "5.0", Symbol: "EURUSD"}, nil)
	if result.Allowed {
		t.Error("5.0 lots should be blocked by basic tier (max 1.0)")
	}
}

func TestCapabilityTierRule_BlocksDisallowedSymbol(t *testing.T) {
	store := &testCapStore{tier: CapabilityTier{
		Name:           "basic",
		MaxVolume:      10.0,
		AllowedSymbols: []string{"EURUSD", "GBPUSD"},
	}}
	rule := &CapabilityTierRule{Store: store}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{UserId: "u1", Volume: "1.0", Symbol: "XAUUSD"}, nil)
	if result.Allowed {
		t.Error("XAUUSD should be blocked by basic tier (only EURUSD, GBPUSD)")
	}
}

func TestCapabilityTierRule_PassesAllowedSymbol(t *testing.T) {
	store := &testCapStore{tier: CapabilityTier{
		Name:           "basic",
		MaxVolume:      10.0,
		AllowedSymbols: []string{"EURUSD", "GBPUSD"},
	}}
	rule := &CapabilityTierRule{Store: store}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{UserId: "u1", Volume: "1.0", Symbol: "EURUSD"}, nil)
	if !result.Allowed {
		t.Errorf("EURUSD should be allowed, got: %s", result.Reason)
	}
}

// ── helpers ─────────────────────────────────────────────────────────────

type testCapStore struct {
	tier CapabilityTier
}

func (s *testCapStore) GetTier(_ context.Context, _ string) (CapabilityTier, error) {
	return s.tier, nil
}

func newDec(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}
