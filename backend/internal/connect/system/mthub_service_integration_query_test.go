//go:build integration

package system

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/mthub"
	"alphaforge/internal/service"
)

func TestMtHub_OpenedOrdersEmptyForNewAccount(t *testing.T) {
	pool := testPG(t)
	ctx := context.Background()
	log := zap.NewNop()

	userID := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
		 VALUES ($1, $2, '$argon2id$v=19$m=65536,t=3,p=2$test$test', 'user', 'active', NOW(), NOW())`,
		userID, fmt.Sprintf("test-empty-orders-%s@anttest.io", uuid.New().String()[:8]),
	)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	var accountID string
	err = pool.QueryRow(ctx,
		`INSERT INTO mt_accounts (user_id, login, password, mt_type, broker_company, broker_server, broker_host, account_status)
		 VALUES ($1, 'emptylogin', 'testpass', 'mt5', 'TestBroker', 'TestServer', 'test.example.com', 'connected')
		 RETURNING id::text`,
		userID,
	).Scan(&accountID)
	if err != nil {
		t.Fatalf("insert test account: %v", err)
	}

	accountSvc := service.NewAccountService(pool, service.NewTestSecretsClient(t))
	platformSvc := service.NewPlatformService(pool, accountSvc)

	hub := mthub.NewHub()
	exec := newTrackedExecutor("mt5")
	hub.Register(accountID, &mthub.Session{AccountID: accountID, CreatedAt: time.Now()}, exec)
	broker := mthub.NewOrderEventBroker()
	svc := mthub.NewMtHubService(hub, broker, mthub.NewAccountProfitBroker(), mthub.NewPositionSnapshotBroker(), nil, nil, nil)
	svc.SetLogger(log)
	svr := NewMtHubServer(svc, platformSvc, nil, nil, log)

	testCtx := context.WithValue(ctx, interceptor.UserIDKey, userID.String())
	testCtx, cancel := context.WithTimeout(testCtx, 10*time.Second)
	defer cancel()

	req := connect.NewRequest(&antv1.OpenedOrdersRequest{AccountId: accountID})
	resp, err := svr.OpenedOrders(testCtx, req)
	if err != nil {
		t.Fatalf("OpenedOrders: %v", err)
	}
	if len(resp.Msg.Orders) != 0 {
		t.Errorf("expected 0 orders for new account, got %d", len(resp.Msg.Orders))
	}
	t.Logf("OpenedOrders for new account: got %d orders (expected 0) PASS", len(resp.Msg.Orders))
}

// ===========================================================================
// Test 4: OrderHistory with time range
// ===========================================================================

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
