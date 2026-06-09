package algo

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewExecutionAlgoServer_Init(t *testing.T) {
	t.Parallel()
	srv := NewExecutionAlgoServer(nil, zap.NewNop())
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if srv.broker != nil {
		t.Fatal("expected nil broker when nil passed")
	}
	if srv.log == nil {
		t.Fatal("expected non-nil logger")
	}
	if srv.active == nil {
		t.Fatal("expected initialized active map")
	}
}
