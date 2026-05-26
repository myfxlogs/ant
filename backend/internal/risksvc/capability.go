// Package risksvc provides Capability tier system (M10-BASE-C1).
//
// Tier 0-3 enumeration with PreCheck that enforces capability constraints
// before any order reaches the broker. The tier is loaded from user_risk_profiles
// and cached for the session lifetime.

package risksvc

import (
	"context"
	"fmt"
	"sync"
)

// CapabilityTier defines the permission level for an account.
type CapabilityTier int

const (
	// Tier0ViewOnly — no orders allowed. Read-only access.
	Tier0ViewOnly CapabilityTier = 0

	// Tier1Paper — paper trading only. No real money.
	Tier1Paper CapabilityTier = 1

	// Tier2LiveLimited — live trading with conservative limits.
	Tier2LiveLimited CapabilityTier = 2

	// Tier3LiveFull — full live trading, highest permissions.
	Tier3LiveFull CapabilityTier = 3
)

// String returns the tier name.
func (t CapabilityTier) String() string {
	switch t {
	case Tier0ViewOnly:
		return "Tier0-ViewOnly"
	case Tier1Paper:
		return "Tier1-Paper"
	case Tier2LiveLimited:
		return "Tier2-LiveLimited"
	case Tier3LiveFull:
		return "Tier3-LiveFull"
	default:
		return fmt.Sprintf("Tier%d-Unknown", int(t))
	}
}

// Capability holds the loaded risk profile for a user/account.
type Capability struct {
	UserID           string
	Tier             CapabilityTier
	OrderTypes       []string // allowed order types; nil = all
	LotPerOrderMax   float64  // 0 = unlimited
	DailyOrderMax    int      // 0 = unlimited
	LeverageMax      float64  // 0 = broker default
	SymbolWhitelist  []string // nil = all symbols allowed
	KillSwitchOn     bool
}

// HasOrderType returns true if the given order type is allowed.
func (c *Capability) HasOrderType(ot string) bool {
	if c.OrderTypes == nil || len(c.OrderTypes) == 0 {
		return true
	}
	for _, allowed := range c.OrderTypes {
		if allowed == ot {
			return true
		}
	}
	return false
}

// TierCheck validates whether an account is allowed to trade based on capability tier.
func (c *Capability) TierCheck() *PreCheckResult {
	if c.KillSwitchOn {
		return &PreCheckResult{Allowed: false, Reason: "killswitch engaged", Rule: "capability"}
	}
	if c.Tier == Tier0ViewOnly {
		return &PreCheckResult{Allowed: false, Reason: "tier 0: view-only access", Rule: "capability"}
	}
	return &PreCheckResult{Allowed: true, Reason: "ok", Rule: "capability"}
}

// CapabilityStore is a thread-safe in-memory cache of per-user capabilities.
// Loaded from user_risk_profiles at session start, refreshed on PG NOTIFY.
type CapabilityStore struct {
	mu    sync.RWMutex
	caps  map[string]*Capability // userID → capability
}

// NewCapabilityStore creates an empty capability store.
func NewCapabilityStore() *CapabilityStore {
	return &CapabilityStore{caps: make(map[string]*Capability)}
}

// Get returns the capability for a user, or a default Tier0 entry.
func (s *CapabilityStore) Get(userID string) *Capability {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if c, ok := s.caps[userID]; ok {
		return c
	}
	return &Capability{UserID: userID, Tier: Tier0ViewOnly}
}

// Set stores a capability entry.
func (s *CapabilityStore) Set(c *Capability) {
	s.mu.Lock()
	s.caps[c.UserID] = c
	s.mu.Unlock()
}

// LoadFromPG loads capabilities from the user_risk_profiles table.
// Called at startup and on PG NOTIFY refresh.
func (s *CapabilityStore) LoadFromPG(ctx context.Context, rows interface{ Scan(dest ...interface{}) error; Next() bool; Close() }) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for rows.Next() {
		var (
			userID          string
			tier            int
			orderTypes      []string
			lotPerOrderMax  *float64
			dailyOrderMax   *int
			leverageMax     *float64
			symbolWhitelist []string
			killswitch      bool
		)
		if err := rows.Scan(&userID, &tier, &orderTypes, &lotPerOrderMax, &dailyOrderMax, &leverageMax, &symbolWhitelist, &killswitch); err != nil {
			return fmt.Errorf("capability: scan row: %w", err)
		}
		c := &Capability{
			UserID:          userID,
			Tier:            CapabilityTier(tier),
			OrderTypes:      orderTypes,
			SymbolWhitelist: symbolWhitelist,
			KillSwitchOn:    killswitch,
		}
		if lotPerOrderMax != nil {
			c.LotPerOrderMax = *lotPerOrderMax
		}
		if dailyOrderMax != nil {
			c.DailyOrderMax = *dailyOrderMax
		}
		if leverageMax != nil {
			c.LeverageMax = *leverageMax
		}
		s.caps[userID] = c
	}
	return nil
}

// Count returns the number of loaded capabilities.
func (s *CapabilityStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.caps)
}
