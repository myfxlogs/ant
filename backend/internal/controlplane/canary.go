package controlplane

import (
	"sync"
	"time"
)

// CanaryConfig defines a canary deployment for a strategy version.
type CanaryConfig struct {
	StrategyID   string   `json:"strategy_id"`
	VersionTag   string   `json:"version_tag"`
	AccountIDs   []string `json:"account_ids"`   // accounts running canary version
	StartAt      string   `json:"start_at"`
	DurationDays int      `json:"duration_days"` // canary period before promotion
	Promoted     bool     `json:"promoted"`
}

// CanaryManager manages canary deployments.
type CanaryManager struct {
	mu      sync.RWMutex
	canaries map[string]*CanaryConfig // keyed by strategy_id
}

// NewCanaryManager creates a canary manager.
func NewCanaryManager() *CanaryManager {
	return &CanaryManager{canaries: make(map[string]*CanaryConfig)}
}

// Set configures a canary deployment for a strategy version.
func (m *CanaryManager) Set(cfg CanaryConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cfg.StartAt == "" {
		cfg.StartAt = time.Now().Format(time.RFC3339)
	}
	cp := cfg
	m.canaries[cfg.StrategyID] = &cp
}

// Get returns the canary config for a strategy.
func (m *CanaryManager) Get(strategyID string) *CanaryConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.canaries[strategyID]
}

// List returns all canary configs.
func (m *CanaryManager) List() []CanaryConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []CanaryConfig
	for _, c := range m.canaries {
		out = append(out, *c)
	}
	return out
}

// Remove deletes a canary config.
func (m *CanaryManager) Remove(strategyID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.canaries, strategyID)
}

// Promote marks a canary as promoted and removes it from the canary list.
func (m *CanaryManager) Promote(strategyID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.canaries[strategyID]; ok {
		c.Promoted = true
	}
}
