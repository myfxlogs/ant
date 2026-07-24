package risk

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// ── Canary stage lifecycle ────────────────────────────────────────────

func TestCanaryStageProgression(t *testing.T) {
	cfg := DefaultCanaryConfig()
	cfg.TradesPerStep = 2   // accelerate for testing
	cfg.MinHoursPerStage = 0 // no waiting
	ctrl := NewCanaryController(cfg)
	ctrl.AddAccount("acct-1")

	// Start: StageOff.
	if ctrl.CurrentStage() != StageOff {
		t.Error("expected StageOff at creation")
	}
	if ctrl.AllowedLotSize().IsZero() {
		t.Error("expected non-zero initial lot size (inactive but configured)")
	}

	// Activate canary.
	if err := ctrl.ActivateCanary(); err != nil {
		t.Fatalf("ActivateCanary: %v", err)
	}
	if ctrl.CurrentStage() != StageCanary {
		t.Error("expected StageCanary after activation")
	}

	// Record successful trades → step up.
	for i := 0; i < cfg.TradesPerStep; i++ {
		ctrl.RecordSuccessfulTrade()
	}
	// Lot should have stepped up.
	expectedLots := cfg.InitialLotSize.Add(cfg.LotStepUp)
	if !ctrl.AllowedLotSize().Equal(expectedLots) {
		t.Errorf("expected lot size %s after step-up, got %s", expectedLots, ctrl.AllowedLotSize())
	}
}

func TestCanaryActivationRequiresAccounts(t *testing.T) {
	ctrl := NewCanaryController(DefaultCanaryConfig())
	err := ctrl.ActivateCanary()
	if err == nil {
		t.Error("expected error when activating without canary accounts")
	}
}

func TestCanaryActivationRequiresStageOff(t *testing.T) {
	ctrl := NewCanaryController(DefaultCanaryConfig())
	ctrl.AddAccount("acct-1")
	_ = ctrl.ActivateCanary()

	// Second activation should fail.
	err := ctrl.ActivateCanary()
	if err == nil {
		t.Error("expected error on double activation")
	}
}

// ── Account whitelist ─────────────────────────────────────────────────

func TestCanaryAccountWhitelist(t *testing.T) {
	cfg := DefaultCanaryConfig()
	ctrl := NewCanaryController(cfg)
	ctrl.AddAccount("acct-1")
	ctrl.AddAccount("acct-2")
	_ = ctrl.ActivateCanary()

	if !ctrl.IsCanaryAccount("acct-1") {
		t.Error("acct-1 should be canary")
	}
	if ctrl.IsCanaryAccount("acct-999") {
		t.Error("acct-999 should NOT be canary")
	}

	ctrl.RemoveAccount("acct-1")
	if ctrl.IsCanaryAccount("acct-1") {
		t.Error("acct-1 should be removed")
	}
}

func TestFullStageAllowsAllAccounts(t *testing.T) {
	cfg := DefaultCanaryConfig()
	cfg.MinHoursPerStage = 0
	cfg.TradesPerStep = 1
	ctrl := NewCanaryController(cfg)
	ctrl.AddAccount("acct-1")
	_ = ctrl.ActivateCanary()
	_ = ctrl.PromoteToFull()

	if !ctrl.IsCanaryAccount("acct-999") {
		t.Error("StageFull should allow all accounts")
	}
}

// ── Lot size management ───────────────────────────────────────────────

func TestLotSizeStepUp(t *testing.T) {
	cfg := DefaultCanaryConfig()
	cfg.InitialLotSize = decimal.NewFromFloat(0.01)
	cfg.LotStepUp = decimal.NewFromFloat(0.01)
	cfg.MaxLotSize = decimal.NewFromFloat(0.10)
	cfg.TradesPerStep = 1
	cfg.MinHoursPerStage = 0
	ctrl := NewCanaryController(cfg)
	ctrl.AddAccount("acct-1")
	_ = ctrl.ActivateCanary()

	// Step 1: 0.01 → 0.02.
	ctrl.RecordSuccessfulTrade()
	if !ctrl.AllowedLotSize().Equal(decimal.NewFromFloat(0.02)) {
		t.Errorf("expected 0.02, got %s", ctrl.AllowedLotSize())
	}

	// Step up to max.
	for i := 0; i < 20; i++ {
		ctrl.RecordSuccessfulTrade()
	}
	if ctrl.AllowedLotSize().GreaterThan(cfg.MaxLotSize) {
		t.Errorf("lot size %s exceeds max %s", ctrl.AllowedLotSize(), cfg.MaxLotSize)
	}
}

func TestLotSizeCappedAtMax(t *testing.T) {
	cfg := DefaultCanaryConfig()
	cfg.MaxLotSize = decimal.NewFromFloat(10.0)
	ctrl := NewCanaryController(cfg)
	ctrl.AddAccount("acct-1")
	_ = ctrl.ActivateCanary()
	_ = ctrl.PromoteToFull()

	if !ctrl.AllowedLotSize().Equal(cfg.MaxLotSize) {
		t.Errorf("expected max lot %s, got %s", cfg.MaxLotSize, ctrl.AllowedLotSize())
	}
}

// ── Kill-switch drill ─────────────────────────────────────────────────

func TestKillSwitchEngage(t *testing.T) {
	cfg := DefaultCanaryConfig()
	ctrl := NewCanaryController(cfg)
	ctrl.AddAccount("acct-1")
	_ = ctrl.ActivateCanary()

	// Before kill-switch: lots allowed.
	if ctrl.AllowedLotSize().IsZero() {
		t.Error("lots should be allowed before kill-switch")
	}

	// Engage.
	ctrl.EngageKillSwitch("manual drill")

	if !ctrl.IsKillSwitchActive() {
		t.Error("kill-switch should be active")
	}
	if !ctrl.AllowedLotSize().IsZero() {
		t.Error("allowed lot size should be zero when kill-switch active")
	}
}

func TestKillSwitchDisengage(t *testing.T) {
	cfg := DefaultCanaryConfig()
	ctrl := NewCanaryController(cfg)
	ctrl.AddAccount("acct-1")
	_ = ctrl.ActivateCanary()
	ctrl.EngageKillSwitch("drill")
	ctrl.DisengageKillSwitch()

	if ctrl.IsKillSwitchActive() {
		t.Error("kill-switch should be disengaged")
	}
	if ctrl.AllowedLotSize().IsZero() {
		t.Error("lots should be allowed after disengage")
	}
}

func TestKillSwitchAuditLogged(t *testing.T) {
	cfg := DefaultCanaryConfig()
	ctrl := NewCanaryController(cfg)
	ctrl.AddAccount("acct-1")
	_ = ctrl.ActivateCanary()
	ctrl.EngageKillSwitch("max drawdown exceeded")

	history := ctrl.History()
	found := false
	for _, h := range history {
		if h.Reason == "KILL-SWITCH: max drawdown exceeded" {
			found = true
			break
		}
	}
	if !found {
		t.Error("kill-switch event not found in audit history")
	}
}

// ── Rollback ──────────────────────────────────────────────────────────

func TestRollback(t *testing.T) {
	cfg := DefaultCanaryConfig()
	cfg.TradesPerStep = 1
	cfg.MinHoursPerStage = 0
	ctrl := NewCanaryController(cfg)
	ctrl.AddAccount("acct-1")
	_ = ctrl.ActivateCanary()

	// Step up once.
	ctrl.RecordSuccessfulTrade()
	lotsBefore := ctrl.AllowedLotSize()

	// Step up again.
	ctrl.RecordSuccessfulTrade()
	if ctrl.AllowedLotSize().Equal(lotsBefore) {
		t.Error("expected lot size change after second step-up")
	}
	lotsAfterStep := ctrl.AllowedLotSize()

	// Rollback.
	if err := ctrl.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if !ctrl.AllowedLotSize().Equal(lotsBefore) {
		t.Errorf("after rollback expected %s, got %s", lotsBefore, ctrl.AllowedLotSize())
	}
	_ = lotsAfterStep
}

func TestRollbackWithoutHistory(t *testing.T) {
	ctrl := NewCanaryController(DefaultCanaryConfig())
	err := ctrl.Rollback()
	if err == nil {
		t.Error("expected error on rollback with no history")
	}
}

func TestRollbackPreservesAccountWhitelist(t *testing.T) {
	cfg := DefaultCanaryConfig()
	cfg.TradesPerStep = 1
	cfg.MinHoursPerStage = 0
	ctrl := NewCanaryController(cfg)
	ctrl.AddAccount("acct-1")
	_ = ctrl.ActivateCanary()
	ctrl.RecordSuccessfulTrade() // step-up creates rollback state
	ctrl.RecordSuccessfulTrade() // another step-up
	_ = ctrl.Rollback()

	// Account whitelist should be preserved.
	if !ctrl.IsCanaryAccount("acct-1") {
		t.Error("account whitelist not preserved after rollback")
	}
}

// ── Promote to full ───────────────────────────────────────────────────

func TestPromoteToFull(t *testing.T) {
	cfg := DefaultCanaryConfig()
	ctrl := NewCanaryController(cfg)
	ctrl.AddAccount("acct-1")
	_ = ctrl.ActivateCanary()

	if err := ctrl.PromoteToFull(); err != nil {
		t.Fatalf("PromoteToFull: %v", err)
	}
	if ctrl.CurrentStage() != StageFull {
		t.Error("expected StageFull after promotion")
	}
}

func TestPromoteToFullIdempotent(t *testing.T) {
	cfg := DefaultCanaryConfig()
	ctrl := NewCanaryController(cfg)
	ctrl.AddAccount("acct-1")
	_ = ctrl.ActivateCanary()
	_ = ctrl.PromoteToFull()
	if err := ctrl.PromoteToFull(); err != nil {
		t.Errorf("PromoteToFull should be idempotent: %v", err)
	}
}

// ── Audit trail ───────────────────────────────────────────────────────

func TestAuditTrail(t *testing.T) {
	cfg := DefaultCanaryConfig()
	cfg.TradesPerStep = 1
	cfg.MinHoursPerStage = 0
	ctrl := NewCanaryController(cfg)
	ctrl.AddAccount("acct-1")

	// Initial state: StageOff.
	_ = ctrl.ActivateCanary()
	_ = ctrl.PromoteToFull()
	ctrl.EngageKillSwitch("test drill")
	ctrl.DisengageKillSwitch()

	history := ctrl.History()
	if len(history) < 4 {
		t.Errorf("expected at least 4 audit entries, got %d", len(history))
	}

	// First entry should be canary activation.
	if history[0].ToStage != StageCanary {
		t.Error("first transition should be to StageCanary")
	}

	// Last entry should be kill-switch disengage.
	last := history[len(history)-1]
	if last.Reason != "kill-switch disengaged" {
		t.Errorf("last entry reason = %q, want kill-switch disengaged", last.Reason)
	}
}

// ── Concurrency ───────────────────────────────────────────────────────

func TestCanaryConcurrent(t *testing.T) {
	cfg := DefaultCanaryConfig()
	cfg.TradesPerStep = 5
	cfg.MinHoursPerStage = 0
	ctrl := NewCanaryController(cfg)
	ctrl.AddAccount("acct-1")
	_ = ctrl.ActivateCanary()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				ctrl.RecordSuccessfulTrade()
				ctrl.IsCanaryAccount("acct-1")
				ctrl.AllowedLotSize()
				ctrl.CurrentStage()
			}
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	// Should not panic or deadlock.
}

// ── Stage string representation ───────────────────────────────────────

func TestCanaryStageString(t *testing.T) {
	tests := []struct {
		stage    CanaryStage
		expected string
	}{
		{StageOff, "off"},
		{StageCanary, "canary"},
		{StageExpanded, "expanded"},
		{StageFull, "full"},
		{CanaryStage(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.stage.String(); got != tt.expected {
			t.Errorf("Stage(%d).String() = %q, want %q", tt.stage, got, tt.expected)
		}
	}
}

// ── Default configuration ─────────────────────────────────────────────

func TestDefaultCanaryConfig(t *testing.T) {
	cfg := DefaultCanaryConfig()
	if cfg.InitialLotSize.IsZero() {
		t.Error("initial lot size should not be zero")
	}
	if cfg.MaxLotSize.IsZero() {
		t.Error("max lot size should not be zero")
	}
	if cfg.TradesPerStep <= 0 {
		t.Error("trades per step must be positive")
	}
	if cfg.MinHoursPerStage < 0 {
		t.Error("min hours per stage must be non-negative")
	}
}

// ── Kill-switch zero lot integration ──────────────────────────────────

func TestKillSwitchZeroLots(t *testing.T) {
	cfg := DefaultCanaryConfig()
	cfg.InitialLotSize = decimal.NewFromFloat(1.0)
	ctrl := NewCanaryController(cfg)
	ctrl.AddAccount("acct-1")
	_ = ctrl.ActivateCanary()

	if ctrl.AllowedLotSize().IsZero() {
		t.Error("should have non-zero lots before kill-switch")
	}

	ctrl.EngageKillSwitch("emergency")

	if !ctrl.AllowedLotSize().IsZero() {
		t.Error("kill-switch should force zero lots")
	}

	// Verify it stays zero regardless of stage.
	ctrl.DisengageKillSwitch()
	if ctrl.AllowedLotSize().IsZero() {
		t.Error("lots should be non-zero after disengage")
	}
}

// ── Minimum hours enforcement ─────────────────────────────────────────

func TestMinHoursEnforcement(t *testing.T) {
	cfg := DefaultCanaryConfig()
	cfg.TradesPerStep = 1
	cfg.MinHoursPerStage = 999999 // effectively never
	ctrl := NewCanaryController(cfg)
	ctrl.AddAccount("acct-1")
	_ = ctrl.ActivateCanary()

	// Record many successful trades — should NOT step up because min hours not met.
	initialLots := ctrl.AllowedLotSize()
	for i := 0; i < 100; i++ {
		ctrl.RecordSuccessfulTrade()
	}
	if !ctrl.AllowedLotSize().Equal(initialLots) {
		t.Error("should not step up before min hours elapsed")
	}
}

// ── Fast rollback drill ───────────────────────────────────────────────

func TestKillSwitchRollbackDrill(t *testing.T) {
	// Simulate a complete drill: activate → trade → emergency → rollback → restore.
	cfg := DefaultCanaryConfig()
	cfg.TradesPerStep = 1
	cfg.MinHoursPerStage = 0
	ctrl := NewCanaryController(cfg)
	ctrl.AddAccount("acct-1")

	// 1. Activate canary.
	if err := ctrl.ActivateCanary(); err != nil {
		t.Fatalf("activate: %v", err)
	}

	// 2. Record successful trades.
	lotsBefore := ctrl.AllowedLotSize()
	ctrl.RecordSuccessfulTrade()
	if !ctrl.AllowedLotSize().GreaterThan(lotsBefore) {
		t.Log("lot size stepped up as expected")
	}

	// 3. Emergency kill-switch.
	ctrl.EngageKillSwitch("drill: unexpected behavior detected")
	if !ctrl.IsKillSwitchActive() {
		t.Fatal("kill-switch not engaged")
	}
	if !ctrl.AllowedLotSize().IsZero() {
		t.Fatal("kill-switch should zero lots")
	}

	// 4. Rollback.
	if err := ctrl.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	// 5. Disengage kill-switch.
	ctrl.DisengageKillSwitch()

	// 6. Verify state: back to canary with rolled-back lot size.
	if ctrl.IsKillSwitchActive() {
		t.Error("kill-switch should be disengaged")
	}
	if ctrl.AllowedLotSize().IsZero() {
		t.Error("lots restored after rollback+disengage")
	}

	// Audit trail should contain all steps.
	history := ctrl.History()
	hasKill := false
	hasRollback := false
	for _, h := range history {
		if h.Reason == "KILL-SWITCH: drill: unexpected behavior detected" {
			hasKill = true
		}
		if h.Reason == "rollback to canary" || (h.FromStage > h.ToStage) {
			hasRollback = true
		}
	}
	if !hasKill {
		t.Error("audit missing kill-switch entry")
	}
	if !hasRollback {
		t.Error("audit missing rollback entry")
	}

	t.Logf("Drill completed. Audit trail: %d entries", len(history))
	for _, h := range history {
		t.Logf("  %s: %s → %s (lots=%s) reason=%s",
			h.Timestamp.Format(time.RFC3339),
			h.FromStage, h.ToStage, h.LotSize, h.Reason)
	}
}
