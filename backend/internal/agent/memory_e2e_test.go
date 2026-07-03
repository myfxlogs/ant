//go:build integration

package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// memE2ETestPG connects to the test database or skips the test.
func memE2ETestPG(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return genE2ETestPG(t)
}

// memE2ETestUser inserts a unique test user and returns its UUID.
func memE2ETestUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	uid := uuid.New()
	email := fmt.Sprintf("agent-mem-%s@anttest.io", uid.String()[:8])
	ctx := context.Background()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
		 VALUES ($1, $2, '$argon2id$v=19$m=65536,t=3,p=2$test$test', 'user', 'active', NOW(), NOW())`,
		uid, email)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM agent_experience WHERE user_id = $1`, uid)
		pool.Exec(context.Background(), `DELETE FROM user_strategy_templates WHERE user_id = $1`, uid)
		pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, uid)
	})
	return uid
}

// TestMemoryE2E_DomainKnowledge verifies domain knowledge CRUD and scope filtering.
func TestMemoryE2E_DomainKnowledge(t *testing.T) {
	t.Parallel()
	pool := memE2ETestPG(t)
	ctx := context.Background()
	store := NewMemoryStore(pool, zap.NewNop())

	// Insert domain knowledge entries with different scopes
	entries := []struct {
		content string
		scope   string
	}{
		{"EURUSD trends during London", `{"symbols":["EURUSD"]}`},
		{"GBPUSD high volatility NFP", `{"symbols":["GBPUSD"]}`},
		{"H1 timeframe best for trend following", `{"timeframes":["H1"]}`},
		{"General trading wisdom", `{}`},
	}
	for _, e := range entries {
		_, err := pool.Exec(ctx,
			`INSERT INTO domain_knowledge (content, scope, status)
			 VALUES ($1, $2::jsonb, 'active')`,
			e.content, e.scope)
		if err != nil {
			t.Fatalf("insert domain knowledge: %v", err)
		}
		t.Cleanup(func() {
			pool.Exec(ctx, `DELETE FROM domain_knowledge WHERE content = $1`, e.content)
		})
	}

	// Insert an inactive entry that should be filtered out
	_, err := pool.Exec(ctx,
		`INSERT INTO domain_knowledge (content, scope, status)
		 VALUES ($1, '{}'::jsonb, 'inactive')`,
		"This should not appear")
	if err != nil {
		t.Fatalf("insert inactive domain knowledge: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM domain_knowledge WHERE content = $1`, "This should not appear")
	})

	// Load with EURUSD/H1 scope — should match EURUSD entry, H1 entry, and general entry
	mem, err := store.LoadSessionMemory(ctx, uuid.New(), "EURUSD", "H1")
	if err != nil {
		t.Fatalf("LoadSessionMemory: %v", err)
	}

	if len(mem.DomainKnowledge) == 0 {
		t.Fatal("expected domain knowledge entries")
	}

	// Verify inactive entries are excluded
	for _, dk := range mem.DomainKnowledge {
		if dk == "This should not appear" {
			t.Error("inactive domain knowledge entry was loaded")
		}
	}

	// Verify EURUSD-scoped entry is loaded
	foundEURUSD := false
	for _, dk := range mem.DomainKnowledge {
		if dk == "EURUSD trends during London" {
			foundEURUSD = true
		}
	}
	if !foundEURUSD {
		t.Error("EURUSD-scoped domain knowledge not loaded")
	}

	// Verify general (empty scope) entry is loaded
	foundGeneral := false
	for _, dk := range mem.DomainKnowledge {
		if dk == "General trading wisdom" {
			foundGeneral = true
		}
	}
	if !foundGeneral {
		t.Error("general domain knowledge not loaded")
	}

	// Verify GBPUSD-scoped entry is NOT loaded when filtering for EURUSD
	for _, dk := range mem.DomainKnowledge {
		if dk == "GBPUSD high volatility NFP" {
			t.Error("GBPUSD-scoped entry should not be loaded for EURUSD")
		}
	}
}

// TestMemoryE2E_UserTemplates verifies user template CRUD operations.
func TestMemoryE2E_UserTemplates(t *testing.T) {
	t.Parallel()
	pool := memE2ETestPG(t)
	uid := memE2ETestUser(t, pool)
	ctx := context.Background()
	store := NewMemoryStore(pool, zap.NewNop())

	// Save a template
	err := store.SaveUserTemplate(ctx, uid, "EMA Strategy", "Use EMA(10) cross EMA(30)", `{"symbols":["EURUSD"]}`)
	if err != nil {
		t.Fatalf("SaveUserTemplate: %v", err)
	}

	// Save another template with different scope
	err = store.SaveUserTemplate(ctx, uid, "RSI Strategy", "RSI oversold bounce", `{"symbols":["GBPUSD"]}`)
	if err != nil {
		t.Fatalf("SaveUserTemplate second: %v", err)
	}

	// List all templates
	templates, err := store.ListUserTemplates(ctx, uid)
	if err != nil {
		t.Fatalf("ListUserTemplates: %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}

	// Verify template names
	names := map[string]bool{}
	for _, tpl := range templates {
		names[tpl.Name] = true
	}
	if !names["EMA Strategy"] {
		t.Error("EMA Strategy template not found")
	}
	if !names["RSI Strategy"] {
		t.Error("RSI Strategy template not found")
	}

	// Update existing template (upsert via ON CONFLICT)
	err = store.SaveUserTemplate(ctx, uid, "EMA Strategy", "Updated EMA(20) cross EMA(50)", `{"symbols":["EURUSD"]}`)
	if err != nil {
		t.Fatalf("SaveUserTemplate update: %v", err)
	}

	// Verify still only 2 templates (upsert, not insert)
	templates, err = store.ListUserTemplates(ctx, uid)
	if err != nil {
		t.Fatalf("ListUserTemplates after update: %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates after upsert, got %d", len(templates))
	}

	// Verify content was updated
	for _, tpl := range templates {
		if tpl.Name == "EMA Strategy" && tpl.Content != "Updated EMA(20) cross EMA(50)" {
			t.Errorf("template content not updated, got %q", tpl.Content)
		}
	}

	// Load session memory with EURUSD scope — should only match EMA Strategy
	mem, err := store.LoadSessionMemory(ctx, uid, "EURUSD", "H1")
	if err != nil {
		t.Fatalf("LoadSessionMemory: %v", err)
	}
	if len(mem.UserTemplates) != 1 {
		t.Fatalf("expected 1 user template for EURUSD scope, got %d", len(mem.UserTemplates))
	}
	if mem.UserTemplates[0] != "[EMA Strategy] Updated EMA(20) cross EMA(50)" {
		t.Errorf("unexpected user template content: %q", mem.UserTemplates[0])
	}

	// Delete one template
	var templateID string
	for _, tpl := range templates {
		if tpl.Name == "RSI Strategy" {
			templateID = tpl.Id
		}
	}
	if templateID == "" {
		t.Fatal("could not find RSI Strategy template ID")
	}
	err = store.DeleteUserTemplate(ctx, uid, templateID)
	if err != nil {
		t.Fatalf("DeleteUserTemplate: %v", err)
	}

	// Verify deletion
	templates, err = store.ListUserTemplates(ctx, uid)
	if err != nil {
		t.Fatalf("ListUserTemplates after delete: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("expected 1 template after delete, got %d", len(templates))
	}
}

// TestMemoryE2E_AgentExperience verifies experience storage, search, and deletion.
func TestMemoryE2E_AgentExperience(t *testing.T) {
	t.Parallel()
	pool := memE2ETestPG(t)
	uid := memE2ETestUser(t, pool)
	ctx := context.Background()
	store := NewMemoryStore(pool, zap.NewNop())

	// Store multiple experiences
	exp1ID, err := store.StoreExperience(ctx, uid,
		"strategy_pattern",
		"EMA crossover with ADX filter works well in trending markets",
		"fp_ema_adx_001",
		[]string{"EMA", "ADX"},
		"crossover")
	if err != nil {
		t.Fatalf("StoreExperience 1: %v", err)
	}

	_, err = store.StoreExperience(ctx, uid,
		"risk_management",
		"Fixed 2% risk per trade with 1:2 RR ratio",
		"fp_risk_fixed",
		[]string{},
		"")
	if err != nil {
		t.Fatalf("StoreExperience 2: %v", err)
	}

	_, err = store.StoreExperience(ctx, uid,
		"strategy_pattern",
		"RSI oversold bounce in range-bound markets",
		"fp_rsi_range",
		[]string{"RSI"},
		"mean_reversion")
	if err != nil {
		t.Fatalf("StoreExperience 3: %v", err)
	}

	// List all experiences
	experiences, err := store.ListAgentExperiences(ctx, uid)
	if err != nil {
		t.Fatalf("ListAgentExperiences: %v", err)
	}
	if len(experiences) != 3 {
		t.Fatalf("expected 3 experiences, got %d", len(experiences))
	}

	// Search by category
	strategyExps, err := store.SearchExperiences(ctx, uid, "", "strategy_pattern", 10)
	if err != nil {
		t.Fatalf("SearchExperiences by category: %v", err)
	}
	if len(strategyExps) != 2 {
		t.Fatalf("expected 2 strategy_pattern experiences, got %d", len(strategyExps))
	}

	// Search by content keyword
	emaExps, err := store.SearchExperiences(ctx, uid, "EMA", "", 10)
	if err != nil {
		t.Fatalf("SearchExperiences by content: %v", err)
	}
	if len(emaExps) != 1 {
		t.Fatalf("expected 1 EMA experience, got %d", len(emaExps))
	}
	if emaExps[0].Category != "strategy_pattern" {
		t.Errorf("expected category strategy_pattern, got %q", emaExps[0].Category)
	}

	// Load session memory — experiences should be included as index lines
	mem, err := store.LoadSessionMemory(ctx, uid, "EURUSD", "H1")
	if err != nil {
		t.Fatalf("LoadSessionMemory: %v", err)
	}
	if len(mem.Experiences) != 3 {
		t.Fatalf("expected 3 experience index lines, got %d", len(mem.Experiences))
	}

	// Verify experience index format contains category and indicators
	foundEMA := false
	for _, exp := range mem.Experiences {
		if strings.Contains(exp, "EMA") && strings.Contains(exp, "strategy_pattern") {
			foundEMA = true
		}
	}
	if !foundEMA {
		t.Error("EMA experience not found in session memory index")
	}

	// Delete one experience
	err = store.DeleteAgentExperience(ctx, uid, exp1ID)
	if err != nil {
		t.Fatalf("DeleteAgentExperience: %v", err)
	}

	// Verify deletion
	experiences, err = store.ListAgentExperiences(ctx, uid)
	if err != nil {
		t.Fatalf("ListAgentExperiences after delete: %v", err)
	}
	if len(experiences) != 2 {
		t.Fatalf("expected 2 experiences after delete, got %d", len(experiences))
	}
}

// TestMemoryE2E_Isolation verifies that user A cannot access user B's memories.
func TestMemoryE2E_UserIsolation(t *testing.T) {
	t.Parallel()
	pool := memE2ETestPG(t)
	uidA := memE2ETestUser(t, pool)
	uidB := memE2ETestUser(t, pool)
	ctx := context.Background()
	store := NewMemoryStore(pool, zap.NewNop())

	// User A saves a template
	err := store.SaveUserTemplate(ctx, uidA, "A's Secret Strategy", "Private content", `{}`)
	if err != nil {
		t.Fatalf("SaveUserTemplate A: %v", err)
	}

	// User A stores an experience
	_, err = store.StoreExperience(ctx, uidA, "strategy_pattern", "A's private experience", "fp_a", []string{"MACD"}, "")
	if err != nil {
		t.Fatalf("StoreExperience A: %v", err)
	}

	// User B lists templates — should be empty
	bTemplates, err := store.ListUserTemplates(ctx, uidB)
	if err != nil {
		t.Fatalf("ListUserTemplates B: %v", err)
	}
	if len(bTemplates) != 0 {
		t.Errorf("user B should have 0 templates, got %d", len(bTemplates))
	}

	// User B lists experiences — should be empty
	bExps, err := store.ListAgentExperiences(ctx, uidB)
	if err != nil {
		t.Fatalf("ListAgentExperiences B: %v", err)
	}
	if len(bExps) != 0 {
		t.Errorf("user B should have 0 experiences, got %d", len(bExps))
	}

	// User B loads session memory — should have no user templates or experiences
	mem, err := store.LoadSessionMemory(ctx, uidB, "EURUSD", "H1")
	if err != nil {
		t.Fatalf("LoadSessionMemory B: %v", err)
	}
	if len(mem.UserTemplates) != 0 {
		t.Errorf("user B should have 0 user templates in session memory, got %d", len(mem.UserTemplates))
	}
	if len(mem.Experiences) != 0 {
		t.Errorf("user B should have 0 experiences in session memory, got %d", len(mem.Experiences))
	}

	// User A loads session memory — should have their data
	memA, err := store.LoadSessionMemory(ctx, uidA, "EURUSD", "H1")
	if err != nil {
		t.Fatalf("LoadSessionMemory A: %v", err)
	}
	if len(memA.UserTemplates) != 1 {
		t.Errorf("user A should have 1 template, got %d", len(memA.UserTemplates))
	}
	if len(memA.Experiences) != 1 {
		t.Errorf("user A should have 1 experience, got %d", len(memA.Experiences))
	}
}

// TestMemoryE2E_InjectIntoPrompt verifies that all three layers are injected.
func TestMemoryE2E_InjectIntoPrompt(t *testing.T) {
	t.Parallel()
	pool := memE2ETestPG(t)
	uid := memE2ETestUser(t, pool)
	ctx := context.Background()
	store := NewMemoryStore(pool, zap.NewNop())

	// Seed all three layers
	pool.Exec(ctx,
		`INSERT INTO domain_knowledge (content, scope, status) VALUES ($1, '{}'::jsonb, 'active')`,
		"Domain knowledge test entry")
	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM domain_knowledge WHERE content = $1`, "Domain knowledge test entry")
	})

	store.SaveUserTemplate(ctx, uid, "Test Template", "Template content", `{}`)
	store.StoreExperience(ctx, uid, "test", "Experience content", "fp_test", []string{"TEST"}, "")

	mem, err := store.LoadSessionMemory(ctx, uid, "EURUSD", "H1")
	if err != nil {
		t.Fatalf("LoadSessionMemory: %v", err)
	}

	var sb strings.Builder
	mem.InjectIntoPrompt(&sb)

	output := sb.String()
	if !strings.Contains(output, "Domain Knowledge") {
		t.Error("output missing Domain Knowledge section")
	}
	if !strings.Contains(output, "Domain knowledge test entry") {
		t.Error("output missing domain knowledge content")
	}
	if !strings.Contains(output, "User Preferences") {
		t.Error("output missing User Preferences section")
	}
	if !strings.Contains(output, "Test Template") {
		t.Error("output missing user template name")
	}
	if !strings.Contains(output, "Past Experiences") {
		t.Error("output missing Past Experiences section")
	}
	if !strings.Contains(output, "Experience content") {
		t.Error("output missing experience content")
	}
}
