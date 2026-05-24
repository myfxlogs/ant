// Package service provides the ant v2 business service layer.
//
// Architecture (ADR-0006): platform-shared tables ∪ user-private tables.
// List methods merge both sources; detail methods route by ID to the correct table.
package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PlatformService provides read access to platform + user strategy/factor/ai data.
type PlatformService struct {
	pg *pgxpool.Pool
}

// NewPlatformService creates a platform service backed by the given pool.
func NewPlatformService(pg *pgxpool.Pool) *PlatformService {
	return &PlatformService{pg: pg}
}

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
			return nil, err
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
			return nil, err
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

// AccountDTO is a lightweight account view for the frontend.
type AccountDTO struct {
	ID, UserID, Platform, Broker, Login, Server string
	IsDisabled                                   bool
}

// ListAccounts returns all accounts.
func (s *PlatformService) ListAccounts(ctx context.Context) ([]AccountDTO, error) {
	rows, err := s.pg.Query(ctx, `
		SELECT id, user_id, mt_type, broker_company, login, broker_server, is_disabled
		FROM mt_accounts ORDER BY mt_type, login
	`)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []AccountDTO
	for rows.Next() {
		var a AccountDTO
		if err := rows.Scan(&a.ID, &a.UserID, &a.Platform, &a.Broker, &a.Login, &a.Server, &a.IsDisabled); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetAccount returns a single account by ID.
func (s *PlatformService) GetAccount(ctx context.Context, accountID string) (*AccountDTO, error) {
	var a AccountDTO
	err := s.pg.QueryRow(ctx, `
		SELECT id, user_id, mt_type, broker_company, login, broker_server, is_disabled
		FROM mt_accounts WHERE id = $1::uuid
	`, accountID).Scan(&a.ID, &a.UserID, &a.Platform, &a.Broker, &a.Login, &a.Server, &a.IsDisabled)
	if err != nil { return nil, err }
	return &a, nil
}

// IsAdmin checks if a user has admin privileges.
func (s *PlatformService) IsAdmin(ctx context.Context, userID string) (bool, error) {
	var count int
	err := s.pg.QueryRow(ctx, "SELECT count(*) FROM admins WHERE user_id = $1", userID).Scan(&count)
	return count > 0, err
}
