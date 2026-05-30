//go:build integration

package ai

import (
	"context"
	"errors"
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
	"anttrader/internal/pkg/secretbox"
	"anttrader/internal/repository"
	systemai "anttrader/internal/service/systemai"
)

// ── test infra ──

func getTestPG(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		password := os.Getenv("DB_PASSWORD")
		if password == "" {
			password = "ant"
		}
		user := os.Getenv("DB_USER")
		if user == "" {
			user = "ant"
		}
		dbname := os.Getenv("DB_NAME")
		if dbname == "" {
			dbname = "ant"
		}
		dsn = "postgres://" + user + ":" + password + "@localhost:5433/" + dbname + "?sslmode=disable"
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

func insertTestUser(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, email string) {
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
}

func authCtx(userID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), interceptor.UserIDKey, userID.String())
}

// ── SystemAI Config CRUD tests ──

func newSystemAIServer(t *testing.T, pool *pgxpool.Pool) *SystemAIServer {
	t.Helper()
	box := secretbox.New([]byte("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
	repo := repository.NewSystemAIConfigRepository(pool)
	svc := systemai.NewService(repo, box)
	return NewSystemAIServer(svc, zap.NewNop())
}

func TestSystemAI_ListConfigs(t *testing.T) {
	t.Parallel()
	pool := getTestPG(t)
	userID := uuid.New()
	insertTestUser(t, pool, userID, fmt.Sprintf("sysai-list-%d@test.io", time.Now().UnixNano()))
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM system_ai_configs WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	srv := newSystemAIServer(t, pool)
	ctx := authCtx(userID)

	resp, err := srv.ListSystemAIConfigs(ctx, connect.NewRequest(&antv1.ListSystemAIConfigsRequest{}))
	if err != nil {
		t.Fatalf("ListSystemAIConfigs: %v", err)
	}
	if len(resp.Msg.Items) == 0 {
		t.Error("expected at least one seeded provider config, got 0")
	}
	t.Logf("ListSystemAIConfigs returned %d configs", len(resp.Msg.Items))
}

func TestSystemAI_GetConfig(t *testing.T) {
	t.Parallel()
	pool := getTestPG(t)
	userID := uuid.New()
	insertTestUser(t, pool, userID, fmt.Sprintf("sysai-get-%d@test.io", time.Now().UnixNano()))
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM system_ai_configs WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	srv := newSystemAIServer(t, pool)
	ctx := authCtx(userID)

	// First list to get seed providers
	listResp, err := srv.ListSystemAIConfigs(ctx, connect.NewRequest(&antv1.ListSystemAIConfigsRequest{}))
	if err != nil {
		t.Fatalf("ListSystemAIConfigs: %v", err)
	}
	if len(listResp.Msg.Items) == 0 {
		t.Skip("no seeded providers")
	}

	// Get a specific provider
	providerID := listResp.Msg.Items[0].ProviderId
	resp, err := srv.GetSystemAIConfig(ctx, connect.NewRequest(&antv1.GetSystemAIConfigRequest{
		ProviderId: providerID,
	}))
	if err != nil {
		t.Fatalf("GetSystemAIConfig(%s): %v", providerID, err)
	}
	if resp.Msg.Item.ProviderId != providerID {
		t.Errorf("expected provider_id %q, got %q", providerID, resp.Msg.Item.ProviderId)
	}
}

func TestSystemAI_UpdateConfig(t *testing.T) {
	t.Parallel()
	pool := getTestPG(t)
	userID := uuid.New()
	insertTestUser(t, pool, userID, fmt.Sprintf("sysai-update-%d@test.io", time.Now().UnixNano()))
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM system_ai_configs WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	srv := newSystemAIServer(t, pool)
	ctx := authCtx(userID)

	// Update the openai config
	newName := "TestOpenAI"
	_, err := srv.UpdateSystemAIConfig(ctx, connect.NewRequest(&antv1.UpdateSystemAIConfigRequest{
		ProviderId: "openai",
		Name:       newName,
		Models:     []string{"gpt-4o", "gpt-4o-mini"},
		Enabled:    true,
	}))
	if err != nil {
		t.Fatalf("UpdateSystemAIConfig: %v", err)
	}

	// Verify the update persisted
	getResp, err := srv.GetSystemAIConfig(ctx, connect.NewRequest(&antv1.GetSystemAIConfigRequest{
		ProviderId: "openai",
	}))
	if err != nil {
		t.Fatalf("GetSystemAIConfig after update: %v", err)
	}
	if getResp.Msg.Item.Name != newName {
		t.Errorf("expected name %q, got %q", newName, getResp.Msg.Item.Name)
	}
	if !getResp.Msg.Item.Enabled {
		t.Error("expected enabled=true after update")
	}
}

func TestSystemAI_UpdateSecret(t *testing.T) {
	t.Parallel()
	pool := getTestPG(t)
	userID := uuid.New()
	insertTestUser(t, pool, userID, fmt.Sprintf("sysai-secret-%d@test.io", time.Now().UnixNano()))
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM system_ai_configs WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	srv := newSystemAIServer(t, pool)
	ctx := authCtx(userID)

	_, err := srv.UpdateSystemAISecret(ctx, connect.NewRequest(&antv1.UpdateSystemAISecretRequest{
		ProviderId: "openai",
		Secret:     "sk-test-secret-key-12345",
	}))
	if err != nil {
		t.Fatalf("UpdateSystemAISecret: %v", err)
	}

	// Verify has_secret is now true
	getResp, err := srv.GetSystemAIConfig(ctx, connect.NewRequest(&antv1.GetSystemAIConfigRequest{
		ProviderId: "openai",
	}))
	if err != nil {
		t.Fatalf("GetSystemAIConfig after secret update: %v", err)
	}
	if !getResp.Msg.Item.HasSecret {
		t.Error("expected has_secret=true after UpdateSystemAISecret")
	}
}

func TestSystemAI_Unauthenticated(t *testing.T) {
	t.Parallel()
	pool := getTestPG(t)
	srv := newSystemAIServer(t, pool)

	// Without auth context, should get Unauthenticated error
	_, err := srv.ListSystemAIConfigs(context.Background(), connect.NewRequest(&antv1.ListSystemAIConfigsRequest{}))
	if err == nil {
		t.Error("expected error for unauthenticated request")
	}
	var ce *connect.Error
	if !errors.As(err, &ce) {
		t.Fatalf("expected connect.Error, got %T", err)
	}
	if ce.Code() != connect.CodeUnauthenticated {
		t.Errorf("expected Unauthenticated, got %v", ce.Code())
	}
}

// ── AI Primary tests ──

func newAIPrimaryServer(t *testing.T, pool *pgxpool.Pool) *AIPrimaryServer {
	t.Helper()
	box := secretbox.New([]byte("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
	repo := repository.NewSystemAIConfigRepository(pool)
	svc := systemai.NewService(repo, box)
	return NewAIPrimaryServer(svc, zap.NewNop())
}

func TestAIPrimary_GetSetPrimary(t *testing.T) {
	t.Parallel()
	pool := getTestPG(t)
	userID := uuid.New()
	insertTestUser(t, pool, userID, fmt.Sprintf("aiprimary-%d@test.io", time.Now().UnixNano()))
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM ai_primary WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	srv := newAIPrimaryServer(t, pool)
	ctx := authCtx(userID)

	// Set primary
	_, err := srv.SetAIPrimary(ctx, connect.NewRequest(&antv1.SetAIPrimaryRequest{
		ProviderId:   "openai",
		Model: "gpt-4o",
	}))
	if err != nil {
		t.Fatalf("SetAIPrimary: %v", err)
	}

	// Get primary
	getResp, err := srv.GetAIPrimary(ctx, connect.NewRequest(&antv1.GetAIPrimaryRequest{}))
	if err != nil {
		t.Fatalf("GetAIPrimary: %v", err)
	}
	if getResp.Msg.ProviderId != "openai" {
		t.Errorf("expected provider_id openai, got %q", getResp.Msg.ProviderId)
	}
	if getResp.Msg.Model != "gpt-4o" {
		t.Errorf("expected model gpt-4o, got %q", getResp.Msg.Model)
	}
}

// ── Conversations CRUD tests ──

func newAIServer(t *testing.T, pool *pgxpool.Pool) *AIServer {
	t.Helper()
	box := secretbox.New([]byte("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
	configRepo := repository.NewSystemAIConfigRepository(pool)
	svc := systemai.NewService(configRepo, box)
	convRepo := repository.NewAIConversationRepository(pool)
	return NewAIServer(svc, convRepo, zap.NewNop())
}

func TestConversations_Lifecycle(t *testing.T) {
	t.Parallel()
	pool := getTestPG(t)
	userID := uuid.New()
	insertTestUser(t, pool, userID, fmt.Sprintf("conv-life-%d@test.io", time.Now().UnixNano()))
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM ai_conversations WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	srv := newAIServer(t, pool)
	ctx := authCtx(userID)

	// 1. List — should be empty
	listResp, err := srv.ListConversations(ctx, connect.NewRequest(&antv1.ListConversationsRequest{}))
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	initialCount := len(listResp.Msg.Conversations)

	// 2. Create
	createResp, err := srv.CreateConversation(ctx, connect.NewRequest(&antv1.CreateConversationRequest{
		Title: "Test Conversation",
	}))
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	convID := createResp.Msg.Conversation.Id
	if convID == "" {
		t.Fatal("expected non-empty conversation id")
	}
	t.Logf("created conversation id=%s", convID)

	// 3. List — should have one more
	listResp2, err := srv.ListConversations(ctx, connect.NewRequest(&antv1.ListConversationsRequest{}))
	if err != nil {
		t.Fatalf("ListConversations after create: %v", err)
	}
	if len(listResp2.Msg.Conversations) != initialCount+1 {
		t.Errorf("expected %d conversations, got %d", initialCount+1, len(listResp2.Msg.Conversations))
	}

	// 4. Get
	getResp, err := srv.GetConversation(ctx, connect.NewRequest(&antv1.GetConversationRequest{
		Id: convID,
	}))
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if getResp.Msg.Conversation.Title != "Test Conversation" {
		t.Errorf("expected title 'Test Conversation', got %q", getResp.Msg.Conversation.Title)
	}

	// 5. Update title
	_, err = srv.UpdateConversationTitle(ctx, connect.NewRequest(&antv1.UpdateConversationTitleRequest{
		Id: convID,
		Title:          "Updated Title",
	}))
	if err != nil {
		t.Fatalf("UpdateConversationTitle: %v", err)
	}

	// 6. Delete
	_, err = srv.DeleteConversation(ctx, connect.NewRequest(&antv1.DeleteConversationRequest{
		Id: convID,
	}))
	if err != nil {
		t.Fatalf("DeleteConversation: %v", err)
	}

	// 7. Verify deleted — should be back to initial count
	listResp3, err := srv.ListConversations(ctx, connect.NewRequest(&antv1.ListConversationsRequest{}))
	if err != nil {
		t.Fatalf("ListConversations after delete: %v", err)
	}
	if len(listResp3.Msg.Conversations) != initialCount {
		t.Errorf("expected %d conversations after delete, got %d", initialCount, len(listResp3.Msg.Conversations))
	}
}

func TestConversations_CrossUserIsolation(t *testing.T) {
	t.Parallel()
	pool := getTestPG(t)

	userA := uuid.New()
	userB := uuid.New()
	insertTestUser(t, pool, userA, fmt.Sprintf("conv-iso-a-%d@test.io", time.Now().UnixNano()))
	insertTestUser(t, pool, userB, fmt.Sprintf("conv-iso-b-%d@test.io", time.Now().UnixNano()))
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM ai_conversations WHERE user_id IN ($1, $2)`, userA, userB)
		pool.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2)`, userA, userB)
	})

	srv := newAIServer(t, pool)

	// User A creates a conversation
	createResp, err := srv.CreateConversation(authCtx(userA), connect.NewRequest(&antv1.CreateConversationRequest{
		Title: "A's Conversation",
	}))
	if err != nil {
		t.Fatalf("CreateConversation for user A: %v", err)
	}
	convAID := createResp.Msg.Conversation.Id

	// User B tries to Get user A's conversation — should fail
	_, err = srv.GetConversation(authCtx(userB), connect.NewRequest(&antv1.GetConversationRequest{
		Id: convAID,
	}))
	if err == nil {
		t.Error("user B should not be able to access user A's conversation")
	}

	// User B tries to Delete user A's conversation — should fail
	_, err = srv.DeleteConversation(authCtx(userB), connect.NewRequest(&antv1.DeleteConversationRequest{
		Id: convAID,
	}))
	if err == nil {
		t.Error("user B should not be able to delete user A's conversation")
	}
}

func TestConversations_InvalidUUID(t *testing.T) {
	t.Parallel()
	pool := getTestPG(t)
	userID := uuid.New()
	insertTestUser(t, pool, userID, fmt.Sprintf("conv-uuid-%d@test.io", time.Now().UnixNano()))
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM ai_conversations WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	srv := newAIServer(t, pool)
	ctx := authCtx(userID)

	_, err := srv.GetConversation(ctx, connect.NewRequest(&antv1.GetConversationRequest{
		Id: "not-a-uuid",
	}))
	if err == nil {
		t.Error("expected error for invalid conversation UUID")
	}
}

func TestConversations_UpdateTitleNotFound(t *testing.T) {
	t.Parallel()
	pool := getTestPG(t)
	userID := uuid.New()
	insertTestUser(t, pool, userID, fmt.Sprintf("conv-404-%d@test.io", time.Now().UnixNano()))
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	srv := newAIServer(t, pool)
	ctx := authCtx(userID)

	_, err := srv.UpdateConversationTitle(ctx, connect.NewRequest(&antv1.UpdateConversationTitleRequest{
		Id: uuid.New().String(),
		Title:          "Ghost",
	}))
	if err == nil {
		t.Error("expected error for non-existent conversation")
	}
}

// ── ListAgents test ──

func TestListAgents(t *testing.T) {
	t.Parallel()
	pool := getTestPG(t)
	userID := uuid.New()
	insertTestUser(t, pool, userID, fmt.Sprintf("ai-listagents-%d@test.io", time.Now().UnixNano()))
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	srv := newAIServer(t, pool)
	ctx := authCtx(userID)

	resp, err := srv.ListAgents(ctx, connect.NewRequest(&antv1.ListAgentsRequest{}))
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	// Each user gets 8 default agents seeded
	if len(resp.Msg.Agents) == 0 {
		t.Error("expected at least one agent, got 0")
	}
	t.Logf("ListAgents returned %d agents", len(resp.Msg.Agents))
}
