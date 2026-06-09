package source

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewLiveSource_NilNATS(t *testing.T) {
	t.Parallel()
	ls := NewLiveSource(nil, zap.NewNop())
	if ls == nil {
		t.Fatal("NewLiveSource should return non-nil even with nil NATS")
	}
}
