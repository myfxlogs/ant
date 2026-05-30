//go:build integration

package user

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
	"anttrader/internal/mdgateway/adapter/brokersearch"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/service"
)

// mockMTTester implements MTConnectionTester for integration tests.
// It avoids actual MT broker connections while letting the handler
// exercise the full flow.
type mockMTTester struct {
	info *mdtick.MTAccountInfo
	err  error
}

func (m *mockMTTester) Test(ctx context.Context, platform, brokerHost, login, password string) (*mdtick.MTAccountInfo, error) {
	return m.info, m.err
}

func (m *mockMTTester) VerifyPassword(ctx context.Context, platform, brokerHost, login, password string) error {
	return m.err
}

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
		// Always use localhost from the test runner (not the Docker service name).
		// The docker port mapping exposes PostgreSQL on localhost:5433.
		port := "5433"
		dsn = "postgres://" + user + ":" + password + "@localhost:" + port + "/" + dbname + "?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping integration test: pg connect: %v", err)
	}
	// Verify the connection is actually usable (pgxpool.New is lazy).
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

// authCtx injects a user ID into the context, simulating an authenticated request.
func authCtx(userID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), interceptor.UserIDKey, userID.String())
}

// newTestAccountServer creates an AccountServer wired with real DB dependencies
// and a mock MT tester.
func newTestAccountServer(t *testing.T, pool *pgxpool.Pool, tester MTConnectionTester) *AccountServer {
	t.Helper()
	svc := service.NewAccountService(pool)
	searcher := brokersearch.New("", "")
	return NewAccountServer(svc, searcher, nil, tester, zap.NewNop())
}

// TestAccountLifecycle covers the full SearchBroker → VerifyAccount → CreateAccount →
// GetAccount → ConnectAccount → DisconnectAccount lifecycle using a mock MT tester.
func TestAccountLifecycle(t *testing.T) {
	t.Parallel()
	pool := getTestPG(t)
	log := zap.NewNop()

	userID := uuid.New()
	insertTestUser(t, pool, userID, "test-lifecycle@anttest.io")
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	mockTester := &mockMTTester{
		info: &mdtick.MTAccountInfo{
			Balance:    10000.0,
			Equity:     10100.0,
			Credit:     0,
			Margin:     500.0,
			FreeMargin: 9600.0,
			Leverage:   100,
			Currency:   "USD",
		},
	}

	svc := service.NewAccountService(pool)
	searcher := brokersearch.New("", "")
	srv := NewAccountServer(svc, searcher, nil, mockTester, log)

	ctx := authCtx(userID)

	// 1. SearchBroker — should return results or static fallback
	t.Run("SearchBroker", func(t *testing.T) {
		searchCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		req := connect.NewRequest(&antv1.SearchBrokerRequest{Company: "Robo", MtType: "mt5"})
		resp, err := srv.SearchBroker(searchCtx, req)
		if err != nil {
			t.Fatalf("SearchBroker unexpected error: %v", err)
		}
		companies := resp.Msg.GetCompanies()
		if len(companies) == 0 {
			t.Error("expected non-empty company results (real search or static fallback)")
		} else {
			t.Logf("SearchBroker returned %d broker companies (fallback or live)", len(companies))
		}
	})

	// 2. VerifyAccount — mock tester returns verified info with balance/equity
	t.Run("VerifyAccount", func(t *testing.T) {
		req := connect.NewRequest(&antv1.VerifyAccountRequest{
			Login:      "12345",
			Password:   "testpass",
			MtType:     "mt5",
			BrokerHost: "demo.example.com:443",
		})
		resp, err := srv.VerifyAccount(ctx, req)
		if err != nil {
			t.Fatalf("VerifyAccount unexpected error: %v", err)
		}
		if !resp.Msg.GetVerified() {
			t.Error("expected verified=true from mock tester")
		}
		if resp.Msg.GetBalance() != 10000.0 {
			t.Errorf("expected balance=10000, got %v", resp.Msg.GetBalance())
		}
		if resp.Msg.GetEquity() != 10100.0 {
			t.Errorf("expected equity=10100, got %v", resp.Msg.GetEquity())
		}
		t.Logf("VerifyAccount: verified=%v balance=%.2f equity=%.2f currency=%s",
			resp.Msg.GetVerified(), resp.Msg.GetBalance(), resp.Msg.GetEquity(), resp.Msg.GetCurrency())
	})

	var accountID string

	// 3. CreateAccount — creates account with status "connecting", mock verifies credentials
	t.Run("CreateAccount", func(t *testing.T) {
		req := connect.NewRequest(&antv1.CreateAccountRequest{
			Login:         "test-lifecycle-login",
			Password:      "testpass",
			MtType:        "mt5",
			BrokerCompany: "TestBroker",
			BrokerServer:  "TestServer",
			BrokerHost:    "demo.example.com:443",
		})
		resp, err := srv.CreateAccount(ctx, req)
		if err != nil {
			t.Fatalf("CreateAccount unexpected error: %v", err)
		}
		acct := resp.Msg
		if acct.GetId() == "" {
			t.Fatal("expected non-empty account ID")
		}
		accountID = acct.GetId()
		if acct.GetLogin() != "test-lifecycle-login" {
			t.Errorf("expected login=test-lifecycle-login, got %s", acct.GetLogin())
		}
		// After successful MT verification, status should be "connected"
		if acct.GetStatus() != "connected" {
			t.Errorf("expected status=connected after successful MT verification, got %s", acct.GetStatus())
		}
		if acct.GetBalance() != 10000.0 {
			t.Errorf("expected balance=10000, got %v", acct.GetBalance())
		}
		t.Logf("CreateAccount: id=%s login=%s status=%s balance=%.2f",
			acct.GetId(), acct.GetLogin(), acct.GetStatus(), acct.GetBalance())
	})

	// 4. GetAccount — verify account exists and has correct fields
	t.Run("GetAccount", func(t *testing.T) {
		req := connect.NewRequest(&antv1.GetAccountRequest{Id: accountID})
		resp, err := srv.GetAccount(ctx, req)
		if err != nil {
			t.Fatalf("GetAccount unexpected error: %v", err)
		}
		acct := resp.Msg
		if acct.GetId() != accountID {
			t.Errorf("expected id=%s, got %s", accountID, acct.GetId())
		}
		if acct.GetLogin() != "test-lifecycle-login" {
			t.Errorf("expected login=test-lifecycle-login, got %s", acct.GetLogin())
		}
		if acct.GetBrokerCompany() != "TestBroker" {
			t.Errorf("expected broker_company=TestBroker, got %s", acct.GetBrokerCompany())
		}
		t.Logf("GetAccount: id=%s login=%s broker=%s status=%s",
			acct.GetId(), acct.GetLogin(), acct.GetBrokerCompany(), acct.GetStatus())
	})

	// 5. ConnectAccount — verify success response
	t.Run("ConnectAccount", func(t *testing.T) {
		req := connect.NewRequest(&antv1.ConnectAccountRequest{Id: accountID})
		resp, err := srv.ConnectAccount(ctx, req)
		if err != nil {
			t.Fatalf("ConnectAccount unexpected error: %v", err)
		}
		if !resp.Msg.GetSuccess() {
			t.Error("expected ConnectAccount success=true")
		}
		t.Logf("ConnectAccount: success=%v message=%s", resp.Msg.GetSuccess(), resp.Msg.GetMessage())
	})

	// 6. DisconnectAccount — verify the account still exists (no error)
	t.Run("DisconnectAccount", func(t *testing.T) {
		req := connect.NewRequest(&antv1.DisconnectAccountRequest{Id: accountID})
		_, err := srv.DisconnectAccount(ctx, req)
		if err != nil {
			t.Fatalf("DisconnectAccount unexpected error: %v", err)
		}
		t.Log("DisconnectAccount: success, account still exists")

		// Verify the account is still accessible after disconnect
		getReq := connect.NewRequest(&antv1.GetAccountRequest{Id: accountID})
		getResp, err := srv.GetAccount(ctx, getReq)
		if err != nil {
			t.Fatalf("GetAccount after disconnect unexpected error: %v", err)
		}
		if getResp.Msg.GetStatus() != "disconnected" {
			t.Errorf("expected status=disconnected, got %s", getResp.Msg.GetStatus())
		}
		t.Logf("Account status after disconnect: %s", getResp.Msg.GetStatus())
	})
}

// TestDuplicateAccountBindingBlocked verifies that binding two accounts with the same
// login+mt_type for the same user returns CodeAlreadyExists.
func TestDuplicateAccountBindingBlocked(t *testing.T) {
	t.Parallel()
	pool := getTestPG(t)

	userID := uuid.New()
	insertTestUser(t, pool, userID, "test-dup@anttest.io")
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	// Use nil MT tester so CreateAccount skips verification (we only test DB uniqueness)
	srv := newTestAccountServer(t, pool, nil)
	ctx := authCtx(userID)

	// First account
	createReq := connect.NewRequest(&antv1.CreateAccountRequest{
		Login:         "test-dup",
		Password:      "pass1",
		MtType:        "mt5",
		BrokerCompany: "TestBroker",
		BrokerServer:  "TestServer",
		BrokerHost:    "demo.example.com:443",
	})
	resp1, err := srv.CreateAccount(ctx, createReq)
	if err != nil {
		t.Fatalf("first CreateAccount unexpected error: %v", err)
	}
	t.Logf("First account created: id=%s", resp1.Msg.GetId())

	// Second account with same login+mt_type — should fail with CodeAlreadyExists
	createReq2 := connect.NewRequest(&antv1.CreateAccountRequest{
		Login:         "test-dup",
		Password:      "pass2",
		MtType:        "mt5",
		BrokerCompany: "TestBroker",
		BrokerServer:  "TestServer",
		BrokerHost:    "demo.example.com:443",
	})
	_, err = srv.CreateAccount(ctx, createReq2)
	if err == nil {
		t.Error("expected CodeAlreadyExists error for duplicate login, got nil")
	} else {
		cerr, ok := err.(*connect.Error)
		if !ok {
			t.Errorf("expected connect.Error, got %T: %v", err, err)
		} else if cerr.Code() != connect.CodeAlreadyExists {
			t.Errorf("expected CodeAlreadyExists, got %s: %v", cerr.Code(), cerr.Message())
		} else {
			t.Logf("duplicate correctly blocked: %s — %s", cerr.Code(), cerr.Message())
		}
	}
}

// TestInvalidUUIDReturnsError verifies that GetAccount with an invalid UUID
// returns an error. The handler returns CodeInternal because the UUID parse
// failure in the service layer is not wrapped as a specific connect error code.
func TestInvalidUUIDReturnsError(t *testing.T) {
	t.Parallel()
	pool := getTestPG(t)

	userID := uuid.New()
	insertTestUser(t, pool, userID, "test-invalid-uuid@anttest.io")
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	srv := newTestAccountServer(t, pool, nil)
	ctx := authCtx(userID)

	req := connect.NewRequest(&antv1.GetAccountRequest{Id: "not-a-valid-uuid"})
	_, err := srv.GetAccount(ctx, req)
	if err == nil {
		t.Error("expected error for invalid UUID, got nil")
	} else {
		cerr, ok := err.(*connect.Error)
		if !ok {
			t.Errorf("expected connect.Error, got %T: %v", err, err)
		} else if cerr.Code() != connect.CodeInternal {
			t.Errorf("expected CodeInternal for invalid UUID, got %s", cerr.Code())
		} else {
			t.Logf("invalid UUID correctly rejected: %s — %s", cerr.Code(), cerr.Message())
		}
	}
}

// TestAccountOwnershipEnforced verifies that user B cannot access an account
// created by user A. The handler returns CodeNotFound because the DB query
// filters by user_id, making the account invisible to non-owners.
func TestAccountOwnershipEnforced(t *testing.T) {
	t.Parallel()
	pool := getTestPG(t)

	userA := uuid.New()
	userB := uuid.New()
	insertTestUser(t, pool, userA, "test-owner-a@anttest.io")
	insertTestUser(t, pool, userB, "test-owner-b@anttest.io")
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE user_id IN ($1, $2)`, userA, userB)
		pool.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2)`, userA, userB)
	})

	srv := newTestAccountServer(t, pool, nil)
	ctxA := authCtx(userA)
	ctxB := authCtx(userB)

	// User A creates an account
	createReq := connect.NewRequest(&antv1.CreateAccountRequest{
		Login:         "test-ownership",
		Password:      "pass",
		MtType:        "mt5",
		BrokerCompany: "TestBroker",
		BrokerServer:  "TestServer",
		BrokerHost:    "demo.example.com:443",
	})
	resp, err := srv.CreateAccount(ctxA, createReq)
	if err != nil {
		t.Fatalf("user A CreateAccount unexpected error: %v", err)
	}
	accountID := resp.Msg.GetId()
	t.Logf("User A created account: id=%s", accountID)

	// User A can access their own account
	getReq := connect.NewRequest(&antv1.GetAccountRequest{Id: accountID})
	_, err = srv.GetAccount(ctxA, getReq)
	if err != nil {
		t.Fatalf("user A GetAccount unexpected error: %v", err)
	}
	t.Log("User A can access their own account")

	// User B cannot access user A's account — should return NotFound
	_, err = srv.GetAccount(ctxB, getReq)
	if err == nil {
		t.Error("expected error when user B accesses user A's account, got nil")
	} else {
		cerr, ok := err.(*connect.Error)
		if !ok {
			t.Errorf("expected connect.Error, got %T: %v", err, err)
		} else if cerr.Code() != connect.CodeNotFound {
			t.Errorf("expected CodeNotFound for cross-user access, got %s", cerr.Code())
		} else {
			t.Logf("ownership enforced: user B access denied — %s: %s", cerr.Code(), cerr.Message())
		}
	}
}

// TestPasswordChangeWithOldPassword verifies the UpdateTradingPassword flow:
// correct oldPassword succeeds, wrong oldPassword fails with password mismatch.
func TestPasswordChangeWithOldPassword(t *testing.T) {
	t.Parallel()
	pool := getTestPG(t)

	userID := uuid.New()
	insertTestUser(t, pool, userID, "test-password@anttest.io")
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM mt_accounts WHERE user_id = $1`, userID)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	// Use nil MT tester — UpdateTradingPassword does not call the MT tester
	srv := newTestAccountServer(t, pool, nil)
	ctx := authCtx(userID)

	knownPassword := "original-pass"

	// Create an account with a known password
	createReq := connect.NewRequest(&antv1.CreateAccountRequest{
		Login:         "test-pw-change",
		Password:      knownPassword,
		MtType:        "mt5",
		BrokerCompany: "TestBroker",
		BrokerServer:  "TestServer",
		BrokerHost:    "demo.example.com:443",
	})
	resp, err := srv.CreateAccount(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateAccount unexpected error: %v", err)
	}
	accountID := resp.Msg.GetId()
	t.Logf("Account created: id=%s login=test-pw-change", accountID)

	// Test 1: Change password with correct oldPassword — should succeed
	t.Run("correct oldPassword succeeds", func(t *testing.T) {
		req := connect.NewRequest(&antv1.UpdateTradingPasswordRequest{
			Id:          accountID,
			OldPassword: knownPassword,
			NewPassword: "new-secure-pass",
		})
		resp, err := srv.UpdateTradingPassword(ctx, req)
		if err != nil {
			t.Fatalf("UpdateTradingPassword with correct oldPassword unexpected error: %v", err)
		}
		if !resp.Msg.GetSuccess() {
			t.Error("expected success=true when oldPassword matches")
		}
		t.Log("password changed successfully with correct oldPassword")
	})

	// Test 2: Change password with wrong oldPassword — should fail
	t.Run("wrong oldPassword fails", func(t *testing.T) {
		req := connect.NewRequest(&antv1.UpdateTradingPasswordRequest{
			Id:          accountID,
			OldPassword: "wrong-old-password",
			NewPassword: "another-new-pass",
		})
		_, err := srv.UpdateTradingPassword(ctx, req)
		if err == nil {
			t.Error("expected error when oldPassword is wrong, got nil")
		} else {
			cerr, ok := err.(*connect.Error)
			if !ok {
				t.Errorf("expected connect.Error, got %T: %v", err, err)
			} else if cerr.Code() != connect.CodeInvalidArgument {
				t.Errorf("expected CodeInvalidArgument for password mismatch, got %s", cerr.Code())
			} else {
				t.Logf("password mismatch correctly rejected: %s — %s", cerr.Code(), cerr.Message())
			}
		}
	})
}

// TestErrorRecovery_PGUnavailable verifies handlers return proper ConnectRPC
// errors (not panics, not nil responses) when the database is unavailable.
func TestErrorRecovery_PGUnavailable(t *testing.T) {
	pool := getTestPG(t)
	

	userID := uuid.New()
	insertTestUser(t, pool, userID, fmt.Sprintf("err-recov-%d@test.io", time.Now().UnixNano()))
	srv := newTestAccountServer(t, pool, nil)
	userCtx := authCtx(userID)

	// Verify normal operation works first.
	req := connect.NewRequest(&antv1.ListAccountsRequest{})
	resp, err := srv.ListAccounts(userCtx, req)
	if err != nil {
		t.Fatalf("ListAccounts should succeed with live DB: %v", err)
	}
	t.Logf("Normal: %d accounts listed", len(resp.Msg.Accounts))

	// Close the pool to simulate PG outage.
	pool.Close()

	// Same request after PG is down — should get Internal error, not panic.
	_, err = srv.ListAccounts(userCtx, req)
	if err == nil {
		t.Error("ListAccounts should return error when PG is unavailable")
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Errorf("expected ConnectRPC error, got %T: %v", err, err)
	} else {
		t.Logf("PG unavailable: correctly returned %v", connectErr.Code())
	}

	// Also verify GetAccount returns error, not nil response.
	req2 := connect.NewRequest(&antv1.GetAccountRequest{Id: uuid.New().String()})
	_, err = srv.GetAccount(userCtx, req2)
	if err == nil {
		t.Error("GetAccount should return error when PG is unavailable")
	} else {
		t.Logf("GetAccount with PG down: correctly returned %v", err)
	}
}
