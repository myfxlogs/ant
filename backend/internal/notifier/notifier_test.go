package notifier

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewEmailNotifier(t *testing.T) {
	t.Parallel()
	cfg := EmailConfig{
		Host: "smtp.example.com",
		Port: "587",
		From: "test@example.com",
	}
	n := NewEmailNotifier(cfg, zap.NewNop())
	if n == nil {
		t.Fatal("expected non-nil notifier")
	}
}

func TestNewEmailNotifier_NilLogger(t *testing.T) {
	t.Parallel()
	cfg := EmailConfig{Host: "h", Port: "25", From: "f@x.com"}
	n := NewEmailNotifier(cfg, nil)
	// Should not panic.
	_ = n
}

func TestAlertMethods_NoPanic(t *testing.T) {
	t.Parallel()
	cfg := EmailConfig{Host: "h", Port: "25", From: "f@x.com"}
	n := NewEmailNotifier(cfg, zap.NewNop())
	// Alert methods should not panic even without SMTP server.
	n.KillSwitchAlert("test reason", "admin")
	n.BreakerTripAlert("strat-1", 15.5)
	n.PromoteToLiveAlert("strat-2", "v2.0")
}
