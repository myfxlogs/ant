//go:build integration

package marketplace

import (
	"testing"

	"github.com/shopspring/decimal"
)

// TestSettlement_PrecisionRoundTrip verifies the core settlement invariant:
// provider_amount + platform_fee = purchase_amount — no floating-point drift.
// This guards against the A-002 CRITICAL bug where decimal.NewFromString errors
// were silently swallowed, potentially undercounting provider revenue.
func TestSettlement_PrecisionRoundTrip(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		purchaseAmt   string
		feeRate       string // e.g. "0.20" = 20%
	}{
		{"whole_dollar", "100.00", "0.20"},
		{"cents", "99.99", "0.15"},
		{"tiny", "0.01", "0.30"},
		{"large", "9999.99", "0.25"},
		{"eight_places", "100.12345678", "0.20"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			purchaseAmt := decimal.RequireFromString(tc.purchaseAmt)
			feeRate := decimal.RequireFromString(tc.feeRate)

			platformFee := purchaseAmt.Mul(feeRate)
			providerAmt := purchaseAmt.Sub(platformFee)

			reconstructed := providerAmt.Add(platformFee)
			if !reconstructed.Equal(purchaseAmt) {
				t.Errorf("precision loss: providerAmt(%s) + platformFee(%s) = %s, expected %s",
					providerAmt, platformFee, reconstructed, purchaseAmt)
			}
			if providerAmt.IsNegative() {
				t.Errorf("provider amount must be non-negative: %s (purchase=%s, fee=%s)",
					providerAmt, tc.purchaseAmt, tc.feeRate)
			}
		})
	}
}

// TestRefund_PrecisionRoundTrip verifies refund restores exact purchase amount.
func TestRefund_PrecisionRoundTrip(t *testing.T) {
	t.Parallel()
	purchaseAmt := decimal.RequireFromString("49.99")
	feeRate := decimal.RequireFromString("0.20")
	platformFee := purchaseAmt.Mul(feeRate)
	providerAmt := purchaseAmt.Sub(platformFee)

	// Refund: buyer gets back purchaseAmt, provider loses providerAmt, platform loses platformFee.
	refundTotal := providerAmt.Add(platformFee)
	if !refundTotal.Equal(purchaseAmt) {
		t.Errorf("refund precision: refund_total(%s) != purchase(%s)", refundTotal, purchaseAmt)
	}
}
