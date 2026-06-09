package autotrading

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/interceptor"
)

func TestUserID_Authenticated(t *testing.T) {
	t.Parallel()
	srv := NewAutoTradingServer(nil, nil, zap.NewNop())
	uid := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, uid)

	got := srv.userID(ctx)
	expected, _ := uuid.Parse(uid)
	if got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestUserID_Missing(t *testing.T) {
	t.Parallel()
	srv := NewAutoTradingServer(nil, nil, zap.NewNop())

	got := srv.userID(context.Background())
	if got != uuid.Nil {
		t.Errorf("expected Nil UUID for missing user, got %s", got)
	}
}

func TestUserID_Invalid(t *testing.T) {
	t.Parallel()
	srv := NewAutoTradingServer(nil, nil, zap.NewNop())
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "not-a-uuid")

	got := srv.userID(ctx)
	if got != uuid.Nil {
		t.Errorf("expected Nil UUID for invalid user, got %s", got)
	}
}

func TestNewAutoTradingServer_NonNil(t *testing.T) {
	t.Parallel()
	srv := NewAutoTradingServer(nil, nil, zap.NewNop())
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if srv.autoRepo != nil {
		t.Fatal("expected nil autoRepo when passed nil")
	}
	if srv.riskPipe != nil {
		t.Fatal("expected nil riskPipe when passed nil")
	}
}
