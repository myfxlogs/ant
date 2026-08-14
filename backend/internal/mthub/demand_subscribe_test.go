package mthub

import (
	"context"
	"testing"
	"time"
)

// TestDEMAND_SUB_SubscribeSymbols_CallsAddSymbols verifies that SubscribeSymbols
// calls AddSymbols on the executor with the correct symbols.
//
// Adversarial proof: Delete the SubscribeSymbols call in live_runner.go
// → AddSymbols is never called → gateway never subscribes to the strategy's
// symbol → OnQuote never delivers bars (RED). With the call → AddSymbols
// invoked with the strategy's symbol (GREEN).
func TestDEMAND_SUB_SubscribeSymbols_CallsAddSymbols(t *testing.T) {
	t.Parallel()

	svc := newTestService()

	var addedSymbols []string
	exec := &mockExecutor{
		platform: "MT5",
		addSymbolsFn: func(_ context.Context, symbols []string) error {
			addedSymbols = symbols
			return nil
		},
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)

	err := svc.SubscribeSymbols(context.Background(), "acc-1", []string{"BTCUSDm"})
	if err != nil {
		t.Fatalf("SubscribeSymbols failed: %v", err)
	}

	if len(addedSymbols) != 1 || addedSymbols[0] != "BTCUSDm" {
		t.Fatalf("DEMAND-SUB: AddSymbols called with %v, want [BTCUSDm] — "+
			"RED: SubscribeSymbols not forwarding to AddSymbols or wrong symbol", addedSymbols)
	}
}
