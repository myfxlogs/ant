package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SettingTier represents the priority level of a setting (ADR-0025 §5).
// Higher priority tiers override lower ones.
type SettingTier int

const (
	TierDefault  SettingTier = 0 // built-in defaults
	TierUser     SettingTier = 10 // per-user override
	TierManaged  SettingTier = 100 // admin-managed, cannot be overridden by users
)

// AgentSetting is a single setting key-value with its tier.
type AgentSetting struct {
	Key   string
	Value string
	Tier  SettingTier
}

// SettingsStore manages tiered agent settings (ADR-0025 §5).
// Priority: Managed > User > Default.
// Tenant tier is not yet implemented (no tenant system).
type SettingsStore struct {
	pool *pgxpool.Pool
}

// NewSettingsStore creates a settings store.
func NewSettingsStore(pool *pgxpool.Pool) *SettingsStore {
	return &SettingsStore{pool: pool}
}

// defaultSettings are the built-in defaults for agent settings.
var defaultSettings = map[string]string{
	"agent.model":               "auto",
	"agent.max_retries":         "3",
	"agent.cost_ceiling_usd":    "0.05",
	"agent.plan_mode":           "plan",
	"agent.memory.enabled":      "true",
	"agent.retrospect.enabled":  "true",
	"agent.capability.can_live": "false",
}

// GetSettings resolves settings for a user by merging tiers (ADR-0025 §5).
// Managed settings override User settings, which override Defaults.
func (s *SettingsStore) GetSettings(ctx context.Context, userID uuid.UUID) (map[string]string, error) {
	result := make(map[string]string)

	// Tier 0: Defaults
	for k, v := range defaultSettings {
		result[k] = v
	}

	// Tier 1: User overrides
	if s.pool != nil {
		userSettings, err := s.loadUserSettings(ctx, userID)
		if err == nil {
			for k, v := range userSettings {
				result[k] = v
			}
		}
	}

	// Tier 2: Managed (admin) overrides — highest priority
	if s.pool != nil {
		managedSettings, err := s.loadManagedSettings(ctx)
		if err == nil {
			for k, v := range managedSettings {
				result[k] = v
			}
		}
	}

	return result, nil
}

// SetUserSetting sets a user-level setting override.
func (s *SettingsStore) SetUserSetting(ctx context.Context, userID uuid.UUID, key, value string) error {
	if s.pool == nil {
		return fmt.Errorf("settings store not configured")
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO agent_user_settings (user_id, key, value)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, key) DO UPDATE SET value = $3, updated_at = NOW()`,
		userID, key, value)
	return err
}

// SetManagedSetting sets an admin-managed setting (highest priority).
func (s *SettingsStore) SetManagedSetting(ctx context.Context, key, value string) error {
	if s.pool == nil {
		return fmt.Errorf("settings store not configured")
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO agent_managed_settings (key, value)
		 VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()`,
		key, value)
	return err
}

// DeleteUserSetting removes a user-level override.
func (s *SettingsStore) DeleteUserSetting(ctx context.Context, userID uuid.UUID, key string) error {
	if s.pool == nil {
		return fmt.Errorf("settings store not configured")
	}
	_, err := s.pool.Exec(ctx,
		`DELETE FROM agent_user_settings WHERE user_id = $1 AND key = $2`,
		userID, key)
	return err
}

// DeleteManagedSetting removes an admin-managed override.
func (s *SettingsStore) DeleteManagedSetting(ctx context.Context, key string) error {
	if s.pool == nil {
		return fmt.Errorf("settings store not configured")
	}
	_, err := s.pool.Exec(ctx,
		`DELETE FROM agent_managed_settings WHERE key = $1`,
		key)
	return err
}

func (s *SettingsStore) loadUserSettings(ctx context.Context, userID uuid.UUID) (map[string]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT key, value FROM agent_user_settings WHERE user_id = $1`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			continue
		}
		result[k] = v
	}
	return result, nil
}

func (s *SettingsStore) loadManagedSettings(ctx context.Context) (map[string]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT key, value FROM agent_managed_settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			continue
		}
		result[k] = v
	}
	return result, nil
}

// ListManagedSettings returns all managed settings for admin UI.
func (s *SettingsStore) ListManagedSettings(ctx context.Context) ([]AgentSetting, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("settings store not configured")
	}
	rows, err := s.pool.Query(ctx,
		`SELECT key, value FROM agent_managed_settings ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []AgentSetting
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			continue
		}
		result = append(result, AgentSetting{Key: k, Value: v, Tier: TierManaged})
	}
	return result, nil
}

// ListUserSettings returns all user-level settings for a user.
func (s *SettingsStore) ListUserSettings(ctx context.Context, userID uuid.UUID) ([]AgentSetting, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("settings store not configured")
	}
	rows, err := s.pool.Query(ctx,
		`SELECT key, value FROM agent_user_settings WHERE user_id = $1 ORDER BY key`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []AgentSetting
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			continue
		}
		result = append(result, AgentSetting{Key: k, Value: v, Tier: TierUser})
	}
	return result, nil
}

// FormatSettings formats resolved settings as a string for prompt injection.
func FormatSettings(settings map[string]string) string {
	if len(settings) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n## Agent Settings\n")
	for k, v := range settings {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
	}
	return sb.String()
}
