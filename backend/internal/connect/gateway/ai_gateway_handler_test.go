//go:build integration

package gateway

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/pkg/secretbox"
	"alphaforge/internal/repository"
)

// FIX-2026-09-08-BYOK-MODEL-PICKER 对抗证明。
//
// SystemModel.provider_id 必须返回 provider 的字符串 id（如 "zhipu"），
// 而不是 system_ai_providers.id UUID：前端模型选择器把该值写入
// users.ai_primary_provider_id，运行时 resolveAllChatProviders /
// resolveGatewayProviders 按字符串比较——存 UUID 则 primary 选择永远无效
// （显示选中 X、运行用默认模型）。
//
// mutation: 还原 ListSystemModels 的 UUID→字符串映射 → 本测试 RED。

func getGatewayTestPG(t *testing.T) *pgxpool.Pool {
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

func newTestAIGatewayServer(t *testing.T, pool *pgxpool.Pool) *AIGatewayServer {
	t.Helper()
	key := []byte("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	box := secretbox.New(key)
	return NewAIGatewayServer(
		repository.NewSystemAIProviderRepository(pool),
		repository.NewAIModelRepository(pool),
		repository.NewAITokenUsageRepository(pool),
		nil, box, zap.NewNop(),
	)
}

func TestListSystemModelsProviderIDIsString(t *testing.T) {
	pool := getGatewayTestPG(t)
	srv := newTestAIGatewayServer(t, pool)
	ctx := context.Background()

	prowID := uuid.New()
	pstr := "zhipu_str_test_" + uuid.NewString()[:8]
	mrowID := uuid.New()
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM ai_models WHERE id=$1`, mrowID)
		_, _ = pool.Exec(ctx, `DELETE FROM system_ai_providers WHERE id=$1`, prowID)
	})

	if _, err := pool.Exec(ctx,
		`INSERT INTO system_ai_providers (id, provider_id, name, base_url, api_key_encrypted, enabled)
		 VALUES ($1, $2, 'StringProviderID Test', 'https://api.example-test.com/v1', $3, true)`,
		prowID, pstr, []byte(nil)); err != nil {
		t.Fatalf("insert provider: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO ai_models (id, provider_id, model_name, display_name, enabled, sort_order)
		 VALUES ($1, $2, 'test-model-string-id', 'Test Model', true, 0)`,
		mrowID, prowID); err != nil {
		t.Fatalf("insert model: %v", err)
	}

	resp, err := srv.ListSystemModels(ctx, connect.NewRequest(&antv1.ListSystemModelsRequest{}))
	if err != nil {
		t.Fatalf("ListSystemModels: %v", err)
	}
	var found *antv1.SystemModel
	for _, m := range resp.Msg.Models {
		if m.ModelName == "test-model-string-id" {
			found = m
			break
		}
	}
	if found == nil {
		t.Fatalf("enabled test model not listed: %d models", len(resp.Msg.Models))
	}
	if found.ProviderId != pstr {
		t.Fatalf("SystemModel.provider_id = %q, want string provider_id %q (UUID leaks into users.ai_primary_provider_id)", found.ProviderId, pstr)
	}
}
