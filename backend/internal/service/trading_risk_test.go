package service

import (
	"context"
	"testing"
	"time"

	"anttrader/internal/model"
)

func TestValidateVolumeStep(t *testing.T) {
	sr := SymbolRule{
		MinLot:  0.01,
		MaxLot:  10,
		LotStep: 0.01,
	}
	if err := validateVolume(sr, 0.02); err != nil {
		t.Fatalf("expected valid volume, got err=%v", err)
	}
	if err := validateVolume(sr, 0.015); err == nil {
		t.Fatalf("expected invalid step volume error")
	}
}

func TestValidateMarketSessionWeekend(t *testing.T) {
	// Deadline trick: validateMarketSession uses deadline-1s when deadline exists.
	sat := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC) // Saturday
	ctx, cancel := context.WithDeadline(context.Background(), sat)
	defer cancel()
	err := validateMarketSession(ctx)
	if err == nil {
		t.Fatalf("expected weekend market closed error")
	}
	re, ok := AsRiskError(err)
	if !ok || re.Code != RiskMarketSessionClosed {
		t.Fatalf("expected RiskMarketSessionClosed, got err=%v", err)
	}
}

func TestValidateFreezeDistance(t *testing.T) {
	sr := SymbolRule{
		FreezeDistancePoints: 50,
		Point:                0.00001,
	}
	price := 1.10000
	stopLoss := 1.09990 // distance 0.00010 < 50*0.00001=0.00050
	err := validateFreezeDistance(sr, price, stopLoss, 0)
	if err == nil {
		t.Fatalf("expected freeze distance rejection")
	}
	re, ok := AsRiskError(err)
	if !ok || re.Code != RiskOrderFrozenZone {
		t.Fatalf("expected RiskOrderFrozenZone, got err=%v", err)
	}
}

func TestValidateOrderCloseRisk(t *testing.T) {
	vctx := &RiskValidationContext{
		AccountRule: AccountRule{TradeEnabled: true},
		SymbolRule:  SymbolRule{TradeEnabled: true},
	}
	req := &OrderCloseRequest{
		AccountID: "acc",
		Ticket:    1001,
		Volume:    1.2,
	}
	position := &PositionResponse{
		Ticket: 1001,
		Volume: 1.0,
	}
	err := validateOrderCloseRisk(vctx, req, position)
	if err == nil {
		t.Fatalf("expected close volume invalid error")
	}
	re, ok := AsRiskError(err)
	if !ok || re.Code != RiskVolumeInvalid {
		t.Fatalf("expected RiskVolumeInvalid, got err=%v", err)
	}
}

func TestResolverBuildsContext(t *testing.T) {
	acc := &model.MTAccount{
		MTType:     "MT5",
		IsDisabled: false,
		IsInvestor: false,
		FreeMargin: 1000,
	}
	resolver := newRiskRuleResolver()
	ctx, err := resolver.resolve(acc, "EURUSD", 3)
	if err != nil {
		t.Fatalf("resolve unexpected error: %v", err)
	}
	if ctx.SymbolRule.Symbol != "EURUSD" {
		t.Fatalf("unexpected symbol: %s", ctx.SymbolRule.Symbol)
	}
	if !ctx.AccountRule.TradeEnabled {
		t.Fatalf("expected trade enabled")
	}
}

func TestRiskDecisionFromError(t *testing.T) {
	err := NewRiskError(RiskVolumeInvalid, "volume out of min/max range", false)
	decision := RiskDecisionFromError(err, model.RiskDecisionSourceManual)
	if decision == nil {
		t.Fatalf("expected decision")
	}
	if decision.Allowed {
		t.Fatalf("expected rejected decision")
	}
	if decision.Source != model.RiskDecisionSourceManual {
		t.Fatalf("unexpected source: %s", decision.Source)
	}
	if decision.Code != string(RiskVolumeInvalid) {
		t.Fatalf("unexpected code: %s", decision.Code)
	}
	if decision.Reason != "volume out of min/max range" {
		t.Fatalf("unexpected reason: %s", decision.Reason)
	}
}
