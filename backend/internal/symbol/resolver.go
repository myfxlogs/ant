// Package symbol — resolver: canonical → broker_symbol lookup.
package symbol

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SymbolInfo holds the resolved broker-native symbol for a canonical name.
type SymbolInfo struct {
	SymbolRaw string `db:"symbol_raw"`
	Canonical string `db:"canonical"`
	TradeMode int32  `db:"trade_mode"` // 0=disabled, 1=long_only, 2=short_only, 3=full
}

// Resolver resolves canonical symbols to broker-specific symbols.
type Resolver struct {
	db *pgxpool.Pool
}

// NewResolver creates a symbol resolver backed by the ant PG database.
func NewResolver(db *pgxpool.Pool) *Resolver {
	return &Resolver{db: db}
}

// ResolveCanonical looks up a canonical symbol for a given account's broker.
// Returns (symbolInfo, tradable, error).
// tradable=false means the symbol exists but is disabled or not found.
func (r *Resolver) ResolveCanonical(ctx context.Context, accountID, canonical string) (*SymbolInfo, bool, error) {
	if r.db == nil {
		return nil, false, fmt.Errorf("symbol resolver: db not available")
	}
	var info SymbolInfo
	err := r.db.QueryRow(ctx, `
			SELECT bs.symbol_raw, bs.canonical, bs.trade_mode
			FROM broker_symbols bs
			JOIN accounts a ON a.broker_id = bs.broker_id
			WHERE a.id = $1 AND bs.canonical = $2
			LIMIT 1
		`, accountID, canonical).Scan(&info.SymbolRaw, &info.Canonical, &info.TradeMode)
	if err != nil {
		return nil, false, fmt.Errorf("symbol %q not found for account %s: %w", canonical, accountID, err)
	}
	if info.TradeMode == 0 {
		return &info, false, fmt.Errorf("symbol %q is disabled on broker (%s)", canonical, info.SymbolRaw)
	}
	return &info, true, nil
}

// ListSupportedCanonicals returns all tradeable canonical symbols for an account.
func (r *Resolver) ListSupportedCanonicals(ctx context.Context, accountID string) ([]string, error) {
	if r.db == nil {
		return nil, fmt.Errorf("symbol resolver: db not available")
	}
	rows, err := r.db.Query(ctx, `
			SELECT DISTINCT bs.canonical
			FROM broker_symbols bs
			JOIN accounts a ON a.broker_id = bs.broker_id
			WHERE a.id = $1 AND bs.trade_mode > 0
			ORDER BY bs.canonical
		`, accountID)
	if err != nil {
		return nil, fmt.Errorf("list symbols: %w", err)
	}
	defer rows.Close()

	var canonicals []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		canonicals = append(canonicals, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return canonicals, nil
}
