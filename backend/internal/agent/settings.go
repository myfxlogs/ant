package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ManagedConfig holds structured admin-managed settings (ADR-0025 §5.2).
// Stored as individual key-value rows in agent_managed_settings, parsed into this struct.
type ManagedConfig struct {
	AllowedModels         []string // "allowed_models" — comma-separated in DB
	EnforceAllowedModels  bool     // "enforce_allowed_models"
	MaxCostCeilingUSD     float64  // "max_cost_ceiling_usd"
	MaxIterations         int      // "max_iterations_per_strategy"
	DisableLiveTrading    bool     // "disable_live_trading"
	RequiredRiskGates     []string // "required_risk_gates" — comma-separated
	AuditRetentionDays    int      // "audit_retention_days"
	AllowManagedRulesOnly bool     // "allow_managed_rules_only"
}

// ResolvedSettings is the fully resolved settings for a user, including parsed managed config.
type ResolvedSettings struct {
	Flat    map[string]string // all key-value settings (for prompt injection)
	Tiers   map[string]string // key -> tier source ("default" | "user" | "managed")
	Managed ManagedConfig     // parsed structured managed config
	Loaded  bool              // true if settings loaded successfully
}

// SettingTier represents the priority level of a setting (ADR-0025 §5).
// Higher priority tiers override lower ones.
type SettingTier int

const (
	TierDefault SettingTier = 0   // built-in defaults
	TierUser    SettingTier = 10  // per-user override
	TierManaged SettingTier = 100 // admin-managed, cannot be overridden by users
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
	"agent.memory.enabled":      strTrue,
	"agent.retrospect.enabled":  strTrue,
	"agent.capability.can_live": "false",
}

// defaultManagedConfig is the fail-closed default when managed settings can't be loaded.
// ADR-0025 §5.2: "解析失败 = fail-closed: disable_live_trading 强制为 true,
// allowed_models 强制为空白名单"
var defaultManagedConfigFailClosed = ManagedConfig{
	AllowedModels:         nil, // empty whitelist
	EnforceAllowedModels:  true,
	MaxCostCeilingUSD:     0.01, // minimal cost
	MaxIterations:         3,
	DisableLiveTrading:    true, // fail-closed
	RequiredRiskGates:     []string{"lookahead", "walkforward", "paper"},
	AuditRetentionDays:    365,
	AllowManagedRulesOnly: true,
}

// ResolveSettings resolves all settings tiers and parses managed config (ADR-0025 §5.2).
// On managed settings load failure, returns fail-closed config.
func (s *SettingsStore) ResolveSettings(ctx context.Context, userID uuid.UUID) (*ResolvedSettings, error) {
	result := make(map[string]string)
	tiers := make(map[string]string)

	// Tier 0: Defaults
	for k, v := range defaultSettings {
		result[k] = v
		tiers[k] = "default"
	}

	// Tier 1: User overrides
	if s.pool != nil {
		userSettings, err := s.loadUserSettings(ctx, userID)
		if err == nil {
			for k, v := range userSettings {
				result[k] = v
				tiers[k] = "user"
			}
		}
	}

	// Tier 2: Managed (admin) overrides — highest priority
	var managed ManagedConfig
	loaded := false
	if s.pool != nil {
		managedSettings, err := s.loadManagedSettings(ctx)
		if err == nil {
			for k, v := range managedSettings {
				result[k] = v
				tiers[k] = "managed"
			}
			managed = parseManagedConfig(managedSettings)
			loaded = true
		}
	}

	// Fail-closed: if managed settings failed to load, use fail-closed defaults
	if !loaded {
		managed = defaultManagedConfigFailClosed
		// Apply fail-closed overrides to flat map
		result["agent.capability.can_live"] = "false"
		tiers["agent.capability.can_live"] = "managed"
		result["agent.cost_ceiling_usd"] = fmt.Sprintf("%.2f", managed.MaxCostCeilingUSD)
		tiers["agent.cost_ceiling_usd"] = "managed"
	}

	return &ResolvedSettings{
		Flat:    result,
		Tiers:   tiers,
		Managed: managed,
		Loaded:  loaded,
	}, nil
}

// parseManagedConfig parses flat key-value managed settings into structured config.
func parseManagedConfig(m map[string]string) ManagedConfig {
	cfg := ManagedConfig{
		MaxCostCeilingUSD:  0.50,
		MaxIterations:      50,
		AuditRetentionDays: 365,
	}
	if v, ok := m["allowed_models"]; ok {
		cfg.AllowedModels = splitCSV(v)
	}
	if v, ok := m["enforce_allowed_models"]; ok {
		cfg.EnforceAllowedModels = v == strTrue
	}
	if v, ok := m["max_cost_ceiling_usd"]; ok {
		if f, err := parseFloat(v); err == nil {
			cfg.MaxCostCeilingUSD = f
		}
	}
	if v, ok := m["max_iterations_per_strategy"]; ok {
		if n, err := parseInt(v); err == nil {
			cfg.MaxIterations = n
		}
	}
	if v, ok := m["disable_live_trading"]; ok {
		cfg.DisableLiveTrading = v == strTrue
	}
	if v, ok := m["required_risk_gates"]; ok {
		cfg.RequiredRiskGates = splitCSV(v)
	}
	if v, ok := m["audit_retention_days"]; ok {
		if n, err := parseInt(v); err == nil {
			cfg.AuditRetentionDays = n
		}
	}
	if v, ok := m["allow_managed_rules_only"]; ok {
		cfg.AllowManagedRulesOnly = v == strTrue
	}
	return cfg
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
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

// GetManagedSetting returns the value of a single managed setting key, or empty string if not set.
func (s *SettingsStore) GetManagedSetting(ctx context.Context, key string) (string, error) {
	if s.pool == nil {
		return "", nil
	}
	var val string
	err := s.pool.QueryRow(ctx,
		`SELECT value FROM agent_managed_settings WHERE key = $1`, key).Scan(&val)
	if err != nil {
		return "", nil // not found = use default
	}
	return val, nil
}
