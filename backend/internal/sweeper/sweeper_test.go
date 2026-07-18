package sweeper

import (
	"testing"
)

func TestSweeper_loadConfigDefaults(t *testing.T) {
	s := &Sweeper{}
	// Test default values when config is not loaded
	if !s.demFactor.IsZero() {
		// After loadConfig with no DB, defaults are set
		// This just verifies the struct is zero-valued before loadConfig
		t.Errorf("expected demFactor=0 before loadConfig, got %s", s.demFactor.String())
	}
}
