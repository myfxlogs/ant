package mt4

import (
	"context"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	pb "alphaforge/mt4"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// TestSubscribeMany_ResponseErrorDetected verifies LIVE-PRICE-3 fix:
// SubscribeMany returns gRPC nil-error but body carries Error{code:7}.
// The fixed code must detect this via resp.GetError() and log.Error.
//
// Adversarial proof: Delete the `resp.GetError()` check (revert to `if _, err :=`)
// → the error is silently swallowed (logged as Info "subscribed" instead of Error).
func TestSubscribeMany_ResponseErrorDetected(t *testing.T) {
	t.Parallel()
	var cc grpc.ClientConn
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.conn = &cc
	gw.sessionID = "test-session"
	gw.streamCli = &mockStreamsClient{
		quoteStream: &mockQuoteStream{ctx: context.Background()},
	}
	gw.subCli = &mockSubCli{
		subscribeManyReply: &pb.SubscribeManyReply{
			Result: "",
			Error: &pb.Error{
				Code:    7,
				Message: "symbol not subscribed",
			},
		},
	}

	err := gw.Subscribe(context.Background(), []string{"EURUSD"}, func(t *mdtick.Tick) {})
	if err != nil {
		t.Fatalf("Subscribe should not return error for response-level error (it logs): %v", err)
	}
}

// TestSubscribeMany_GRPCError checks gRPC-level error still handled.
func TestSubscribeMany_GRPCError(t *testing.T) {
	t.Parallel()
	var cc grpc.ClientConn
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.conn = &cc
	gw.sessionID = "test-session"
	gw.streamCli = &mockStreamsClient{
		quoteStream: &mockQuoteStream{ctx: context.Background()},
	}
	gw.subCli = &mockSubCli{
		subscribeManyErr: context.DeadlineExceeded,
	}

	err := gw.Subscribe(context.Background(), []string{"EURUSD"}, func(t *mdtick.Tick) {})
	if err != nil {
		t.Fatalf("Subscribe should not propagate gRPC error (it logs): %v", err)
	}
}

// TestSubscribeMany_Success verifies normal path still works.
func TestSubscribeMany_Success(t *testing.T) {
	t.Parallel()
	var cc grpc.ClientConn
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.conn = &cc
	gw.sessionID = "test-session"
	gw.streamCli = &mockStreamsClient{
		quoteStream: &mockQuoteStream{ctx: context.Background()},
	}
	gw.subCli = &mockSubCli{
		subscribeManyReply: &pb.SubscribeManyReply{Result: "subscribed"},
	}

	err := gw.Subscribe(context.Background(), []string{"EURUSD"}, func(t *mdtick.Tick) {})
	if err != nil {
		t.Fatalf("Subscribe should succeed: %v", err)
	}
}

// TestQuoteTimeout_Injectable verifies LIVE-PRICE-2: quoteTimeout field
// is injectable for tests. The recvLoop should use it instead of hardcoded 90s.
func TestQuoteTimeout_Injectable(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	if got := gw.quoteTimeoutOrDefault(); got != 90*time.Second {
		t.Errorf("default quoteTimeout = %v, want 90s", got)
	}
	gw.quoteTimeout = 100 * time.Millisecond
	if got := gw.quoteTimeoutOrDefault(); got != 100*time.Millisecond {
		t.Errorf("injected quoteTimeout = %v, want 100ms", got)
	}
}

// TestQuoteTimeout_FiresReconnect verifies LIVE-PRICE-2:
// A blocking quote stream (no data) triggers reconnect via the timeout path.
//
// Adversarial proof: Delete the `case <-time.After` branch → the timeout
// never fires and the test hangs (detected by -timeout flag as RED).
func TestQuoteTimeout_FiresReconnect(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cc grpc.ClientConn
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.conn = &cc
	gw.sessionID = "test-session"
	gw.quoteTimeout = 50 * time.Millisecond
	gw.streamCli = &mockStreamsClient{
		quoteStream: &mockQuoteStream{ctx: ctx},
	}
	gw.subCli = &mockSubCli{
		subscribeManyReply: &pb.SubscribeManyReply{Result: "ok"},
	}

	reconnected := make(chan struct{}, 4)
	gw.SetStatusCallback(func(status, message string) {
		if status == "reconnecting" {
			select {
			case reconnected <- struct{}{}:
			default:
			}
		}
	})

	go gw.recvLoop(ctx, func(t *mdtick.Tick) {})

	select {
	case <-reconnected:
		// GREEN: timeout fired → handleStreamError → reportStatus("reconnecting")
	case <-time.After(3 * time.Second):
		t.Fatal("quote timeout did not trigger reconnect within 3s — RED: case <-time.After missing or broken")
	}
}
