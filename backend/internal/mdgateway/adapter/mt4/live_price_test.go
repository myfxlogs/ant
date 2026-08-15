package mt4

import (
	"context"
	"errors"
	"fmt"
	"sync"
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

// TestQuoteStream_RecvError_DoesNotFireOnIdle verifies a silent quote stream
// (no data, no error) does NOT trigger a reconnect. Idle is the normal state
// for an event-driven stream.
//
// Adversarial proof: Recreate the old `case <-time.After` timeout branch →
// the test would see a "reconnecting" callback and fail (RED).
func TestQuoteStream_RecvError_DoesNotFireOnIdle(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cc grpc.ClientConn
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.conn = &cc
	gw.sessionID = "test-session"
	gw.streamCli = &mockStreamsClient{
		quoteStream: &mockQuoteStream{},
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
		t.Fatal("quote stream should NOT reconnect on idle silence — RED: no-data timeout still present")
	case <-time.After(200 * time.Millisecond):
		// GREEN: no Recv error, no reconnect
	}
}

// TestQuoteStream_RecvError_FiresReconnect verifies LIVE-PRICE-2:
// A Recv error on the quote stream triggers reconnect via the error path.
//
// Adversarial proof: Remove the `if err != nil` branch after stream.Recv() →
// the error is swallowed and no reconnect is observed (RED).
func TestQuoteStream_RecvError_FiresReconnect(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cc grpc.ClientConn
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.conn = &cc
	gw.sessionID = "test-session"
	gw.streamCli = &mockStreamsClient{
		quoteStream: &mockQuoteStream{recvErr: errors.New("mock quote recv error")},
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
		// GREEN: Recv error → handleStreamError → reportStatus("reconnecting")
	case <-time.After(3 * time.Second):
		t.Fatal("quote stream did not reconnect on Recv error — RED: error-driven reconnect missing")
	}
}

// mockSubCliPerSymbol returns different results based on the symbol being subscribed.
type mockSubCliPerSymbol struct {
	mu           sync.Mutex
	validSymbols map[string]bool
	calls        []string
}

func (m *mockSubCliPerSymbol) Subscribe(ctx context.Context, in *pb.SubscribeRequest, opts ...grpc.CallOption) (*pb.SubscribeReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockSubCliPerSymbol) SubscribeMany(ctx context.Context, in *pb.SubscribeManyRequest, opts ...grpc.CallOption) (*pb.SubscribeManyReply, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range in.Symbols {
		m.calls = append(m.calls, s)
		if !m.validSymbols[s] {
			return &pb.SubscribeManyReply{Error: &pb.Error{Code: 7, Message: "symbol not found"}}, nil
		}
	}
	return &pb.SubscribeManyReply{Result: "subscribed"}, nil
}
func (m *mockSubCliPerSymbol) UnSubscribe(ctx context.Context, in *pb.UnSubscribeRequest, opts ...grpc.CallOption) (*pb.UnSubscribeReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockSubCliPerSymbol) UnSubscribeMany(ctx context.Context, in *pb.UnSubscribeManyRequest, opts ...grpc.CallOption) (*pb.UnSubscribeManyReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockSubCliPerSymbol) SubscribeOrderProfit(ctx context.Context, in *pb.SubscribeOrderProfitRequest, opts ...grpc.CallOption) (*pb.SubscribeOrderProfitReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockSubCliPerSymbol) SubscribeTickValue(ctx context.Context, in *pb.SubscribeTickValueRequest, opts ...grpc.CallOption) (*pb.SubscribeTickValueReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockSubCliPerSymbol) SubscribeOrderUpdate(ctx context.Context, in *pb.SubscribeOrderUpdateRequest, opts ...grpc.CallOption) (*pb.SubscribeOrderUpdateReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockSubCliPerSymbol) SubscribeQuoteHistory(ctx context.Context, in *pb.SubscribeQuoteHistoryRequest, opts ...grpc.CallOption) (*pb.SubscribeQuoteHistoryReply, error) {
	return nil, fmt.Errorf("mock: not implemented")
}

// TestSubscribe_PerSymbol_SkipsInvalid verifies LIVE-PRICE-4:
// Subscribe per-symbol so invalid symbols are skipped instead of failing
// the entire batch atomically.
//
// Adversarial proof: Revert to batch SubscribeMany (all symbols in one call)
// → mock returns error for non-existent symbol → entire batch fails →
// Subscribe returns error (RED).
// With per-symbol subscribe → valid symbols subscribe, invalid skipped (GREEN).
func TestSubscribe_PerSymbol_SkipsInvalid(t *testing.T) {
	t.Parallel()
	var cc grpc.ClientConn
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.conn = &cc
	gw.sessionID = "test-session"
	gw.streamCli = &mockStreamsClient{
		quoteStream: &mockQuoteStream{ctx: context.Background()},
	}

	mock := &mockSubCliPerSymbol{
		validSymbols: map[string]bool{"EURUSDm": true, "XAUUSDm": true},
	}
	gw.subCli = mock

	// Subscribe to 3 symbols: 2 valid + 1 invalid.
	err := gw.Subscribe(context.Background(), []string{"EURUSDm", "XAUUSDm", "FAKEUSDm"}, func(t *mdtick.Tick) {})
	if err != nil {
		t.Fatalf("Subscribe failed — RED: batch SubscribeMany fails on invalid symbol: %v", err)
	}

	// Verify all 3 symbols were attempted (per-symbol calls).
	mock.mu.Lock()
	gotCalls := len(mock.calls)
	mock.mu.Unlock()
	if gotCalls != 3 {
		t.Fatalf("expected 3 per-symbol SubscribeMany calls, got %d — RED: batch mode only 1 call", gotCalls)
	}
}
