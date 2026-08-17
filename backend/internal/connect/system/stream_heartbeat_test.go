//go:build integration

package system

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/mthub"
	"alphaforge/internal/service"
)

// TestSubscribeOrderUpdates_Heartbeat verifies STREAM-FREEZE-1 Task B:
// SubscribeOrderUpdates sends a heartbeat (empty OrderUpdateEvent) every
// heartbeatInterval to keep the SSE connection alive during idle periods.
//
// Adversarial proof: Delete the `case <-heartbeat.C` branch →
// no heartbeat Send within 2s → test times out (RED).
func TestSubscribeOrderUpdates_Heartbeat(t *testing.T) {
	original := orderProfitHeartbeatInterval
	orderProfitHeartbeatInterval = 100 * time.Millisecond
	defer func() { orderProfitHeartbeatInterval = original }()

	pool := testPG(t)
	accountSvc := service.NewAccountService(pool, nil)
	platformSvc := service.NewPlatformService(pool, accountSvc)
	broker := mthub.NewOrderEventBroker()
	svc := mthub.NewMtHubService(nil, broker, nil, nil, nil, nil, nil)

	userID := uuid.New().String()
	accountID := insertHeartbeatAccount(t, pool, userID)

	srv := &StreamServer{
		svc:      svc,
		platform: platformSvc,
		log:      zap.NewNop(),
	}

	mux := http.NewServeMux()
	mux.Handle(antv1c.NewStreamServiceHandler(srv,
		connect.WithInterceptors(&fakeAuthInterceptorSys{userID: userID})))

	server := httptest.NewServer(mux)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := antv1c.NewStreamServiceClient(http.DefaultClient, server.URL)
	stream, err := client.SubscribeOrderUpdates(ctx,
		connect.NewRequest(&antv1.SubscribeOrderUpdatesRequest{AccountId: accountID}))
	if err != nil {
		t.Fatalf("SubscribeOrderUpdates failed: %v", err)
	}

	heartbeatReceived := false
	deadline := time.After(2 * time.Second)
	for !heartbeatReceived {
		select {
		case <-deadline:
			t.Fatal("no heartbeat received within 2s — RED: case <-heartbeat.C missing or broken")
		default:
		}
		ok := stream.Receive()
		if err := stream.Err(); err != nil {
			t.Fatalf("Receive failed waiting for heartbeat: %v", err)
		}
		if !ok {
			t.Fatal("stream closed before heartbeat")
		}
		ev := stream.Msg()
		if ev.GetAccountId() == "" && ev.GetTicket() == 0 {
			heartbeatReceived = true
		}
	}

	if !heartbeatReceived {
		t.Fatal("heartbeat event was not received")
	}
}

// TestSubscribeProfitUpdates_Heartbeat verifies STREAM-FREEZE-1 Task B:
// SubscribeProfitUpdates sends a heartbeat (empty ProfitUpdateEvent) every
// heartbeatInterval.
//
// Adversarial proof: Delete the `case <-heartbeat.C` branch →
// no heartbeat Send within 2s → test times out (RED).
func TestSubscribeProfitUpdates_Heartbeat(t *testing.T) {
	original := orderProfitHeartbeatInterval
	orderProfitHeartbeatInterval = 100 * time.Millisecond
	defer func() { orderProfitHeartbeatInterval = original }()

	pool := testPG(t)
	accountSvc := service.NewAccountService(pool, nil)
	platformSvc := service.NewPlatformService(pool, accountSvc)
	profitBroker := mthub.NewAccountProfitBroker()
	svc := mthub.NewMtHubService(nil, nil, profitBroker, nil, nil, nil, nil)

	userID := uuid.New().String()
	accountID := insertHeartbeatAccount(t, pool, userID)

	srv := &StreamServer{
		svc:      svc,
		platform: platformSvc,
		log:      zap.NewNop(),
	}

	mux := http.NewServeMux()
	mux.Handle(antv1c.NewStreamServiceHandler(srv,
		connect.WithInterceptors(&fakeAuthInterceptorSys{userID: userID})))

	server := httptest.NewServer(mux)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := antv1c.NewStreamServiceClient(http.DefaultClient, server.URL)
	stream, err := client.SubscribeProfitUpdates(ctx,
		connect.NewRequest(&antv1.SubscribeProfitUpdatesRequest{AccountId: accountID}))
	if err != nil {
		t.Fatalf("SubscribeProfitUpdates failed: %v", err)
	}

	heartbeatReceived := false
	deadline := time.After(2 * time.Second)
	for !heartbeatReceived {
		select {
		case <-deadline:
			t.Fatal("no heartbeat received within 2s — RED: case <-heartbeat.C missing or broken")
		default:
		}
		ok := stream.Receive()
		if err := stream.Err(); err != nil {
			t.Fatalf("Receive failed waiting for heartbeat: %v", err)
		}
		if !ok {
			t.Fatal("stream closed before heartbeat")
		}
		ev := stream.Msg()
		if ev.GetAccountId() == "" {
			heartbeatReceived = true
		}
	}

	if !heartbeatReceived {
		t.Fatal("heartbeat event was not received")
	}
}

// fakeAuthInterceptorSys injects a user ID into context for testing.
type fakeAuthInterceptorSys struct {
	userID string
}

func (f *fakeAuthInterceptorSys) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		ctx = context.WithValue(ctx, interceptor.UserIDKey, f.userID)
		return next(ctx, req)
	}
}

func (f *fakeAuthInterceptorSys) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (f *fakeAuthInterceptorSys) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		ctx = context.WithValue(ctx, interceptor.UserIDKey, f.userID)
		return next(ctx, conn)
	}
}

// insertHeartbeatAccount inserts a test account owned by userID and returns its UUID string.
func insertHeartbeatAccount(t *testing.T, pool *pgxpool.Pool, userID string) string {
	t.Helper()
	ctx := context.Background()
	var accountID string
	err := pool.QueryRow(ctx,
		`INSERT INTO mt_accounts (user_id, login, password, mt_type, broker_company, broker_server, broker_host, account_status)
		 VALUES ($1, 'hbtest', 'hbpass', 'mt5', 'TestBroker', 'TestServer', 'test.example.com', 'connected')
		 RETURNING id::text`,
		userID,
	).Scan(&accountID)
	if err != nil {
		t.Fatalf("insertHeartbeatAccount failed: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE id::text = $1`, accountID)
	})
	return accountID
}
