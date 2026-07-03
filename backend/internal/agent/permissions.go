package agent

import (
	"context"
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

// PermissionEngine evaluates capabilities based on settings and user context (ADR-0025 §9).
type PermissionEngine struct {
	settings *SettingsStore
}

// NewPermissionEngine creates a permission engine.
func NewPermissionEngine(settings *SettingsStore) *PermissionEngine {
	return &PermissionEngine{settings: settings}
}

// Can checks if a user has permission for a capability.
// Uses resolved settings to determine permission.
func (p *PermissionEngine) Can(ctx context.Context, userID uuid.UUID, cap Capability) bool {
	if p.settings == nil {
		// No settings store — allow basic capabilities, deny sensitive ones
		switch cap {
		case CapLiveDeploy:
			return false
		default:
			return true
		}
	}

	settings, err := p.settings.GetSettings(ctx, userID)
	if err != nil {
		// Failed to load settings — fail safe for sensitive capabilities
		switch cap {
		case CapLiveDeploy:
			return false
		default:
			return true
		}
	}

	switch cap {
	case CapGenerateStrategy, CapSubmitStrategy, CapViewBacktest:
		return true // always allowed
	case CapSearchExperience, CapStoreExperience, CapManageMemory:
		return settings["agent.memory.enabled"] != "false"
	case CapLiveDeploy:
		return settings["agent.capability.can_live"] == "true"
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
