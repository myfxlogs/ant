// Package mdgateway — dependency container.
// Provides a minimal Deps struct so mdgateway does not depend on
// an external bootstrap package.
package mdgateway

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Deps is the dependency container for the gateway runner.
type Deps struct {
	RDB *redis.Client
	PG  *PGClient
	Log *zap.Logger
}

// PGClient wraps a pgxpool.Pool with optional role support.
type PGClient struct {
	Pool *pgxpool.Pool
}

// SetRole sets the PostgreSQL role for the current session.
// In ant (no RLS), this is a no-op kept for compatibility with
// the alfq code pattern.
func (p *PGClient) SetRole(ctx interface{}, role string) error {
	// No-op in ant: tenant RLS is not used.
	return nil
}
