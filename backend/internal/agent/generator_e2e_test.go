//go:build integration

package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/pkg/secretbox"
	"anttrader/internal/repository"
	systemai "anttrader/internal/service/systemai"
)

// genE2ETestPG connects to the test database or skips the test.
func genE2ETestPG(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		dsn = "postgres://ant:ant@localhost:5432/ant?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping integration test: pg connect: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skipping integration test: pg ping: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// genE2ETestUser inserts a unique test user and returns its UUID.
func genE2ETestUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	uid := uuid.New()
	email := fmt.Sprintf("agent-gen-%s@anttest.io", uid.String()[:8])
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
		pool.Exec(context.Background(), `DELETE FROM agent_experience WHERE user_id = $1`, uid)
		pool.Exec(context.Background(), `DELETE FROM user_strategy_templates WHERE user_id = $1`, uid)
		pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, uid)
	})
	return uid
}

// genE2ETestAI creates a systemai.Service backed by the test DB.
// No provider configs are seeded with secrets, so all LLM calls will fail
// with "AI 未配置" — this is intentional for testing error/degradation paths.
func genE2ETestAI(t *testing.T, pool *pgxpool.Pool) *systemai.Service {
	t.Helper()
	box := secretbox.New([]byte("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
	repo := repository.NewSystemAIConfigRepository(pool)
	return systemai.NewService(repo, box)
}

// collectChunks is a stream callback that collects all chunks for assertion.
func collectChunks(out *[]*antv1.AgentGenerateStrategyChunk) func(*antv1.AgentGenerateStrategyChunk) error {
	return func(c *antv1.AgentGenerateStrategyChunk) error {
		*out = append(*out, c)
		return nil
	}
}

// TestGeneratorE2E_PlanMode_NoLLM verifies the Plan Mode flow when LLM is unavailable.
// The generator should: stream "planning" → attempt profile (fail gracefully) →
// attempt plan (fail) → stream "done" with error.
func TestGeneratorE2E_PlanMode_NoLLM(t *testing.T) {
	t.Parallel()
	pool := genE2ETestPG(t)
	uid := genE2ETestUser(t, pool)
	ctx := context.Background()

	aiSvc := genE2ETestAI(t, pool)
	cache := NewLLCache(5 * time.Minute)
	memStore := NewMemoryStore(pool, zap.NewNop())
	settingsStore := NewSettingsStore(pool)
	hooks := NewHookEngine(zap.NewNop())

	gen := NewGenerator(aiSvc, zap.NewNop(),
		NewProfiler(aiSvc, cache),
		NewInterpreter(aiSvc, cache),
		cache,
		memStore,
		hooks,
		settingsStore,
	)

	var chunks []*antv1.AgentGenerateStrategyChunk
	err := gen.Generate(ctx, uid,
		&antv1.AgentGenerateStrategyRequest{
			Message:   "EMA crossover strategy on EURUSD H1",
			Symbol:    "EURUSD",
			Timeframe: "H1",
			PlanMode:  "plan",
		},
		collectChunks(&chunks),
	)

	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// First chunk should be "planning" phase
	if chunks[0].Phase != "planning" {
		t.Errorf("first chunk phase = %q, want %q", chunks[0].Phase, "planning")
	}

	// Last chunk should be "done" with an error (LLM unavailable)
	last := chunks[len(chunks)-1]
	if last.Phase != "done" {
		t.Errorf("last chunk phase = %q, want %q", last.Phase, "done")
	}
	if last.Error == "" {
		t.Error("last chunk should contain an error message (LLM unavailable)")
	}
	if last.Plan != nil {
		t.Error("plan should be nil when LLM fails")
	}
}

// TestGeneratorE2E_GenerateMode_CostCeiling verifies that the cost ceiling
// from managed settings is enforced, limiting the number of retry attempts.
func TestGeneratorE2E_GenerateMode_CostCeiling(t *testing.T) {
	t.Parallel()
	pool := genE2ETestPG(t)
	uid := genE2ETestUser(t, pool)
	ctx := context.Background()

	// Set managed settings: very low cost ceiling + max_iterations=1
	settingsStore := NewSettingsStore(pool)
	if err := settingsStore.SetManagedSetting(ctx, "max_cost_ceiling_usd", "0.001"); err != nil {
		t.Fatalf("set managed cost ceiling: %v", err)
	}
	if err := settingsStore.SetManagedSetting(ctx, "max_iterations_per_strategy", "1"); err != nil {
		t.Fatalf("set managed max iterations: %v", err)
	}
	t.Cleanup(func() {
		settingsStore.DeleteManagedSetting(ctx, "max_cost_ceiling_usd")
		settingsStore.DeleteManagedSetting(ctx, "max_iterations_per_strategy")
	})

	aiSvc := genE2ETestAI(t, pool)
	cache := NewLLCache(5 * time.Minute)
	memStore := NewMemoryStore(pool, zap.NewNop())
	hooks := NewHookEngine(zap.NewNop())

	gen := NewGenerator(aiSvc, zap.NewNop(),
		NewProfiler(aiSvc, cache),
		NewInterpreter(aiSvc, cache),
		cache,
		memStore,
		hooks,
		settingsStore,
	)

	var chunks []*antv1.AgentGenerateStrategyChunk
	err := gen.Generate(ctx, uid,
		&antv1.AgentGenerateStrategyRequest{
			Message:   "RSI oversold bounce",
			Symbol:    "EURUSD",
			Timeframe: "H1",
			PlanMode:  "generate",
		},
		collectChunks(&chunks),
	)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	// With max_iterations=1, only one attempt should be made.
	// The LLM call will fail (no AI configured), so we expect:
	// planning → generating (attempt 1) → done with error
	hasGenerating := false
	hasDone := false
	for _, c := range chunks {
		if c.Phase == "generating" {
			hasGenerating = true
			if c.Attempts > 1 {
				t.Errorf("expected at most 1 attempt, got %d", c.Attempts)
			}
		}
		if c.Phase == "done" {
			hasDone = true
		}
	}
	if !hasGenerating {
		t.Error("expected at least one 'generating' phase chunk")
	}
	if !hasDone {
		t.Error("expected a 'done' phase chunk")
	}
}

// TestGeneratorE2E_PlanMode_WithMemory verifies that session memory is loaded
// from the database and the generator proceeds through the plan phase.
func TestGeneratorE2E_PlanMode_WithMemory(t *testing.T) {
	t.Parallel()
	pool := genE2ETestPG(t)
	uid := genE2ETestUser(t, pool)
	ctx := context.Background()

	// Seed domain knowledge
	_, err := pool.Exec(ctx,
		`INSERT INTO domain_knowledge (content, scope, status)
		 VALUES ($1, '{"symbols":["EURUSD"]}'::jsonb, 'active')`,
		"EURUSD tends to trend during London session")
	if err != nil {
		t.Fatalf("insert domain knowledge: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM domain_knowledge WHERE content = $1`, "EURUSD tends to trend during London session")
	})

	// Seed user template
	memStore := NewMemoryStore(pool, zap.NewNop())
	if err := memStore.SaveUserTemplate(ctx, uid, "EMA Trend", "Use EMA crossover with ADX filter", `{"symbols":["EURUSD"]}`); err != nil {
		t.Fatalf("save user template: %v", err)
	}

	// Seed agent experience
	if _, err := memStore.StoreExperience(ctx, uid, "strategy_pattern", "EMA crossover works well in trending markets", "fp_ema_cross", []string{"EMA", "ADX"}, "crossover"); err != nil {
		t.Fatalf("store experience: %v", err)
	}

	// Verify memory loads correctly
	sessionMem, err := memStore.LoadSessionMemory(ctx, uid, "EURUSD", "H1")
	if err != nil {
		t.Fatalf("load session memory: %v", err)
	}
	if len(sessionMem.DomainKnowledge) == 0 {
		t.Error("expected domain knowledge to be loaded")
	}
	if len(sessionMem.UserTemplates) == 0 {
		t.Error("expected user templates to be loaded")
	}
	if len(sessionMem.Experiences) == 0 {
		t.Error("expected experiences to be loaded")
	}

	// Verify InjectIntoPrompt produces non-empty output
	var sb strings.Builder
	sessionMem.InjectIntoPrompt(&sb)
	if sb.Len() == 0 {
		t.Error("InjectIntoPrompt produced empty output")
	}

	// Now run the generator — it will load memory but LLM will fail
	aiSvc := genE2ETestAI(t, pool)
	cache := NewLLCache(5 * time.Minute)
	settingsStore := NewSettingsStore(pool)
	hooks := NewHookEngine(zap.NewNop())

	gen := NewGenerator(aiSvc, zap.NewNop(),
		NewProfiler(aiSvc, cache),
		NewInterpreter(aiSvc, cache),
		cache,
		memStore,
		hooks,
		settingsStore,
	)

	var chunks []*antv1.AgentGenerateStrategyChunk
	err = gen.Generate(ctx, uid,
		&antv1.AgentGenerateStrategyRequest{
			Message:   "EMA crossover with ADX filter",
			Symbol:    "EURUSD",
			Timeframe: "H1",
			PlanMode:  "plan",
		},
		collectChunks(&chunks),
	)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	// Memory should be loaded without error, even though LLM fails.
	// The generator should still stream planning → done with error.
	last := chunks[len(chunks)-1]
	if last.Phase != "done" {
		t.Errorf("last chunk phase = %q, want %q", last.Phase, "done")
	}
}

// TestGeneratorE2E_GenerateMode_WithConfirmedPlan verifies the generate path
// when a confirmed plan is provided (planMode="generate" with ConfirmedPlan).
func TestGeneratorE2E_GenerateMode_WithConfirmedPlan(t *testing.T) {
	t.Parallel()
	pool := genE2ETestPG(t)
	uid := genE2ETestUser(t, pool)
	ctx := context.Background()

	aiSvc := genE2ETestAI(t, pool)
	cache := NewLLCache(5 * time.Minute)
	memStore := NewMemoryStore(pool, zap.NewNop())
	settingsStore := NewSettingsStore(pool)
	hooks := NewHookEngine(zap.NewNop())

	gen := NewGenerator(aiSvc, zap.NewNop(),
		NewProfiler(aiSvc, cache),
		NewInterpreter(aiSvc, cache),
		cache,
		memStore,
		hooks,
		settingsStore,
	)

	plan := &antv1.StrategyPlan{
		Type:    "dual_ema_crossover",
		Entry:   "EMA(10) crosses above EMA(30)",
		Exit:    "EMA(10) crosses below EMA(30)",
		Risk:    "2% per trade, RR 1:2",
		Market:  "trending, ADX>25, H1",
	}

	var chunks []*antv1.AgentGenerateStrategyChunk
	err := gen.Generate(ctx, uid,
		&antv1.AgentGenerateStrategyRequest{
			Message:       "EMA crossover strategy",
			Symbol:        "EURUSD",
			Timeframe:     "H1",
			PlanMode:      "generate",
			ConfirmedPlan: plan,
		},
		collectChunks(&chunks),
	)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	// Should stream: planning → generating → done (LLM fails)
	hasGenerating := false
	for _, c := range chunks {
		if c.Phase == "generating" {
			hasGenerating = true
		}
	}
	if !hasGenerating {
		t.Error("expected 'generating' phase when planMode=generate with confirmed plan")
	}

	last := chunks[len(chunks)-1]
	if last.Phase != "done" {
		t.Errorf("last chunk phase = %q, want %q", last.Phase, "done")
	}
}
