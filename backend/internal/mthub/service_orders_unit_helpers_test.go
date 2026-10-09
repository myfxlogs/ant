package mthub

import (
	"context"

	"github.com/shopspring/decimal"

	"alphaforge/internal/risk"
)

// mockKillSwitch implements KillSwitchGate for testing.
type mockKillSwitch struct{ engaged bool }

func (m *mockKillSwitch) IsEngaged() bool { return m.engaged }

func mustDec(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

// newTestService creates a minimally wired MtHubService for unit testing.
// Includes a permissive gate + state provider so PlaceOrder/CloseOrder
// pass the gate check (DEPLOY-LIVE-4 fail-closed). Tests that need to
// test nil-gate behavior should use newTestServiceNoGate().

// newTestService creates a minimally wired MtHubService for unit testing.
// Includes a permissive gate + state provider so PlaceOrder/CloseOrder
// pass the gate check (DEPLOY-LIVE-4 fail-closed). Tests that need to
// test nil-gate behavior should use newTestServiceNoGate().
func newTestService() *MtHubService {
	hub := NewHub()
	svc := NewMtHubService(hub, NewOrderEventBroker(), NewAccountProfitBroker(), NewPositionSnapshotBroker(), nil, nil, nil)
	svc.SetGate(risk.NewDefaultGate())
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{Balance: dec(100000), Equity: dec(100000), FreeMargin: dec(95000)}, nil
	})
	return svc
}

// newTestServiceNoGate creates a service without gate/state provider for
// testing fail-closed behavior (DEPLOY-LIVE-4).

// newTestServiceNoGate creates a service without gate/state provider for
// testing fail-closed behavior (DEPLOY-LIVE-4).
func newTestServiceNoGate() *MtHubService {
	hub := NewHub()
	return NewMtHubService(hub, NewOrderEventBroker(), NewAccountProfitBroker(), NewPositionSnapshotBroker(), nil, nil, nil)
}

// --- Service order tests ---
