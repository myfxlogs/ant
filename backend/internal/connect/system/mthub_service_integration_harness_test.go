//go:build integration

package system

import (
	"context"
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

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/mthub"
	"alphaforge/internal/service"
)

func testPG(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		password := os.Getenv("DB_PASSWORD")
		user := os.Getenv("DB_USER")
		if user == "" {
			user = "ant"
		}
		dbname := os.Getenv("DB_NAME")
		if dbname == "" {
			dbname = "ant"
		}
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

func (e *trackedExecutor) PlaceOrder(_ context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
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
	return rec, nil
}

func (e *trackedExecutor) CloseOrder(_ context.Context, ticket int64, _ decimal.Decimal) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.orders, ticket)
	return nil
}

func (e *trackedExecutor) DeleteOrder(_ context.Context, ticket int64) error {
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

func (e *trackedExecutor) FetchAllSymbols(_ context.Context) ([]string, error) {
	return nil, nil
}

func (e *trackedExecutor) FetchPriceHistory(_ context.Context, _, _ string, _, _ int64, _ int) ([]*mthub.Bar, error) {
	return nil, nil
}

func (e *trackedExecutor) AddSymbols(_ context.Context, _ []string) error {
	return nil
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
	accountSvc := service.NewAccountService(pool, service.NewTestSecretsClient(t))
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

// mockStreamConn implements connect.StreamingHandlerConn and captures Send calls.
type mockStreamConn struct {
	ch  chan *antv1.StreamEvent
	ctx context.Context
}

func (m *mockStreamConn) Spec() connect.Spec { return connect.Spec{} }

func (m *mockStreamConn) Peer() connect.Peer { return connect.Peer{} }

func (m *mockStreamConn) Receive(any) error { return nil }

func (m *mockStreamConn) RequestHeader() http.Header { return http.Header{} }

func (m *mockStreamConn) ResponseHeader() http.Header { return http.Header{} }

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
