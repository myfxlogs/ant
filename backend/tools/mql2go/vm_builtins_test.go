package mql2go

import (
	"testing"

	"alphaforge/tools/mql2go/interp"
)

// TestAllBuiltinsWired verifies that every entry in builtinRegistry has a non-nil fn,
// except for symbols registered as StatusUnsupported in the API registry.
func TestAllBuiltinsWired(t *testing.T) {
	var unwired []string
	for _, entry := range builtinRegistry {
		if entry.fn != nil {
			continue
		}
		if sym, ok := interp.LookupAPI(entry.name); ok && sym.Status == interp.StatusUnsupported {
			continue
		}
		unwired = append(unwired, entry.name)
	}
	if len(unwired) > 0 {
		t.Errorf("%d builtins have nil fn (not wired, not unsupported): %v", len(unwired), unwired)
	}
}

// TestNoDuplicateBuiltins verifies no duplicate names in builtinRegistry.
func TestNoDuplicateBuiltins(t *testing.T) {
	seen := map[string]int{}
	for _, entry := range builtinRegistry {
		seen[entry.name]++
	}
	var dups []string
	for name, count := range seen {
		if count > 1 {
			dups = append(dups, name)
		}
	}
	if len(dups) > 0 {
		t.Errorf("duplicate builtin names: %v", dups)
	}
}

// TestBuiltinCount verifies the total number of registered builtins.
func TestBuiltinCount(t *testing.T) {
	total := len(builtinRegistry)
	wired := 0
	for _, entry := range builtinRegistry {
		if entry.fn != nil {
			wired++
		}
	}
	t.Logf("builtinRegistry: %d total, %d wired, %d blind spots", total, wired, total-wired)
	if total < 300 {
		t.Errorf("expected at least 300 builtins, got %d", total)
	}
}

// TestImplementedNamesHaveVMHandlers verifies that every name in the
// implemented* slices (builtin_registry.go) has a corresponding entry
// in builtinRegistry with a non-nil fn. This catches false "implemented"
// status where the registry claims a function is implemented but no VM
// handler exists.
func TestImplementedNamesHaveVMHandlers(t *testing.T) {
	builtinMap := make(map[string]bool, len(builtinRegistry))
	for _, entry := range builtinRegistry {
		builtinMap[entry.name] = entry.fn != nil
	}

	check := func(names []string, label string) {
		for _, name := range names {
			// Skip names that are MQL constants (M1, H1, etc.) — these are in
			// MQLConstants, not builtinRegistry.
			if _, isConst := interp.LookupMQLConstant(name); isConst {
				continue
			}
			wired, ok := builtinMap[name]
			if !ok {
				t.Errorf("%s: %q not in builtinRegistry", label, name)
				continue
			}
			if !wired {
				t.Errorf("%s: %q has nil fn (not wired)", label, name)
			}
		}
	}

	check(interp.ImplementedMarketData(), "implementedMarketData")
	check(interp.ImplementedIndicators(), "implementedIndicators")
	check(interp.ImplementedMQL4Trade(), "implementedMQL4Trade")
	check(interp.ImplementedMQL5Position(), "implementedMQL5Position")
	check(interp.ImplementedAccount(), "implementedAccount")
	check(interp.ImplementedPlatform(), "implementedPlatform")

	// CTrade methods are registered with "CTrade." prefix in builtinRegistry.
	for _, name := range interp.ImplementedCTradeMethods() {
		fullName := "CTrade." + name
		wired, ok := builtinMap[fullName]
		if !ok {
			t.Errorf("implementedCTradeMethods: %q not in builtinRegistry", fullName)
			continue
		}
		if !wired {
			t.Errorf("implementedCTradeMethods: %q has nil fn (not wired)", fullName)
		}
	}
}
