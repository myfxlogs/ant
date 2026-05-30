//go:build integration

package strategy

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/interceptor"
	"anttrader/internal/service"
)

func ptr[T any](v T) *T { return &v }

func strategyTestPG(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		password := os.Getenv("DB_PASSWORD")
		if password == "" { password = "ant" }
		user := os.Getenv("DB_USER")
		if user == "" { user = "ant" }
		dbname := os.Getenv("DB_NAME")
		if dbname == "" { dbname = "ant" }
		dsn = "postgres://" + user + ":" + password + "@localhost:5433/" + dbname + "?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("skipping integration test: pg connect: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("skipping integration test: pg ping: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func insertStratTestUser(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, email string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
		 VALUES ($1, $2, '$argon2id$v=19$m=65536,t=3,p=2$test$test', 'user', 'active', NOW(), NOW())
		 ON CONFLICT (id) DO NOTHING`,
		userID, email,
	)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	accID := uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO mt_accounts (id, user_id, login, password, mt_type, broker_company, broker_host, account_status, balance, equity, created_at, updated_at)
		 VALUES ($1, $2, $3, 'test-pass', 'mt5', 'TestBroker', 'localhost', 'connected', 10000, 10000, NOW(), NOW())
		 ON CONFLICT (id) DO NOTHING`,
		accID, userID, "testlogin-"+userID.String()[:8],
	)
	if err != nil {
		t.Fatalf("insert test account: %v", err)
	}
	return accID
}

func stratAuthCtx(userID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), interceptor.UserIDKey, userID.String())
}

func newStratServer(t *testing.T, pool *pgxpool.Pool) *StrategyServer {
	t.Helper()
	svc := service.NewStrategySvc(pool)
	return NewStrategyServer(svc, zap.NewNop())
}

// ── Template Lifecycle ──

func TestStrategy_TemplateLifecycle(t *testing.T) {
	t.Parallel()
	pool := strategyTestPG(t)
	userID := uuid.New()
	accID := insertStratTestUser(t, pool, userID, fmt.Sprintf("tpl-life-%d@test.io", time.Now().UnixNano()))
	_ = accID
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM strategy_schedules WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM strategy_templates WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	srv := newStratServer(t, pool)
	ctx := stratAuthCtx(userID)

	listResp, err := srv.ListTemplates(ctx, connect.NewRequest(&antv1.ListTemplatesRequest{}))
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	initialCount := len(listResp.Msg.Templates)

	createResp, err := srv.CreateTemplate(ctx, connect.NewRequest(&antv1.CreateTemplateRequest{
		Name:        "Test Template",
		Description: "Integration test template",
		Code:        "print('hello world')",
		Tags:        []string{"test"},
	}))
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	tplID := createResp.Msg.Id
	if tplID == "" {
		t.Fatal("expected non-empty template ID")
	}

	listResp2, err := srv.ListTemplates(ctx, connect.NewRequest(&antv1.ListTemplatesRequest{}))
	if err != nil {
		t.Fatalf("ListTemplates after create: %v", err)
	}
	if len(listResp2.Msg.Templates) != initialCount+1 {
		t.Errorf("expected %d templates, got %d", initialCount+1, len(listResp2.Msg.Templates))
	}

	getResp, err := srv.GetTemplate(ctx, connect.NewRequest(&antv1.GetTemplateRequest{Id: tplID}))
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if getResp.Msg.Name != "Test Template" {
		t.Errorf("expected 'Test Template', got %q", getResp.Msg.Name)
	}

	_, err = srv.UpdateTemplate(ctx, connect.NewRequest(&antv1.UpdateTemplateRequest{
		Id:          tplID,
		Name:        ptr("Updated Template"),
		Description: ptr("Updated description"),
		Code:        ptr("print('updated')"),
	}))
	if err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}

	_, err = srv.DeleteTemplate(ctx, connect.NewRequest(&antv1.DeleteTemplateRequest{Id: tplID}))
	if err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}

	_, err = srv.GetTemplate(ctx, connect.NewRequest(&antv1.GetTemplateRequest{Id: tplID}))
	if err == nil {
		t.Error("expected error getting deleted template")
	}
}

// ── Template Validation ──

func TestStrategy_TemplateValidation(t *testing.T) {
	t.Parallel()
	pool := strategyTestPG(t)
	userID := uuid.New()
	accID := insertStratTestUser(t, pool, userID, fmt.Sprintf("tpl-val-%d@test.io", time.Now().UnixNano()))
	_ = accID
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM strategy_templates WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	srv := newStratServer(t, pool)
	ctx := stratAuthCtx(userID)

	_, err := srv.CreateTemplate(ctx, connect.NewRequest(&antv1.CreateTemplateRequest{
		Name: "",
		Code: "print('test')",
	}))
	if err == nil {
		t.Error("expected error for empty template name")
	}

	_, err = srv.CreateTemplate(ctx, connect.NewRequest(&antv1.CreateTemplateRequest{
		Name: "Valid Name",
		Code: "",
	}))
	if err == nil {
		t.Error("expected error for empty template code")
	}
}

// ── Template Draft Workflow ──

func TestStrategy_TemplateDraftWorkflow(t *testing.T) {
	t.Parallel()
	pool := strategyTestPG(t)
	userID := uuid.New()
	accID := insertStratTestUser(t, pool, userID, fmt.Sprintf("tpl-draft-%d@test.io", time.Now().UnixNano()))
	_ = accID
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM strategy_templates WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	srv := newStratServer(t, pool)
	ctx := stratAuthCtx(userID)

	draftResp, err := srv.CreateTemplateDraft(ctx, connect.NewRequest(&antv1.CreateTemplateDraftRequest{
		Name: "Draft Template",
	}))
	if err != nil {
		t.Fatalf("CreateTemplateDraft: %v", err)
	}
	draftID := draftResp.Msg.Id
	if draftID == "" {
		t.Fatal("expected non-empty draft ID")
	}

	_, err = srv.UpdateTemplateDraft(ctx, connect.NewRequest(&antv1.UpdateTemplateDraftRequest{
		Id:          draftID,
		Name:        ptr("Draft Template v2"),
		Description: ptr("Improved version"),
		Code:        ptr("print('draft v2')"),
	}))
	if err != nil {
		t.Fatalf("UpdateTemplateDraft: %v", err)
	}

	pubResp, err := srv.PublishTemplateDraft(ctx, connect.NewRequest(&antv1.PublishTemplateDraftRequest{Id: draftID}))
	if err != nil {
		t.Fatalf("PublishTemplateDraft: %v", err)
	}
	if pubResp.Msg.Status != "published" {
		t.Errorf("expected 'published', got %q", pubResp.Msg.Status)
	}

	cancelResp, err := srv.CreateTemplateDraft(ctx, connect.NewRequest(&antv1.CreateTemplateDraftRequest{
		Name: "To Cancel",
	}))
	if err != nil {
		t.Fatalf("CreateTemplateDraft for cancel: %v", err)
	}
	_, err = srv.CancelTemplateDraft(ctx, connect.NewRequest(&antv1.CancelTemplateDraftRequest{Id: cancelResp.Msg.Id}))
	if err != nil {
		t.Fatalf("CancelTemplateDraft: %v", err)
	}
}

// ── Schedule Lifecycle ──

func TestStrategy_ScheduleLifecycle(t *testing.T) {
	t.Parallel()
	pool := strategyTestPG(t)
	userID := uuid.New()
	accID := insertStratTestUser(t, pool, userID, fmt.Sprintf("sch-life-%d@test.io", time.Now().UnixNano()))
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM strategy_execution_logs WHERE schedule_id IN (SELECT id FROM strategy_schedules WHERE user_id = $1)`, userID)
		pool.Exec(ctx, `DELETE FROM strategy_schedules WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM strategy_templates WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	srv := newStratServer(t, pool)
	ctx := stratAuthCtx(userID)

	tplResp, err := srv.CreateTemplate(ctx, connect.NewRequest(&antv1.CreateTemplateRequest{
		Name: "Schedule Template",
		Code: "print('schedule')",
	}))
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	createResp, err := srv.CreateSchedule(ctx, connect.NewRequest(&antv1.CreateScheduleRequest{
		TemplateId:   tplResp.Msg.Id,
		AccountId:    accID.String(),
		Name:         "Test Schedule",
		Symbol:       "EURUSD",
		Timeframe:    "H1",
		ScheduleType: "interval",
		ScheduleConfig: &antv1.ScheduleConfig{
			IntervalMs: 60000,
		},
	}))
	if err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}
	schID := createResp.Msg.Id
	if schID == "" {
		t.Fatal("expected non-empty schedule ID")
	}

	listResp, err := srv.ListSchedules(ctx, connect.NewRequest(&antv1.ListSchedulesRequest{}))
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	found := false
	for _, s := range listResp.Msg.Schedules {
		if s.Id == schID {
			found = true
			break
		}
	}
	if !found {
		t.Error("created schedule not found in list")
	}

	getResp, err := srv.GetSchedule(ctx, connect.NewRequest(&antv1.GetScheduleRequest{Id: schID}))
	if err != nil {
		t.Fatalf("GetSchedule: %v", err)
	}
	if getResp.Msg.Name != "Test Schedule" {
		t.Errorf("expected 'Test Schedule', got %q", getResp.Msg.Name)
	}

	toggleResp, err := srv.ToggleSchedule(ctx, connect.NewRequest(&antv1.ToggleScheduleRequest{Id: schID}))
	if err != nil {
		t.Fatalf("ToggleSchedule: %v", err)
	}
	if !toggleResp.Msg.IsActive {
		t.Error("expected is_active=true after toggle")
	}

	_, err = srv.UpdateSchedule(ctx, connect.NewRequest(&antv1.UpdateScheduleRequest{
		Id:   schID,
		Name: ptr("Updated Schedule"),
	}))
	if err != nil {
		t.Fatalf("UpdateSchedule: %v", err)
	}

	_, err = srv.DeleteSchedule(ctx, connect.NewRequest(&antv1.DeleteScheduleRequest{Id: schID}))
	if err != nil {
		t.Fatalf("DeleteSchedule: %v", err)
	}

	listResp2, err := srv.ListSchedules(ctx, connect.NewRequest(&antv1.ListSchedulesRequest{}))
	if err != nil {
		t.Fatalf("ListSchedules after delete: %v", err)
	}
	for _, s := range listResp2.Msg.Schedules {
		if s.Id == schID {
			t.Error("schedule should have been deleted")
		}
	}
}

// ── Schedule Validation ──

func TestStrategy_ScheduleValidation(t *testing.T) {
	t.Parallel()
	pool := strategyTestPG(t)
	userID := uuid.New()
	accID := insertStratTestUser(t, pool, userID, fmt.Sprintf("sch-val-%d@test.io", time.Now().UnixNano()))
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM strategy_schedules WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM strategy_templates WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	srv := newStratServer(t, pool)
	ctx := stratAuthCtx(userID)

	tplResp, err := srv.CreateTemplate(ctx, connect.NewRequest(&antv1.CreateTemplateRequest{
		Name: "Validation Test",
		Code: "print('val')",
	}))
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	// Missing symbol
	_, err = srv.CreateSchedule(ctx, connect.NewRequest(&antv1.CreateScheduleRequest{
		TemplateId:   tplResp.Msg.Id,
		AccountId:    accID.String(),
		Name:         "Bad Schedule",
		Symbol:       "",
		ScheduleType: "interval",
		ScheduleConfig: &antv1.ScheduleConfig{IntervalMs: 60000},
	}))
	if err == nil {
		t.Error("expected error for empty symbol")
	}

	// Invalid schedule type
	_, err = srv.CreateSchedule(ctx, connect.NewRequest(&antv1.CreateScheduleRequest{
		TemplateId:   tplResp.Msg.Id,
		AccountId:    accID.String(),
		Name:         "Bad Schedule",
		Symbol:       "EURUSD",
		ScheduleType: "invalid_type",
	}))
	if err == nil {
		t.Error("expected error for invalid schedule type")
	}
}

// ── Delete Template With Schedule ──

func TestStrategy_DeleteTemplateWithSchedule(t *testing.T) {
	t.Parallel()
	pool := strategyTestPG(t)
	userID := uuid.New()
	accID := insertStratTestUser(t, pool, userID, fmt.Sprintf("tpl-sch-%d@test.io", time.Now().UnixNano()))
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM strategy_schedules WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM strategy_templates WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	srv := newStratServer(t, pool)
	ctx := stratAuthCtx(userID)

	tplResp, err := srv.CreateTemplate(ctx, connect.NewRequest(&antv1.CreateTemplateRequest{
		Name: "Template With Schedule",
		Code: "print('has schedule')",
	}))
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	_, err = srv.CreateSchedule(ctx, connect.NewRequest(&antv1.CreateScheduleRequest{
		TemplateId:   tplResp.Msg.Id,
		AccountId:    accID.String(),
		Name:         "Attached Schedule",
		Symbol:       "EURUSD",
		ScheduleType: "interval",
		ScheduleConfig: &antv1.ScheduleConfig{IntervalMs: 30000},
	}))
	if err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}

	_, err = srv.DeleteTemplate(ctx, connect.NewRequest(&antv1.DeleteTemplateRequest{Id: tplResp.Msg.Id}))
	if err != nil {
		t.Logf("delete template with schedule rejected (expected): %v", err)
	} else {
		t.Log("delete template with schedule succeeded")
	}
}
