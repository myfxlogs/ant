//go:build integration

package admin

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/repository"
	"anttrader/internal/service"

	"connectrpc.com/connect"
)

func resetTestPG(t *testing.T) *pgxpool.Pool {
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

func TestResetUserPasswordNoPlaintextInResponse(t *testing.T) {
	pool := resetTestPG(t)
	ctx := context.Background()
	log := zap.NewNop()

	adminRepo := repository.NewAdminRepository(pool)
	resetRepo := repository.NewPasswordResetRepo(pool)
	platformSvc := service.NewPlatformService(pool)

	// Create test user
	userID := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
		 VALUES ($1, $2, '$argon2id$v=19$m=65536,t=3,p=2$test$test', 'user', 'active', NOW(), NOW())
		 ON CONFLICT (id) DO NOTHING`,
		userID, "test-resetpw@anttest.io",
	)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}

	// Make user an admin (needed for admin interceptor to pass)
	_, err = pool.Exec(ctx,
		`INSERT INTO admins (user_id, scope, created_at) VALUES ($1, 'platform:admin', NOW()) ON CONFLICT DO NOTHING`,
		userID,
	)
	if err != nil {
		t.Fatalf("insert test admin: %v", err)
	}

	_ = platformSvc

	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM password_reset_tokens WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM admins WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	// Run migration
	_, _ = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS password_reset_tokens (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			token TEXT NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			consumed BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)

	server := NewAdminUserServer(adminRepo, resetRepo, log)

	// Test: ResetUserPassword must NOT return plaintext password
	req := connect.NewRequest(&antv1.ResetUserPasswordRequest{Id: userID.String()})
	resp, err := server.ResetUserPassword(ctx, req)
	if err != nil {
		t.Fatalf("ResetUserPassword failed: %v", err)
	}

	if resp.Msg.NewPassword != "" {
		t.Errorf("expected empty NewPassword in response, got %q", resp.Msg.NewPassword)
	} else {
		t.Log("NewPassword is empty in response: PASS")
	}

	// Verify DB stores valid argon2id hash (not plaintext)
	var passwordHash string
	err = pool.QueryRow(ctx,
		`SELECT password_hash FROM users WHERE id = $1`, userID,
	).Scan(&passwordHash)
	if err != nil {
		t.Fatalf("query password_hash: %v", err)
	}

	if len(passwordHash) < 10 || passwordHash[:10] != "$argon2id$" {
		t.Errorf("expected argon2id hash in DB, got: %s", passwordHash[:min(len(passwordHash), 40)])
	} else {
		t.Logf("DB stores valid argon2id hash: %s...", passwordHash[:40])
	}

	// Verify reset token was created
	var tokenCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM password_reset_tokens WHERE user_id = $1 AND consumed = FALSE`,
		userID,
	).Scan(&tokenCount)
	if err != nil {
		t.Fatalf("count reset tokens: %v", err)
	}
	if tokenCount == 0 {
		t.Error("expected at least 1 reset token created")
	} else {
		t.Logf("reset tokens created: %d PASS", tokenCount)
	}
}
