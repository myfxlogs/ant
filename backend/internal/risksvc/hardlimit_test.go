package risksvc

import (
	"context"
	"testing"
	"time"
)

func TestHardLimit_MarginFloor_Allowed(t *testing.T) {
	t.Parallel()
	rule := &MarginFloorRule{FloorRatio: 1.0}
	req := &HardLimitRequest{
		Volume:     0.1,
		Price:      1.0850,
		FreeMargin: 5000,
	}
	if err := rule.Check(context.Background(), req); err != nil {
		t.Fatalf("expected pass, got: %v", err)
	}
}

func TestHardLimit_MarginFloor_Blocked(t *testing.T) {
	t.Parallel()
	rule := &MarginFloorRule{FloorRatio: 1.0}
	req := &HardLimitRequest{
		Volume:     10.0,
		Price:      1.0850,
		FreeMargin: 5,
	}
	err := rule.Check(context.Background(), req)
	if err == nil {
		t.Fatal("expected block, got nil")
	}
	if he, ok := err.(*HardLimitError); ok {
		if he.Rule != "margin_floor" {
			t.Fatalf("expected rule=margin_floor, got %s", he.Rule)
		}
	}
}

func TestHardLimit_MarginFloor_DefaultRatio(t *testing.T) {
	t.Parallel()
	rule := &MarginFloorRule{} // FloorRatio defaults to 1.0
	req := &HardLimitRequest{
		Volume:     0.01,
		Price:      1.0850,
		FreeMargin: 100,
	}
	if err := rule.Check(context.Background(), req); err != nil {
		t.Fatalf("expected pass with default ratio, got: %v", err)
	}
}

func TestHardLimit_ContractExpiry_Spot(t *testing.T) {
	t.Parallel()
	rule := &ContractExpiryRule{CoolingOffHours: 24}
	req := &HardLimitRequest{
		ContractExpiry: time.Time{}, // zero = spot
	}
	if err := rule.Check(context.Background(), req); err != nil {
		t.Fatalf("spot instrument should pass: %v", err)
	}
}

func TestHardLimit_ContractExpiry_FarFuture(t *testing.T) {
	t.Parallel()
	rule := &ContractExpiryRule{CoolingOffHours: 24}
	req := &HardLimitRequest{
		ContractExpiry: time.Now().Add(30 * 24 * time.Hour), // 30 days away
	}
	if err := rule.Check(context.Background(), req); err != nil {
		t.Fatalf("far future expiry should pass: %v", err)
	}
}

func TestHardLimit_ContractExpiry_TooClose(t *testing.T) {
	t.Parallel()
	rule := &ContractExpiryRule{CoolingOffHours: 24}
	req := &HardLimitRequest{
		ContractExpiry: time.Now().Add(1 * time.Hour), // 1 hour away
	}
	err := rule.Check(context.Background(), req)
	if err == nil {
		t.Fatal("imminent expiry should block")
	}
	if he, ok := err.(*HardLimitError); ok {
		if he.Rule != "contract_expiry" {
			t.Fatalf("expected rule=contract_expiry, got %s", he.Rule)
		}
	}
}

func TestHardLimit_ContractExpiry_DefaultWindow(t *testing.T) {
	t.Parallel()
	rule := &ContractExpiryRule{} // defaults to 24h
	req := &HardLimitRequest{
		ContractExpiry: time.Now().Add(23 * time.Hour), // within 24h
	}
	if err := rule.Check(context.Background(), req); err == nil {
		t.Fatal("should block within default 24h window")
	}
}

func TestHardLimitEvaluator_AllPass(t *testing.T) {
	t.Parallel()
	e := NewHardLimitEvaluator(
		&MarginFloorRule{FloorRatio: 1.0},
		&ContractExpiryRule{CoolingOffHours: 24},
	)
	req := &HardLimitRequest{
		Volume:         0.1,
		Price:          1.0850,
		FreeMargin:     5000,
		ContractExpiry: time.Now().Add(30 * 24 * time.Hour),
	}
	if err := e.Evaluate(context.Background(), req); err != nil {
		t.Fatalf("all rules should pass: %v", err)
	}
}

func TestHardLimitEvaluator_FirstBlocks(t *testing.T) {
	t.Parallel()
	e := NewHardLimitEvaluator(
		&MarginFloorRule{FloorRatio: 1.0},
		&ContractExpiryRule{CoolingOffHours: 24},
	)
	req := &HardLimitRequest{
		Volume:         10.0,
		Price:          1.0850,
		FreeMargin:     5,
		ContractExpiry: time.Now().Add(1 * time.Hour),
	}
	err := e.Evaluate(context.Background(), req)
	if err == nil {
		t.Fatal("margin floor should block first")
	}
	if he, ok := err.(*HardLimitError); ok {
		if he.Rule != "margin_floor" {
			t.Fatalf("expected margin_floor to block first, got %s", he.Rule)
		}
	}
}

func TestHardLimit_KycJurisdiction_Name(t *testing.T) {
	t.Parallel()
	rule := &KycJurisdictionRule{}
	if rule.Name() != "kyc_jurisdiction" {
		t.Fatalf("expected kyc_jurisdiction, got %s", rule.Name())
	}
}

func TestHardLimit_KillSwitch_Name(t *testing.T) {
	t.Parallel()
	rule := &KillSwitchRule{}
	if rule.Name() != "kill_switch" {
		t.Fatalf("expected kill_switch, got %s", rule.Name())
	}
}
