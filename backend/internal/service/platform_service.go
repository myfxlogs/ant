// Package service provides the ant v2 business service layer.
//
// Architecture (ADR-0006): platform-shared tables ∪ user-private tables.
// List methods merge both sources; detail methods route by ID to the correct table.
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// PlatformService provides read access to platform + user strategy/factor/ai data.
type PlatformService struct {
	pg         *pgxpool.Pool
	log        *zap.Logger
	accountSvc *AccountService
}

// NewPlatformService creates a platform service backed by the given pool.
func NewPlatformService(pg *pgxpool.Pool, accountSvc *AccountService) *PlatformService {
	return &PlatformService{pg: pg, accountSvc: accountSvc}
}

// SetLogger injects a logger into the service.
func (s *PlatformService) SetLogger(log *zap.Logger) { s.log = log }

// Strategy represents a strategy visible to a user (platform ∪ user-private).
type Strategy struct {
	ID          string
	Name        string
	Description string
	Expression  string
	Category    string
	Source      string // "platform" or "user"
}

// ListStrategies returns all official platform strategies plus the user's own.
func (s *PlatformService) ListStrategies(ctx context.Context, userID string) ([]Strategy, error) {
	rows, err := s.pg.Query(ctx, `
			SELECT ps.id, ps.name, ps.description, ps.expression, ps.category, 'platform' as source
			FROM platform_strategies ps
			WHERE ps.is_official = true
			UNION ALL
			SELECT usp.strategy_id, st.name, st.description, st.expression, st.category, 'user' as source
			FROM user_strategy_publishes usp
			JOIN strategy_templates st ON st.id::text = usp.strategy_id
			WHERE usp.publisher_user_id = $1
			ORDER BY name
		`, userID)
	if err != nil {
		return nil, fmt.Errorf("service: list strategies: %w", err)
	}
	defer rows.Close()

	var out []Strategy
	for rows.Next() {
		var s Strategy
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Expression, &s.Category, &s.Source); err != nil {
			return nil, fmt.Errorf("service: list strategies: scan: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// UserSubscription represents a user's subscription to a platform strategy.
type UserSubscription struct {
	ID               string
	TargetUserID     string
	TargetStrategyID string
	Kind             string // "copy_trade" | "signal" | "follow"
}

// ListSubscriptions returns all active subscriptions for a user.
func (s *PlatformService) ListSubscriptions(ctx context.Context, userID string) ([]UserSubscription, error) {
	rows, err := s.pg.Query(ctx, `
			SELECT id, target_user_id, target_strategy_id, kind
			FROM user_subscriptions
			WHERE subscriber_user_id = $1 AND active = true
			ORDER BY created_at DESC
		`, userID)
	if err != nil {
		return nil, fmt.Errorf("service: list subscriptions: %w", err)
	}
	defer rows.Close()

	var out []UserSubscription
	for rows.Next() {
		var sub UserSubscription
		if err := rows.Scan(&sub.ID, &sub.TargetUserID, &sub.TargetStrategyID, &sub.Kind); err != nil {
			return nil, fmt.Errorf("service: list subscriptions: scan: %w", err)
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

// Admin represents a platform administrator.
type Admin struct {
	UserID string
	Scope  string // "platform:admin"
}

// IsAdmin checks whether a user is a platform administrator.
func (s *PlatformService) IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	var count int
	err := s.pg.QueryRow(ctx, "SELECT count(*) FROM admins WHERE user_id = $1", userID).Scan(&count)
	return count > 0, err
}

// --- Account method delegation (for callers in system/ that go through PlatformService) ---

// UserOwnsAccount checks if an account belongs to the given user.
func (s *PlatformService) UserOwnsAccount(ctx context.Context, userID, accountID string) (bool, error) {
	return s.accountSvc.UserOwnsAccount(ctx, userID, accountID)
}

// GetAccount returns a single account by ID.
func (s *PlatformService) GetAccount(ctx context.Context, userID uuid.UUID, accountID string) (*AccountDTO, error) {
	return s.accountSvc.GetAccount(ctx, userID, accountID)
}

// GetUserAccountIDs returns all account IDs belonging to a user.
func (s *PlatformService) GetUserAccountIDs(ctx context.Context, userID string) ([]string, error) {
	return s.accountSvc.GetUserAccountIDs(ctx, userID)
}

// GetUserAccountSnapshots returns current state for all of a user's MT accounts.
func (s *PlatformService) GetUserAccountSnapshots(ctx context.Context, userID string) ([]AccountSnapshot, error) {
	return s.accountSvc.GetUserAccountSnapshots(ctx, userID)
}

// GetUserAccountsSummary computes an aggregated summary of a user's accounts.
func (s *PlatformService) GetUserAccountsSummary(ctx context.Context, userID string) (*UserAccountsSummary, error) {
	return s.accountSvc.GetUserAccountsSummary(ctx, userID)
}
