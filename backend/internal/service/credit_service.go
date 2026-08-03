package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/repository"
)

// CreditPerDollar = 100 (1 credit = $0.01)
var CreditPerDollar = decimal.NewFromInt(100)

// CreditRepoInterface is the interface for credit account/transaction operations.
// Implemented by *repository.CreditRepository.
type CreditRepoInterface interface {
	GetOrCreateAccount(ctx context.Context, userID uuid.UUID) (*repository.CreditAccount, error)
	GetBalance(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error)
	AddCredits(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, txType, source, description string, operatorID *uuid.UUID) (*repository.CreditTransaction, error)
	HoldCredits(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, description string) (*repository.CreditTransaction, error)
	SettleCredits(ctx context.Context, userID uuid.UUID, holdAmount, actualCost decimal.Decimal, description string) error
	ListTransactions(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*repository.CreditTransaction, int64, error)
}

// CreditModelRepo is the interface for AI model pricing lookups.
type CreditModelRepo interface {
	GetByProviderAndModel(ctx context.Context, providerID, modelName string) (*repository.AIModel, error)
}

// CreditService manages credit-based AI billing with pre-hold and settlement.
type CreditService struct {
	repo   CreditRepoInterface
	models CreditModelRepo
	log    *zap.Logger

	// pendingHolds tracks active pre-holds per session for settlement.
	mu   sync.Mutex
	holds map[string]decimal.Decimal // sessionID → holdAmount
}

func NewCreditService(repo CreditRepoInterface, models CreditModelRepo, log *zap.Logger) *CreditService {
	return &CreditService{
		repo:   repo,
		models: models,
		log:    log,
		holds:  make(map[string]decimal.Decimal),
	}
}

// PreHold freezes credits at session start based on model P90 estimated cost.
// estimatedCostUSD is the conservative pre-hold amount in USD (vendor cost).
// The actual hold amount in credits = estimatedCostUSD × markupRate × CreditPerDollar.
func (s *CreditService) PreHold(ctx context.Context, userID uuid.UUID, sessionID string, providerID, modelName string) error {
	estimatedCostUSD := s.estimateSessionCost(ctx, providerID, modelName)
	markupRate := s.getMarkupRate(ctx, providerID, modelName)
	holdCredits := estimatedCostUSD.Mul(markupRate).Mul(CreditPerDollar)

	if holdCredits.LessThanOrEqual(decimal.Zero) {
		return nil // no hold needed for free/BYO models
	}

	_, err := s.repo.HoldCredits(ctx, userID, holdCredits, fmt.Sprintf("AI session %s", sessionID))
	if err != nil {
		return fmt.Errorf("credit pre-hold failed: %w", err)
	}

	s.mu.Lock()
	s.holds[sessionID] = holdCredits
	s.mu.Unlock()

	s.log.Info("credit pre-hold",
		zap.String("user_id", userID.String()),
		zap.String("session_id", sessionID),
		zap.String("hold_credits", holdCredits.StringFixed(2)))
	return nil
}

// Settle finalizes billing after a session: deducts actual cost, releases unused hold.
// actualCostUSD is the real vendor cost for the session.
func (s *CreditService) Settle(ctx context.Context, userID uuid.UUID, sessionID, providerID, modelName string, inputTokens, outputTokens int) error {
	actualCostUSD := s.computeCost(ctx, providerID, modelName, inputTokens, outputTokens)
	markupRate := s.getMarkupRate(ctx, providerID, modelName)
	actualCredits := actualCostUSD.Mul(markupRate).Mul(CreditPerDollar)

	s.mu.Lock()
	holdAmount := s.holds[sessionID]
	delete(s.holds, sessionID)
	s.mu.Unlock()

	if holdAmount.Equal(decimal.Zero) {
		// No prior hold — direct deduction.
		if actualCredits.LessThanOrEqual(decimal.Zero) {
			return nil
		}
		_, err := s.repo.AddCredits(ctx, userID, actualCredits.Neg(), "ai_usage", "direct_deduction", fmt.Sprintf("AI session %s", sessionID), nil)
		return err
	}

	if err := s.repo.SettleCredits(ctx, userID, holdAmount, actualCredits, fmt.Sprintf("AI session %s (%s)", sessionID, modelName)); err != nil {
		return fmt.Errorf("credit settlement failed: %w", err)
	}

	s.log.Info("credit settled",
		zap.String("user_id", userID.String()),
		zap.String("session_id", sessionID),
		zap.String("hold_credits", holdAmount.StringFixed(2)),
		zap.String("actual_credits", actualCredits.StringFixed(2)))
	return nil
}

// CheckBalance returns whether the user has enough credits for a minimum hold.
func (s *CreditService) CheckBalance(ctx context.Context, userID uuid.UUID, minCredits decimal.Decimal) error {
	bal, err := s.repo.GetBalance(ctx, userID)
	if err != nil {
		return nil // fail-open on DB errors
	}
	if bal.LessThan(minCredits) {
		return fmt.Errorf("insufficient credits: have %s, need %s", bal.StringFixed(0), minCredits.StringFixed(0))
	}
	return nil
}

// estimateSessionCost returns a conservative P90 estimate for a single session.
// Cold-start: uses fixed estimates per model tier until historical data exists.
func (s *CreditService) estimateSessionCost(ctx context.Context, providerID, modelName string) decimal.Decimal {
	// Cold-start estimates (USD): flagship ~$0.15, lightweight ~$0.03
	tier := s.getModelTier(ctx, providerID, modelName)
	switch tier {
	case "lightweight":
		return decimal.NewFromFloat(0.03)
	default:
		return decimal.NewFromFloat(0.15)
	}
}

// computeCost calculates the actual vendor cost in USD for a call.
func (s *CreditService) computeCost(ctx context.Context, providerID, modelName string, inputTokens, outputTokens int) decimal.Decimal {
	m, err := s.models.GetByProviderAndModel(ctx, providerID, modelName)
	if err != nil || m == nil {
		return decimal.Zero
	}
	ip, err := decimal.NewFromString(m.PricePer1MInput)
	if err != nil {
		return decimal.Zero
	}
	op, err := decimal.NewFromString(m.PricePer1MOutput)
	if err != nil {
		return decimal.Zero
	}
	million := decimal.NewFromInt(1_000_000)
	inCost := decimal.NewFromInt(int64(inputTokens)).Div(million).Mul(ip)
	outCost := decimal.NewFromInt(int64(outputTokens)).Div(million).Mul(op)
	return inCost.Add(outCost)
}

// getMarkupRate returns the pricing multiplier for a model (1.5x flagship, 2.5x lightweight).
func (s *CreditService) getMarkupRate(ctx context.Context, providerID, modelName string) decimal.Decimal {
	m, err := s.models.GetByProviderAndModel(ctx, providerID, modelName)
	if err != nil || m == nil {
		return decimal.NewFromFloat(1.5) // default
	}
	if m.MarkupRate != "" {
		if rate, err := decimal.NewFromString(m.MarkupRate); err == nil && rate.IsPositive() {
			return rate
		}
	}
	tier := s.getModelTier(ctx, providerID, modelName)
	switch tier {
	case "lightweight":
		return decimal.NewFromFloat(2.5)
	default:
		return decimal.NewFromFloat(1.5)
	}
}

// getModelTier returns the model tier from the DB.
func (s *CreditService) getModelTier(ctx context.Context, providerID, modelName string) string {
	m, err := s.models.GetByProviderAndModel(ctx, providerID, modelName)
	if err != nil || m == nil {
		return "flagship"
	}
	if m.ModelTier != "" {
		return m.ModelTier
	}
	return "flagship"
}

// ReleaseHold cancels a pre-hold without settlement (e.g. session failed before any API call).
func (s *CreditService) ReleaseHold(ctx context.Context, userID uuid.UUID, sessionID string) error {
	s.mu.Lock()
	holdAmount := s.holds[sessionID]
	delete(s.holds, sessionID)
	s.mu.Unlock()

	if holdAmount.Equal(decimal.Zero) {
		return nil
	}

	// Release = settle with zero actual cost.
	if err := s.repo.SettleCredits(ctx, userID, holdAmount, decimal.Zero, fmt.Sprintf("AI session %s cancelled", sessionID)); err != nil {
		return fmt.Errorf("credit release failed: %w", err)
	}

	s.log.Info("credit hold released",
		zap.String("user_id", userID.String()),
		zap.String("session_id", sessionID),
		zap.String("released_credits", holdAmount.StringFixed(2)))
	return nil
}
