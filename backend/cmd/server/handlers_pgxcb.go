package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// pgxCB adapts *pgxpool.Pool to systemai.cbExecutor for circuit breaker queries.
type pgxCB struct{ p *pgxpool.Pool }

func (a *pgxCB) Exec(ctx context.Context, sql string, args ...any) (any, error) {
	return a.p.Exec(ctx, sql, args...)
}

func (a *pgxCB) QueryRow(ctx context.Context, sql string, args ...any) interface{ Scan(dest ...any) error } {
	return a.p.QueryRow(ctx, sql, args...)
}
