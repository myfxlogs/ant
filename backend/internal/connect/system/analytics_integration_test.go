//go:build integration

package system

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

// --- test helpers ---

func getTestPG(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		password := os.Getenv("DB_PASSWORD")
		user := os.Getenv("DB_USER")
		if user == "" { user = "ant" }
		dbname := os.Getenv("DB_NAME")
		if dbname == "" { dbname = "ant" }
		port := "5433"
		dsn = "postgres://" + user + ":" + password + "@localhost:" + port + "/" + dbname + "?sslmode=disable"
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

func getTestRedis(t *testing.T) *goredis.Client {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	client := goredis.NewClient(&goredis.Options{
		Addr: addr,
		DB:   1, // use DB 1 for tests to avoid polluting dev data
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Skipf("skipping integration test: redis connect: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

// createTestUser inserts a user row and returns the user ID.
func createTestUser(t *testing.T, pool *pgxpool.Pool, email string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	ctx := context.Background()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
		 VALUES ($1, $2, '$argon2id$v=19$m=65536,t=3,p=2$test$test', 'user', 'active', NOW(), NOW())
		 ON CONFLICT (id) DO NOTHING`,
		id, email,
	)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

// createTestAccount inserts an mt_account row for the given user and returns the account ID.
func createTestAccount(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, login string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	ctx := context.Background()
	_, err := pool.Exec(ctx,
		`INSERT INTO mt_accounts (
			id, user_id, mt_type, broker_company, broker_server, broker_host,
			login, password, alias, is_disabled, balance, credit, equity,
			margin, free_margin, margin_level, leverage, currency,
			account_method, account_type, is_investor, account_status,
			stream_status, mt_token, broker_margin_call_pct, broker_stop_out_pct,
			last_error, created_at, updated_at
		) VALUES (
			$1, $2, 'MT5', 'TestBroker', 'TestServer', 'test.example.com',
			$3, 'encrypted', 'test-account', false, 10000, 0, 10000,
			0, 10000, 0, 100, 'USD',
			'hedging', 'demo', false, 'connected',
			'inactive', '', 100, 50,
			'', NOW(), NOW()
		) ON CONFLICT (id) DO NOTHING`,
		id, userID, login,
	)
	if err != nil {
		t.Fatalf("insert test account: %v", err)
	}
	// Insert a balance history record so equity curve queries work.
	_, err = pool.Exec(ctx,
		`INSERT INTO account_balance_history (account_id, balance, equity, margin, free_margin, recorded_at)
		 VALUES ($1, 10000, 10000, 0, 10000, NOW())`,
		id,
	)
	if err != nil {
		t.Fatalf("insert balance history: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM account_balance_history WHERE account_id = $1`, id)
		pool.Exec(context.Background(), `DELETE FROM trade_records WHERE account_id = $1`, id)
		pool.Exec(context.Background(), `DELETE FROM mt_accounts WHERE id = $1`, id)
	})
	return id
}

// insertTradeRecord inserts a single trade record directly via SQL.
func insertTradeRecord(t *testing.T, pool *pgxpool.Pool, accountID uuid.UUID, ticket int64, symbol string,
	orderType string, volume float64, openPrice float64, closePrice float64,
	profit float64, openTime time.Time, closeTime time.Time) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx,
		`INSERT INTO trade_records (
			id, account_id, ticket, symbol, order_type, volume,
			open_price, close_price, profit, swap, commission,
			open_time, close_time, stop_loss, take_profit,
			order_comment, magic_number, platform, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, 0, 0,
			$10, $11, 0, 0, '', 0, 'MT5', NOW(), NOW()
		) ON CONFLICT (account_id, ticket, close_time) DO UPDATE SET
			profit = EXCLUDED.profit,
			updated_at = NOW()`,
		uuid.New(), accountID, ticket, symbol, orderType, volume,
		openPrice, closePrice, profit, openTime, closeTime,
	)
	if err != nil {
		t.Fatalf("insert trade record: %v", err)
	}
}

// buildTestAnalyticsServer creates a real AnalyticsServer for integration tests.
func buildTestAnalyticsServer(t *testing.T, pool *pgxpool.Pool, redisClient *goredis.Client) *AnalyticsServer {
	t.Helper()
	log := zap.NewNop()
	analyticsRepo := repository.NewAnalyticsRepository(pool)
	accountSvc := service.NewAccountService(pool)
	platformSvc := service.NewPlatformService(pool, accountSvc)
	var cache *service.AnalyticsCache
	if redisClient != nil {
		cache = service.NewAnalyticsCache(redisClient, log)
	}
	return NewAnalyticsServer(analyticsRepo, platformSvc, cache, log)
}

// ctxWithUserID returns a context with the given user ID set (bypasses JWT auth).
func ctxWithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, interceptor.UserIDKey, userID.String())
}

// --- Test 1: Analytics cache hit/miss ---

func TestAnalyticsCacheHitMiss(t *testing.T) {
	pool := getTestPG(t)
	redisClient := getTestRedis(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Flush test DB before test to ensure clean cache state.
	if err := redisClient.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flush redis test db: %v", err)
	}

	userID := createTestUser(t, pool, fmt.Sprintf("cache-hit-miss-%d@test.io", rand.Intn(100000)))
	accountID := createTestAccount(t, pool, userID, fmt.Sprintf("login-%d", rand.Intn(100000)))

	// Insert some winning and losing trades.
	now := time.Now()
	insertTradeRecord(t, pool, accountID, 1001, "EURUSD", "buy", 0.1, 1.0800, 1.0850, 50.0,
		now.Add(-48*time.Hour), now.Add(-47*time.Hour))
	insertTradeRecord(t, pool, accountID, 1002, "EURUSD", "sell", 0.1, 1.0860, 1.0830, 30.0,
		now.Add(-24*time.Hour), now.Add(-23*time.Hour))
	insertTradeRecord(t, pool, accountID, 1003, "GBPUSD", "buy", 0.2, 1.2500, 1.2480, -40.0,
		now.Add(-12*time.Hour), now.Add(-11*time.Hour))

	server := buildTestAnalyticsServer(t, pool, redisClient)
	userCtx := ctxWithUserID(ctx, userID)

	// First call — cache miss, should go through full query path.
	start1 := time.Now()
	req1 := connect.NewRequest(&antv1.GetAccountAnalyticsRequest{
		AccountId: accountID.String(),
	})
	resp1, err := server.GetAccountAnalytics(userCtx, req1)
	dur1 := time.Since(start1)
	if err != nil {
		t.Fatalf("first GetAccountAnalytics failed: %v", err)
	}
	ts1 := resp1.Msg.GetTradeStats()
	if ts1 == nil {
		t.Fatal("first response has nil trade stats")
	}
	if ts1.TotalTrades != 3 {
		t.Errorf("first call: expected 3 total trades, got %d", ts1.TotalTrades)
	}
	if ts1.NetProfit == 0 {
		t.Error("first call: expected non-zero net profit")
	}
	t.Logf("First call (cache miss) took %v, total_trades=%d", dur1, ts1.TotalTrades)

	// Second call — cache hit, should be much faster.
	start2 := time.Now()
	req2 := connect.NewRequest(&antv1.GetAccountAnalyticsRequest{
		AccountId: accountID.String(),
	})
	resp2, err := server.GetAccountAnalytics(userCtx, req2)
	dur2 := time.Since(start2)
	if err != nil {
		t.Fatalf("second GetAccountAnalytics failed: %v", err)
	}
	ts2 := resp2.Msg.GetTradeStats()
	if ts2 == nil {
		t.Fatal("second response has nil trade stats")
	}
	if ts2.TotalTrades != 3 {
		t.Errorf("second call: expected 3 total trades, got %d", ts2.TotalTrades)
	}
	t.Logf("Second call (cache hit) took %v", dur2)

	// Verify cache hit is fast (< 50ms).
	if dur2 > 50*time.Millisecond {
		t.Errorf("cache hit took %v, expected < 50ms", dur2)
	}
}

// --- Test 2: Analytics cache invalidation ---

func TestAnalyticsCacheInvalidation(t *testing.T) {
	pool := getTestPG(t)
	redisClient := getTestRedis(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := redisClient.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flush redis test db: %v", err)
	}

	userID := createTestUser(t, pool, fmt.Sprintf("cache-inval-%d@test.io", rand.Intn(100000)))
	accountID := createTestAccount(t, pool, userID, fmt.Sprintf("login-%d", rand.Intn(100000)))

	now := time.Now()
	insertTradeRecord(t, pool, accountID, 2001, "EURUSD", "buy", 0.1, 1.0800, 1.0850, 50.0,
		now.Add(-48*time.Hour), now.Add(-47*time.Hour))
	insertTradeRecord(t, pool, accountID, 2002, "GBPUSD", "sell", 0.2, 1.2500, 1.2480, -40.0,
		now.Add(-24*time.Hour), now.Add(-23*time.Hour))

	server := buildTestAnalyticsServer(t, pool, redisClient)
	userCtx := ctxWithUserID(ctx, userID)

	// First call — fill the cache.
	req1 := connect.NewRequest(&antv1.GetAccountAnalyticsRequest{
		AccountId: accountID.String(),
	})
	resp1, err := server.GetAccountAnalytics(userCtx, req1)
	if err != nil {
		t.Fatalf("first GetAccountAnalytics failed: %v", err)
	}
	if resp1.Msg.GetTradeStats().TotalTrades != 2 {
		t.Fatalf("expected 2 trades before invalidation, got %d", resp1.Msg.GetTradeStats().TotalTrades)
	}
	t.Logf("Before invalidation: %d trades", resp1.Msg.GetTradeStats().TotalTrades)

	// Add a new trade record.
	insertTradeRecord(t, pool, accountID, 2003, "USDJPY", "buy", 0.3, 150.00, 151.00, 100.0,
		now.Add(-6*time.Hour), now.Add(-5*time.Hour))

	// Invalidate the cache.
	if server.cache != nil {
		server.cache.Invalidate(ctx, accountID.String())
		t.Log("Cache invalidated")
	}

	// Second call — should get fresh data including the new trade.
	req2 := connect.NewRequest(&antv1.GetAccountAnalyticsRequest{
		AccountId: accountID.String(),
	})
	resp2, err := server.GetAccountAnalytics(userCtx, req2)
	if err != nil {
		t.Fatalf("second GetAccountAnalytics after invalidation failed: %v", err)
	}
	ts2 := resp2.Msg.GetTradeStats()
	if ts2.TotalTrades != 3 {
		t.Errorf("after invalidation: expected 3 total trades, got %d", ts2.TotalTrades)
	}
	if ts2.NetProfit != 110.0 {
		t.Errorf("after invalidation: expected net profit 110, got %f", ts2.NetProfit)
	}
	t.Logf("After invalidation: %d trades, net_profit=%f", ts2.TotalTrades, ts2.NetProfit)
}

// --- Test 3: Cross-account auth enforcement ---

func TestAnalyticsCrossAccountAuth(t *testing.T) {
	pool := getTestPG(t)
	// No Redis needed for this test — auth check happens before cache.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create two users with one account each.
	userA := createTestUser(t, pool, fmt.Sprintf("cross-auth-a-%d@test.io", rand.Intn(100000)))
	userB := createTestUser(t, pool, fmt.Sprintf("cross-auth-b-%d@test.io", rand.Intn(100000)))

	accountA := createTestAccount(t, pool, userA, fmt.Sprintf("login-a-%d", rand.Intn(100000)))
	accountB := createTestAccount(t, pool, userB, fmt.Sprintf("login-b-%d", rand.Intn(100000)))

	// Insert some trades for both accounts.
	now := time.Now()
	insertTradeRecord(t, pool, accountA, 3001, "EURUSD", "buy", 0.1, 1.0800, 1.0850, 50.0,
		now.Add(-48*time.Hour), now.Add(-47*time.Hour))
	insertTradeRecord(t, pool, accountB, 3002, "GBPUSD", "sell", 0.2, 1.2500, 1.2480, -40.0,
		now.Add(-24*time.Hour), now.Add(-23*time.Hour))

	server := buildTestAnalyticsServer(t, pool, nil)

	// User A tries to access User B's account — should be denied.
	userACtx := ctxWithUserID(ctx, userA)
	req := connect.NewRequest(&antv1.GetAccountAnalyticsRequest{
		AccountId: accountB.String(),
	})
	_, err := server.GetAccountAnalytics(userACtx, req)
	if err == nil {
		t.Error("expected PermissionDenied for cross-account access, got nil error")
	} else {
		cerr, ok := err.(*connect.Error)
		if !ok {
			t.Errorf("expected connect.Error, got %T: %v", err, err)
		} else if cerr.Code() != connect.CodePermissionDenied {
			t.Errorf("expected PermissionDenied, got %s: %v", cerr.Code(), cerr.Message())
		} else {
			t.Logf("cross-account access correctly denied: %s", cerr.Code())
		}
	}

	// User A accessing their own account should succeed.
	ownReq := connect.NewRequest(&antv1.GetAccountAnalyticsRequest{
		AccountId: accountA.String(),
	})
	resp, err := server.GetAccountAnalytics(userACtx, ownReq)
	if err != nil {
		t.Errorf("user A accessing own account should succeed, got: %v", err)
	} else if resp.Msg.GetTradeStats().TotalTrades != 1 {
		t.Errorf("user A own account: expected 1 trade, got %d", resp.Msg.GetTradeStats().TotalTrades)
	} else {
		t.Log("user A accessing own account: OK")
	}

	// User B accessing their own account should also succeed.
	userBCtx := ctxWithUserID(ctx, userB)
	ownReqB := connect.NewRequest(&antv1.GetAccountAnalyticsRequest{
		AccountId: accountB.String(),
	})
	respB, err := server.GetAccountAnalytics(userBCtx, ownReqB)
	if err != nil {
		t.Errorf("user B accessing own account should succeed, got: %v", err)
	} else if respB.Msg.GetTradeStats().TotalTrades != 1 {
		t.Errorf("user B own account: expected 1 trade, got %d", respB.Msg.GetTradeStats().TotalTrades)
	} else {
		t.Log("user B accessing own account: OK")
	}
}

// --- Test 4: GetRecentTrades with pagination ---

func TestAnalyticsGetRecentTrades(t *testing.T) {
	pool := getTestPG(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID := createTestUser(t, pool, fmt.Sprintf("recent-trades-%d@test.io", rand.Intn(100000)))
	accountID := createTestAccount(t, pool, userID, fmt.Sprintf("login-%d", rand.Intn(100000)))

	// Insert 12 trade records with distinct close times (newest first in DESC order).
	now := time.Now()
	for i := int64(1); i <= 12; i++ {
		symbol := "EURUSD"
		if i%2 == 0 {
			symbol = "GBPUSD"
		}
		profit := float64(i*10) - 20 // mix of positive and negative
		insertTradeRecord(t, pool, accountID, 4000+i, symbol, "buy", 0.1,
			1.0800, 1.0850, profit,
			now.Add(-time.Duration(i)*time.Hour-time.Minute),
			now.Add(-time.Duration(i)*time.Hour))
	}

	server := buildTestAnalyticsServer(t, pool, nil)
	userCtx := ctxWithUserID(ctx, userID)

	// Page 1, pageSize 5.
	req := connect.NewRequest(&antv1.GetRecentTradesRequest{
		AccountId: accountID.String(),
		Page:      1,
		PageSize:  5,
	})
	resp, err := server.GetRecentTrades(userCtx, req)
	if err != nil {
		t.Fatalf("GetRecentTrades failed: %v", err)
	}
	if resp.Msg.Total != 12 {
		t.Errorf("expected total=12, got %d", resp.Msg.Total)
	}
	if len(resp.Msg.Trades) != 5 {
		t.Errorf("expected 5 trades on page 1, got %d", len(resp.Msg.Trades))
	}
	t.Logf("Page 1: got %d trades, total=%d", len(resp.Msg.Trades), resp.Msg.Total)

	// Page 3 (last page, should have 2 trades: 12 - 2*5 = 2).
	req3 := connect.NewRequest(&antv1.GetRecentTradesRequest{
		AccountId: accountID.String(),
		Page:      3,
		PageSize:  5,
	})
	resp3, err := server.GetRecentTrades(userCtx, req3)
	if err != nil {
		t.Fatalf("GetRecentTrades page 3 failed: %v", err)
	}
	if resp3.Msg.Total != 12 {
		t.Errorf("page 3: expected total=12, got %d", resp3.Msg.Total)
	}
	if len(resp3.Msg.Trades) != 2 {
		t.Errorf("page 3: expected 2 trades, got %d", len(resp3.Msg.Trades))
	}
	t.Logf("Page 3: got %d trades, total=%d", len(resp3.Msg.Trades), resp3.Msg.Total)
}

// --- Test 5: Monthly PnL calculation ---

func TestAnalyticsMonthlyPnL(t *testing.T) {
	pool := getTestPG(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID := createTestUser(t, pool, fmt.Sprintf("monthly-pnl-%d@test.io", rand.Intn(100000)))
	accountID := createTestAccount(t, pool, userID, fmt.Sprintf("login-%d", rand.Intn(100000)))

	// Insert trades in Jan, Feb, Mar 2025.
	// January: 2 winning trades = +100
	insertTradeRecord(t, pool, accountID, 5001, "EURUSD", "buy", 0.1, 1.0800, 1.0850, 50.0,
		time.Date(2025, 1, 10, 10, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 10, 14, 0, 0, 0, time.UTC))
	insertTradeRecord(t, pool, accountID, 5002, "GBPUSD", "sell", 0.2, 1.2500, 1.2480, 50.0,
		time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC))

	// February: 1 losing trade = -30
	insertTradeRecord(t, pool, accountID, 5003, "USDJPY", "buy", 0.3, 150.00, 149.50, -30.0,
		time.Date(2025, 2, 5, 10, 0, 0, 0, time.UTC),
		time.Date(2025, 2, 5, 14, 0, 0, 0, time.UTC))

	// March: 2 trades, +80
	insertTradeRecord(t, pool, accountID, 5004, "AUDUSD", "buy", 0.1, 0.6500, 0.6550, 50.0,
		time.Date(2025, 3, 8, 10, 0, 0, 0, time.UTC),
		time.Date(2025, 3, 8, 14, 0, 0, 0, time.UTC))
	insertTradeRecord(t, pool, accountID, 5005, "NZDUSD", "sell", 0.2, 0.6100, 0.6080, 30.0,
		time.Date(2025, 3, 20, 10, 0, 0, 0, time.UTC),
		time.Date(2025, 3, 20, 14, 0, 0, 0, time.UTC))

	server := buildTestAnalyticsServer(t, pool, nil)
	userCtx := ctxWithUserID(ctx, userID)

	req := connect.NewRequest(&antv1.GetMonthlyPnLRequest{
		AccountId: accountID.String(),
		Year:      2025,
	})
	resp, err := server.GetMonthlyPnL(userCtx, req)
	if err != nil {
		t.Fatalf("GetMonthlyPnL failed: %v", err)
	}
	items := resp.Msg.GetMonthlyPnl()
	// Should return 12 months (all months with 0 for empty months).
	if len(items) != 12 {
		t.Fatalf("expected 12 monthly items, got %d", len(items))
	}

	// January (month 1) — profit=100, trades=2.
	if items[0].Month != 1 {
		t.Errorf("Jan: expected month=1, got %d", items[0].Month)
	}
	if items[0].Profit != 100.0 {
		t.Errorf("Jan: expected profit=100, got %f", items[0].Profit)
	}
	if items[0].Trades != 2 {
		t.Errorf("Jan: expected trades=2, got %d", items[0].Trades)
	}

	// February (month 2) — profit=-30, trades=1.
	if items[1].Profit != -30.0 {
		t.Errorf("Feb: expected profit=-30, got %f", items[1].Profit)
	}
	if items[1].Trades != 1 {
		t.Errorf("Feb: expected trades=1, got %d", items[1].Trades)
	}

	// March (month 3) — profit=80, trades=2.
	if items[2].Profit != 80.0 {
		t.Errorf("Mar: expected profit=80, got %f", items[2].Profit)
	}
	if items[2].Trades != 2 {
		t.Errorf("Mar: expected trades=2, got %d", items[2].Trades)
	}

	// April-December should be zero.
	for i := 3; i < 12; i++ {
		if items[i].Profit != 0 {
			t.Errorf("month %d: expected profit=0, got %f", i+1, items[i].Profit)
		}
		if items[i].Trades != 0 {
			t.Errorf("month %d: expected trades=0, got %d", i+1, items[i].Trades)
		}
	}

	t.Logf("Monthly PnL verified: Jan=%f/%d, Feb=%f/%d, Mar=%f/%d",
		items[0].Profit, items[0].Trades,
		items[1].Profit, items[1].Trades,
		items[2].Profit, items[2].Trades)
}
