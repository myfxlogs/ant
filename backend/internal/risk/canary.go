// Package risk — canary rollout controller (T4.2).
//
// Manages graduated deployment of live trading:
//   - Canary accounts (whitelist)
//   - Graduated lot size (start small, step up after N successful trades)
//   - Kill-switch integration (global emergency stop)
//   - Rollback (revert to previous safe configuration)
//
// Usage:
//
//	ctrl := NewCanaryController(CanaryConfig{...})
//	ctrl.AddAccount("acct-1")           // whitelist canary account
//	ctrl.StepUpLotSize()                // after successful trades
//	ctrl.EngageKillSwitch("drawdown")   // emergency stop
//	ctrl.Rollback()                     // revert to previous state
package risk

import (
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// ── Canary configuration ──────────────────────────────────────────────

// CanaryStage represents the current rollout stage.
type CanaryStage int

const (
	StageOff       CanaryStage = iota // no live trading
	StageCanary                       // canary accounts only, min lots
	StageExpanded                     // more accounts, stepped-up lots
	StageFull                         // all accounts, full lots
)

func (s CanaryStage) String() string {
	switch s {
	case StageOff:
		return "off"
	case StageCanary:
		return "canary"
	case StageExpanded:
		return "expanded"
	case StageFull:
		return "full"
	default:
		return "unknown"
	}
}

// CanaryConfig holds the rollout parameters.
type CanaryConfig struct {
	// Initial lot size for canary stage.
	InitialLotSize decimal.Decimal

	// Lot size step-up increment after each successful batch.
	LotStepUp decimal.Decimal

	// Maximum lot size (full production).
	MaxLotSize decimal.Decimal

	// Number of successful trades required before stepping up.
	TradesPerStep int

	// Minimum hours at current stage before stepping up.
	MinHoursPerStage float64
}

// DefaultCanaryConfig returns sensible rollout defaults.
func DefaultCanaryConfig() CanaryConfig {
	return CanaryConfig{
		InitialLotSize:   decimal.NewFromFloat(0.01),
		LotStepUp:        decimal.NewFromFloat(0.01),
		MaxLotSize:       decimal.NewFromFloat(10.0),
		TradesPerStep:    10,
		MinHoursPerStage: 24,
	}
}

// ── Canary controller ─────────────────────────────────────────────────

// CanaryController manages the canary rollout state machine.
// Safe for concurrent use.
type CanaryController struct {
	mu     sync.RWMutex
	config CanaryConfig

	stage        CanaryStage
	currentLots  decimal.Decimal
	canaryAccts  map[string]bool   // whitelisted account IDs
	successTrades int               // successful trades at current stage
	stageEntered time.Time         // when current stage was entered
	killSwitch   bool              // emergency stop
	rollbackLots decimal.Decimal   // lot size before last step-up
	rollbackStage CanaryStage      // stage before last transition

	// Audit log of stage transitions.
	history []StageTransition
}

// StageTransition records a state change for audit.
type StageTransition struct {
	Timestamp  time.Time
	FromStage  CanaryStage
	ToStage    CanaryStage
	LotSize    decimal.Decimal
	Reason     string
}

// NewCanaryController creates a controller with the given config.
func NewCanaryController(cfg CanaryConfig) *CanaryController {
	return &CanaryController{
		config:       cfg,
		stage:        StageOff,
		currentLots:  cfg.InitialLotSize,
		canaryAccts:  make(map[string]bool),
		stageEntered: time.Now(),
		history:      make([]StageTransition, 0),
	}
}

// ── Account management ────────────────────────────────────────────────

// AddAccount adds an account to the canary whitelist.
func (c *CanaryController) AddAccount(accountID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.canaryAccts[accountID] = true
}

// RemoveAccount removes an account from the canary whitelist.
func (c *CanaryController) RemoveAccount(accountID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.canaryAccts, accountID)
}

// IsCanaryAccount checks if an account is whitelisted for the current stage.
func (c *CanaryController) IsCanaryAccount(accountID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.stage == StageFull {
		return true // all accounts allowed
	}
	return c.canaryAccts[accountID]
}

// ── Lot size management ───────────────────────────────────────────────

// AllowedLotSize returns the maximum lot size for the current stage.
func (c *CanaryController) AllowedLotSize() decimal.Decimal {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.killSwitch {
		return decimal.Zero
	}
	return c.currentLots
}

// CurrentStage returns the current rollout stage.
func (c *CanaryController) CurrentStage() CanaryStage {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stage
}

// ── Stage transitions ─────────────────────────────────────────────────

// ActivateCanary moves from StageOff to StageCanary.
func (c *CanaryController) ActivateCanary() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stage != StageOff {
		return fmt.Errorf("canary activation requires StageOff, current: %s", c.stage)
	}
	if len(c.canaryAccts) == 0 {
		return fmt.Errorf("no canary accounts configured")
	}

	return c.transitionTo(StageCanary, "canary activation")
}

// RecordSuccessfulTrade increments the success counter and checks
// if a step-up is warranted.
func (c *CanaryController) RecordSuccessfulTrade() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.successTrades++
	if c.stage >= StageFull {
		return
	}

	hoursElapsed := time.Since(c.stageEntered).Hours()
	if c.successTrades >= c.config.TradesPerStep &&
		hoursElapsed >= c.config.MinHoursPerStage {
		_ = c.stepUp()
	}
}

// StepUpLotSize manually advances the lot size (for testing/forced advance).
func (c *CanaryController) StepUpLotSize() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stepUp()
}

func (c *CanaryController) stepUp() error {
	c.rollbackLots = c.currentLots
	c.rollbackStage = c.stage

	newLots := c.currentLots.Add(c.config.LotStepUp)
	if newLots.GreaterThan(c.config.MaxLotSize) {
		newLots = c.config.MaxLotSize
	}

	// Determine new stage.
	newStage := c.stage
	if newLots.GreaterThanOrEqual(c.config.MaxLotSize) {
		newStage = StageFull
	} else if c.stage == StageCanary && c.successTrades >= c.config.TradesPerStep*2 {
		newStage = StageExpanded
	}

	c.currentLots = newLots
	return c.transitionTo(newStage, fmt.Sprintf("step-up to %s lots", newLots))
}

// PromoteToFull immediately moves to StageFull (all accounts, full lots).
// Use only after manual review.
func (c *CanaryController) PromoteToFull() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stage == StageFull {
		return nil
	}

	c.rollbackLots = c.currentLots
	c.rollbackStage = c.stage
	c.currentLots = c.config.MaxLotSize
	return c.transitionTo(StageFull, "manual promotion to full")
}

// ── Kill-switch ───────────────────────────────────────────────────────

// EngageKillSwitch immediately stops all live trading.
func (c *CanaryController) EngageKillSwitch(reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.killSwitch = true
	c.history = append(c.history, StageTransition{
		Timestamp: time.Now(),
		FromStage: c.stage,
		ToStage:   StageOff,
		LotSize:   decimal.Zero,
		Reason:    fmt.Sprintf("KILL-SWITCH: %s", reason),
	})
}

// DisengageKillSwitch releases the kill-switch.
func (c *CanaryController) DisengageKillSwitch() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.killSwitch = false
	c.history = append(c.history, StageTransition{
		Timestamp: time.Now(),
		FromStage: StageOff,
		ToStage:   c.stage,
		LotSize:   c.currentLots,
		Reason:    "kill-switch disengaged",
	})
}

// IsKillSwitchActive returns true if the emergency stop is engaged.
func (c *CanaryController) IsKillSwitchActive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.killSwitch
}

// ── Rollback ──────────────────────────────────────────────────────────

// Rollback reverts to the previous stage and lot size.
func (c *CanaryController) Rollback() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.rollbackLots.IsZero() && c.rollbackStage == StageOff {
		return fmt.Errorf("no rollback state available")
	}

	prevLots := c.rollbackLots
	prevStage := c.rollbackStage

	c.currentLots = prevLots
	return c.transitionTo(prevStage, fmt.Sprintf("rollback to %s (%s lots)", prevStage, prevLots))
}

// ── Audit ─────────────────────────────────────────────────────────────

// History returns the stage transition audit log.
func (c *CanaryController) History() []StageTransition {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]StageTransition, len(c.history))
	copy(out, c.history)
	return out
}

// ── Internals ─────────────────────────────────────────────────────────

func (c *CanaryController) transitionTo(stage CanaryStage, reason string) error {
	transition := StageTransition{
		Timestamp: time.Now(),
		FromStage: c.stage,
		ToStage:   stage,
		LotSize:   c.currentLots,
		Reason:    reason,
	}

	c.stage = stage
	c.stageEntered = time.Now()
	c.successTrades = 0
	c.history = append(c.history, transition)

	return nil
}
