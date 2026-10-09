package mthub

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	"alphaforge/internal/interceptor"
	"alphaforge/internal/usermgr"
)

type mockMarginExecutor struct {
	mockExecutor
	marginRequired decimal.Decimal
	marginErr      error
}

func (m *mockMarginExecutor) RequiredMargin(_ context.Context, _ string, _ decimal.Decimal, _ Side, _ decimal.Decimal) (decimal.Decimal, error) {
	if m.marginErr != nil {
		return decimal.Zero, m.marginErr
	}
	return m.marginRequired, nil
}

// --- helper to create context with user ID ---

func ctxWithUser(userID string) context.Context {
	return context.WithValue(context.Background(), interceptor.UserIDKey, userID)
}

// --- preTradeChecks coverage ---

func newSaturatedLimiter() *usermgr.UserLimiter {
	l := usermgr.NewUserLimiter(usermgr.Config{MaxEntries: 10, OrderPerUserMax: 1})
	l.AllowOrder("user-1") // exhaust the quota
	return l
}

// --- Broker drop path coverage ---

func newFailingRedisClient() *goredis.Client {
	return goredis.NewClient(&goredis.Options{
		Addr:        "localhost:0", // invalid port — connection refused
		DialTimeout: 50 * time.Millisecond,
		ReadTimeout: 50 * time.Millisecond,
	})
}
