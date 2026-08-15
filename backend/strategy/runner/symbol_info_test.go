package runner

import (
	"testing"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// ── Task 4: Symbol info injection ────────────────────────────────────
//
// These tests verify that UpdateSymbolInfo injects Point/Digits/ContractSize/
// StopsLevel into the context, and that Point()/Digits()/Broker().SymbolInfo()
// return the injected values in harness mode (no executor).
// Remove UpdateSymbolInfo or the live* fields → these tests go red.

// TestUpdateSymbolInfo_Point verifies that UpdateSymbolInfo injects Point
// into the context, and Point() returns the injected value in harness mode.
func TestUpdateSymbolInfo_Point(t *testing.T) {
	r := New(Config{})
	r.UpdateSymbolInfo("0.00001", 5, "100000", "30")

	if r.ctx.Point().String() != "0.00001" {
		t.Errorf("Point()=%s, want 0.00001", r.ctx.Point().String())
	}
}

// TestUpdateSymbolInfo_Digits verifies that UpdateSymbolInfo injects Digits.
func TestUpdateSymbolInfo_Digits(t *testing.T) {
	r := New(Config{})
	r.UpdateSymbolInfo("0.001", 3, "1000", "50")

	if r.ctx.Digits() != 3 {
		t.Errorf("Digits()=%d, want 3", r.ctx.Digits())
	}
}

// TestUpdateSymbolInfo_BrokerSymbolInfo verifies that broker.SymbolInfo
// returns injected values in harness mode (no executor).
func TestUpdateSymbolInfo_BrokerSymbolInfo(t *testing.T) {
	r := New(Config{})
	r.UpdateSymbolInfo("0.001", 3, "1000", "50")

	info, err := r.broker.SymbolInfo("EURUSD")
	if err != nil {
		t.Fatalf("SymbolInfo error: %v", err)
	}
	if info.Point.String() != "0.001" {
		t.Errorf("Point=%s, want 0.001", info.Point.String())
	}
	if info.Digits != 3 {
		t.Errorf("Digits=%d, want 3", info.Digits)
	}
	if !info.ContractSize.Equal(decimal.NewFromInt(1000)) {
		t.Errorf("ContractSize=%s, want 1000", info.ContractSize.String())
	}
	if info.StopsLevel != 50 {
		t.Errorf("StopsLevel=%d, want 50", info.StopsLevel)
	}
}

// TestUpdateSymbolInfo_EmptyDefaults verifies that without UpdateSymbolInfo,
// harness mode returns zero values (not panic).
func TestUpdateSymbolInfo_EmptyDefaults(t *testing.T) {
	r := New(Config{})
	info, err := r.broker.SymbolInfo("EURUSD")
	if err != nil {
		t.Fatalf("SymbolInfo error: %v", err)
	}
	if !info.Point.IsZero() {
		t.Errorf("Point=%s, want 0 (default)", info.Point.String())
	}
	if info.Digits != 0 {
		t.Errorf("Digits=%d, want 0 (default)", info.Digits)
	}
}

// TestUpdateLiveState_MarginFreeMargin verifies that margin/free_margin
// are returned from broker.Account() in harness mode.
func TestUpdateLiveState_MarginFreeMargin(t *testing.T) {
	r := New(Config{})
	r.UpdateLiveState("10000", "10500", "500", "9500", nil)

	info := r.broker.Account()
	if !info.Margin.Equal(decimal.NewFromInt(500)) {
		t.Errorf("Margin=%s, want 500", info.Margin.String())
	}
	if !info.FreeMargin.Equal(decimal.NewFromInt(9500)) {
		t.Errorf("FreeMargin=%s, want 9500", info.FreeMargin.String())
	}
}

// TestUpdateLiveState_EmptyMargin verifies that empty margin strings
// default to zero (not panic).
func TestUpdateLiveState_EmptyMargin(t *testing.T) {
	r := New(Config{})
	r.UpdateLiveState("10000", "10500", "", "", nil)

	info := r.broker.Account()
	if !info.Margin.IsZero() {
		t.Errorf("Margin=%s, want 0 (empty default)", info.Margin.String())
	}
	if !info.FreeMargin.IsZero() {
		t.Errorf("FreeMargin=%s, want 0 (empty default)", info.FreeMargin.String())
	}
}

// ── Task 2: Account info with margin in harness mode ─────────────────

// TestBrokerImpl_Account_WithExecutor_StillUsesExecutor verifies that
// with an executor, Account() delegates to the executor (not harness path).
func TestBrokerImpl_Account_WithExecutor_StillUsesExecutor(t *testing.T) {
	r := New(Config{})
	exec := &mockExecutor{accountInfo: sdk.AccountInfo{
		Balance:    decimal.NewFromInt(99999),
		Equity:     decimal.NewFromInt(88888),
		Margin:     decimal.NewFromInt(77777),
		FreeMargin: decimal.NewFromInt(66666),
	}}
	r.broker.executor = exec

	info := r.broker.Account()
	if !info.Balance.Equal(decimal.NewFromInt(99999)) {
		t.Errorf("Balance=%s, want 99999 (executor)", info.Balance.String())
	}
	if !info.Margin.Equal(decimal.NewFromInt(77777)) {
		t.Errorf("Margin=%s, want 77777 (executor)", info.Margin.String())
	}
}
