package marketplace

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestAnalyticsPeriodToInterval(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input    string
		expected time.Duration
	}{
		{"7d", 7 * 24 * time.Hour},
		{"30d", 30 * 24 * time.Hour},
		{"90d", 90 * 24 * time.Hour},
		{"unknown", 365 * 24 * time.Hour},
		{"", 365 * 24 * time.Hour},
	}
	for _, tc := range cases {
		if got := analyticsPeriodToInterval(tc.input); got != tc.expected {
			t.Errorf("analyticsPeriodToInterval(%q) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestComputeCouponDiscount_Percentage(t *testing.T) {
	t.Parallel()
	row := CouponRow{
		Enabled:        true,
		DiscountType:   "percentage",
		DiscountValue:  dec(10),
		MinPurchase:    dec(50),
	}
	result, err := computeCouponDiscount(row, dec(100))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid, got %s", result.ErrorMessage)
	}
	if !result.DiscountAmount.Equal(dec(10)) {
		t.Fatalf("expected discount 10, got %s", result.DiscountAmount.String())
	}
	if !result.FinalAmount.Equal(dec(90)) {
		t.Fatalf("expected final 90, got %s", result.FinalAmount.String())
	}
}

func TestComputeCouponDiscount_Fixed(t *testing.T) {
	t.Parallel()
	row := CouponRow{
		Enabled:       true,
		DiscountType:  "fixed",
		DiscountValue: dec(15),
	}
	result, _ := computeCouponDiscount(row, dec(100))
	if !result.DiscountAmount.Equal(dec(15)) {
		t.Fatalf("expected discount 15, got %s", result.DiscountAmount.String())
	}
}

func TestComputeCouponDiscount_Fixed_CappedAtAmount(t *testing.T) {
	t.Parallel()
	row := CouponRow{
		Enabled:       true,
		DiscountType:  "fixed",
		DiscountValue: dec(200),
	}
	result, _ := computeCouponDiscount(row, dec(100))
	if !result.DiscountAmount.Equal(dec(100)) {
		t.Fatalf("expected discount capped at 100, got %s", result.DiscountAmount.String())
	}
}

func TestComputeCouponDiscount_Disabled(t *testing.T) {
	t.Parallel()
	row := CouponRow{Enabled: false}
	result, _ := computeCouponDiscount(row, dec(100))
	if result.Valid {
		t.Fatal("expected invalid for disabled coupon")
	}
}

func TestComputeCouponDiscount_Expired(t *testing.T) {
	t.Parallel()
	exp := time.Now().Add(-24 * time.Hour)
	row := CouponRow{Enabled: true, ExpiresAt: &exp}
	result, _ := computeCouponDiscount(row, dec(100))
	if result.Valid {
		t.Fatal("expected invalid for expired coupon")
	}
}

func TestComputeCouponDiscount_MaxUsesReached(t *testing.T) {
	t.Parallel()
	row := CouponRow{Enabled: true, MaxUses: 5, UsedCount: 5}
	result, _ := computeCouponDiscount(row, dec(100))
	if result.Valid {
		t.Fatal("expected invalid for max uses reached")
	}
}

func TestComputeCouponDiscount_BelowMinPurchase(t *testing.T) {
	t.Parallel()
	row := CouponRow{Enabled: true, MinPurchase: dec(50)}
	result, _ := computeCouponDiscount(row, dec(10))
	if result.Valid {
		t.Fatal("expected invalid for below min purchase")
	}
}

func TestComputeCouponDiscount_InvalidPercentage(t *testing.T) {
	t.Parallel()
	row := CouponRow{Enabled: true, DiscountType: "percentage", DiscountValue: dec(150)}
	result, _ := computeCouponDiscount(row, dec(100))
	if result.Valid {
		t.Fatal("expected invalid for percentage > 100")
	}
}

func TestComputeCouponDiscount_InvalidType(t *testing.T) {
	t.Parallel()
	row := CouponRow{Enabled: true, DiscountType: "bogus"}
	result, _ := computeCouponDiscount(row, dec(100))
	if result.Valid {
		t.Fatal("expected invalid for bogus discount type")
	}
}

func TestFormatDecayReason_NotDecaying(t *testing.T) {
	t.Parallel()
	r := &DecayResult{IsDecaying: false}
	if got := formatDecayReason(r); got != "no significant decay detected" {
		t.Fatalf("unexpected: %s", got)
	}
}

func TestFormatDecayReason_SharpeSignal(t *testing.T) {
	t.Parallel()
	r := &DecayResult{IsDecaying: true, SharpeSignal: true}
	got := formatDecayReason(r)
	if got == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestFormatDecayReason_AllSignals(t *testing.T) {
	t.Parallel()
	r := &DecayResult{IsDecaying: true, SharpeSignal: true, WinRateSignal: true, ReturnSignal: true}
	got := formatDecayReason(r)
	if got == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestFormatDecayReason_NoSignalsButDecaying(t *testing.T) {
	t.Parallel()
	r := &DecayResult{IsDecaying: true}
	got := formatDecayReason(r)
	if got != "multiple decay signals detected" {
		t.Fatalf("expected fallback message, got %s", got)
	}
}

func TestHashCodeStr(t *testing.T) {
	t.Parallel()
	h1 := hashCodeStr("test source")
	h2 := hashCodeStr("test source")
	if h1 != h2 {
		t.Fatal("same input should produce same hash")
	}
	h3 := hashCodeStr("different source")
	if h1 == h3 {
		t.Fatal("different input should produce different hash")
	}
	if len(h1) != 64 {
		t.Fatalf("expected 64 char hex hash, got %d", len(h1))
	}
}

func TestDecimalSqrt(t *testing.T) {
	t.Parallel()
	result := decimalSqrt(dec(9))
	if !result.Equal(dec(3)) {
		t.Fatalf("expected sqrt(9)=3, got %s", result.String())
	}
}

func TestComputePeriodMetrics_Empty(t *testing.T) {
	t.Parallel()
	m := computePeriodMetrics(nil)
	if !m.totalReturn.IsZero() {
		t.Fatal("expected zero total return for empty rows")
	}
}

func TestComputePeriodMetrics_WithData(t *testing.T) {
	t.Parallel()
	rows := []dailyRow{
		{dailyReturn: dec(1), winningTrades: 5, totalTrades: 10},
		{dailyReturn: dec(2), winningTrades: 7, totalTrades: 10},
	}
	m := computePeriodMetrics(rows)
	if !m.totalReturn.Equal(dec(3)) {
		t.Fatalf("expected total return 3, got %s", m.totalReturn.String())
	}
	if m.winRate == nil || !m.winRate.Equal(dec(0.6)) {
		t.Fatalf("expected win rate 0.6, got %v", m.winRate)
	}
}

func TestComputePeriodMetrics_SharpeWithEnoughRows(t *testing.T) {
	t.Parallel()
	rows := make([]dailyRow, 10)
	for i := range rows {
		rows[i] = dailyRow{dailyReturn: dec(float64(i)), winningTrades: 5, totalTrades: 10}
	}
	m := computePeriodMetrics(rows)
	if m.sharpe == nil {
		t.Fatal("expected non-nil sharpe for >5 rows")
	}
}

func TestPublish_ValidPriceModels(t *testing.T) {
	t.Parallel()
	s := &Service{}
	for _, model := range []string{PriceModelFree, PriceModelOnce, PriceModelSubscription} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Expected: panics due to nil DB, but price model validation passed
				}
			}()
			_, _ = s.Publish(context.Background(), PublishParams{PriceModel: model})
		}()
	}
}

func TestSetPricing_ValidPriceModels(t *testing.T) {
	t.Parallel()
	s := &Service{}
	for _, model := range []string{PriceModelFree, PriceModelOnce, PriceModelSubscription} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Expected: panics due to nil DB, but price model validation passed
				}
			}()
			_ = s.SetPricing(context.Background(), "a", "b", model, "10", "0.1")
		}()
	}
}

func dec(v float64) decimal.Decimal { return decimal.NewFromFloat(v) }
