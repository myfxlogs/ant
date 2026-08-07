package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/repository"
)

// ── Mock repos ──

type mockCreditRepo struct {
	balance      decimal.Decimal
	holdErr      error
	settleErr    error
	holdAmount   decimal.Decimal
	settleHold   decimal.Decimal
	settleActual decimal.Decimal
}

func (m *mockCreditRepo) GetOrCreateAccount(ctx context.Context, userID uuid.UUID) (*repository.CreditAccount, error) {
	return &repository.CreditAccount{Balance: m.balance.StringFixed(8)}, nil
}
func (m *mockCreditRepo) GetBalance(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	return m.balance, nil
}
func (m *mockCreditRepo) AddCredits(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, txType, source, description string, operatorID *uuid.UUID) (*repository.CreditTransaction, error) {
	m.balance = m.balance.Add(amount)
	return &repository.CreditTransaction{Amount: amount.StringFixed(8), BalanceAfter: m.balance.StringFixed(8)}, nil
}
func (m *mockCreditRepo) HoldCredits(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, sessionID, description string) (*repository.CreditTransaction, error) {
	if m.holdErr != nil {
		return nil, m.holdErr
	}
	m.holdAmount = amount
	return &repository.CreditTransaction{TxType: "ai_hold", Amount: amount.StringFixed(8)}, nil
}
func (m *mockCreditRepo) SettleCredits(ctx context.Context, userID uuid.UUID, holdAmount, actualCost decimal.Decimal, description string) error {
	if m.settleErr != nil {
		return m.settleErr
	}
	m.settleHold = holdAmount
	m.settleActual = actualCost
	return nil
}
func (m *mockCreditRepo) ListTransactions(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*repository.CreditTransaction, int64, error) {
	return nil, 0, nil
}
func (m *mockCreditRepo) GetStaleHolds(ctx context.Context) ([]repository.StaleHold, error) {
	return nil, nil
}
func (m *mockCreditRepo) MarkHoldSettled(ctx context.Context, txID uuid.UUID) error {
	return nil
}

type mockModelRepo struct {
	model *repository.AIModel
	err   error
}

func (m *mockModelRepo) GetByProviderAndModel(ctx context.Context, providerID, modelName string) (*repository.AIModel, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.model, nil
}

// ── Tests ──

func TestCreditService_PreHold_Success(t *testing.T) {
	repo := &mockCreditRepo{balance: decimal.NewFromInt(1000)}
	models := &mockModelRepo{model: &repository.AIModel{
		PricePer1MInput:  "0.15000000",
		PricePer1MOutput: "0.60000000",
		MarkupRate:       "1.5",
		ModelTier:        "flagship",
	}}
	svc := NewCreditService(repo, models, zap.NewNop())
	uid := uuid.New()
	err := svc.PreHold(context.Background(), uid, "session-1", "openai", "gpt-4o")
	if err != nil {
		t.Fatalf("PreHold failed: %v", err)
	}
	if !repo.holdAmount.GreaterThan(decimal.Zero) {
		t.Fatal("hold amount should be positive")
	}
}

func TestCreditService_PreHold_ZeroBalance_SkipsHold(t *testing.T) {
	repo := &mockCreditRepo{balance: decimal.Zero, holdErr: fmt.Errorf("should not be called")}
	models := &mockModelRepo{model: &repository.AIModel{ModelTier: "flagship"}}
	svc := NewCreditService(repo, models, zap.NewNop())
	uid := uuid.New()
	err := svc.PreHold(context.Background(), uid, "session-1", "openai", "gpt-4o")
	if err != nil {
		t.Fatalf("PreHold should skip for zero balance, got: %v", err)
	}
	if repo.holdAmount.GreaterThan(decimal.Zero) {
		t.Fatal("should not hold credits for zero-balance user")
	}
}

func TestCreditService_PreHold_HoldError(t *testing.T) {
	repo := &mockCreditRepo{balance: decimal.NewFromInt(1000), holdErr: fmt.Errorf("insufficient balance")}
	models := &mockModelRepo{model: &repository.AIModel{ModelTier: "flagship"}}
	svc := NewCreditService(repo, models, zap.NewNop())
	uid := uuid.New()
	err := svc.PreHold(context.Background(), uid, "session-1", "openai", "gpt-4o")
	if err == nil {
		t.Fatal("should error on hold failure")
	}
}

func TestCreditService_Settle_Success(t *testing.T) {
	repo := &mockCreditRepo{balance: decimal.NewFromInt(1000)}
	models := &mockModelRepo{model: &repository.AIModel{
		PricePer1MInput:  "0.15000000",
		PricePer1MOutput: "0.60000000",
		MarkupRate:       "1.5",
		ModelTier:        "flagship",
	}}
	svc := NewCreditService(repo, models, zap.NewNop())
	uid := uuid.New()

	// Pre-hold first
	_ = svc.PreHold(context.Background(), uid, "session-2", "openai", "gpt-4o")

	// Settle with some token usage
	err := svc.Settle(context.Background(), uid, "session-2", "openai", "gpt-4o", 50000, 10000)
	if err != nil {
		t.Fatalf("Settle failed: %v", err)
	}
	if !repo.settleHold.GreaterThan(decimal.Zero) {
		t.Fatal("settle hold should be positive")
	}
}

func TestCreditService_Settle_NoPriorHold(t *testing.T) {
	repo := &mockCreditRepo{balance: decimal.NewFromInt(1000)}
	models := &mockModelRepo{model: &repository.AIModel{
		PricePer1MInput:  "0.15000000",
		PricePer1MOutput: "0.60000000",
		MarkupRate:       "1.5",
	}}
	svc := NewCreditService(repo, models, zap.NewNop())
	uid := uuid.New()

	// Settle without prior hold — should do direct deduction
	err := svc.Settle(context.Background(), uid, "session-no-hold", "openai", "gpt-4o", 50000, 10000)
	if err != nil {
		t.Fatalf("Settle without hold failed: %v", err)
	}
}

func TestCreditService_ReleaseHold_Success(t *testing.T) {
	repo := &mockCreditRepo{balance: decimal.NewFromInt(1000)}
	models := &mockModelRepo{model: &repository.AIModel{ModelTier: "flagship"}}
	svc := NewCreditService(repo, models, zap.NewNop())
	uid := uuid.New()

	_ = svc.PreHold(context.Background(), uid, "session-3", "openai", "gpt-4o")
	err := svc.ReleaseHold(context.Background(), uid, "session-3")
	if err != nil {
		t.Fatalf("ReleaseHold failed: %v", err)
	}
	if !repo.settleActual.Equals(decimal.Zero) {
		t.Fatal("settle actual should be zero on release")
	}
}

func TestCreditService_ReleaseHold_NoHold(t *testing.T) {
	repo := &mockCreditRepo{}
	models := &mockModelRepo{}
	svc := NewCreditService(repo, models, zap.NewNop())
	uid := uuid.New()
	err := svc.ReleaseHold(context.Background(), uid, "nonexistent-session")
	if err != nil {
		t.Fatalf("ReleaseHold on nonexistent session should not error: %v", err)
	}
}

func TestCreditService_CheckBalance_Sufficient(t *testing.T) {
	repo := &mockCreditRepo{balance: decimal.NewFromInt(500)}
	models := &mockModelRepo{}
	svc := NewCreditService(repo, models, zap.NewNop())
	uid := uuid.New()
	err := svc.CheckBalance(context.Background(), uid, decimal.NewFromInt(100))
	if err != nil {
		t.Fatalf("should pass with sufficient balance: %v", err)
	}
}

func TestCreditService_CheckBalance_Insufficient(t *testing.T) {
	repo := &mockCreditRepo{balance: decimal.NewFromInt(50)}
	models := &mockModelRepo{}
	svc := NewCreditService(repo, models, zap.NewNop())
	uid := uuid.New()
	err := svc.CheckBalance(context.Background(), uid, decimal.NewFromInt(100))
	if err == nil {
		t.Fatal("should error on insufficient balance")
	}
}

func TestCreditService_GetMarkupRate_FromDB(t *testing.T) {
	models := &mockModelRepo{model: &repository.AIModel{
		MarkupRate: "2.0",
		ModelTier:  "lightweight",
	}}
	svc := NewCreditService(&mockCreditRepo{}, models, zap.NewNop())
	rate := svc.getMarkupRate(context.Background(), "deepseek", "deepseek-chat")
	if !rate.Equals(decimal.NewFromFloat(2.0)) {
		t.Fatalf("markup rate should be 2.0 from DB, got %s", rate.String())
	}
}

func TestCreditService_GetMarkupRate_Default(t *testing.T) {
	models := &mockModelRepo{err: fmt.Errorf("not found")}
	svc := NewCreditService(&mockCreditRepo{}, models, zap.NewNop())
	rate := svc.getMarkupRate(context.Background(), "unknown", "unknown")
	if !rate.Equals(decimal.NewFromFloat(1.5)) {
		t.Fatalf("default markup rate should be 1.5, got %s", rate.String())
	}
}

func TestCreditService_GetMarkupRate_LightweightDefault(t *testing.T) {
	models := &mockModelRepo{model: &repository.AIModel{ModelTier: "lightweight"}}
	svc := NewCreditService(&mockCreditRepo{}, models, zap.NewNop())
	rate := svc.getMarkupRate(context.Background(), "deepseek", "deepseek-chat")
	if !rate.Equals(decimal.NewFromFloat(2.5)) {
		t.Fatalf("lightweight default rate should be 2.5, got %s", rate.String())
	}
}

func TestCreditService_GetModelTier_FromDB(t *testing.T) {
	models := &mockModelRepo{model: &repository.AIModel{ModelTier: "lightweight"}}
	svc := NewCreditService(&mockCreditRepo{}, models, zap.NewNop())
	tier := svc.getModelTier(context.Background(), "deepseek", "deepseek-chat")
	if tier != "lightweight" {
		t.Fatalf("tier should be lightweight, got %s", tier)
	}
}

func TestCreditService_GetModelTier_Default(t *testing.T) {
	models := &mockModelRepo{err: fmt.Errorf("not found")}
	svc := NewCreditService(&mockCreditRepo{}, models, zap.NewNop())
	tier := svc.getModelTier(context.Background(), "unknown", "unknown")
	if tier != "flagship" {
		t.Fatalf("default tier should be flagship, got %s", tier)
	}
}

func TestCreditService_ComputeCost_WithModel(t *testing.T) {
	models := &mockModelRepo{model: &repository.AIModel{
		PricePer1MInput:  "0.15000000",
		PricePer1MOutput: "0.60000000",
	}}
	svc := NewCreditService(&mockCreditRepo{}, models, zap.NewNop())
	cost := svc.computeCost(context.Background(), "openai", "gpt-4o", 1_000_000, 500_000)
	// input: 1M * 0.15 = 0.15; output: 500K * 0.60 = 0.30; total = 0.45
	expected := decimal.NewFromFloat(0.45)
	if !cost.Equals(expected) {
		t.Fatalf("cost should be 0.45, got %s", cost.String())
	}
}

func TestCreditService_ComputeCost_NoModel(t *testing.T) {
	models := &mockModelRepo{err: fmt.Errorf("not found")}
	svc := NewCreditService(&mockCreditRepo{}, models, zap.NewNop())
	cost := svc.computeCost(context.Background(), "unknown", "unknown", 1000, 500)
	if !cost.Equals(decimal.Zero) {
		t.Fatalf("cost should be 0 for unknown model, got %s", cost.String())
	}
}

func TestCreditService_EstimateSessionCost_Flagship(t *testing.T) {
	models := &mockModelRepo{model: &repository.AIModel{ModelTier: "flagship"}}
	svc := NewCreditService(&mockCreditRepo{}, models, zap.NewNop())
	cost := svc.estimateSessionCost(context.Background(), "openai", "gpt-4o")
	if !cost.Equals(decimal.NewFromFloat(0.15)) {
		t.Fatalf("flagship estimate should be 0.15, got %s", cost.String())
	}
}

func TestCreditService_EstimateSessionCost_Lightweight(t *testing.T) {
	models := &mockModelRepo{model: &repository.AIModel{ModelTier: "lightweight"}}
	svc := NewCreditService(&mockCreditRepo{}, models, zap.NewNop())
	cost := svc.estimateSessionCost(context.Background(), "deepseek", "deepseek-chat")
	if !cost.Equals(decimal.NewFromFloat(0.03)) {
		t.Fatalf("lightweight estimate should be 0.03, got %s", cost.String())
	}
}
