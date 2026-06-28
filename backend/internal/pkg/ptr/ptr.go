// Package ptr provides helper functions for creating pointers to primitive values.
package ptr

import "github.com/shopspring/decimal"

// Str returns a pointer to s, or nil if s is empty.
func Str(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// F64 returns a pointer to v.
func F64(v float64) *float64 { return &v }

// Bool returns a pointer to v.
func Bool(v bool) *bool { return &v }

// Decimal returns a pointer to v.
func Decimal(v decimal.Decimal) *decimal.Decimal { return &v }
