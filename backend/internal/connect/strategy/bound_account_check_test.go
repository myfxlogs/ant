package strategy

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/service"
)

// mockBoundSvc is a test stub for BoundAccountChecker.
type mockBoundSvc struct {
	err error
}

func (m *mockBoundSvc) EnsureBoundAccount(ctx context.Context, userID, accountID uuid.UUID) error {
	return m.err
}

// TestRunLiveStrategy_BoundAccountCheck_RejectsUnbound is the adversarial proof:
// A free-tier user (max_mt_accounts=1) with one account already bound
// must be rejected when attempting to start a live strategy on a second account.
//
// Adversarial proof: remove the bound account check in RunLiveStrategy → this test goes red.
func TestRunLiveStrategy_BoundAccountCheck_RejectsUnbound(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, zap.NewNop())
	srv.SetBoundSvc(&mockBoundSvc{err: service.ErrAccountLimitExceeded})

	cfg := LiveStrategyConfig{
		RunID:     uuid.New(),
		AccountID: uuid.New().String(),
		UserID:    uuid.New().String(),
		Mode:      "live",
		Symbol:    "EURUSD",
	}

	err := srv.RunLiveStrategy(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error from bound account check, got nil — adversarial proof: check is missing!")
	}
	if !errors.Is(err, service.ErrAccountLimitExceeded) {
		t.Fatalf("expected ErrAccountLimitExceeded, got: %v", err)
	}
}

// TestRunLiveStrategy_BoundAccountCheck_PaperModeSkips verifies that paper mode
// does NOT trigger the bound account check (paper trading doesn't use real MT accounts).
func TestRunLiveStrategy_BoundAccountCheck_PaperModeSkips(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, zap.NewNop())
	// boundSvc always returns error — if paper mode triggers the check, test fails.
	srv.SetBoundSvc(&mockBoundSvc{err: service.ErrAccountLimitExceeded})

	cfg := LiveStrategyConfig{
		RunID:     uuid.New(),
		AccountID: uuid.New().String(),
		UserID:    uuid.New().String(),
		Mode:      "paper",
		Symbol:    "EURUSD",
	}

	err := srv.RunLiveStrategy(context.Background(), cfg)
	// Paper mode should skip the bound account check and fail on a different check
	// (no barSource configured), NOT on bound account.
	if err != nil && errors.Is(err, service.ErrAccountLimitExceeded) {
		t.Fatal("paper mode should NOT trigger bound account check")
	}
}

// TestRunLiveStrategy_BoundAccountCheck_NilBoundSvcSkips verifies that nil boundSvc
// is a no-op (backward compatibility for environments without subscription service).
func TestRunLiveStrategy_BoundAccountCheck_NilBoundSvcSkips(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, zap.NewNop())
	// boundSvc is nil — check should be skipped

	cfg := LiveStrategyConfig{
		RunID:     uuid.New(),
		AccountID: uuid.New().String(),
		UserID:    uuid.New().String(),
		Mode:      "live",
		Symbol:    "EURUSD",
	}

	err := srv.RunLiveStrategy(context.Background(), cfg)
	// Should fail on barSource/gate, NOT on bound account
	if err != nil && errors.Is(err, service.ErrAccountLimitExceeded) {
		t.Fatal("nil boundSvc should NOT trigger bound account check")
	}
}

// TestCheckBoundAccount_NilAccountIDSkips verifies that uuid.Nil accountID
// is a no-op (some paths may not have an account ID yet).
func TestCheckBoundAccount_NilAccountIDSkips(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, zap.NewNop())
	srv.SetBoundSvc(&mockBoundSvc{err: service.ErrAccountLimitExceeded})

	err := srv.checkBoundAccount(context.Background(), uuid.New(), uuid.Nil)
	if err != nil {
		t.Fatalf("nil accountID should skip check, got: %v", err)
	}
}
