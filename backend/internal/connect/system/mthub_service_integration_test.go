//go:build integration

package system

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/interceptor"
	"anttrader/internal/mthub"
	"anttrader/internal/service"
)

// ------------------------------ test helper: pg pool ----------------------------------

func testPG(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		password := os.Getenv("DB_PASSWORD")
		user := os.Getenv("DB_USER")
		if user == "" { user = "ant" }
		dbname := os.Getenv("DB_NAME")
		if dbname == "" { dbname = "ant" }
		dsn = "postgres://" + user + ":" + password + "@localhost:5433/" + dbname + "?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("skipping integration test: pg connect: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("skipping integration test: pg ping: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// ------------------------------ stateful mock executor ----------------------------------

// trackedExecutor is an OrderExecutor that maintains opened positions so the
// PlaceOrder -> OpenedOrders -> CloseOrder -> OpenedOrders lifecycle is realistic.
type trackedExecutor struct {
	mu       sync.Mutex
	nextID   int64
	orders   map[int64]*mthub.OrderRecord
	platform string
}

func newTrackedExecutor(platform string) *trackedExecutor {
	return &trackedExecutor{
		nextID:   100000,
		orders:   make(map[int64]*mthub.OrderRecord),
		platform: platform,
	}
}

func (e *trackedExecutor) Platform() string { return e.platform }

func (e *trackedExecutor) PlaceOrder(_ context.Context, req *mthub.OrderRequest) (int64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	ticket := e.nextID
	e.nextID++

	rec := &mthub.OrderRecord{
		Ticket:     ticket,
		AccountID:  req.AccountID,
		SymbolRaw:  req.Canonical,
		Canonical:  req.Canonical,
		Side:       req.Side,
		OrderType:  req.OrderType,
		Volume:     req.Volume,
		OpenPrice:  req.Price,
		ClosePrice: decimal.Zero,
		Profit:     decimal.Zero,
		Commission: decimal.Zero,
		Swap:       decimal.Zero,
		OpenTime:   time.Now(),
		CloseTime:  time.Time{},
		Comment:    req.Comment,
		Magic:      req.Magic,
		State:      mthub.OrderStateOpen,
	}
	e.orders[ticket] = rec
	return ticket, nil
}

func (e *trackedExecutor) CloseOrder(_ context.Context, ticket int64, _ decimal.Decimal) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.orders, ticket)
	return nil
}

func (e *trackedExecutor) ModifyOrder(_ context.Context, _ int64, _, _, _ decimal.Decimal) error {
	return nil
}

func (e *trackedExecutor) FetchOpenedOrders(_ context.Context) ([]*mthub.OrderRecord, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]*mthub.OrderRecord, 0, len(e.orders))
	for _, rec := range e.orders {
		out = append(out, rec)
	}
	return out, nil
}

func (e *trackedExecutor) FetchOrderHistory(_ context.Context, _, _ time.Time) ([]*mthub.OrderRecord, error) {
	return nil, nil // history is always empty in test harness
}

func (e *trackedExecutor) FetchSymbolParams(_ context.Context, _ []string) ([]*mthub.SymbolParam, error) {
	return nil, nil
}

func (e *trackedExecutor) SubscribeOrderEvents(_ context.Context, _ mthub.OrderEventHandler) error {
	return nil
}

// ------------------------------ test infra builder ----------------------------------

type mtHubTestHarness struct {
	pool      *pgxpool.Pool
	userID    uuid.UUID
	accountID string // the mt_accounts row ID (UUID)
	hub       *mthub.Hub
	exec      *trackedExecutor
	svc       *mthub.MtHubService
	platform  *service.PlatformService
	server    *MtHubServer
	streamSrv *StreamServer
}

func newMtHubTestHarness(t *testing.T) *mtHubTestHarness {
	t.Helper()
	pool := testPG(t)
	ctx := context.Background()
	log := zap.NewNop()

	// Create test user.
	userID := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
		 VALUES ($1, $2, '$argon2id$v=19$m=65536,t=3,p=2$test$test', 'user', 'active', NOW(), NOW())
		 ON CONFLICT (id) DO NOTHING`,
		userID, fmt.Sprintf("test-mthub-%s@anttest.io", uuid.New().String()[:8]),
	)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	// Create test account with balance/equity for SSE snapshot tests.
	var accountID string
	err = pool.QueryRow(ctx,
		`INSERT INTO mt_accounts (user_id, login, password, mt_type, broker_company, broker_server, broker_host, account_status,
			balance, equity, credit, margin, free_margin)
		 VALUES ($1, 'testlogin', 'testpass', 'mt5', 'TestBroker', 'TestServer', 'test.example.com', 'connected',
		 	10000, 10100, 0, 1000, 9100)
		 RETURNING id::text`,
		userID,
	).Scan(&accountID)
	if err != nil {
		t.Fatalf("insert test account: %v", err)
	}

	// Build services.
	accountSvc := service.NewAccountService(pool)
	platformSvc := service.NewPlatformService(pool, accountSvc)

	hub := mthub.NewHub()
	exec := newTrackedExecutor("mt5")
	hub.Register(accountID, &mthub.Session{AccountID: accountID, CreatedAt: time.Now()}, exec)

	broker := mthub.NewOrderEventBroker()
	accountBroker := mthub.NewAccountProfitBroker()
	snapshotBroker := mthub.NewPositionSnapshotBroker()
	svc := mthub.NewMtHubService(hub, broker, accountBroker, snapshotBroker, nil, nil, nil)
	svc.SetLogger(log)

	svr := NewMtHubServer(svc, platformSvc, nil, nil, log)
	streamSrv := NewStreamServer(svc, platformSvc, log)

	return &mtHubTestHarness{
		pool:      pool,
		userID:    userID,
		accountID: accountID,
		hub:       hub,
		exec:      exec,
		svc:       svc,
		platform:  platformSvc,
		server:    svr,
		streamSrv: streamSrv,
	}
}

func (h *mtHubTestHarness) ctx() context.Context {
	return context.WithValue(context.Background(), interceptor.UserIDKey, h.userID.String())
}

func (h *mtHubTestHarness) ctxWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(h.ctx(), 10*time.Second)
}

// ===========================================================================
// Test 1: PlaceOrder -> CloseOrder lifecycle
// ===========================================================================

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

	accountSvc := service.NewAccountService(pool)
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
