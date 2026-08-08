package interp

import "testing"

// TestCompatFixes_clrGreen verifies that clrGreen (a clr* prefixed alias)
// is transparently resolved to the canonical Green color value via the L0
// CompatFixes registry. This is the adversarial proof: if the clrGreen entry
// is removed from CompatFixes, LookupMQLConstant will return false, and the
// compiler will treat it as an unknown constant → blind spot. The test
// proves the L0 fix works with zero tokens and zero blind spots.
func TestCompatFixes_clrGreen(t *testing.T) {
	t.Parallel()
	// clrGreen should NOT be in MQLConstants directly (moved to CompatFixes).
	if _, ok := MQLConstants["clrGreen"]; ok {
		t.Fatal("clrGreen should not be in MQLConstants — it belongs in CompatFixes")
	}
	// clrGreen should resolve via CompatFixes → Green → MQLConstants.
	v, ok := LookupMQLConstant("clrGreen")
	if !ok {
		t.Fatal("LookupMQLConstant(clrGreen) failed — CompatFixes L0 lookup not working")
	}
	// Value should match Green's RGB value (32768).
	if v.Kind != ValInt || v.Int != 32768 {
		t.Errorf("clrGreen value = %+v, want ValInt(32768) (same as Green)", v)
	}
	// IsMQLConstant should also return true for clrGreen.
	if !IsMQLConstant("clrGreen") {
		t.Fatal("IsMQLConstant(clrGreen) = false, want true — CompatFixes not checked")
	}
}

// TestCompatFixes_MODE_SENKOU_A verifies that MODE_SENKOU_A (MQL5 naming variant)
// is transparently resolved to MODE_SENKOUA (MQL4 canonical) via CompatFixes.
func TestCompatFixes_MODE_SENKOU_A(t *testing.T) {
	t.Parallel()
	// MODE_SENKOU_A should NOT be in MQLConstants directly.
	if _, ok := MQLConstants["MODE_SENKOU_A"]; ok {
		t.Fatal("MODE_SENKOU_A should not be in MQLConstants — it belongs in CompatFixes")
	}
	// MODE_SENKOU_A should resolve via CompatFixes → MODE_SENKOUA → MQLConstants.
	v, ok := LookupMQLConstant("MODE_SENKOU_A")
	if !ok {
		t.Fatal("LookupMQLConstant(MODE_SENKOU_A) failed — CompatFixes L0 lookup not working")
	}
	// Value should match MODE_SENKOUA's value (3).
	if v.Kind != ValInt || v.Int != 3 {
		t.Errorf("MODE_SENKOU_A value = %+v, want ValInt(3) (same as MODE_SENKOUA)", v)
	}
}

// TestCompatFixes_Adversarial_RemoveEntry proves that removing a CompatFixes
// entry causes the lookup to fail (blind spot would reappear). This is the
// adversarial proof: if the registry entry is removed, the test fails.
func TestCompatFixes_Adversarial_RemoveEntry(t *testing.T) {
	t.Parallel()
	// Save and remove clrGreen from CompatFixes to prove it's needed.
	original := CompatFixes["clrGreen"]
	delete(CompatFixes, "clrGreen")
	defer func() { CompatFixes["clrGreen"] = original }()

	// Without the CompatFixes entry, clrGreen should NOT resolve.
	_, ok := LookupMQLConstant("clrGreen")
	if ok {
		t.Fatal("clrGreen should NOT resolve without CompatFixes entry — adversarial proof failed")
	}
	if IsMQLConstant("clrGreen") {
		t.Fatal("IsMQLConstant(clrGreen) should be false without CompatFixes entry — adversarial proof failed")
	}
}

// TestCompatFixes_NoneMatch verifies that a truly unknown identifier
// does NOT match any compat fix (falls through to blind spot detection).
func TestCompatFixes_NoneMatch(t *testing.T) {
	t.Parallel()
	_, ok := LookupCompatFix("totallyUnknownIdentifier")
	if ok {
		t.Fatal("LookupCompatFix should return false for unknown identifiers")
	}
	_, ok = LookupMQLConstant("totallyUnknownIdentifier")
	if ok {
		t.Fatal("LookupMQLConstant should return false for unknown identifiers")
	}
}
