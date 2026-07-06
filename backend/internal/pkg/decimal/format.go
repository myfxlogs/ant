// Package decimal provides shared decimal.Decimal utilities used across connect handlers.
package decimal

import "github.com/shopspring/decimal"

// FormatPrice formats a decimal.Decimal price with dynamic decimal precision:
//   - Price > 100:  3 digits (JPY pairs, e.g. 149.250)
//   - Price > 1:    5 digits (standard forex, e.g. 1.12345)
//   - Price <= 1:   6 digits (crypto or fractional assets)
func FormatPrice(p decimal.Decimal) string {
	switch {
	case p.GreaterThan(decimal.NewFromInt(100)):
		return p.StringFixed(3)
	case p.GreaterThan(decimal.NewFromInt(1)):
		return p.StringFixed(5)
	default:
		return p.StringFixed(6)
	}
}
