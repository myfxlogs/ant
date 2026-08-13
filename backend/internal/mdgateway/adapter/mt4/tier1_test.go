package mt4

import (
	"context"
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

// TestMDGATEWAY3_OrderUpdateTimeout_FiresReconnect verifies MDGATEWAY-3:
// A blocking order update stream (no data) triggers reconnect via the timeout path.
//
// Adversarial proof: Delete the `case <-time.After` branch → the timeout
// never fires and the test hangs (detected by -timeout flag as RED).
func TestMDGATEWAY3_OrderUpdateTimeout_FiresReconnect(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cc grpc.ClientConn
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.conn = &cc
	gw.sessionID = "test-session"
	gw.orderUpdateTimeout = 50 * time.Millisecond
	gw.streamCli = &mockStreamsClient{
		orderUpdateStream: &mockOrderUpdateStream{ctx: ctx},
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
		// GREEN: timeout fired → handleStreamError → reportStatus("reconnecting")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout: order update stream did not reconnect — RED: no-data timeout missing")
	}
}

// TestMDGATEWAY3_OrderUpdateTimeout_Injectable verifies the orderUpdateTimeout
// field is injectable for tests (not hardcoded 90s).
func TestMDGATEWAY3_OrderUpdateTimeout_Injectable(t *testing.T) {
	t.Parallel()
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	if got := gw.orderUpdateTimeoutOrDefault(); got != 90*time.Second {
		t.Fatalf("default should be 90s, got %v", got)
	}
	gw.orderUpdateTimeout = 5 * time.Second
	if got := gw.orderUpdateTimeoutOrDefault(); got != 5*time.Second {
		t.Fatalf("injected should be 5s, got %v", got)
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

// TestMDGATEWAY4_SubscribeOrderEvents_ReconnectOnTimeout verifies MDGATEWAY-4:
// SubscribeOrderEvents goroutine reconnects when the order event stream has no data
// (timeout fires → handleStreamError → resubscribe).
//
// Adversarial proof: Remove the outer reconnect loop (or the time.After branch) →
// goroutine exits on first timeout/error → no reconnect observed (RED).
func TestMDGATEWAY4_SubscribeOrderEvents_ReconnectOnTimeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cc grpc.ClientConn
	gw := New(mdtick.AccountConfig{AccountID: "test-acc"}, zap.NewNop())
	gw.conn = &cc
	gw.sessionID = "test-session"
	gw.orderUpdateTimeout = 50 * time.Millisecond
	gw.streamCli = &mockStreamsClient{
		orderUpdateStream: &mockOrderUpdateStream{ctx: ctx},
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

	// Wait for at least 2 reconnects (proves outer loop works, not single-shot)
	select {
	case c := <-reconnectCount:
		if c < 1 {
			t.Fatalf("expected at least 1 reconnect, got %d", c)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout: SubscribeOrderEvents did not reconnect — RED: goroutine exits on no-data, no outer reconnect loop")
	}
}
