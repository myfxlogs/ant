//go:build integration

package system

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
)

func TestMtHub_PlaceOrderCloseOrderLifecycle(t *testing.T) {
	harness := newMtHubTestHarness(t)

	// Step 1: PlaceOrder.
	ctx, cancel := harness.ctxWithTimeout()
	defer cancel()

	placeReq := connect.NewRequest(&antv1.PlaceOrderRequest{
		AccountId:  harness.accountID,
		Canonical:  "EURUSD",
		Side:       antv1.Side_SIDE_BUY,
		OrderType:  antv1.OrderType_ORDER_TYPE_MARKET,
		Volume:     "0.1",
		Price:      "1.08500",
		StopLoss:   "1.08000",
		TakeProfit: "1.09000",
		Comment:    "integration-test",
	})
	placeResp, err := harness.server.PlaceOrder(ctx, placeReq)
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}

	if placeResp.Msg.Ticket <= 0 {
		t.Fatalf("expected positive ticket, got %d", placeResp.Msg.Ticket)
	}
	if placeResp.Msg.Status != "submitted" {
		t.Errorf("expected status 'submitted', got %q", placeResp.Msg.Status)
	}
	t.Logf("PlaceOrder succeeded: ticket=%d", placeResp.Msg.Ticket)

	// Step 2: OpenedOrders — position should appear.
	ctx2, cancel2 := harness.ctxWithTimeout()
	defer cancel2()

	openReq := connect.NewRequest(&antv1.OpenedOrdersRequest{AccountId: harness.accountID})
	openResp, err := harness.server.OpenedOrders(ctx2, openReq)
	if err != nil {
		t.Fatalf("OpenedOrders: %v", err)
	}
	if len(openResp.Msg.Orders) != 1 {
		t.Fatalf("expected 1 opened order, got %d", len(openResp.Msg.Orders))
	}
	order := openResp.Msg.Orders[0]
	if order.Ticket != placeResp.Msg.Ticket {
		t.Errorf("OpenedOrders ticket mismatch: want %d, got %d", placeResp.Msg.Ticket, order.Ticket)
	}
	if order.Canonical != "EURUSD" {
		t.Errorf("OpenedOrders canonical mismatch: want EURUSD, got %s", order.Canonical)
	}
	t.Logf("OpenedOrders: found position ticket=%d canonical=%s", order.Ticket, order.Canonical)

	// Step 3: CloseOrder.
	ctx3, cancel3 := harness.ctxWithTimeout()
	defer cancel3()

	closeReq := connect.NewRequest(&antv1.CloseOrderRequest{
		AccountId: harness.accountID,
		Ticket:    placeResp.Msg.Ticket,
	})
	closeResp, err := harness.server.CloseOrder(ctx3, closeReq)
	if err != nil {
		t.Fatalf("CloseOrder: %v", err)
	}
	if closeResp.Msg.Status != "closed" {
		t.Errorf("expected status 'closed', got %q", closeResp.Msg.Status)
	}
	t.Logf("CloseOrder succeeded: status=%s", closeResp.Msg.Status)

	// Step 4: OpenedOrders should be empty now.
	ctx4, cancel4 := harness.ctxWithTimeout()
	defer cancel4()

	openReq2 := connect.NewRequest(&antv1.OpenedOrdersRequest{AccountId: harness.accountID})
	openResp2, err := harness.server.OpenedOrders(ctx4, openReq2)
	if err != nil {
		t.Fatalf("OpenedOrders after close: %v", err)
	}
	if len(openResp2.Msg.Orders) != 0 {
		t.Errorf("expected 0 opened orders after close, got %d", len(openResp2.Msg.Orders))
	}
	t.Logf("OpenedOrders after close: empty PASS")
}

// ===========================================================================
// Test 2: PlaceOrder with invalid parameters
// ===========================================================================

func TestMtHub_PlaceOrderWithEmptyCanonical(t *testing.T) {
	harness := newMtHubTestHarness(t)

	ctx, cancel := harness.ctxWithTimeout()
	defer cancel()

	req := connect.NewRequest(&antv1.PlaceOrderRequest{
		AccountId: harness.accountID,
		Canonical: "", // empty — should fail
		Side:      antv1.Side_SIDE_BUY,
		OrderType: antv1.OrderType_ORDER_TYPE_MARKET,
		Volume:    "0.1",
	})
	_, err := harness.server.PlaceOrder(ctx, req)
	if err == nil {
		t.Fatal("expected error for empty canonical, got nil")
	}
	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T: %v", err, err)
	}
	if connectErr.Code() != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", connectErr.Code())
	}
	t.Logf("empty canonical -> %v (code=%v) PASS", connectErr.Message(), connectErr.Code())
}

func TestMtHub_PlaceOrderWithNegativeVolume(t *testing.T) {
	harness := newMtHubTestHarness(t)

	ctx, cancel := harness.ctxWithTimeout()
	defer cancel()

	req := connect.NewRequest(&antv1.PlaceOrderRequest{
		AccountId: harness.accountID,
		Canonical: "EURUSD",
		Side:      antv1.Side_SIDE_BUY,
		OrderType: antv1.OrderType_ORDER_TYPE_MARKET,
		Volume:    "-0.1", // negative — should fail
	})
	_, err := harness.server.PlaceOrder(ctx, req)
	if err == nil {
		t.Fatal("expected error for negative volume, got nil")
	}
	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T: %v", err, err)
	}
	if connectErr.Code() != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", connectErr.Code())
	}
	t.Logf("negative volume -> %v (code=%v) PASS", connectErr.Message(), connectErr.Code())
}

func TestMtHub_PlaceOrderWithInvalidPrice(t *testing.T) {
	harness := newMtHubTestHarness(t)

	ctx, cancel := harness.ctxWithTimeout()
	defer cancel()

	req := connect.NewRequest(&antv1.PlaceOrderRequest{
		AccountId: harness.accountID,
		Canonical: "EURUSD",
		Side:      antv1.Side_SIDE_BUY,
		OrderType: antv1.OrderType_ORDER_TYPE_MARKET,
		Volume:    "0.1",
		Price:     "not-a-number", // invalid — should fail
	})
	_, err := harness.server.PlaceOrder(ctx, req)
	if err == nil {
		t.Fatal("expected error for invalid price, got nil")
	}
	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T: %v", err, err)
	}
	if connectErr.Code() != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", connectErr.Code())
	}
	t.Logf("invalid price -> %v (code=%v) PASS", connectErr.Message(), connectErr.Code())
}

// ===========================================================================
// Test 3: OpenedOrders returns empty for new account
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
