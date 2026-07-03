//go:build integration

package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// adminE2ETestPG connects to the test database or skips the test.
func adminE2ETestPG(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return genE2ETestPG(t)
}

// adminE2ETestUser inserts a unique test user and returns its UUID.
func adminE2ETestUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	uid := uuid.New()
	email := fmt.Sprintf("agent-admin-%s@anttest.io", uid.String()[:8])
	ctx := context.Background()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
		 VALUES ($1, $2, '$argon2id$v=19$m=65536,t=3,p=2$test$test', 'user', 'active', NOW(), NOW())`,
		uid, email)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM agent_user_settings WHERE user_id = $1`, uid)
		pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, uid)
	})
	return uid
}

// cleanupManagedSetting removes a managed setting and registers cleanup.
func cleanupManagedSetting(t *testing.T, store *SettingsStore, ctx context.Context, key string) {
	t.Cleanup(func() {
		store.DeleteManagedSetting(ctx, key)
	})
}

// TestAdminE2E_ManagedSettings_CRUD verifies admin-managed settings CRUD operations.
func TestAdminE2E_ManagedSettings_CRUD(t *testing.T) {
	t.Parallel()
	pool := adminE2ETestPG(t)
	ctx := context.Background()
	store := NewSettingsStore(pool)

	// Set a managed setting
	err := store.SetManagedSetting(ctx, "max_cost_ceiling_usd", "0.50")
	if err != nil {
		t.Fatalf("SetManagedSetting: %v", err)
	}
	cleanupManagedSetting(t, store, ctx, "max_cost_ceiling_usd")

	// Resolve and verify
	rs, err := store.ResolveSettings(ctx, uuid.New())
	if err != nil {
		t.Fatalf("ResolveSettings: %v", err)
	}
	if !rs.Loaded {
		t.Fatal("expected settings to be loaded")
	}
	if rs.Flat["max_cost_ceiling_usd"] != "0.50" {
		t.Errorf("max_cost_ceiling_usd = %q, want %q", rs.Flat["max_cost_ceiling_usd"], "0.50")
	}
	if rs.Tiers["max_cost_ceiling_usd"] != "managed" {
		t.Errorf("tier = %q, want %q", rs.Tiers["max_cost_ceiling_usd"], "managed")
	}
	if rs.Managed.MaxCostCeilingUSD != 0.50 {
		t.Errorf("MaxCostCeilingUSD = %v, want 0.50", rs.Managed.MaxCostCeilingUSD)
	}

	// Update the managed setting
	err = store.SetManagedSetting(ctx, "max_cost_ceiling_usd", "1.00")
	if err != nil {
		t.Fatalf("SetManagedSetting update: %v", err)
	}
	rs, err = store.ResolveSettings(ctx, uuid.New())
	if err != nil {
		t.Fatalf("ResolveSettings after update: %v", err)
	}
	if rs.Managed.MaxCostCeilingUSD != 1.00 {
		t.Errorf("MaxCostCeilingUSD after update = %v, want 1.00", rs.Managed.MaxCostCeilingUSD)
	}

	// Delete the managed setting
	err = store.DeleteManagedSetting(ctx, "max_cost_ceiling_usd")
	if err != nil {
		t.Fatalf("DeleteManagedSetting: %v", err)
	}
	rs, err = store.ResolveSettings(ctx, uuid.New())
	if err != nil {
		t.Fatalf("ResolveSettings after delete: %v", err)
	}
	// After deletion, should fall back to default or fail-closed
	if rs.Managed.MaxCostCeilingUSD == 1.00 {
		t.Error("managed setting was not deleted")
	}
}

// TestAdminE2E_TieredResolution verifies the priority: managed > user > default.
func TestAdminE2E_TieredResolution(t *testing.T) {
	t.Parallel()
	pool := adminE2ETestPG(t)
	uid := adminE2ETestUser(t, pool)
	ctx := context.Background()
	store := NewSettingsStore(pool)

	// 1. Default only — agent.model should be "auto"
	rs, err := store.ResolveSettings(ctx, uid)
	if err != nil {
		t.Fatalf("ResolveSettings defaults: %v", err)
	}
	if rs.Flat["agent.model"] != "auto" {
		t.Errorf("default agent.model = %q, want %q", rs.Flat["agent.model"], "auto")
	}
	if rs.Tiers["agent.model"] != "default" {
		t.Errorf("default tier = %q, want %q", rs.Tiers["agent.model"], "default")
	}

	// 2. User override — agent.model = "gpt-4"
	err = store.SetUserSetting(ctx, uid, "agent.model", "gpt-4")
	if err != nil {
		t.Fatalf("SetUserSetting: %v", err)
	}
	rs, err = store.ResolveSettings(ctx, uid)
	if err != nil {
		t.Fatalf("ResolveSettings user override: %v", err)
	}
	if rs.Flat["agent.model"] != "gpt-4" {
		t.Errorf("user agent.model = %q, want %q", rs.Flat["agent.model"], "gpt-4")
	}
	if rs.Tiers["agent.model"] != "user" {
		t.Errorf("user tier = %q, want %q", rs.Tiers["agent.model"], "user")
	}

	// 3. Managed override — agent.model = "claude-3"
	err = store.SetManagedSetting(ctx, "agent.model", "claude-3")
	if err != nil {
		t.Fatalf("SetManagedSetting: %v", err)
	}
	cleanupManagedSetting(t, store, ctx, "agent.model")
	rs, err = store.ResolveSettings(ctx, uid)
	if err != nil {
		t.Fatalf("ResolveSettings managed override: %v", err)
	}
	if rs.Flat["agent.model"] != "claude-3" {
		t.Errorf("managed agent.model = %q, want %q", rs.Flat["agent.model"], "claude-3")
	}
	if rs.Tiers["agent.model"] != "managed" {
		t.Errorf("managed tier = %q, want %q", rs.Tiers["agent.model"], "managed")
	}

	// 4. Delete user setting — managed should still win
	err = store.DeleteUserSetting(ctx, uid, "agent.model")
	if err != nil {
		t.Fatalf("DeleteUserSetting: %v", err)
	}
	rs, err = store.ResolveSettings(ctx, uid)
	if err != nil {
		t.Fatalf("ResolveSettings after user delete: %v", err)
	}
	if rs.Flat["agent.model"] != "claude-3" {
		t.Errorf("managed agent.model after user delete = %q, want %q", rs.Flat["agent.model"], "claude-3")
	}

	// 5. Delete managed setting — should fall back to default
	store.DeleteManagedSetting(ctx, "agent.model")
	rs, err = store.ResolveSettings(ctx, uid)
	if err != nil {
		t.Fatalf("ResolveSettings after managed delete: %v", err)
	}
	if rs.Flat["agent.model"] != "auto" {
		t.Errorf("default agent.model after all deletes = %q, want %q", rs.Flat["agent.model"], "auto")
	}
	if rs.Tiers["agent.model"] != "default" {
		t.Errorf("default tier after all deletes = %q, want %q", rs.Tiers["agent.model"], "default")
	}
}

// TestAdminE2E_Permissions_Default verifies default permissions with no managed settings.
func TestAdminE2E_Permissions_Default(t *testing.T) {
	t.Parallel()
	pool := adminE2ETestPG(t)
	uid := adminE2ETestUser(t, pool)
	ctx := context.Background()
	store := NewSettingsStore(pool)
	engine := NewPermissionEngine(store)

	// With no managed settings, fail-closed config applies:
	// - Generate/Submit/ViewBacktest: allowed
	// - LiveDeploy: denied (fail-closed)
	// - Memory ops: allowed
	caps := engine.CapabilitiesForUser(ctx, uid)

	if !caps[CapGenerateStrategy] {
		t.Error("CapGenerateStrategy should be allowed by default")
	}
	if !caps[CapSubmitStrategy] {
		t.Error("CapSubmitStrategy should be allowed by default")
	}
	if !caps[CapViewBacktest] {
		t.Error("CapViewBacktest should be allowed by default")
	}
	if caps[CapLiveDeploy] {
		t.Error("CapLiveDeploy should be denied (fail-closed) when no managed settings")
	}
	if !caps[CapSearchExperience] {
		t.Error("CapSearchExperience should be allowed by default")
	}
	if !caps[CapStoreExperience] {
		t.Error("CapStoreExperience should be allowed by default")
	}
}

// TestAdminE2E_Permissions_ManagedRules verifies permission rules from managed settings.
func TestAdminE2E_Permissions_ManagedRules(t *testing.T) {
	t.Parallel()
	pool := adminE2ETestPG(t)
	uid := adminE2ETestUser(t, pool)
	ctx := context.Background()
	store := NewSettingsStore(pool)
	engine := NewPermissionEngine(store)

	// Set managed settings with permission rules
	// Rule 1: DENY live_deploy for all
	err := store.SetManagedSetting(ctx, "permission.rule.1", "DENY live_deploy(*:*)")
	if err != nil {
		t.Fatalf("SetManagedSetting rule 1: %v", err)
	}
	cleanupManagedSetting(t, store, ctx, "permission.rule.1")

	// Rule 2: ALLOW generate_strategy for all
	err = store.SetManagedSetting(ctx, "permission.rule.2", "ALLOW generate_strategy(*:*)")
	if err != nil {
		t.Fatalf("SetManagedSetting rule 2: %v", err)
	}
	cleanupManagedSetting(t, store, ctx, "permission.rule.2")

	// Rule 3: ALLOW live_deploy for specific resource "safe-account"
	err = store.SetManagedSetting(ctx, "permission.rule.3", "ALLOW live_deploy(safe-account:*)")
	if err != nil {
		t.Fatalf("SetManagedSetting rule 3: %v", err)
	}
	cleanupManagedSetting(t, store, ctx, "permission.rule.3")

	// Enable live trading in managed config
	err = store.SetManagedSetting(ctx, "disable_live_trading", "false")
	if err != nil {
		t.Fatalf("SetManagedSetting disable_live_trading: %v", err)
	}
	cleanupManagedSetting(t, store, ctx, "disable_live_trading")

	err = store.SetManagedSetting(ctx, "agent.capability.can_live", "true")
	if err != nil {
		t.Fatalf("SetManagedSetting can_live: %v", err)
	}
	cleanupManagedSetting(t, store, ctx, "agent.capability.can_live")

	// DENY should take priority over ALLOW for live_deploy
	canLive := engine.Can(ctx, uid, CapLiveDeploy)
	if canLive {
		t.Error("CapLiveDeploy should be denied — DENY rule takes priority")
	}

	// But for specific resource "safe-account", ALLOW should match
	// However, DENY(*:*) also matches "safe-account", so DENY still wins
	canLiveSafe := engine.CanWithResource(ctx, uid, CapLiveDeploy, "safe-account")
	if canLiveSafe {
		t.Error("CapLiveDeploy for safe-account should be denied — DENY(*:*) matches all resources")
	}

	// Generate strategy should be allowed
	canGen := engine.Can(ctx, uid, CapGenerateStrategy)
	if !canGen {
		t.Error("CapGenerateStrategy should be allowed by ALLOW rule")
	}

	// Remove the DENY rule — now ALLOW for safe-account should work
	store.DeleteManagedSetting(ctx, "permission.rule.1")

	canLiveSafe2 := engine.CanWithResource(ctx, uid, CapLiveDeploy, "safe-account")
	if !canLiveSafe2 {
		t.Error("CapLiveDeploy for safe-account should be allowed after removing DENY rule")
	}

	// But generic live_deploy (resource "*") should fall back to defaultCan
	canLiveGeneric := engine.Can(ctx, uid, CapLiveDeploy)
	if !canLiveGeneric {
		t.Error("CapLiveDeploy should be allowed via defaultCan (can_live=true, disable_live_trading=false)")
	}
}

// TestAdminE2E_Permissions_DisableLiveTrading verifies that disable_live_trading
// managed setting overrides can_live.
func TestAdminE2E_Permissions_DisableLiveTrading(t *testing.T) {
	t.Parallel()
	pool := adminE2ETestPG(t)
	uid := adminE2ETestUser(t, pool)
	ctx := context.Background()
	store := NewSettingsStore(pool)
	engine := NewPermissionEngine(store)

	// Set can_live=true but disable_live_trading=true
	err := store.SetManagedSetting(ctx, "agent.capability.can_live", "true")
	if err != nil {
		t.Fatalf("SetManagedSetting can_live: %v", err)
	}
	cleanupManagedSetting(t, store, ctx, "agent.capability.can_live")

	err = store.SetManagedSetting(ctx, "disable_live_trading", "true")
	if err != nil {
		t.Fatalf("SetManagedSetting disable_live_trading: %v", err)
	}
	cleanupManagedSetting(t, store, ctx, "disable_live_trading")

	// LiveDeploy should be denied because disable_live_trading takes priority
	canLive := engine.Can(ctx, uid, CapLiveDeploy)
	if canLive {
		t.Error("CapLiveDeploy should be denied when disable_live_trading=true")
	}

	// Other capabilities should still work
	canGen := engine.Can(ctx, uid, CapGenerateStrategy)
	if !canGen {
		t.Error("CapGenerateStrategy should still be allowed")
	}
}

// TestAdminE2E_Permissions_AllowManagedRulesOnly verifies that when
// allow_managed_rules_only=true, user-tier permission rules are ignored.
func TestAdminE2E_Permissions_AllowManagedRulesOnly(t *testing.T) {
	t.Parallel()
	pool := adminE2ETestPG(t)
	uid := adminE2ETestUser(t, pool)
	ctx := context.Background()
	store := NewSettingsStore(pool)
	engine := NewPermissionEngine(store)

	// Enable allow_managed_rules_only
	err := store.SetManagedSetting(ctx, "allow_managed_rules_only", "true")
	if err != nil {
		t.Fatalf("SetManagedSetting allow_managed_rules_only: %v", err)
	}
	cleanupManagedSetting(t, store, ctx, "allow_managed_rules_only")

	// Set a user-tier rule: ALLOW live_deploy
	err = store.SetUserSetting(ctx, uid, "permission.rule.1", "ALLOW live_deploy(*:*)")
	if err != nil {
		t.Fatalf("SetUserSetting: %v", err)
	}

	// Set managed config to allow live trading (so defaultCan would allow it)
	err = store.SetManagedSetting(ctx, "disable_live_trading", "false")
	if err != nil {
		t.Fatalf("SetManagedSetting: %v", err)
	}
	cleanupManagedSetting(t, store, ctx, "disable_live_trading")

	err = store.SetManagedSetting(ctx, "agent.capability.can_live", "true")
	if err != nil {
		t.Fatalf("SetManagedSetting can_live: %v", err)
	}
	cleanupManagedSetting(t, store, ctx, "agent.capability.can_live")

	// User-tier ALLOW rule should be ignored because allow_managed_rules_only=true
	// But defaultCan should still allow live_deploy (can_live=true, not disabled)
	canLive := engine.Can(ctx, uid, CapLiveDeploy)
	if !canLive {
		t.Error("CapLiveDeploy should be allowed via defaultCan (user rule ignored but defaults allow)")
	}

	// Now add a managed DENY rule — should deny even though user has ALLOW
	err = store.SetManagedSetting(ctx, "permission.rule.1", "DENY live_deploy(*:*)")
	if err != nil {
		t.Fatalf("SetManagedSetting deny rule: %v", err)
	}
	cleanupManagedSetting(t, store, ctx, "permission.rule.1")

	canLive = engine.Can(ctx, uid, CapLiveDeploy)
	if canLive {
		t.Error("CapLiveDeploy should be denied by managed DENY rule (user ALLOW ignored)")
	}
}

// TestAdminE2E_ManagedConfig_Parsing verifies that managed settings are parsed
// into the structured ManagedConfig correctly.
func TestAdminE2E_ManagedConfig_Parsing(t *testing.T) {
	t.Parallel()
	pool := adminE2ETestPG(t)
	ctx := context.Background()
	store := NewSettingsStore(pool)

	// Set various managed settings
	settings := map[string]string{
		"allowed_models":              "gpt-4,claude-3,gemini-pro",
		"enforce_allowed_models":      "true",
		"max_cost_ceiling_usd":        "0.25",
		"max_iterations_per_strategy": "10",
		"disable_live_trading":        "true",
		"required_risk_gates":         "lookahead,walkforward",
		"audit_retention_days":        "180",
		"allow_managed_rules_only":    "true",
	}
	for k, v := range settings {
		err := store.SetManagedSetting(ctx, k, v)
		if err != nil {
			t.Fatalf("SetManagedSetting %s: %v", k, err)
		}
		cleanupManagedSetting(t, store, ctx, k)
	}

	rs, err := store.ResolveSettings(ctx, uuid.New())
	if err != nil {
		t.Fatalf("ResolveSettings: %v", err)
	}
	cfg := rs.Managed

	if len(cfg.AllowedModels) != 3 {
		t.Errorf("AllowedModels len = %d, want 3", len(cfg.AllowedModels))
	}
	if !cfg.EnforceAllowedModels {
		t.Error("EnforceAllowedModels should be true")
	}
	if cfg.MaxCostCeilingUSD != 0.25 {
		t.Errorf("MaxCostCeilingUSD = %v, want 0.25", cfg.MaxCostCeilingUSD)
	}
	if cfg.MaxIterations != 10 {
		t.Errorf("MaxIterations = %d, want 10", cfg.MaxIterations)
	}
	if !cfg.DisableLiveTrading {
		t.Error("DisableLiveTrading should be true")
	}
	if len(cfg.RequiredRiskGates) != 2 {
		t.Errorf("RequiredRiskGates len = %d, want 2", len(cfg.RequiredRiskGates))
	}
	if cfg.AuditRetentionDays != 180 {
		t.Errorf("AuditRetentionDays = %d, want 180", cfg.AuditRetentionDays)
	}
	if !cfg.AllowManagedRulesOnly {
		t.Error("AllowManagedRulesOnly should be true")
	}
}
