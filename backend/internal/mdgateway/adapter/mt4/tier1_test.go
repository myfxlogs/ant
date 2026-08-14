package mt4

import (
	"context"
	"errors"
	"testing"
	"time"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mthub"
	pb "alphaforge/mt4"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// TestMDGATEWAY1_FetchAccountInfo_AppErrorRejected verifies MDGATEWAY-1:
// When mtapi returns gRPC-OK + body Error{code:7}, FetchAccountInfo must
// return an error (fail-closed), NOT degrade to IsInvestor:true.
//
// Adversarial proof: Remove the resp.GetError() check → mock returns
// gRPC nil-error + body Error → old code falls through to result==nil →
// returns IsInvestor:true (RED, misdiagnoses as read-only).
// With check → returns error with code/msg (GREEN).
func TestMDGATEWAY1_FetchAccountInfo_AppErrorRejected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.sessionID = "test-session"
	gw.client = &mockMT4Client{
		accountSummaryRes: &pb.AccountSummaryReply{
			Result: nil,
			Error:  &pb.Error{Code: 7, Message: "session expired"},
		},
	}

	info, err := gw.FetchAccountInfo(context.Background())
	if err == nil {
		t.Fatalf("FetchAccountInfo should return error on app error — RED: error swallowed, got info=%+v", info)
	}
	if info != nil {
		t.Fatalf("FetchAccountInfo should return nil info on error — RED: got info=%+v", info)
	}
}

// TestMDGATEWAY1_FetchAccountInfo_Success verifies normal path still works.
func TestMDGATEWAY1_FetchAccountInfo_Success(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.sessionID = "test-session"
	gw.client = &mockMT4Client{
		accountSummaryRes: &pb.AccountSummaryReply{
			Result: &pb.AccountSummary{
				Balance:    10000.0,
				Equity:     10500.0,
				Leverage:   100,
				Currency:   "USD",
				IsInvestor: false,
			},
		},
	}

	info, err := gw.FetchAccountInfo(context.Background())
	if err != nil {
		t.Fatalf("FetchAccountInfo should succeed — unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("FetchAccountInfo should return non-nil info")
	}
	if !info.Balance.Equal(decimal.NewFromFloat(10000.0)) {
		t.Fatalf("expected balance 10000, got %s", info.Balance)
	}
	if info.IsInvestor {
		t.Fatal("should not be investor")
	}
}

// TestMDGATEWAY3_OrderUpdateRecvError_DoesNotFireOnIdle verifies a silent
// order update stream does NOT reconnect. Idle is the normal state for an
// event-driven stream.
//
// Adversarial proof: Recreate the old `case <-time.After` timeout branch →
// a "reconnecting" callback would fire and the test would fail (RED).
func TestMDGATEWAY3_OrderUpdateRecvError_DoesNotFireOnIdle(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cc grpc.ClientConn
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.conn = &cc
	gw.sessionID = "test-session"
	gw.streamCli = &mockStreamsClient{
		orderUpdateStream: &mockOrderUpdateStream{},
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

	go gw.orderUpdateRecvLoop(ctx, func(u *mdtick.OrderUpdate) {})

	select {
	case <-reconnected:
		t.Fatal("order update stream should NOT reconnect on idle silence — RED: no-data timeout still present")
	case <-time.After(200 * time.Millisecond):
		// GREEN: no Recv error, no reconnect
	}
}

// TestMDGATEWAY3_OrderUpdateRecvError_FiresReconnect verifies MDGATEWAY-3:
// A Recv error on the order update stream triggers reconnect.
//
// Adversarial proof: Delete the `if err != nil` branch after stream.Recv() →
// the error is swallowed and no reconnect is observed (RED).
func TestMDGATEWAY3_OrderUpdateRecvError_FiresReconnect(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cc grpc.ClientConn
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.conn = &cc
	gw.sessionID = "test-session"
	gw.streamCli = &mockStreamsClient{
		orderUpdateStream: &mockOrderUpdateStream{recvErr: errors.New("mock order update recv error")},
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

	go gw.orderUpdateRecvLoop(ctx, func(u *mdtick.OrderUpdate) {})

	select {
	case <-reconnected:
		// GREEN: Recv error → handleStreamError → reportStatus("reconnecting")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout: order update stream did not reconnect on Recv error — RED: error-driven reconnect missing")
	}
}

// TestMDGATEWAY2_HealthCheck_AppErrorRejected verifies MDGATEWAY-2:
// When mtapi Ping returns gRPC-OK + body Error{code:7}, HealthCheck must
// return an error (fail-closed), NOT silently return nil (healthy).
//
// Adversarial proof: Remove the resp.GetError() check → mock returns
// gRPC nil-error + body Error → old code returns nil (RED, dead session judged healthy).
// With check → returns error with code/msg (GREEN).
func TestMDGATEWAY2_HealthCheck_AppErrorRejected(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.sessionID = "test-session"
	gw.serviceCli = &mockServiceClient{
		pingRes: &pb.PingReply{
			Result: "",
			Error:  &pb.Error{Code: 7, Message: "session expired"},
		},
	}

	err := gw.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("HealthCheck should return error on app error — RED: error swallowed, dead session judged healthy")
	}
}

// TestMDGATEWAY2_HealthCheck_Success verifies normal path still works.
func TestMDGATEWAY2_HealthCheck_Success(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.sessionID = "test-session"
	gw.serviceCli = &mockServiceClient{
		pingRes: &pb.PingReply{Result: "ok"},
	}

	err := gw.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck should succeed — unexpected error: %v", err)
	}
}

// TestMDGATEWAY4_SubscribeOrderEvents_ReconnectOnRecvError verifies MDGATEWAY-4:
// SubscribeOrderEvents goroutine reconnects when the order event stream Recv
// returns an error (Recv error → handleStreamError → resubscribe).
//
// Adversarial proof: Remove the outer reconnect loop (or the `if err != nil`
// branch after Recv) → goroutine exits on first error → no reconnect observed (RED).
func TestMDGATEWAY4_SubscribeOrderEvents_ReconnectOnRecvError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cc grpc.ClientConn
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.conn = &cc
	gw.sessionID = "test-session"
	gw.streamCli = &mockStreamsClient{
		orderUpdateStream: &mockOrderUpdateStream{recvErr: errors.New("mock order event recv error")},
	}

	reconnectCount := make(chan int, 10)
	count := 0
	gw.SetStatusCallback(func(status, message string) {
		if status == "reconnecting" {
			count++
			select {
			case reconnectCount <- count:
			default:
			}
		}
	})

	err := gw.SubscribeOrderEvents(ctx, func(e *mthub.OrderEvent) {})
	if err != nil {
		t.Fatalf("SubscribeOrderEvents failed: %v", err)
	}

	select {
	case c := <-reconnectCount:
		if c < 1 {
			t.Fatalf("expected at least 1 reconnect, got %d", c)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout: SubscribeOrderEvents did not reconnect on Recv error — RED: goroutine exits on error, no outer reconnect loop")
	}
}
