package notification

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewSender(t *testing.T) {
	t.Parallel()
	s := NewSender(nil, nil, zap.NewNop())
	if s == nil {
		t.Fatal("expected non-nil sender")
	}
}
