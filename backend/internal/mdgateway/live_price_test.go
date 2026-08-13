package mdgateway

import (
	"testing"

	"alphaforge/internal/mdgateway/adapter/mdtick"

	"github.com/shopspring/decimal"
)

// TestOnTickCallbackFired verifies LIVE-PRICE-1:
// When a tick flows through HandleTick, the onTick callback is invoked.
//
// Adversarial proof: Delete the `if m.onTick != nil { m.onTick(t) }` lines
// in manager_tick.go → tickReceived never gets set → test fails (RED).
func TestOnTickCallbackFired(t *testing.T) {
	t.Parallel()

	tickReceived := make(chan *mdtick.Tick, 1)
	mgr := testManager()
	mgr.onTick = func(t *mdtick.Tick) {
		select {
		case tickReceived <- t:
		default:
		}
	}

	tk := &mdtick.Tick{
		UserID:    "u1",
		AccountID: "a1",
		Broker:    "test",
		Platform:  "mt4",
		SymbolRaw: "EURUSD",
		Canonical: "EURUSD",
		TsUnixMs:  1000,
		Bid:       decimal.NewFromFloat(1.10),
		Ask:       decimal.NewFromFloat(1.11),
		BidVolume: 1000,
		AskVolume: 1000,
	}
	mgr.HandleTick(tk)

	select {
	case got := <-tickReceived:
		if got.SymbolRaw != "EURUSD" {
			t.Errorf("onTick received SymbolRaw = %q, want EURUSD", got.SymbolRaw)
		}
	default:
		t.Fatal("onTick callback was not fired — RED: m.onTick(t) call missing in manager_tick.go")
	}
}

// TestOnTickNilSafe verifies onTick is nil-safe (no panic when not set).
func TestOnTickNilSafe(t *testing.T) {
	t.Parallel()
	mgr := testManager()
	// onTick is nil by default — must not panic
	tk := &mdtick.Tick{
		AccountID: "a1",
		Broker:    "test",
		SymbolRaw: "EURUSD",
		TsUnixMs:  1000,
	}
	mgr.HandleTick(tk)
}
