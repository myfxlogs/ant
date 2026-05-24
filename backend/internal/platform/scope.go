package platform

import "context"

// PlatformScope resolves the current platform identifier.
// Default implementation returns "ant" (single platform).
// M10+ multi-tenant implementations extract from JWT/header.
type PlatformScope interface {
    Current(ctx context.Context) string
}

type defaultScope struct{}

func (defaultScope) Current(ctx context.Context) string { return "ant" }

// New returns the default single-platform scope.
func New() PlatformScope { return defaultScope{} }
