package paper

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/connect/strategy"
	"alphaforge/internal/interceptor"
	papereng "alphaforge/internal/paper"
	"alphaforge/internal/repository"
)

// ── Test stubs ──

type stubStrategyRunner struct {
	mu    sync.Mutex
	calls []strategy.LiveStrategyConfig
	err   error
}

func (s *stubStrategyRunner) RunLiveStrategy(_ context.Context, cfg strategy.LiveStrategyConfig) error {
	s.mu.Lock()
	s.calls = append(s.calls, cfg)
	s.mu.Unlock()
	return s.err
}

func (s *stubStrategyRunner) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func (s *stubStrategyRunner) lastCall() *strategy.LiveStrategyConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.calls) == 0 {
		return nil
	}
	return &s.calls[len(s.calls)-1]
}

// stubPaperRepo implements paperRepository for handler tests.
type stubPaperRepo struct {
	accounts map[string]*repository.PaperAccount
}

func newHandlerStubRepo() *stubPaperRepo {
	return &stubPaperRepo{accounts: make(map[string]*repository.PaperAccount)}
}

func (s *stubPaperRepo) CreateAccount(_ context.Context, userID, name string, initialBalance decimal.Decimal) (*repository.PaperAccount, error) {
	a := &repository.PaperAccount{
		ID:             fmt.Sprintf("pa-%s-%s", userID, name),
		UserID:         userID,
		Name:           name,
		InitialBalance: initialBalance,
		CurrentBalance: initialBalance,
		Equity:         initialBalance,
		Currency:       "USD",
	}
	s.accounts[a.ID] = a
	return a, nil
}

func (s *stubPaperRepo) ListAccounts(_ context.Context, _ string) ([]*repository.PaperAccount, error) {
	out := make([]*repository.PaperAccount, 0, len(s.accounts))
	for _, a := range s.accounts {
		out = append(out, a)
	}
	return out, nil
}

func (s *stubPaperRepo) CreateOrder(_ context.Context, _ *repository.PaperOrder) error { return nil }
func (s *stubPaperRepo) GetAccount(_ context.Context, id string) (*repository.PaperAccount, error) {
	if a, ok := s.accounts[id]; ok { return a, nil }
	return nil, nil
}

func (s *stubPaperRepo) UpdateAccountBalance(_ context.Context, _ string, _, _ decimal.Decimal) error {
	return nil
}
func (s *stubPaperRepo) GetOrder(_ context.Context, _ string) (*repository.PaperOrder, error) {
	return nil, nil
}
func (s *stubPaperRepo) UpdateOrder(_ context.Context, _ *repository.PaperOrder) error {
	return nil
}
func (s *stubPaperRepo) FindOpenOrder(_ context.Context, _, _ string) (*repository.PaperOrder, error) {
	return nil, fmt.Errorf("not found")
}

func (s *stubPaperRepo) ListOrders(_ context.Context, _ string) ([]*repository.PaperOrder, error) {
	return nil, nil
}
func (s *stubPaperRepo) CreateStrategy(_ context.Context, _ *repository.PaperStrategy) error { return nil }
func (s *stubPaperRepo) ListStrategies(_ context.Context, _ string) ([]*repository.PaperStrategy, error) {
	return nil, nil
}
func (s *stubPaperRepo) UpdateOrderState(_ context.Context, _ string, _ string) error { return nil }

var _ paperRepo = (*stubPaperRepo)(nil)

func authCtx(userID string) context.Context {
	return context.WithValue(context.Background(), interceptor.UserIDKey, userID)
}

func testHandlerLogger() *zap.Logger { return zap.NewNop() }

func newTestHandler(runner StrategyRunner) *Handler {
	repo := newHandlerStubRepo()
	eng := papereng.New(repo, nil, testHandlerLogger())
	return NewHandler(repo, eng, runner, testHandlerLogger(), nil)
}

// ── Tests ──

func TestHandler_StartPaperStrategy_Deprecated(t *testing.T) {
	t.Parallel()
	h := newTestHandler(&stubStrategyRunner{})

	req := connect.NewRequest(&antv1.StartPaperStrategyRequest{
		PaperAccountId: "pa-1",
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		StrategyCode:   "print('hello')",
	})
	resp, err := h.StartPaperStrategy(authCtx("u1"), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Success {
		t.Fatal("expected deprecated failure")
	}
}

func TestHandler_StartPaperStrategy_MissingAccountID(t *testing.T) {
	t.Parallel()
	h := newTestHandler(&stubStrategyRunner{})

	req := connect.NewRequest(&antv1.StartPaperStrategyRequest{
		PaperAccountId: "", // empty
		Symbol:         "EURUSD",
		StrategyCode:   "print('x')",
	})
	resp, err := h.StartPaperStrategy(authCtx("u1"), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Success {
		t.Fatal("expected deprecated failure")
	}
}

func TestHandler_StartPaperStrategy_NilRunner(t *testing.T) {
	t.Parallel()
	h := newTestHandler(nil)

	req := connect.NewRequest(&antv1.StartPaperStrategyRequest{
		PaperAccountId: "pa-x",
		Symbol:         "EURUSD",
		StrategyCode:   "print('x')",
	})
	resp, err := h.StartPaperStrategy(authCtx("u1"), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Success {
		t.Fatal("expected deprecated failure")
	}
}

func TestHandler_StartPaperStrategy_DuplicateStart(t *testing.T) {
	t.Parallel()
	h := newTestHandler(&stubStrategyRunner{})

	req := connect.NewRequest(&antv1.StartPaperStrategyRequest{
		PaperAccountId: "pa-dup",
		Symbol:         "EURUSD",
		StrategyCode:   "print('x')",
	})
	resp, _ := h.StartPaperStrategy(authCtx("u1"), req)
	if resp.Msg.Success {
		t.Fatal("expected deprecated failure")
	}
}

func TestHandler_StopPaperStrategy_Deprecated(t *testing.T) {
	t.Parallel()
	h := newTestHandler(&stubStrategyRunner{})

	stopReq := connect.NewRequest(&antv1.StopPaperStrategyRequest{
		PaperAccountId: "pa-stop",
	})
	stopResp, err := h.StopPaperStrategy(authCtx("u1"), stopReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stopResp.Msg.Success {
		t.Fatal("expected deprecated failure")
	}
}

func TestHandler_StopPaperStrategy_NotRunning(t *testing.T) {
	t.Parallel()
	h := newTestHandler(&stubStrategyRunner{})

	req := connect.NewRequest(&antv1.StopPaperStrategyRequest{
		PaperAccountId: "pa-never-started",
	})
	resp, err := h.StopPaperStrategy(authCtx("u1"), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Success {
		t.Fatal("expected deprecated failure")
	}
}

func TestHandler_StartPaperStrategy_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := newTestHandler(&stubStrategyRunner{})

	req := connect.NewRequest(&antv1.StartPaperStrategyRequest{
		PaperAccountId: "pa-x",
		Symbol:         "EURUSD",
		StrategyCode:   "print('x')",
	})
	resp, err := h.StartPaperStrategy(context.Background(), req) // no userID in ctx
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Deprecated stub returns failure (not auth error) regardless of auth.
	if resp.Msg.Success {
		t.Fatal("expected deprecated failure")
	}
}

func TestHandler_CreatePaperAccount_Success(t *testing.T) {
	t.Parallel()
	runner := &stubStrategyRunner{}
	h := newTestHandler(runner)

	req := connect.NewRequest(&antv1.CreatePaperAccountRequest{
		Name:           "Test Account",
		InitialBalance: "50000",
	})
	resp, err := h.CreatePaperAccount(authCtx("u1"), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	acct := resp.Msg
	if acct.Name != "Test Account" {
		t.Errorf("expected name 'Test Account', got %s", acct.Name)
	}
	if acct.InitialBalance != "50000" {
		t.Errorf("expected balance 50000, got %s", acct.InitialBalance)
	}
}

func TestHandler_CreatePaperAccount_DefaultBalance(t *testing.T) {
	t.Parallel()
	runner := &stubStrategyRunner{}
	h := newTestHandler(runner)

	req := connect.NewRequest(&antv1.CreatePaperAccountRequest{
		Name:           "Default Balance",
		InitialBalance: "", // empty → default 10000
	})
	resp, err := h.CreatePaperAccount(authCtx("u1"), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.InitialBalance != "10000" {
		t.Errorf("expected default balance 10000, got %s", resp.Msg.InitialBalance)
	}
}

func TestHandler_ListPaperAccounts_Success(t *testing.T) {
	t.Parallel()
	runner := &stubStrategyRunner{}
	h := newTestHandler(runner)

	// Create one account first.
	createReq := connect.NewRequest(&antv1.CreatePaperAccountRequest{
		Name:           "A1",
		InitialBalance: "1000",
	})
	h.CreatePaperAccount(authCtx("u1"), createReq)

	listReq := connect.NewRequest(&antv1.ListPaperAccountsRequest{})
	resp, err := h.ListPaperAccounts(authCtx("u1"), listReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(resp.Msg.Accounts))
	}
	if resp.Msg.Accounts[0].Name != "A1" {
		t.Errorf("expected account name A1, got %s", resp.Msg.Accounts[0].Name)
	}
}
