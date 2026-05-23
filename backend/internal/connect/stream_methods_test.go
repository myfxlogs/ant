package connect

import (
	"context"
	"testing"
)

func TestIsCanceledErr_ContextCanceled(t *testing.T) {
	if !isCanceledErr(context.Canceled) {
		t.Fatalf("expected isCanceledErr(context.Canceled)=true")
	}
}

func TestIsCanceledErr_DeadlineExceeded(t *testing.T) {
	if !isCanceledErr(context.DeadlineExceeded) {
		t.Fatalf("expected isCanceledErr(context.DeadlineExceeded)=true")
	}
}

func TestIsCanceledErr_Nil(t *testing.T) {
	if isCanceledErr(nil) {
		t.Fatalf("expected isCanceledErr(nil)=false")
	}
}
