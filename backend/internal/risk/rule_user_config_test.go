package risk

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// mockConfigStore implements a simple in-memory config for testing.
type mockConfigStore struct {
	config *UserRiskConfig
	err    error
}

func (m *mockConfigStore) store() func(ctx context.Context, accountID string) (*UserRiskConfig, error) {
	return func(_ context.Context, _ string) (*UserRiskConfig, error) {
		return m.config, m.err
	}
}

func TestUserRiskConfigRule_NilStorePasses(t *testing.T) {
	rule := &UserRiskConfigRule{Store: nil}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{AccountId: "test", Volume: "100", Type: "buy"}, nil)
	if !result.Allowed {
		t.Error("nil store should pass")
	}
}

func TestUserRiskConfigRule_NoConfigPasses(t *testing.T) {
	store := &mockConfigStore{config: nil}
	rule := &UserRiskConfigRule{Store: store.store()}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{AccountId: "test", Volume: "1.0", Type: "buy"}, nil)
	if !result.Allowed {
		t.Error("nil config should pass (no restriction)")
	}
}

func TestUserRiskConfigRule_BlocksOverMaxLotSize(t *testing.T) {
	store := &mockConfigStore{config: &UserRiskConfig{MaxLotSize: decimal.NewFromFloat(5.0)}}
	rule := &UserRiskConfigRule{Store: store.store()}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{AccountId: "test", Volume: "10.0", Type: "buy"}, nil)
	if result.Allowed {
		t.Error("10 lots should be blocked by max 5.0")
	}
}

func TestUserRiskConfigRule_PassesWithinMaxLotSize(t *testing.T) {
	store := &mockConfigStore{config: &UserRiskConfig{MaxLotSize: decimal.NewFromFloat(5.0)}}
	rule := &UserRiskConfigRule{Store: store.store()}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{AccountId: "test", Volume: "3.0", Type: "buy"}, nil)
	if !result.Allowed {
		t.Errorf("3 lots should pass max 5.0, got: %s", result.Reason)
	}
}

func TestUserRiskConfigRule_BlocksOverMaxPositions(t *testing.T) {
	store := &mockConfigStore{config: &UserRiskConfig{MaxPositions: 3}}
	rule := &UserRiskConfigRule{Store: store.store()}
	state := &AccountState{OpenPositions: 3}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{AccountId: "test", Volume: "1.0", Type: "buy"}, state)
	if result.Allowed {
		t.Error("opening 4th position should be blocked when max is 3")
	}
}

func TestUserRiskConfigRule_PassesUnderMaxPositions(t *testing.T) {
	store := &mockConfigStore{config: &UserRiskConfig{MaxPositions: 5}}
	rule := &UserRiskConfigRule{Store: store.store()}
	state := &AccountState{OpenPositions: 3}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{AccountId: "test", Volume: "1.0", Type: "buy"}, state)
	if !result.Allowed {
		t.Errorf("opening 4th position should pass max 5, got: %s", result.Reason)
	}
}

func TestUserRiskConfigRule_BlocksOverMaxDrawdown(t *testing.T) {
	store := &mockConfigStore{config: &UserRiskConfig{MaxDrawdownPercent: decimal.NewFromFloat(10.0)}}
	rule := &UserRiskConfigRule{Store: store.store()}
	state := &AccountState{
		PeakEquity: newDec("100000"),
		Equity:     newDec("85000"),
	}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{AccountId: "test", Volume: "1.0", Type: "buy"}, state)
	if result.Allowed {
		t.Error("15% drawdown should be blocked at 10% limit")
	}
}

func TestUserRiskConfigRule_PassesUnderMaxDrawdown(t *testing.T) {
	store := &mockConfigStore{config: &UserRiskConfig{MaxDrawdownPercent: decimal.NewFromFloat(20.0)}}
	rule := &UserRiskConfigRule{Store: store.store()}
	state := &AccountState{
		PeakEquity: newDec("100000"),
		Equity:     newDec("95000"),
	}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{AccountId: "test", Volume: "1.0", Type: "buy"}, state)
	if !result.Allowed {
		t.Errorf("5%% drawdown should pass 20%% limit, got: %s", result.Reason)
	}
}

func TestUserRiskConfigRule_BlocksOverRiskPercent(t *testing.T) {
	store := &mockConfigStore{config: &UserRiskConfig{MaxRiskPercent: decimal.NewFromFloat(2.0)}}
	rule := &UserRiskConfigRule{Store: store.store()}
	state := &AccountState{Equity: newDec("10000")}
	// 1 lot × 500 = 500 notional / 10000 equity = 5% risk
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{AccountId: "test", Volume: "1.0", Price: "500", Type: "buy"}, state)
	if result.Allowed {
		t.Error("5% risk per trade should be blocked at 2% limit")
	}
}

func TestUserRiskConfigRule_PassesUnderRiskPercent(t *testing.T) {
	store := &mockConfigStore{config: &UserRiskConfig{MaxRiskPercent: decimal.NewFromFloat(5.0)}}
	rule := &UserRiskConfigRule{Store: store.store()}
	state := &AccountState{Equity: newDec("10000")}
	result := rule.Check(context.Background(),
		&antv1.OrderIntent{AccountId: "test", Volume: "1.0", Price: "400", Type: "buy"}, state)
	if !result.Allowed {
		t.Errorf("4%% risk should pass 5%% limit, got: %s", result.Reason)
	}
}
