package reconcile

import (
	"testing"
)

func TestReconciler_Defaults(t *testing.T) {
	r := &Reconciler{}
	if r.shortageThreshold != 0 {
		t.Errorf("expected shortageThreshold=0 before loadConfig, got %f", r.shortageThreshold)
	}
	if r.surplusThreshold != 0 {
		t.Errorf("expected surplusThreshold=0 before loadConfig, got %f", r.surplusThreshold)
	}
}
