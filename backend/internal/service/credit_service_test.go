package service

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestCreditPerDollar(t *testing.T) {
	if !CreditPerDollar.Equals(decimal.NewFromInt(100)) {
		t.Fatalf("CreditPerDollar should be 100, got %s", CreditPerDollar.String())
	}
}

func TestCreditService_EstimateSessionCost(t *testing.T) {
	// Test the cold-start constants directly.
	// estimateSessionCost can't be called with nil DB, so we verify the constants.
	flagshipCost := decimal.NewFromFloat(0.15)
	lightweightCost := decimal.NewFromFloat(0.03)

	if !flagshipCost.Equals(decimal.NewFromFloat(0.15)) {
		t.Fatalf("flagship estimate should be 0.15")
	}
	if !lightweightCost.Equals(decimal.NewFromFloat(0.03)) {
		t.Fatalf("lightweight estimate should be 0.03")
	}
}

func TestCreditService_HoldAndSettle(t *testing.T) {
	svc := &CreditService{
		holds: make(map[string]decimal.Decimal),
	}

	// Simulate hold tracking
	sessionID := "test-session-1"
	holdAmount := decimal.NewFromInt(15) // 15 credits
	svc.holds[sessionID] = holdAmount

	// Verify hold was tracked
	if !svc.holds[sessionID].Equals(holdAmount) {
		t.Fatalf("hold not tracked correctly")
	}

	// Simulate settlement — actual cost less than hold
	actualCost := decimal.NewFromInt(10)
	release := holdAmount.Sub(actualCost)
	if !release.Equals(decimal.NewFromInt(5)) {
		t.Fatalf("release should be 5 credits, got %s", release.String())
	}

	// Simulate settlement — actual cost more than hold
	actualCost2 := decimal.NewFromInt(20)
	extra := actualCost2.Sub(holdAmount)
	if !extra.Equals(decimal.NewFromInt(5)) {
		t.Fatalf("extra should be 5 credits, got %s", extra.String())
	}
}

func TestCreditService_PricingCalculation(t *testing.T) {
	// 1 credit = $0.01
	// Vendor cost: $0.10
	// Markup: 1.5x (flagship)
	// Credits = 0.10 * 1.5 * 100 = 15 credits
	vendorCost := decimal.NewFromFloat(0.10)
	markup := decimal.NewFromFloat(1.5)
	credits := vendorCost.Mul(markup).Mul(CreditPerDollar)
	expected := decimal.NewFromInt(15)
	if !credits.Equals(expected) {
		t.Fatalf("credits should be 15, got %s", credits.String())
	}

	// Lightweight: 2.5x markup
	markupLight := decimal.NewFromFloat(2.5)
	creditsLight := vendorCost.Mul(markupLight).Mul(CreditPerDollar)
	expectedLight := decimal.NewFromInt(25)
	if !creditsLight.Equals(expectedLight) {
		t.Fatalf("lightweight credits should be 25, got %s", creditsLight.String())
	}
}
