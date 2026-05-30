//go:build integration

package admin

import (
	"context"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/interceptor"
	"anttrader/internal/service"
)

func getTestPG(t *testing.T) *pgxpool.Pool {
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

func generateTestJWT(t *testing.T, userID uuid.UUID, secret string) string {
	t.Helper()
	claims := &interceptor.JWTClaims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign JWT: %v", err)
	}
	return signed
}

func TestAdminRequiresAdminRole(t *testing.T) {
	pool := getTestPG(t)
	ctx := context.Background()
	log := zap.NewNop()

	jwtSecret := "test-integration-secret-32-bytes!!"
	platformSvc := service.NewPlatformService(pool, nil)
	authInterceptor := interceptor.NewAuthInterceptor(jwtSecret, nil)
	adminInterceptor := interceptor.NewAdminInterceptor(platformSvc, log)

	// Create test users
	nonAdminID := uuid.New()
	adminID := uuid.New()

	// Insert users
	for _, u := range []struct {
		id    uuid.UUID
		email string
		role  string
	}{
		{nonAdminID, "test-nonadmin@anttest.io", "user"},
		{adminID, "test-admin@anttest.io", "admin"},
	} {
		_, err := pool.Exec(ctx,
			`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
			 VALUES ($1, $2, '$argon2id$v=19$m=65536,t=3,p=2$test$test', $3, 'active', NOW(), NOW())
			 ON CONFLICT (id) DO NOTHING`,
			u.id, u.email, u.role,
		)
		if err != nil {
			t.Fatalf("insert test user: %v", err)
		}
	}

	// Make adminID an admin
	_, err := pool.Exec(ctx,
		`INSERT INTO admins (user_id, scope, created_at) VALUES ($1, 'platform:admin', NOW()) ON CONFLICT DO NOTHING`,
		adminID,
	)
	if err != nil {
		t.Fatalf("insert test admin: %v", err)
	}

	// Cleanup after test
	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM admins WHERE user_id IN ($1, $2)`, nonAdminID, adminID)
		pool.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2)`, nonAdminID, adminID)
	})

	// Build a wrapped handler: auth → admin → simple ping handler
	pingHandler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return connect.NewResponse(&antv1.GetDashboardResponse{
			Stats: &antv1.DashboardStats{TotalUsers: 1},
		}), nil
	}

	wrapped := authInterceptor.WrapUnary(adminInterceptor.WrapUnary(pingHandler))

	// Test 1: Non-admin user must get PermissionDenied
	nonAdminToken := generateTestJWT(t, nonAdminID, jwtSecret)
	nonAdminReq := connect.NewRequest(&antv1.GetDashboardRequest{})
	nonAdminReq.Header().Set("Authorization", "Bearer "+nonAdminToken)

	// Build a proper connect request with procedure spec
	_, err = wrapped(ctx, nonAdminReq)
	if err == nil {
		t.Error("expected PermissionDenied for non-admin user, got nil error")
	} else {
		cerr, ok := err.(*connect.Error)
		if !ok {
			t.Errorf("expected connect.Error, got %T: %v", err, err)
		} else if cerr.Code() != connect.CodePermissionDenied {
			t.Errorf("expected PermissionDenied, got %s: %v", cerr.Code(), cerr.Message())
		} else {
			t.Logf("non-admin correctly denied: %s", cerr.Code())
		}
	}

	// Test 2: Admin user must succeed
	adminToken := generateTestJWT(t, adminID, jwtSecret)
	adminReq := connect.NewRequest(&antv1.GetDashboardRequest{})
	adminReq.Header().Set("Authorization", "Bearer "+adminToken)

	resp, err := wrapped(ctx, adminReq)
	if err != nil {
		t.Errorf("expected success for admin user, got: %v", err)
	} else if resp == nil {
		t.Error("expected non-nil response for admin user")
	} else {
		t.Log("admin correctly allowed")
	}

	// Test 3: No auth header must get Unauthenticated
	noAuthReq := connect.NewRequest(&antv1.GetDashboardRequest{})
	_, err = wrapped(ctx, noAuthReq)
	if err == nil {
		t.Error("expected Unauthenticated for missing auth header")
	} else {
		cerr, ok := err.(*connect.Error)
		if !ok {
			t.Errorf("expected connect.Error, got %T: %v", err, err)
		} else if cerr.Code() != connect.CodeUnauthenticated {
			t.Errorf("expected Unauthenticated, got %s", cerr.Code())
		} else {
			t.Log("missing auth correctly denied:", cerr.Code())
		}
	}
}
