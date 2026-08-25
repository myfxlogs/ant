//go:build integration

package system

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
)

func TestMtHub_OrderHistoryWithTimeRange(t *testing.T) {
	harness := newMtHubTestHarness(t)

	ctx, cancel := harness.ctxWithTimeout()
	defer cancel()

	now := time.Now()
	req := connect.NewRequest(&antv1.OrderHistoryRequest{
		AccountId: harness.accountID,
		From:      timestamppb.New(now.Add(-7 * 24 * time.Hour)),
		To:        timestamppb.New(now),
	})
	resp, err := harness.server.OrderHistory(ctx, req)
	if err != nil {
		t.Fatalf("OrderHistory: %v", err)
	}

	if resp.Msg.Orders == nil {
		t.Error("expected non-nil Orders slice, got nil")
	}
	t.Logf("OrderHistory: got %d orders (may be empty with mock executor) PASS", len(resp.Msg.Orders))
}

// ===========================================================================
// Test 5: SSE stream events (SubscribeEvents)
// ===========================================================================

func TestMtHub_SubscribeEventsReceivesAccountStatus(t *testing.T) {
	harness := newMtHubTestHarness(t)

	ctx, cancel := context.WithCancel(harness.ctx())
	defer cancel()

	eventCh := make(chan *antv1.StreamEvent, 64)
	stream := newTestServerStream(eventCh, ctx)

	req := connect.NewRequest(&antv1.SubscribeEventsRequest{
		AccountIds: []string{harness.accountID},
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- harness.streamSrv.SubscribeEvents(ctx, req, stream)
	}()

	// SubscribeEvents sends initial snapshots (profit_update + account_status)
	// via sendInitialSnapshot which reads GetUserAccountSnapshots from DB.
	var gotStatus bool
	timeout := time.After(5 * time.Second)

	for !gotStatus {
		select {
		case ev, ok := <-eventCh:
			if !ok {
				t.Fatal("event channel closed unexpectedly")
			}
			t.Logf("received SSE event: type=%s account=%s", ev.GetType(), ev.AccountId)
			if ev.GetType() == "account_status" {
				gotStatus = true
				t.Logf("received account_status event for account %s PASS", ev.AccountId)
			}
		case err := <-errCh:
			t.Fatalf("SubscribeEvents returned early: %v", err)
		case <-timeout:
			t.Fatal("timed out waiting for account_status event")
		}
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Logf("SubscribeEvents returned after cancel: %v (expected)", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SubscribeEvents to finish after cancel")
	}
}

// ===========================================================================
// Test 5b: SSE stream connection established (verify multiple event types)
// ===========================================================================

func TestMtHub_SubscribeEventsConnectionEstablished(t *testing.T) {
	harness := newMtHubTestHarness(t)

	ctx, cancel := context.WithCancel(harness.ctx())
	defer cancel()

	eventCh := make(chan *antv1.StreamEvent, 64)
	stream := newTestServerStream(eventCh, ctx)

	req := connect.NewRequest(&antv1.SubscribeEventsRequest{
		AccountIds: []string{harness.accountID},
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- harness.streamSrv.SubscribeEvents(ctx, req, stream)
	}()

	receivedTypes := make(map[string]bool)
	timeout := time.After(5 * time.Second)

collectLoop:
	for len(receivedTypes) < 2 {
		select {
		case ev, ok := <-eventCh:
			if !ok {
				t.Fatal("event channel closed unexpectedly")
			}
			receivedTypes[ev.GetType()] = true
			t.Logf("received SSE event: type=%s account=%s", ev.GetType(), ev.AccountId)
		case err := <-errCh:
			t.Fatalf("SubscribeEvents returned early: %v", err)
		case <-timeout:
			t.Logf("collected events before timeout: %v", receivedTypes)
			break collectLoop
		}
	}

	if receivedTypes["account_status"] {
		t.Log("connection established: account_status received PASS")
	} else {
		t.Error("connection was NOT properly established: missing account_status event")
	}
	if receivedTypes["profit_update"] {
		t.Log("profit_update received PASS")
	}

	cancel()
	<-errCh
}

// ===========================================================================
// Helper: construct a real connect.ServerStream with a mock StreamingHandlerConn
// ===========================================================================

// mockStreamConn implements connect.StreamingHandlerConn and captures Send calls.
type mockStreamConn struct {
	ch  chan *antv1.StreamEvent
	ctx context.Context
}

func (m *mockStreamConn) Spec() connect.Spec           { return connect.Spec{} }
func (m *mockStreamConn) Peer() connect.Peer           { return connect.Peer{} }
func (m *mockStreamConn) Receive(any) error            { return nil }
func (m *mockStreamConn) RequestHeader() http.Header   { return http.Header{} }
func (m *mockStreamConn) ResponseHeader() http.Header  { return http.Header{} }
func (m *mockStreamConn) ResponseTrailer() http.Header { return http.Header{} }

func (m *mockStreamConn) Send(msg any) error {
	ev, ok := msg.(*antv1.StreamEvent)
	if !ok {
		return fmt.Errorf("unexpected message type: %T", msg)
	}
	select {
	case m.ch <- ev:
		return nil
	case <-m.ctx.Done():
		return m.ctx.Err()
	}
}

// ===========================================================================
// Test 9: SymbolParams — with session and without session
// ===========================================================================

func TestMtHub_SymbolParams(t *testing.T) {
	harness := newMtHubTestHarness(t)
	ctx, cancel := harness.ctxWithTimeout()
	defer cancel()

	// SymbolParams with valid account + session should succeed (mock returns nil).
	req := connect.NewRequest(&antv1.SymbolParamsRequest{
		AccountId:  harness.accountID,
		Canonicals: []string{"EURUSD", "GBPUSD"},
	})
	resp, err := harness.server.SymbolParams(ctx, req)
	if err != nil {
		t.Fatalf("SymbolParams with valid session: %v", err)
	}
	if resp.Msg == nil {
		t.Fatal("expected non-nil response")
	}
	t.Logf("SymbolParams returned %d params (mock returns empty)", len(resp.Msg.Params))
}

func TestMtHub_SymbolParamsNoSession(t *testing.T) {
	harness := newMtHubTestHarness(t)
	ctx, cancel := harness.ctxWithTimeout()
	defer cancel()

	// SymbolParams for an account that has no session should fail.
	req := connect.NewRequest(&antv1.SymbolParamsRequest{
		AccountId:  uuid.New().String(),
		Canonicals: []string{"EURUSD"},
	})
	_, err := harness.server.SymbolParams(ctx, req)
	if err == nil {
		t.Fatal("expected error for account with no session")
	}
	var ce *connect.Error
	if !errors.As(err, &ce) {
		t.Fatalf("expected connect.Error, got %T: %v", err, err)
	}
	if ce.Code() != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v", ce.Code())
	}
	t.Logf("SymbolParams without session correctly returned %v", ce.Code())
}

// ===========================================================================
// Test 10: Error recovery — PlaceOrder with unknown account
// ===========================================================================

func TestMtHub_PlaceOrderUnknownAccount(t *testing.T) {
	harness := newMtHubTestHarness(t)
	ctx, cancel := harness.ctxWithTimeout()
	defer cancel()

	req := connect.NewRequest(&antv1.PlaceOrderRequest{
		AccountId: uuid.New().String(),
		Canonical: "EURUSD",
		Side:      antv1.Side_SIDE_BUY,
		OrderType: antv1.OrderType_ORDER_TYPE_MARKET,
		Volume:    "0.1",
		Price:     "1.08500",
	})
	_, err := harness.server.PlaceOrder(ctx, req)
	if err == nil {
		t.Fatal("expected error for unknown account")
	}
	var ce *connect.Error
	if !errors.As(err, &ce) {
		t.Fatalf("expected connect.Error, got %T", err)
	}
	if ce.Code() != connect.CodeNotFound {
		t.Errorf("expected NotFound, got %v", ce.Code())
	}
	t.Logf("PlaceOrder with unknown account correctly returned %v", ce.Code())
}

// newTestServerStream creates a *connect.ServerStream backed by a mock connection.
// Uses reflect+unsafe to set the unexported conn field (standard test-only pattern).
func newTestServerStream(eventCh chan *antv1.StreamEvent, parentCtx context.Context) *connect.ServerStream[antv1.StreamEvent] {
	conn := &mockStreamConn{ch: eventCh, ctx: parentCtx}
	stream := &connect.ServerStream[antv1.StreamEvent]{}

	// connect.ServerStream has a single unexported field: conn StreamingHandlerConn
	// Use unsafe to set it — standard test-only pattern for this library.
	v := reflect.ValueOf(stream).Elem()
	field := v.FieldByName("conn")
	field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
	field.Set(reflect.ValueOf(conn))
	return stream
}
