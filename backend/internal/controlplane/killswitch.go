// Package controlplane provides the SRE control plane (M10-BASE-F7).
// Three mechanisms: Kill Switch, Strategy Breaker, Canary.
package controlplane

import (
	"sync"
	"time"
)

// KillSwitch provides emergency stop-all for every trading account.
type KillSwitch struct {
	mu        sync.RWMutex
	engaged   bool
	reason    string
	operator  string
	engagedAt time.Time
}

// NewKillSwitch creates a disarmed kill switch.
func NewKillSwitch() *KillSwitch { return &KillSwitch{} }

// Engage arms the kill switch. All trading must stop immediately.
func (ks *KillSwitch) Engage(reason, operator string) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.engaged = true
	ks.reason = reason
	ks.operator = operator
	ks.engagedAt = time.Now()
}

// Disengage disarms the kill switch. Trading may resume.
func (ks *KillSwitch) Disengage() {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.engaged = false
	ks.reason = ""
	ks.operator = ""
}

// IsEngaged returns true when the kill switch is active.
func (ks *KillSwitch) IsEngaged() bool {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.engaged
}

// Status reports the current kill-switch state.
type KillSwitchStatus struct {
	Engaged   bool   `json:"engaged"`
	Reason    string `json:"reason,omitempty"`
	Operator  string `json:"operator,omitempty"`
	EngagedAt string `json:"engaged_at,omitempty"`
}

// Status returns a snapshot of the kill-switch state.
func (ks *KillSwitch) Status() KillSwitchStatus {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	s := KillSwitchStatus{Engaged: ks.engaged, Reason: ks.reason, Operator: ks.operator}
	if ks.engaged {
		s.EngagedAt = ks.engagedAt.Format(time.RFC3339)
	}
	return s
}
