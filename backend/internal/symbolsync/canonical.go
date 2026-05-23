// Package symbolsync — canonical symbol name normalisation.
// Ported from alfq. Handles broker-specific suffixes → canonical form.
package symbolsync

import "strings"

// Canonicalize converts a broker-specific symbol name to canonical form.
//
// Strategy (ordered by specificity):
//  1. Strip dotted suffixes: .ECN, .RAW, .PRO, .STP, .M, .I, .X
//  2. Strip bare suffixes: M, I (only when preceded by a letter, to avoid
//     stripping digit-suffix pairs like US30 → US3)
//  3. Strip trailing special chars: #, !
//
// Examples:
//
//	BTCUSDm    → BTCUSD
//	BTCUSD.M   → BTCUSD
//	BTCUSD.x   → BTCUSD
//	BTCUSDpro  → NOT stripped (need canonical_symbols dict lookup)
//	BTCUSD#    → BTCUSD
//	EURUSD.ECN → EURUSD
//	EURUSDc    → NOT stripped (different contract, not position alias)
//
// Note: .c suffix is intentionally NOT stripped — EURUSDc is a different
// contract type, not a position alias. Those go through partial=true + manual review.
func Canonicalize(raw string) string {
	raw = strings.ToUpper(raw)

	// Ordered by specificity: longer/dotted suffixes first
	suffixes := []string{".ECN", ".RAW", ".PRO", ".STP", ".M", ".I", ".X"}
	for _, s := range suffixes {
		if strings.HasSuffix(raw, s) && len(raw) > len(s)+4 {
			return strings.TrimSuffix(raw, s)
		}
	}

	// Bare suffixes (no dot) — only strip if the base is long enough to be a real symbol
	// and the character before the suffix is a letter (not a digit)
	bareSuffixes := []string{"M", "I"}
	for _, s := range bareSuffixes {
		if strings.HasSuffix(raw, s) && len(raw) > len(s)+5 {
			prev := raw[len(raw)-len(s)-1]
			if prev >= 'A' && prev <= 'Z' {
				return strings.TrimSuffix(raw, s)
			}
		}
	}

	// Strip trailing special chars: #, ! (e.g. BTCUSD# → BTCUSD)
	if last := raw[len(raw)-1]; last == '#' || last == '!' {
		if len(raw) > 6 {
			return raw[:len(raw)-1]
		}
	}

	return raw
}
