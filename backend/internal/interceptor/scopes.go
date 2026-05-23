package interceptor

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
)

// Standard API key scopes (industry standard naming)
const (
	ScopeMarketRead     = "market:read"
	ScopeTradeRead      = "trade:read"
	ScopeTradeWrite     = "trade:write"
	ScopeAccountRead    = "account:read"
	ScopeAccountWrite   = "account:write"
	ScopeStrategyRun    = "strategy:run"
	ScopeStreamEvents   = "stream:events"
	ScopeStreamQuotes   = "stream:quotes"
	ScopeStreamOrders   = "stream:orders"
	ScopeStreamProfits  = "stream:profits"
)

// RequireScopes checks if the current context has all required scopes.
// Returns a frontend-friendly error if missing any scope.
func RequireScopes(ctx context.Context, requiredScopes ...string) error {
	// Only enforce scopes for API key authentication.
	// JWT-authenticated frontend users should not be blocked by API key scopes.
	if !IsAPIKeyAuthenticated(ctx) {
		return nil
	}

	scopes, ok := GetAPIScopes(ctx)
	if !ok || len(scopes) == 0 {
		return newScopeError(requiredScopes, nil)
	}

	missing := missingScopes(scopes, requiredScopes)
	if len(missing) > 0 {
		return newScopeError(requiredScopes, scopes)
	}
	return nil
}

// missingScopes returns required scopes not present in provided scopes.
func missingScopes(provided, required []string) []string {
	providedSet := make(map[string]bool, len(provided))
	for _, s := range provided {
		providedSet[s] = true
	}

	var missing []string
	for _, r := range required {
		if !providedSet[r] {
			missing = append(missing, r)
		}
	}
	return missing
}

// newScopeError builds a frontend-friendly permission error.
func newScopeError(required, provided []string) error {
	err := &ScopeError{
		Code:           "INSUFFICIENT_SCOPE",
		Message:        "API key lacks required scope(s)",
		RequiredScopes: required,
		ProvidedScopes: provided,
	}
	return connect.NewError(connect.CodePermissionDenied, err)
}

// ScopeError is returned when API key lacks required scopes.
type ScopeError struct {
	Code           string   `json:"code"`
	Message        string   `json:"message"`
	RequiredScopes []string `json:"required_scopes"`
	ProvidedScopes []string `json:"provided_scopes,omitempty"`
}

func (e *ScopeError) Error() string {
	if len(e.RequiredScopes) == 0 {
		return e.Message
	}
	return fmt.Sprintf("%s: requires %v", e.Message, e.RequiredScopes)
}

// HasAnyScope returns true if context has at least one of the given scopes.
func HasAnyScope(ctx context.Context, scopes ...string) bool {
	provided, ok := GetAPIScopes(ctx)
	if !ok {
		return false
	}
	providedSet := make(map[string]bool, len(provided))
	for _, s := range provided {
		providedSet[s] = true
	}
	for _, s := range scopes {
		if providedSet[s] {
			return true
		}
	}
	return false
}

// HasAllScopes returns true if context has all of the given scopes.
func HasAllScopes(ctx context.Context, scopes ...string) bool {
	provided, ok := GetAPIScopes(ctx)
	if !ok {
		return false
	}
	providedSet := make(map[string]bool, len(provided))
	for _, s := range provided {
		providedSet[s] = true
	}
	for _, s := range scopes {
		if !providedSet[s] {
			return false
		}
	}
	return true
}
