package symbolsync

import "testing"

func TestCanonicalize_DottedSuffixes(t *testing.T) {
	tests := []struct{ raw, expected string }{
		{"EURUSD.M", "EURUSD"},
		{"EURUSD.m", "EURUSD"},
		{"EURUSD.I", "EURUSD"},
		{"EURUSD.X", "EURUSD"},
		{"EURUSD.x", "EURUSD"},
		{"EURUSD.ECN", "EURUSD"},
		{"EURUSD.ecn", "EURUSD"},
		{"EURUSD.RAW", "EURUSD"},
		{"EURUSD.PRO", "EURUSD"},
		{"EURUSD.STP", "EURUSD"},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := Canonicalize(tt.raw); got != tt.expected {
				t.Errorf("Canonicalize(%q) = %q, want %q", tt.raw, got, tt.expected)
			}
		})
	}
}

func TestCanonicalize_BareSuffixes(t *testing.T) {
	tests := []struct{ raw, expected string }{
		{"BTCUSDm", "BTCUSD"},
		{"BTCUSDM", "BTCUSD"},
		{"XAUUSDi", "XAUUSD"},
		{"XAUUSDI", "XAUUSD"},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := Canonicalize(tt.raw); got != tt.expected {
				t.Errorf("Canonicalize(%q) = %q, want %q", tt.raw, got, tt.expected)
			}
		})
	}
}

func TestCanonicalize_TrailingSpecialChars(t *testing.T) {
	tests := []struct{ raw, expected string }{
		{"BTCUSD#", "BTCUSD"},
		{"BTCUSD!", "BTCUSD"},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := Canonicalize(tt.raw); got != tt.expected {
				t.Errorf("Canonicalize(%q) = %q, want %q", tt.raw, got, tt.expected)
			}
		})
	}
}

func TestCanonicalize_PreserveNonSuffix(t *testing.T) {
	// These should NOT be stripped (will be uppercased)
	tests := []struct{ raw, expected string }{
		{"EURUSDc", "EURUSDC"},
		{"EURUSDC", "EURUSDC"},
		{"US30", "US30"},
		{"BTCUSDpro", "BTCUSDPRO"},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got := Canonicalize(tt.raw)
			if got != tt.expected {
				t.Errorf("Canonicalize(%q) = %q, want %q", tt.raw, got, tt.expected)
			}
		})
	}
}

func TestCanonicalize_ShortSymbol(t *testing.T) {
	// Very short symbols should not lose suffix stripping
	if got := Canonicalize("XAU"); got != "XAU" {
		t.Errorf("Canonicalize(%q) = %q, want %q", "XAU", got, "XAU")
	}
}

func TestCanonicalize_AlreadyCanonical(t *testing.T) {
	tests := []string{"EURUSD", "BTCUSD", "XAUUSD", "USDJPY", "GBPUSD"}
	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			if got := Canonicalize(raw); got != raw {
				t.Errorf("Canonicalize(%q) = %q, want unchanged %q", raw, got, raw)
			}
		})
	}
}
