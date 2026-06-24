package paper

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"anttrader/internal/repository"
)

// stubPaperRepo implements paperRepository for tests.
type stubPaperRepo struct {
	accounts map[string]*repository.PaperAccount
	orders   []*repository.PaperOrder
	createErr error
	listErr   error
	updateErr error
}

func newStubPaperRepo() *stubPaperRepo {
	return &stubPaperRepo{
		accounts: make(map[string]*repository.PaperAccount),
	}
}

func (s *stubPaperRepo) CreateOrder(_ context.Context, o *repository.PaperOrder) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.orders = append(s.orders, o)
	return nil
}

func (s *stubPaperRepo) ListAccounts(_ context.Context, _ string) ([]*repository.PaperAccount, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	out := make([]*repository.PaperAccount, 0, len(s.accounts))
	for _, a := range s.accounts {
		out = append(out, a)
	}
	return out, nil
}

func (s *stubPaperRepo) GetAccount(_ context.Context, id string) (*repository.PaperAccount, error) {
	for _, a := range s.accounts {
		if a.ID == id { return a, nil }
	}
	return nil, nil
}

func (s *stubPaperRepo) GetOrder(_ context.Context, _ string) (*repository.PaperOrder, error) {
	return nil, nil
}
func (s *stubPaperRepo) UpdateOrder(_ context.Context, _ *repository.PaperOrder) error {
	return nil
}
func (s *stubPaperRepo) FindOpenOrder(_ context.Context, _, _ string) (*repository.PaperOrder, error) {
	return nil, nil
}

func (s *stubPaperRepo) UpdateAccountBalance(_ context.Context, accountID string, balance, equity decimal.Decimal) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	if a, ok := s.accounts[accountID]; ok {
		a.CurrentBalance = balance
		a.Equity = equity
	}
	return nil
}

func (s *stubPaperRepo) seedAccount(id string, balance float64) {
	s.accounts[id] = &repository.PaperAccount{
		ID:             id,
		CurrentBalance: decimal.NewFromFloat(balance),
		Equity:         decimal.NewFromFloat(balance),
	}
}

func testLogger() *zap.Logger { return zap.NewNop() }

func TestNew_CreatesEmptySubscribers(t *testing.T) {
	t.Parallel()
	repo := newStubPaperRepo()
	engine := New(repo, nil, testLogger())
	if engine.subscribers == nil {
		t.Fatal("subscribers map should be initialized")
	}
	if len(engine.subscribers) != 0 {
		t.Fatalf("expected 0 subscribers, got %d", len(engine.subscribers))
	}
}

func TestPlacePaperOrder_BuyFillsAtAsk(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := newStubPaperRepo()
	repo.seedAccount("pa-1", 10000)
	engine := New(repo, nil, testLogger())

	err := engine.PlacePaperOrder(ctx, "pa-1", "EURUSD", "buy",
		decimal.NewFromFloat(0.1), 1.1000, 1.1005)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(repo.orders))
	}
	o := repo.orders[0]
	if o.Side != "buy" {
		t.Errorf("expected side buy, got %s", o.Side)
	}
	// Buy fills at ask.
	if !o.FillPrice.Equal(decimal.NewFromFloat(1.1005)) {
		t.Errorf("expected fill at ask 1.1005, got %s", o.FillPrice)
	}
	// Balance reduced by margin.
	acct := repo.accounts["pa-1"]
	margin := decimal.NewFromFloat(0.1).Mul(decimal.NewFromFloat(1.1005)).Mul(decimal.NewFromFloat(0.01))
	expected := decimal.NewFromFloat(10000).Sub(margin)
	if !acct.CurrentBalance.Equal(expected) {
		t.Errorf("expected balance %s, got %s", expected, acct.CurrentBalance)
	}
}

func TestPlacePaperOrder_SellFillsAtBid(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := newStubPaperRepo()
	repo.seedAccount("pa-2", 20000)
	engine := New(repo, nil, testLogger())

	err := engine.PlacePaperOrder(ctx, "pa-2", "GBPUSD", "sell",
		decimal.NewFromFloat(0.2), 1.2500, 1.2505)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(repo.orders))
	}
	o := repo.orders[0]
	if o.Side != "sell" {
		t.Errorf("expected side sell, got %s", o.Side)
	}
	if !o.FillPrice.Equal(decimal.NewFromFloat(1.2500)) {
		t.Errorf("expected fill at bid 1.2500, got %s", o.FillPrice)
	}
}

func TestPlacePaperOrder_FallbackToMidPrice(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := newStubPaperRepo()
	repo.seedAccount("pa-3", 5000)
	engine := New(repo, nil, testLogger())

	// Both bid and ask are 0.
	err := engine.PlacePaperOrder(ctx, "pa-3", "XAUUSD", "buy",
		decimal.NewFromFloat(1), 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(repo.orders))
	}
	// Mid = (0+0)/2 = 0. Fill should be 0.
	if !repo.orders[0].FillPrice.Equal(decimal.Zero) {
		t.Errorf("expected fill price 0 when bid=ask=0, got %s", repo.orders[0].FillPrice)
	}
}

func TestPlacePaperOrder_AccountNotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := newStubPaperRepo()
	// No accounts seeded.
	engine := New(repo, nil, testLogger())

	err := engine.PlacePaperOrder(ctx, "nonexistent", "EURUSD", "buy",
		decimal.NewFromFloat(0.1), 1.1000, 1.1005)
	if err != nil {
		t.Fatalf("expected no error (graceful), got %v", err)
	}
	// Order is still created even if account not found.
	if len(repo.orders) != 1 {
		t.Fatalf("order should be created even without account, got %d", len(repo.orders))
	}
}

func TestPlacePaperOrder_CreateOrderError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := newStubPaperRepo()
	repo.createErr = context.DeadlineExceeded
	engine := New(repo, nil, testLogger())

	err := engine.PlacePaperOrder(ctx, "pa-1", "EURUSD", "buy",
		decimal.NewFromFloat(0.1), 1.1000, 1.1005)
	if err == nil {
		t.Fatal("expected error from CreateOrder, got nil")
	}
}

func TestSubscribe_ReceivesUpdateAfterOrder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := newStubPaperRepo()
	repo.seedAccount("pa-sub", 10000)
	engine := New(repo, nil, testLogger())

	ch, unsubscribe := engine.Subscribe("pa-sub")
	defer unsubscribe()

	// Place order should trigger broadcast.
	err := engine.PlacePaperOrder(ctx, "pa-sub", "EURUSD", "buy",
		decimal.NewFromFloat(0.1), 1.1000, 1.1005)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case acct := <-ch:
		if acct.ID != "pa-sub" {
			t.Errorf("expected account pa-sub, got %s", acct.ID)
		}
	default:
		t.Fatal("expected to receive account update on channel")
	}
}

func TestSubscribe_UnsubscribeStopsReceiving(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := newStubPaperRepo()
	repo.seedAccount("pa-unsub", 10000)
	engine := New(repo, nil, testLogger())

	ch, unsubscribe := engine.Subscribe("pa-unsub")
	unsubscribe()

	err := engine.PlacePaperOrder(ctx, "pa-unsub", "EURUSD", "buy",
		decimal.NewFromFloat(0.1), 1.1000, 1.1005)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("channel should be closed after unsubscribe")
		}
	default:
		// Channel closed — expected.
	}
}
