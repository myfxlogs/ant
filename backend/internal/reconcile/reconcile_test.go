package reconcile

import (
	"testing"
)

func TestReconciler_Defaults(t *testing.T) {
	r := &Reconciler{}
	if r.alertThreshold != 0 {
		t.Errorf("expected alertThreshold=0 before loadConfig, got %f", r.alertThreshold)
	}
}
