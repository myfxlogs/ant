package mt4

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"alphaforge/internal/mdgateway/adapter/mdtick"
)

// TestDEMAND_SUB_NilSymbolsNoSubscribeMany verifies that Subscribe with nil
// symbols does NOT call SubscribeMany (only starts recvLoop).
//
// Adversarial proof: Revert runner_gateway.go to pass FetchAllSymbols result
// (462 symbols) → Subscribe is called with 462 symbols → SubscribeMany is
// called 462 times (RED). With nil → 0 calls (GREEN).
func TestDEMAND_SUB_NilSymbolsNoSubscribeMany(t *testing.T) {
	t.Parallel()

	var cc grpc.ClientConn
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.conn = &cc
	gw.sessionID = "test-session"
	gw.streamCli = &mockStreamsClient{
		quoteStream: &mockQuoteStream{ctx: context.Background()},
	}

	mock := &mockSubCliPerSymbol{
		validSymbols: map[string]bool{"EURUSDm": true},
	}
	gw.subCli = mock

	// Subscribe with nil symbols — demand-driven: only recvLoop, no SubscribeMany.
	err := gw.Subscribe(context.Background(), nil, func(t *mdtick.Tick) {})
	if err != nil {
		t.Fatalf("Subscribe with nil symbols failed: %v", err)
	}

	if len(mock.calls) != 0 {
		t.Fatalf("DEMAND-SUB: SubscribeMany called %d times with nil symbols — "+
			"RED: should be 0 (demand-driven, no pre-subscription)", len(mock.calls))
	}
}

// TestDEMAND_SUB_NonNilSymbolsCallsSubscribeMany verifies that Subscribe with
// non-nil symbols DOES call SubscribeMany (the on-demand path via AddSymbols
// or initial subscription).
func TestDEMAND_SUB_NonNilSymbolsCallsSubscribeMany(t *testing.T) {
	t.Parallel()

	var cc grpc.ClientConn
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.conn = &cc
	gw.sessionID = "test-session"
	gw.streamCli = &mockStreamsClient{
		quoteStream: &mockQuoteStream{ctx: context.Background()},
	}

	mock := &mockSubCliPerSymbol{
		validSymbols: map[string]bool{"EURUSDm": true},
	}
	gw.subCli = mock

	err := gw.Subscribe(context.Background(), []string{"EURUSDm"}, func(t *mdtick.Tick) {})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 SubscribeMany call, got %d", len(mock.calls))
	}
}
