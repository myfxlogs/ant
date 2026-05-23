package symbol

import "testing"

func TestCanonicalize(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		// Dotted suffixes
		{"BTCUSDm", "BTCUSD"},
		{"BTCUSD.m", "BTCUSD"},
		{"EURUSD.ecn", "EURUSD"},
		{"EURUSD.raw", "EURUSD"},
		{"EURUSD.pro", "EURUSD"},
		{"EURUSD.stp", "EURUSD"},
		{"EURUSD.x", "EURUSD"},
		{"EURUSD.i", "EURUSD"},

		// Bare suffixes
		{"EURUSDm", "EURUSD"},
		{"EURUSDi", "EURUSD"},

		// Trailing special chars
		{"BTCUSD#", "BTCUSD"},
		{"BTCUSD!", "BTCUSD"},

		// No change
		{"EURUSD", "EURUSD"},
		{"BTCUSD", "BTCUSD"},
		{"XAUUSD", "XAUUSD"},
		{"US30", "US30"},

		// Should NOT strip .c (different contract — case-normalised to upper)
		{"EURUSDc", "EURUSDC"},

		// Don't strip M from digit-prefixed (US30 is not US3 + 0M)
		{"US30", "US30"},
		{"GER40", "GER40"},
	}
	for _, tt := range tests {
		got := Canonicalize(tt.raw)
		if got != tt.want {
			t.Errorf("Canonicalize(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestSeedCanonicals_Count(t *testing.T) {
	seeds := SeedCanonicals()
	if len(seeds) < 45 {
		t.Errorf("SeedCanonicals() returned %d entries, want ≥45 (mainstream coverage)", len(seeds))
	}
	// Verify uniqueness
	seen := make(map[string]bool)
	for _, s := range seeds {
		if seen[s.Canonical] {
			t.Errorf("duplicate canonical %q", s.Canonical)
		}
		seen[s.Canonical] = true
	}
}

func TestSeedCanonicals_Coverage(t *testing.T) {
	seeds := SeedCanonicals()
	// Build a set for fast lookup
	set := make(map[string]bool)
	for _, s := range seeds {
		set[s.Canonical] = true
	}
	mustHave := []string{"EURUSD", "GBPUSD", "USDJPY", "XAUUSD", "BTCUSD", "ETHUSD", "US30", "US500"}
	for _, c := range mustHave {
		if !set[c] {
			t.Errorf("SeedCanonicals missing %q", c)
		}
	}
}
