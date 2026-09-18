// vm_live_account_fields_test.go — LIVE-ACCOUNT-FIELDS-1 (2026-09-18).
//
// Harness-mode VM-executed assertions: SetAccountIdentity feeds
// Leverage/Currency/Company/Mode through brokerImpl.Account() so the MQL
// builtins (AccountLeverage/AccountCurrency/AccountCompany/AccountInfoInteger)
// read real values instead of zeros / fail-closed errors. Assertions run
// through VM execution (compiled MQL → runner.OnBar → GetGlobal), not Go
// field reads.
//
// Adversarial proofs (M1/M2): delete broker.go's four Account() fields → T1
// RED (leverage 0 / errors); restore sdkAccountMode to a direct cast → T2
// RED (bogus mode accepted).
package mql2go

import (
	"context"
	"strings"
	"testing"

	"alphaforge/strategy/runner"
)

const accountFieldsSource = `
int g_lev = -1;
int g_hedge = -1;
int g_mm = -1;
string g_cur = "";
string g_co = "";

int OnInit() { return 0; }

void OnBar()
{
    g_lev = AccountLeverage();
    g_cur = AccountCurrency();
    g_co = AccountCompany();
    g_mm = AccountInfoInteger(ACCOUNT_MARGIN_MODE);
    g_hedge = AccountInfoInteger(ACCOUNT_HEDGE_ALLOWED);
}
`

// newAccountFieldsRunner compiles the probe source and wires it into a
// harness-mode Runner with the given identity.
func newAccountFieldsRunner(t *testing.T, leverage int32, currency, company, accountMode string) (*runner.Runner, *VMRunner) {
	t.Helper()
	vmRunner, err := CompileMQL(accountFieldsSource)
	if err != nil {
		t.Fatalf("CompileMQL failed: %v", err)
	}
	r := runner.New(runner.Config{})
	r.SetStrategy(vmRunner)
	r.UpdateLiveState("10000", "10500", "0", "9500", nil, nil)
	r.SetAccountIdentity(leverage, currency, company, accountMode)
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	return r, vmRunner
}

// TestHarness_AccountIdentityViaVM — T1: harness-mode identity reaches the VM
// through brokerImpl.Account(); all five builtins read real values.
//
// Adversarial (M1): remove the Leverage/Currency/Company/Mode lines from
// broker.go Account() → leverage 0 + currency/company/margin-mode errors → RED.
func TestHarness_AccountIdentityViaVM(t *testing.T) {
	r, vmRunner := newAccountFieldsRunner(t, 200, "USD", "BrokerCo", "hedging")
	if _, err := r.OnBar(context.Background(), nil, "M15"); err != nil {
		t.Fatalf("OnBar failed: %v", err)
	}

	for name, want := range map[string]int32{
		"g_lev":   200,
		"g_hedge": 1,   // hedging → ACCOUNT_HEDGE_ALLOWED = 1
		"g_mm":    2,   // ACCOUNT_MARGIN_MODE_RETAIL_HEDGING = 2
	} {
		v, ok := vmRunner.GetGlobal(name)
		if !ok {
			t.Fatalf("global %s not found", name)
		}
		if got := v.ToInt(); got != want {
			t.Fatalf("%s = %d, want %d (VM must read the injected identity)", name, got, want)
		}
	}
	if v, ok := vmRunner.GetGlobal("g_cur"); !ok || v.ToString() != "USD" {
		t.Fatalf("g_cur = %v, want USD", v)
	}
	if v, ok := vmRunner.GetGlobal("g_co"); !ok || v.ToString() != "BrokerCo" {
		t.Fatalf("g_co = %v, want BrokerCo", v)
	}
}

// TestHarness_BogusAccountModeFailsClosed — T2: a non-whitelisted account
// mode string maps to "" and the VM margin-mode prop fails closed (no
// misclassification into hedging/netting).
//
// Adversarial (M2): restore sdkAccountMode to a direct cast → "bogus"
// becomes a valid sdk.AccountMode → margin mode resolves via default → RED.
func TestHarness_BogusAccountModeFailsClosed(t *testing.T) {
	r, _ := newAccountFieldsRunner(t, 200, "USD", "BrokerCo", "bogus")
	_, err := r.OnBar(context.Background(), nil, "M15")
	if err == nil {
		t.Fatal("OnBar err = nil, want error (bogus account mode must fail closed on margin-mode prop)")
	}
	if !strings.Contains(err.Error(), "VM fatal") {
		t.Fatalf("err = %v, want it to contain 'VM fatal'", err)
	}
}
