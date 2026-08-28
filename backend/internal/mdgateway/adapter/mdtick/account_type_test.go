package mdtick

import "testing"

// TRUST-1 adversarial proofs for account-type normalization helpers.
// These tests guard S1 (adapter reads broker Type) and S2 (struct fields).

// T1: TestMt4AccountTypeToString guards the MT4 enum→string helper (S1).
// RED mutation: swap any case return value → test fails.
func TestMt4AccountTypeToString(t *testing.T) {
	if got := Mt4AccountTypeToString(0); got != "real" {
		t.Fatalf("MT4 Type=0 must be real, got %q", got)
	}
	if got := Mt4AccountTypeToString(1); got != "contest" {
		t.Fatalf("MT4 Type=1 must be contest, got %q", got)
	}
	if got := Mt4AccountTypeToString(2); got != "demo" {
		t.Fatalf("MT4 Type=2 must be demo, got %q", got)
	}
	if got := Mt4AccountTypeToString(99); got != "unknown" {
		t.Fatalf("MT4 Type=99 must be unknown (fail-closed), got %q", got)
	}
}

// T2: TestNormalizeAccountType guards the MT5 string normalization helper (S1).
// RED mutation: remove a case or change a return → test fails.
func TestNormalizeAccountType(t *testing.T) {
	cases := map[string]string{
		"real":    "real",
		"DEMO":    "demo",
		"Contest": "contest",
		"":        "unknown",
		"foo":     "unknown",
		"  Real ": "real",
	}
	for in, want := range cases {
		if got := NormalizeAccountType(in); got != want {
			t.Fatalf("NormalizeAccountType(%q) = %q, want %q", in, got, want)
		}
	}
}

// T3: TestMTAccountInfo_HasAccountTypeField guards S2 (MTAccountInfo.AccountType field).
// RED mutation: remove the AccountType field → compile error (strongest proof).
func TestMTAccountInfo_HasAccountTypeField(t *testing.T) {
	info := MTAccountInfo{AccountType: "demo"}
	if info.AccountType != "demo" {
		t.Fatal("MTAccountInfo must have AccountType field (S2)")
	}
}

// T3b: TestBrokerInfo_HasAccountTypeField guards S2 (BrokerInfo.AccountType field).
// RED mutation: remove the AccountType field → compile error.
func TestBrokerInfo_HasAccountTypeField(t *testing.T) {
	bi := BrokerInfo{AccountType: "real"}
	if bi.AccountType != "real" {
		t.Fatal("BrokerInfo must have AccountType field (S2)")
	}
}
