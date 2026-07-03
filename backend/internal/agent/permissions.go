package agent

import (
	"context"
	"path"
	"strings"

	"github.com/google/uuid"
)

// Capability represents an agent capability that can be permitted or denied (ADR-0025 §9).
type Capability string

const (
	CapGenerateStrategy  Capability = "generate_strategy"
	CapSubmitStrategy    Capability = "submit_strategy"
	CapLiveDeploy        Capability = "live_deploy"
	CapSearchExperience  Capability = "search_experience"
	CapStoreExperience   Capability = "store_experience"
	CapManageMemory      Capability = "manage_memory"
	CapViewBacktest      Capability = "view_backtest"
)

// Effect is the outcome of a permission rule (ADR-0025 §5.3).
type Effect string

const (
	EffectAllow Effect = "ALLOW"
	EffectDeny  Effect = "DENY"
)

// PermissionRule is a single rule: Effect Action(Resource:Selector).
// Example: DENY live_trading(*:leverage>100)
type PermissionRule struct {
	Effect   Effect
	Action   string // e.g. "live_trading", "model", "backtest"
	Resource string // e.g. "*", "gpt-4"
	Selector string // e.g. "*", "leverage>100"
	Tier     SettingTier
}

// ParseRule parses a rule string like "ALLOW live_trading(*:volume<=1.0)".
func ParseRule(s string, tier SettingTier) (PermissionRule, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return PermissionRule{}, false
	}

	var effect Effect
	if strings.HasPrefix(s, "ALLOW ") {
		effect = EffectAllow
		s = strings.TrimPrefix(s, "ALLOW ")
	} else if strings.HasPrefix(s, "DENY ") {
		effect = EffectDeny
		s = strings.TrimPrefix(s, "DENY ")
	} else {
		return PermissionRule{}, false
	}

	// Parse Action(Resource:Selector)
	parenIdx := strings.Index(s, "(")
	if parenIdx < 0 {
		// No parens — just action with wildcard
		return PermissionRule{
			Effect:   effect,
			Action:   strings.TrimSpace(s),
			Resource: "*",
			Selector: "*",
			Tier:     tier,
		}, true
	}

	action := strings.TrimSpace(s[:parenIdx])
	rest := s[parenIdx+1:]
	closeIdx := strings.Index(rest, ")")
	if closeIdx < 0 {
		return PermissionRule{}, false
	}
	inner := rest[:closeIdx]

	// Split Resource:Selector
	colonIdx := strings.Index(inner, ":")
	if colonIdx < 0 {
		return PermissionRule{
			Effect:   effect,
			Action:   action,
			Resource: strings.TrimSpace(inner),
			Selector: "*",
			Tier:     tier,
		}, true
	}

	return PermissionRule{
		Effect:   effect,
		Action:   action,
		Resource: strings.TrimSpace(inner[:colonIdx]),
		Selector: strings.TrimSpace(inner[colonIdx+1:]),
		Tier:     tier,
	}, true
}

// matches checks if a rule matches the given action and resource.
func (r PermissionRule) matches(action, resource string) bool {
	if r.Action != action && r.Action != "*" {
		return false
	}
	if r.Resource == "*" || r.Resource == resource {
		return true
	}
	// glob match with path.Match
	matched, err := path.Match(r.Resource, resource)
	return err == nil && matched
}

// PermissionEngine evaluates capabilities based on settings and rules (ADR-0025 §5.3).
// Deny always takes priority over Allow. Rules are merged across tiers.
type PermissionEngine struct {
	settings *SettingsStore
}

// NewPermissionEngine creates a permission engine.
func NewPermissionEngine(settings *SettingsStore) *PermissionEngine {
	return &PermissionEngine{settings: settings}
}

// Can checks if a user has permission for a capability.
// Evaluates permission rules with Deny-priority, then falls back to managed config.
func (p *PermissionEngine) Can(ctx context.Context, userID uuid.UUID, cap Capability) bool {
	return p.CanWithResource(ctx, userID, cap, "*")
}

// CanWithResource checks permission for a capability against a specific resource.
func (p *PermissionEngine) CanWithResource(ctx context.Context, userID uuid.UUID, cap Capability, resource string) bool {
	if p.settings == nil {
		return failClosedCan(cap)
	}

	rs, err := p.settings.ResolveSettings(ctx, userID)
	if err != nil || !rs.Loaded {
		return failClosedCan(cap)
	}

	// Build rules from managed settings (stored as "permission.rule.<n>" keys)
	rules := p.buildRules(rs)

	// Evaluate rules: Deny-priority — if any DENY matches, deny.
	// Then if any ALLOW matches, allow. Default deny.
	action := string(cap)
	for _, rule := range rules {
		if rule.Effect == EffectDeny && rule.matches(action, resource) {
			return false
		}
	}
	for _, rule := range rules {
		if rule.Effect == EffectAllow && rule.matches(action, resource) {
			return true
		}
	}

	// No explicit rules — fall back to managed config defaults
	return p.defaultCan(cap, rs)
}

// buildRules extracts permission rules from resolved settings.
// Rules are stored as "permission.rule.1", "permission.rule.2", etc. in managed settings.
// When allowManagedRulesOnly=true (ADR-0025 §5.3), only managed-tier rules are included.
func (p *PermissionEngine) buildRules(rs *ResolvedSettings) []PermissionRule {
	var rules []PermissionRule
	for k, v := range rs.Flat {
		if !strings.HasPrefix(k, "permission.rule.") {
			continue
		}
		// Determine the tier of this rule
		tierStr := rs.Tiers[k]
		tier := TierManaged // default to managed for permission rules
		switch tierStr {
		case "user":
			tier = TierUser
		case "default":
			tier = TierDefault
		}
		// If allowManagedRulesOnly, skip non-managed rules
		if rs.Managed.AllowManagedRulesOnly && tier != TierManaged {
			continue
		}
		if rule, ok := ParseRule(v, tier); ok {
			rules = append(rules, rule)
		}
	}
	return rules
}

// defaultCan provides fallback capability checks when no explicit rules match.
func (p *PermissionEngine) defaultCan(cap Capability, rs *ResolvedSettings) bool {
	// Managed config overrides
	if cap == CapLiveDeploy {
		if rs.Managed.DisableLiveTrading {
			return false
		}
		return rs.Flat["agent.capability.can_live"] == "true"
	}

	switch cap {
	case CapGenerateStrategy, CapSubmitStrategy, CapViewBacktest:
		return true
	case CapSearchExperience, CapStoreExperience, CapManageMemory:
		return rs.Flat["agent.memory.enabled"] != "false"
	default:
		return false
	}
}

// failClosedCan returns the fail-closed permission for a capability.
// ADR-0025 §5.2: settings parse failure = lock down sensitive operations.
func failClosedCan(cap Capability) bool {
	switch cap {
	case CapGenerateStrategy, CapSubmitStrategy, CapViewBacktest:
		return true
	case CapLiveDeploy:
		return false // fail-closed
	case CapSearchExperience, CapStoreExperience, CapManageMemory:
		return true
	default:
		return false
	}
}

// CapabilitiesForUser returns all capabilities and their allowed status for a user.
func (p *PermissionEngine) CapabilitiesForUser(ctx context.Context, userID uuid.UUID) map[Capability]bool {
	caps := []Capability{
		CapGenerateStrategy, CapSubmitStrategy, CapLiveDeploy,
		CapSearchExperience, CapStoreExperience, CapManageMemory, CapViewBacktest,
	}
	result := make(map[Capability]bool, len(caps))
	for _, c := range caps {
		result[c] = p.Can(ctx, userID, c)
	}
	return result
}

// FormatCapabilities formats capabilities as a string for prompt injection.
func FormatCapabilities(caps map[Capability]bool) string {
	if len(caps) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n## Agent Capabilities\n")
	for cap, allowed := range caps {
		status := "denied"
		if allowed {
			status = "allowed"
		}
		sb.WriteString(string(cap) + ": " + status + "\n")
	}
	return sb.String()
}
