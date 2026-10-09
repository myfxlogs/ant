// mutation_coordinator_test.go — Production-wiring adversarial tests for
// LIVE-ORDER-REENTRY-1 (B8). These tests exercise the REAL production call
// chain: dispatchLiveSignal → submitOrder → coordinateMutation → broker RPC
// → PositionSnapshotBroker subscription → confirmation.
//
// All tests use channel-based synchronization — NO time.Sleep for concurrency.
// Cutting the production wiring (submitOrder→coordinateMutation or
// OnOrderUpdate→barrier) must make these tests RED.

package strategy

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/mthub"
	"alphaforge/internal/risk"
)

// mockKillSwitch implements mthub.KillSwitchGate for testing.
type mockKillSwitch struct{ engaged bool }

func (m *mockKillSwitch) IsEngaged() bool { return m.engaged }

// toInt converts a numeric interface{} (int, int64, etc.) from a zaptest
// observer ContextMap to an int.

// toInt converts a numeric interface{} (int, int64, etc.) from a zaptest
// observer ContextMap to an int.
func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case int32:
		return int(n), true
	default:
		return 0, false
	}
}

// prodMockExecutor is a controllable mock for production-wiring tests.
// Each function is injectable per-test. Call counts are tracked atomically.

// prodMockExecutor is a controllable mock for production-wiring tests.
// Each function is injectable per-test. Call counts are tracked atomically.
type prodMockExecutor struct {
	placeFn  func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error)
	closeFn  func(ctx context.Context, ticket int64, lots decimal.Decimal) error
	deleteFn func(ctx context.Context, ticket int64) error
	modifyFn func(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error
	fetchFn  func(ctx context.Context) ([]*mthub.OrderRecord, error)

	placeCount  atomic.Int64
	closeCount  atomic.Int64
	deleteCount atomic.Int64
	modifyCount atomic.Int64
}

func (m *prodMockExecutor) Platform() string { return "mock" }

func (m *prodMockExecutor) PlaceOrder(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
	m.placeCount.Add(1)
	if m.placeFn != nil {
		return m.placeFn(ctx, req)
	}
	return &mthub.OrderRecord{Ticket: 1, AccountID: req.AccountID, Canonical: req.Canonical, State: mthub.OrderStateOpen}, nil
}

func (m *prodMockExecutor) CloseOrder(ctx context.Context, ticket int64, lots decimal.Decimal) error {
	m.closeCount.Add(1)
	if m.closeFn != nil {
		return m.closeFn(ctx, ticket, lots)
	}
	return nil
}

func (m *prodMockExecutor) DeleteOrder(ctx context.Context, ticket int64) error {
	m.deleteCount.Add(1)
	if m.deleteFn != nil {
		return m.deleteFn(ctx, ticket)
	}
	return nil
}

func (m *prodMockExecutor) ModifyOrder(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
	m.modifyCount.Add(1)
	if m.modifyFn != nil {
		return m.modifyFn(ctx, ticket, sl, tp, price)
	}
	return nil
}

func (m *prodMockExecutor) FetchOpenedOrders(ctx context.Context) ([]*mthub.OrderRecord, error) {
	if m.fetchFn != nil {
		return m.fetchFn(ctx)
	}
	return nil, nil
}

func (m *prodMockExecutor) FetchOrderHistory(_ context.Context, _, _ time.Time) ([]*mthub.OrderRecord, error) {
	return nil, nil
}

func (m *prodMockExecutor) FetchSymbolParams(_ context.Context, c []string) ([]*mthub.SymbolParam, error) {
	if len(c) == 0 {
		return nil, nil
	}
	return []*mthub.SymbolParam{{Canonical: c[0], ContractSize: decimal.NewFromInt(100000), LotSize: decimal.NewFromInt(100000)}}, nil
}

func (m *prodMockExecutor) FetchAllSymbols(_ context.Context) ([]string, error) { return nil, nil }

func (m *prodMockExecutor) FetchPriceHistory(_ context.Context, _ string, _ string, _, _ int64, _ int) ([]*mthub.Bar, error) {
	return nil, nil
}

func (m *prodMockExecutor) AddSymbols(_ context.Context, _ []string) error { return nil }

func (m *prodMockExecutor) SubscribeOrderEvents(_ context.Context, _ mthub.OrderEventHandler) error {
	return nil
}

// testCoordinatorSetup creates a full production wiring chain for testing.
// Returns the server, mtHub service, mock executor, and snapshot broker.

// testCoordinatorSetup creates a full production wiring chain for testing.
// Returns the server, mtHub service, mock executor, and snapshot broker.
func testCoordinatorSetup(exec *prodMockExecutor) (*StrategyExecutionServer, *mthub.MtHubService, *mthub.PositionSnapshotBroker) {
	hub := mthub.NewHub()
	broker := mthub.NewPositionSnapshotBroker()
	svc := mthub.NewMtHubService(hub, mthub.NewOrderEventBroker(), mthub.NewAccountProfitBroker(), broker, nil, nil, nil)
	svc.SetLogger(zap.NewNop())
	svc.SetGate(risk.NewDefaultGate())
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{Balance: decimal.NewFromInt(100000), Equity: decimal.NewFromInt(100000)}, nil
	})
	hub.Register("acct-1", &mthub.Session{AccountID: "acct-1", CreatedAt: time.Now()}, exec)
	srv := &StrategyExecutionServer{log: zap.NewNop(), mtHub: svc}
	return srv, svc, broker
}

func testLiveCfg() LiveStrategyConfig {
	return LiveStrategyConfig{
		AccountID:  "acct-1",
		UserID:     "user-1",
		Symbol:     "EURUSD",
		Mode:       "live",
		RunID:      uuid.New(),
		TickSeq:    new(atomic.Int64),
		ScheduleID: uuid.New(),
	}
}

func testActiveSess() *ActiveSession {
	return &ActiveSession{barrier: NewTradeBarrier(zap.NewNop())}
}

// publishOrderUpdate publishes an OnOrderUpdate event to the snapshot broker,
// simulating a broker push. This exercises the REAL subscription wiring.

// publishOrderUpdate publishes an OnOrderUpdate event to the snapshot broker,
// simulating a broker push. This exercises the REAL subscription wiring.
func publishOrderUpdate(broker *mthub.PositionSnapshotBroker, accountID string, ticket int64, magic int32, updateType string) {
	broker.Publish(&mthub.PositionSnapshot{
		AccountID:              accountID,
		FinancialsSource:       "order_stream",
		CapturedAt:             time.Now(),
		PositionsAuthoritative: true,
		PositionsCapturedAt:    time.Now(),
		PositionsSource:        "order_stream",
		Positions:              []mthub.PositionSnapshotItem{{Ticket: ticket, Magic: magic}},
		UpdateTicket:           ticket,
		UpdateType:             updateType,
		UpdateMagic:            magic,
	})
}

// T1-PROD: 100 concurrent tick signals, only 1 PlaceOrder call.
// Uses REAL dispatchLiveSignal → submitOrder → coordinateMutation wiring.
// Cutting submitOrder→coordinateMutation must RED.
