//go:build integration

package service

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func updateAccountTestPG(t *testing.T) *pgxpool.Pool {
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

func TestUpdateAccountParamsCorrect(t *testing.T) {
	pool := updateAccountTestPG(t)
	ctx := context.Background()
	svc := NewAccountService(pool)

	userID := uuid.New()
	accID := uuid.New()

	// Create test user
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
		 VALUES ($1, $2, '$argon2id$v=19$m=65536,t=3,p=2$test$test', 'user', 'active', NOW(), NOW())`,
		userID, "test-updateacct-"+userID.String()[:8]+"@anttest.io",
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Create test account
	_, err = pool.Exec(ctx,
		`INSERT INTO mt_accounts (id, user_id, mt_type, login, password, broker_company, broker_server, broker_host, account_status)
		 VALUES ($1, $2, 'mt5', $3, 'test-pass', 'OriginalBroker', 'OriginalServer', 'demo.mt5.com', 'connected')`,
		accID, userID, "testlogin-"+userID.String()[:8],
	)
	if err != nil {
		t.Fatalf("insert account: %v", err)
	}

	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE id = $1`, accID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	// Call UpdateAccount with new values
	err = svc.UpdateAccount(ctx, userID, accID.String(), "NewBroker", "NewServer", "live.mt5.com", nil)
	if err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}

	// Verify broker_company is the correct value (not a UUID)
	var brokerCompany, brokerServer, brokerHost string
	err = pool.QueryRow(ctx,
		`SELECT broker_company, broker_server, broker_host FROM mt_accounts WHERE id = $1`, accID,
	).Scan(&brokerCompany, &brokerServer, &brokerHost)
	if err != nil {
		t.Fatalf("query result: %v", err)
	}

	if brokerCompany != "NewBroker" {
		t.Errorf("broker_company: expected 'NewBroker', got '%s'", brokerCompany)
	} else {
		t.Log("broker_company correct:", brokerCompany)
	}

	if brokerServer != "NewServer" {
		t.Errorf("broker_server: expected 'NewServer', got '%s'", brokerServer)
	} else {
		t.Log("broker_server correct:", brokerServer)
	}

	if brokerHost != "live.mt5.com" {
		t.Errorf("broker_host: expected 'live.mt5.com', got '%s'", brokerHost)
	} else {
		t.Log("broker_host correct:", brokerHost)
	}

	// Verify is_disabled unchanged (we passed nil, meaning COALESCE should keep existing value)
	var isDisabled bool
	err = pool.QueryRow(ctx,
		`SELECT is_disabled FROM mt_accounts WHERE id = $1`, accID,
	).Scan(&isDisabled)
	if err != nil {
		t.Fatalf("query is_disabled: %v", err)
	}
	if isDisabled {
		t.Errorf("is_disabled: expected false (unchanged), got true")
	} else {
		t.Log("is_disabled unchanged: PASS (COALESCE nil → kept false)")
	}
}
