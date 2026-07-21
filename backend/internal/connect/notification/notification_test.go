package notification

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewNotificationServer_Init(t *testing.T) {
	t.Parallel()
	srv := NewNotificationServer(nil, nil, nil, zap.NewNop())
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
}
