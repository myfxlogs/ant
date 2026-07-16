// rules_risksvc.go — Gate rules ported from the legacy risksvc pipeline.
//
// These rules provide the unique capabilities of risksvc (KYC/jurisdiction,
// contract expiry, margin floor, capability tier) as standard risk.Rule
// implementations so they can be added to the Gate and evaluated alongside
// the existing 11 Gate rules.
//
// Once verified, the risksvc pipeline in mthub/service_orders_risk.go can
// be removed and all orders will pass through a single Gate.

package risk

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/risksvc"
)

// --- KycJurisdictionGateRule ---

// KycJurisdictionGateRule blocks orders from users who have not completed KYC
// or are in sanctioned jurisdictions.  Wraps the existing risksvc.JurisdictionGate.
type KycJurisdictionGateRule struct {
	Gate      *risksvc.JurisdictionGate
	UserIDFn  func(ctx context.Context) string  // extracts user ID from context
	ClientIPFn func(ctx context.Context) string // extracts client IP from context
}

func (r *KycJurisdictionGateRule) Name() string { return "kyc_jurisdiction" }

func (r *KycJurisdictionGateRule) Check(ctx context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
	if r.Gate == nil {
		return passResult(r.Name())
	}
	uid := ""
	ip := ""
	if r.UserIDFn != nil {
		uid = r.UserIDFn(ctx)
	}
	if r.ClientIPFn != nil {
		ip = r.ClientIPFn(ctx)
	}
	if err := r.Gate.Check(ctx, uid, ip); err != nil {
		return blockResult(r.Name(), err.Error())
	}
	return passResult(r.Name())
}

// --- ContractExpiryRule ---

// ContractExpiryRule blocks orders on instruments whose contract expires
// within the CoolingOff window (e.g., 2 hours before expiry).
// This prevents strategy from opening positions that would be force-closed
// by the broker at expiry.
type ContractExpiryRule struct {
	// CoolingOff is the window before expiry during which new positions
	// are blocked.  Default: 2 hours.
	CoolingOffHours int

	// ExpiryProvider returns the contract expiry time for a symbol.
	// Returns zero time if the symbol has no expiry (e.g., spot FX).
	ExpiryProvider func(symbol string) (expiryUnixMs int64)
}

func (r *ContractExpiryRule) Name() string { return "contract_expiry" }

func (r *ContractExpiryRule) Check(ctx context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
	if r.ExpiryProvider == nil {
		return passResult(r.Name())
	}
	coolingOff := r.CoolingOffHours
	if coolingOff <= 0 {
		coolingOff = 2
	}
	expiryMs := r.ExpiryProvider(intent.GetSymbol())
	if expiryMs == 0 {
		return passResult(r.Name()) // no expiry — spot FX, always allowed
	}
	nowMs := currentTimeMillis()
	deadlineMs := expiryMs - int64(coolingOff)*3600_000
	if nowMs >= deadlineMs {
		return blockResult(r.Name(),
			fmt.Sprintf("contract for %s expires within %dh (expiry: %d)",
				intent.GetSymbol(), coolingOff, expiryMs))
	}
	return passResult(r.Name())
}

// --- MarginFloorRule ---

// MarginFloorRule blocks orders when free margin falls below a floor ratio.
// This is tighter than MarginPreCheck (which only checks if margin is positive).
// FloorRatio: minimum free_margin / required_margin. Default 1.0 (must have
// at least 100% of required margin as free margin).
type MarginFloorRule struct {
	FloorRatio float64
}

func (r *MarginFloorRule) Name() string { return "margin_floor" }

func (r *MarginFloorRule) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
	if state == nil {
		return passResult(r.Name()) // can't check without state
	}
	ratio := r.FloorRatio
	if ratio <= 0 {
		ratio = 1.0
	}
	vol := parseDecimal(intent.GetVolume())
	price := parseDecimal(intent.GetPrice())
	if vol.LessThanOrEqual(decimal.Zero) || price.LessThanOrEqual(decimal.Zero) {
		return passResult(r.Name()) // skip for market orders (price unknown)
	}
	required := vol.Mul(price)
	if state.FreeMargin.LessThan(decimal.NewFromFloat(ratio).Mul(required)) {
		return blockResult(r.Name(),
			fmt.Sprintf("free margin %s < required %s (ratio=%.1f)", state.FreeMargin.String(), decimal.NewFromFloat(ratio).Mul(required).String(), ratio))
	}
	return passResult(r.Name())
}

// --- CapabilityTierRule ---

// CapabilityTierRule restricts trading based on the user's assigned
// capability tier.  Each tier defines limits on position size, leverage,
// and permitted symbols.  Wraps risksvc.CapabilityStore.
type CapabilityTierRule struct {
	Store CapabilityStore
}

// CapabilityStore provides per-user trading limits.
type CapabilityStore interface {
	GetTier(ctx context.Context, userID string) (CapabilityTier, error)
}

// CapabilityTier defines limits for a user tier.
type CapabilityTier struct {
	Name           string
	MaxVolume      decimal.Decimal
	MaxLeverage    int
	MaxPositions   int
	AllowedSymbols []string
}

// NewCapabilityTierRule creates a rule backed by a risksvc.CapabilityStore.
func NewCapabilityTierRule(store *risksvc.CapabilityStore) *CapabilityTierRule {
	return &CapabilityTierRule{Store: &risksvcCapStoreAdapter{store}}
}

// risksvcCapStoreAdapter adapts *risksvc.CapabilityStore to risk.CapabilityStore.
type risksvcCapStoreAdapter struct {
	store *risksvc.CapabilityStore
}

func (a *risksvcCapStoreAdapter) GetTier(_ context.Context, userID string) (CapabilityTier, error) {
	cap := a.store.Get(userID)
	if cap == nil {
		return CapabilityTier{}, nil
	}
	return CapabilityTier{
		Name:           cap.Tier.String(),
		MaxVolume:      cap.LotPerOrderMax,
		MaxLeverage:    int(cap.LeverageMax),
		AllowedSymbols: cap.SymbolWhitelist,
	}, nil
}

func (r *CapabilityTierRule) Name() string { return "capability_tier" }

func (r *CapabilityTierRule) Check(ctx context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
	if r.Store == nil {
		return passResult(r.Name())
	}
	uid := intent.GetUserId()
	if uid == "" {
		return passResult(r.Name())
	}
	tier, err := r.Store.GetTier(ctx, uid)
	if err != nil || tier.Name == "" {
		return passResult(r.Name()) // no tier assigned = no restriction
	}
	vol := parseDecimal(intent.GetVolume())
	if tier.MaxVolume.GreaterThan(decimal.Zero) && vol.GreaterThan(tier.MaxVolume) {
		return blockResult(r.Name(),
			fmt.Sprintf("volume %s exceeds tier max %s (tier: %s)", vol.String(), tier.MaxVolume.String(), tier.Name))
	}
	if len(tier.AllowedSymbols) > 0 {
		sym := intent.GetSymbol()
		allowed := false
		for _, s := range tier.AllowedSymbols {
			if s == sym {
				allowed = true
				break
			}
		}
		if !allowed {
			return blockResult(r.Name(),
				fmt.Sprintf("symbol %s not allowed in tier %s", sym, tier.Name))
		}
	}
	return passResult(r.Name())
}

// --- helpers ---

func passResult(_ string) *RuleResult {
	return &RuleResult{Allowed: true}
}

func blockResult(_, reason string) *RuleResult {
	return &RuleResult{Allowed: false, Reason: reason}
}

func parseDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func currentTimeMillis() int64 {
	return time.Now().UnixMilli()
}
