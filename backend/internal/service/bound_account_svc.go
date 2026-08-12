package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"alphaforge/internal/repository"
)

var (
	ErrAccountLimitExceeded = errors.New("exceeded tier account limit — upgrade your plan to bind more accounts")
	ErrAccountNotOwned      = errors.New("MT account not found or not owned by user")
)

// BoundAccountService enforces tier-based MT account binding limits.
// Used by CreateSchedule to ensure only bound accounts execute strategies.
type BoundAccountService struct {
	boundRepo *repository.BoundAccountRepository
	subRepo   *repository.SubscriptionRepository
	pg        *pgxpool.Pool
	log       *zap.Logger
}

func NewBoundAccountService(boundRepo *repository.BoundAccountRepository, subRepo *repository.SubscriptionRepository, pg *pgxpool.Pool, log *zap.Logger) *BoundAccountService {
	return &BoundAccountService{boundRepo: boundRepo, subRepo: subRepo, pg: pg, log: log}
}

// EnsureBoundAccount checks if the account is bound; if not, auto-binds it
// if the user's tier allows. Returns error if limit exceeded or account not owned.
func (s *BoundAccountService) EnsureBoundAccount(ctx context.Context, userID, accountID uuid.UUID) error {
	bound, err := s.boundRepo.IsAccountBound(ctx, userID, accountID)
	if err != nil {
		return fmt.Errorf("ensure bound: check bound: %w", err)
	}
	if bound {
		return nil
	}

	// Verify the account belongs to the user and is not deleted.
	var exists bool
	err = s.pg.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM mt_accounts WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL)`,
		accountID, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("ensure bound: check ownership: %w", err)
	}
	if !exists {
		return ErrAccountNotOwned
	}

	// Get the user's plan limit.
	limit, err := s.getAccountLimit(ctx, userID)
	if err != nil {
		return fmt.Errorf("ensure bound: get limit: %w", err)
	}

	// 0 = unlimited.
	if limit == 0 {
		return s.boundRepo.BindAccount(ctx, userID, accountID)
	}

	count, err := s.boundRepo.CountBoundAccounts(ctx, userID)
	if err != nil {
		return fmt.Errorf("ensure bound: count: %w", err)
	}

	if count >= limit {
		return ErrAccountLimitExceeded
	}

	return s.boundRepo.BindAccount(ctx, userID, accountID)
}

// ListBoundAccounts returns all bound MT accounts for a user.
func (s *BoundAccountService) ListBoundAccounts(ctx context.Context, userID uuid.UUID) ([]repository.BoundAccountRow, error) {
	return s.boundRepo.ListBoundAccounts(ctx, userID)
}

// UnbindAccount removes an MT account binding.
func (s *BoundAccountService) UnbindAccount(ctx context.Context, userID, accountID uuid.UUID) error {
	return s.boundRepo.UnbindAccount(ctx, userID, accountID)
}

// GetAccountLimit returns the user's tier MT account limit (0 = unlimited).
func (s *BoundAccountService) GetAccountLimit(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.getAccountLimit(ctx, userID)
}

func (s *BoundAccountService) getAccountLimit(ctx context.Context, userID uuid.UUID) (int, error) {
	sub, err := s.subRepo.GetActiveSubscription(ctx, userID)
	if err != nil {
		return 0, err
	}

	var planID uuid.UUID
	if sub != nil {
		planID = sub.PlanID
	} else {
		// No active subscription — use free plan.
		plan, err := s.subRepo.GetPlanByName(ctx, "free")
		if err != nil {
			return 0, err
		}
		if plan == nil {
			return 1, nil // safe default: free = 1
		}
		planID = plan.ID
	}

	plan, err := s.subRepo.GetPlanByID(ctx, planID)
	if err != nil {
		return 0, err
	}
	if plan == nil {
		return 1, nil
	}
	return plan.MaxMTAccounts, nil
}
