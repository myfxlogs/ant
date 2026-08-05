package interp

import (
	"sort"
	"testing"
)

// TestAPIRegistryNoDuplicates ensures no duplicate names in unsupportedSymbols.
func TestAPIRegistryNoDuplicates(t *testing.T) {
	seen := make(map[string]bool)
	for _, s := range unsupportedSymbols {
		if seen[s.Name] {
			t.Errorf("duplicate unsupported symbol: %s", s.Name)
		}
		seen[s.Name] = true
	}
}

// TestAPIRegistryImplementedConsistency verifies that every name in the
// implemented* slices in builtin_registry.go appears in the registry
// with StatusImplemented.
func TestAPIRegistryImplementedConsistency(t *testing.T) {
	check := func(slice []string, label string) {
		for _, name := range slice {
			s, ok := registryMap[name]
			if !ok {
				t.Errorf("%s: %q missing from registry", label, name)
				continue
			}
			if s.Status != StatusImplemented {
				t.Errorf("%s: %q has status %d, expected StatusImplemented", label, name, s.Status)
			}
		}
	}

	check(implementedMarketData, "implementedMarketData")
	check(implementedIndicators, "implementedIndicators")
	check(implementedMQL4Trade, "implementedMQL4Trade")
	check(implementedMQL5Position, "implementedMQL5Position")
	check(implementedAccount, "implementedAccount")
	check(implementedPlatform, "implementedPlatform")
	check(implementedCTradeMethods, "implementedCTradeMethods")
}

// TestAPIRegistryConstantsConsistency verifies that every constant in
// MQLConstants appears in the registry with StatusImplemented.
func TestAPIRegistryConstantsConsistency(t *testing.T) {
	for name := range MQLConstants {
		s, ok := registryMap[name]
		if !ok {
			t.Errorf("constant %q missing from registry", name)
			continue
		}
		if s.Status != StatusImplemented {
			t.Errorf("constant %q has status %d, expected StatusImplemented", name, s.Status)
		}
		if s.Category != CatConstant {
			t.Errorf("constant %q has category %d, expected CatConstant", name, s.Category)
		}
	}
}

// TestAPIRegistryNoConflict ensures no name is both in implemented* and
// unsupportedSymbols (would be a logic error).
func TestAPIRegistryNoConflict(t *testing.T) {
	implSet := make(map[string]bool)
	for _, name := range implementedMarketData {
		implSet[name] = true
	}
	for _, name := range implementedIndicators {
		implSet[name] = true
	}
	for _, name := range implementedMQL4Trade {
		implSet[name] = true
	}
	for _, name := range implementedMQL5Position {
		implSet[name] = true
	}
	for _, name := range implementedAccount {
		implSet[name] = true
	}
	for _, name := range implementedPlatform {
		implSet[name] = true
	}
	for _, name := range implementedCTradeMethods {
		implSet[name] = true
	}

	for _, s := range unsupportedSymbols {
		if implSet[s.Name] {
			t.Errorf("symbol %q is both implemented and unsupported — logic conflict", s.Name)
		}
	}
}

// TestAPIRegistryLookup verifies basic lookup behavior.
func TestAPIRegistryLookup(t *testing.T) {
	// Known implemented
	if !IsAPIImplemented("OrderSend") {
		t.Error("OrderSend should be implemented")
	}
	if !IsAPIImplemented("iMA") {
		t.Error("iMA should be implemented")
	}
	if !IsAPIImplemented("MODE_SIGNAL") {
		t.Error("MODE_SIGNAL should be implemented (constant)")
	}

	// Known unsupported
	if !IsAPIUnsupported("iCustom") {
		t.Error("iCustom should be unsupported")
	}
	if !IsAPIUnsupported("FileOpen") {
		t.Error("FileOpen should be unsupported")
	}

	// Unknown symbol
	s, ok := LookupAPI("ThisDoesNotExist")
	if ok {
		t.Errorf("ThisDoesNotExist should not be in registry, got %+v", s)
	}
}

// TestAPIRegistryCompleteness verifies that the registry has a reasonable
// number of entries (sanity check that init() ran correctly).
func TestAPIRegistryCompleteness(t *testing.T) {
	implCount := len(AllImplementedFunctions())
	if implCount < 100 {
		t.Errorf("expected at least 100 implemented functions, got %d", implCount)
	}

	constCount := len(AllRegisteredConstants())
	if constCount < 100 {
		t.Errorf("expected at least 100 constants, got %d", constCount)
	}

	unsupCount := len(AllUnsupportedFunctions())
	if unsupCount < 20 {
		t.Errorf("expected at least 20 unsupported functions, got %d", unsupCount)
	}
}

// TestAPIRegistrySortedOutput ensures deterministic output for debugging.
func TestAPIRegistrySortedOutput(t *testing.T) {
	impl := AllImplementedFunctions()
	sorted := make([]string, len(impl))
	copy(sorted, impl)
	sort.Strings(sorted)

	// Just verify it doesn't panic and produces non-empty result
	if len(sorted) == 0 {
		t.Error("AllImplementedFunctions returned empty list")
	}
}
