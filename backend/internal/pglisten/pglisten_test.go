package pglisten

import (
	"testing"

	"go.uber.org/zap"
)

func TestNew_ReturnsListener(t *testing.T) {
	t.Parallel()
	l := New(nil, zap.NewNop())
	if l == nil {
		t.Fatal("expected non-nil listener")
	}
	if l.pool != nil {
		t.Fatal("expected nil pool when passed nil")
	}
}
