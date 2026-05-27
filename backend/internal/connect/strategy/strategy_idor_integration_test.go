//go:build integration

package strategy

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"anttrader/internal/service"
)

func idorTestPG(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		dsn = "postgres://ant:ant@localhost:5432/ant?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("skipping integration test: pg connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestStrategyIDOR(t *testing.T) {
	pool := idorTestPG(t)
	ctx := context.Background()
	log := zap.NewNop()

	svc := service.NewStrategySvc(pool)
	server := NewStrategyServer(svc, log)

	userA := uuid.New()
	userB := uuid.New()

	// Create test users
	for _, u := range []struct {
		id    uuid.UUID
		email string
	}{
		{userA, "test-idor-a-" + uuid.NewString()[:8] + "@anttest.io"},
		{userB, "test-idor-b-" + uuid.NewString()[:8] + "@anttest.io"},
	} {
		_, err := pool.Exec(ctx,
			`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
			 VALUES ($1, $2, '$argon2id$v=19$m=65536,t=3,p=2$test$test', 'user', 'active', NOW(), NOW())
			 ON CONFLICT (id) DO NOTHING`,
			u.id, u.email,
		)
		if err != nil {
			t.Fatalf("insert test user: %v", err)
		}
	}

	// Create mt_accounts for both users (needed for signal IDOR tests)
	for _, a := range []struct {
		userID uuid.UUID
		accID  uuid.UUID
	}{
		{userA, uuid.New()},
		{userB, uuid.New()},
	} {
		_, err := pool.Exec(ctx,
			`INSERT INTO mt_accounts (id, user_id, login, password, mt_type, broker_company, broker_host, account_status, created_at, updated_at)
			 VALUES ($1, $2, $3, 'test-pass', 'mt5', 'TestBroker', 'localhost', 'connected', NOW(), NOW())
			 ON CONFLICT (id) DO NOTHING`,
			a.accID, a.userID, "testlogin-"+a.accID.String()[:8],
		)
		if err != nil {
			t.Fatalf("insert test account: %v", err)
		}
	}

	// Ensure tables exist
	pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS strategy_templates (
		id UUID PRIMARY KEY, user_id UUID, name TEXT, description TEXT, code TEXT, status TEXT,
		parameters JSONB DEFAULT '[]', is_public BOOLEAN DEFAULT FALSE, is_system BOOLEAN DEFAULT FALSE,
		tags TEXT[] DEFAULT '{}', use_count INT DEFAULT 0, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)`)
	pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS strategy_schedules (
		id UUID PRIMARY KEY, user_id UUID, template_id UUID, account_id UUID, name TEXT,
		symbol TEXT, timeframe TEXT, parameters JSONB, schedule_type TEXT, schedule_config JSONB,
		backtest_metrics JSONB, risk_score INT, risk_level TEXT, risk_reasons JSONB, risk_warnings JSONB,
		last_backtest_at TIMESTAMPTZ, is_active BOOLEAN DEFAULT FALSE,
		last_run_at TIMESTAMPTZ, next_run_at TIMESTAMPTZ, run_count INT DEFAULT 0,
		last_error TEXT DEFAULT '', enable_count INT DEFAULT 0, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)`)

	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM strategy_execution_logs WHERE schedule_id IN (SELECT id FROM strategy_schedules WHERE user_id IN ($1, $2))`, userA, userB)
		pool.Exec(ctx, `DELETE FROM strategy_signals WHERE account_id IN (SELECT id FROM mt_accounts WHERE user_id IN ($1, $2))`, userA, userB)
		pool.Exec(ctx, `DELETE FROM strategy_schedules WHERE user_id IN ($1, $2)`, userA, userB)
		pool.Exec(ctx, `DELETE FROM strategy_templates WHERE user_id IN ($1, $2)`, userA, userB)
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE user_id IN ($1, $2)`, userA, userB)
		pool.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2)`, userA, userB)
	})

	// --- Template IDOR tests ---

	// Test the service layer directly for IDOR ownership checks.

	// Create template as userA
	tA := &service.TemplateRow{
		ID:     uuid.New(),
		UserID: userA,
		Name:   "A's Private Template",
		Code:   "print('hello')",
		Status: "published",
	}
	if err := svc.CreateTemplate(ctx, tA); err != nil {
		t.Fatalf("create template A: %v", err)
	}

	// User A can read their own template
	_, err := svc.GetTemplate(ctx, tA.ID, userA)
	if err != nil {
		t.Errorf("userA should be able to read own template: %v", err)
	} else {
		t.Log("userA reads own template: PASS")
	}

	// User B CANNOT read user A's private template
	_, err = svc.GetTemplate(ctx, tA.ID, userB)
	if err == nil {
		t.Error("userB should NOT be able to read userA's private template (IDOR)")
	} else if err == service.ErrTemplateNotFound {
		t.Log("userB denied access to userA's template: PASS")
	} else {
		t.Logf("userB denied access (error: %v): PASS", err)
	}

	// User B CANNOT delete user A's template
	err = svc.DeleteTemplate(ctx, tA.ID, userB)
	if err == nil {
		t.Error("userB should NOT be able to delete userA's template (IDOR)")
	} else {
		t.Logf("userB cannot delete userA template: PASS")
	}

	// --- Schedule IDOR tests ---

	// Get userA's account ID for schedule FK
	var accAID uuid.UUID
	_ = pool.QueryRow(ctx,
		`SELECT id FROM mt_accounts WHERE user_id = $1 LIMIT 1`, userA,
	).Scan(&accAID)

	// Create schedule as userA (references tA template + userA account)
	sA := &service.ScheduleRow{
		ID:             uuid.New(),
		UserID:         userA,
		TemplateID:     tA.ID,
		AccountID:      accAID,
		Name:           "A's Schedule",
		ScheduleType:   "interval",
		ScheduleConfig: []byte(`{"interval_ms": 60000}`),
	}
	if err := svc.CreateSchedule(ctx, sA); err != nil {
		t.Fatalf("create schedule A: %v", err)
	}

	// User A can read own schedule
	_, err = svc.GetSchedule(ctx, sA.ID, userA)
	if err != nil {
		t.Errorf("userA should be able to read own schedule: %v", err)
	} else {
		t.Log("userA reads own schedule: PASS")
	}

	// User B CANNOT read user A's schedule
	_, err = svc.GetSchedule(ctx, sA.ID, userB)
	if err == nil {
		t.Error("userB should NOT be able to read userA's schedule (IDOR)")
	} else {
		t.Logf("userB denied access to userA's schedule: PASS")
	}

	// User B CANNOT delete user A's schedule
	err = svc.DeleteSchedule(ctx, sA.ID, userB)
	if err == nil {
		t.Error("userB should NOT be able to delete userA's schedule (IDOR)")
	} else {
		t.Logf("userB cannot delete userA schedule: PASS")
	}

	// --- Public template is readable by userB ---

	tPub := &service.TemplateRow{
		ID:       uuid.New(),
		UserID:   userA,
		Name:     "A's Public Template",
		Code:     "print('public')",
		Status:   "published",
		IsPublic: true,
	}
	if err := svc.CreateTemplate(ctx, tPub); err != nil {
		t.Fatalf("create public template: %v", err)
	}
	row, err := svc.GetTemplate(ctx, tPub.ID, userB)
	if err != nil {
		t.Errorf("userB should be able to read public template: %v", err)
	} else {
		t.Logf("userB reads userA's public template (name=%s): PASS", row.Name)
	}

	_ = server // mark as used
}
