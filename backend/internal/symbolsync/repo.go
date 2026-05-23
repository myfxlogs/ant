// Package symbolsync — broker_symbols UPSERT repository.
package symbolsync

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Repo handles broker_symbols table operations.
type Repo struct {
	db *sqlx.DB
}

// NewRepo creates a symbolsync repository.
func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

// Upsert inserts or updates a broker_symbol row.
// Uses ON CONFLICT (broker_id, symbol_raw) DO UPDATE.
func (r *Repo) Upsert(ctx context.Context, s BrokerSymbol) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO broker_symbols (
			broker_id, symbol_raw, canonical,
			digits, point, tick_size,
			min_lot, max_lot, lot_step,
			swap_long, swap_short,
			trade_mode, description,
			raw_payload, partial, updated_at
		) VALUES (
			:broker_id, :symbol_raw, :canonical,
			:digits, :point, :tick_size,
			:min_lot, :max_lot, :lot_step,
			:swap_long, :swap_short,
			:trade_mode, :description,
			:raw_payload, :partial, now()
		)
		ON CONFLICT (broker_id, symbol_raw) DO UPDATE SET
			canonical     = EXCLUDED.canonical,
			digits        = EXCLUDED.digits,
			point         = EXCLUDED.point,
			tick_size     = EXCLUDED.tick_size,
			min_lot       = EXCLUDED.min_lot,
			max_lot       = EXCLUDED.max_lot,
			lot_step      = EXCLUDED.lot_step,
			swap_long     = EXCLUDED.swap_long,
			swap_short    = EXCLUDED.swap_short,
			trade_mode    = EXCLUDED.trade_mode,
			description   = EXCLUDED.description,
			raw_payload   = EXCLUDED.raw_payload,
			partial       = EXCLUDED.partial,
			updated_at    = now()
	`, s)
	if err != nil {
		return fmt.Errorf("symbolsync upsert %s/%s: %w", s.BrokerID, s.SymbolRaw, err)
	}
	return nil
}

// ListByBroker returns all symbols for a given broker.
func (r *Repo) ListByBroker(ctx context.Context, brokerID string) ([]BrokerSymbol, error) {
	var rows []BrokerSymbol
	err := r.db.SelectContext(ctx, &rows,
		`SELECT * FROM broker_symbols WHERE broker_id = $1 ORDER BY symbol_raw`, brokerID)
	if err != nil {
		return nil, fmt.Errorf("symbolsync list broker %s: %w", brokerID, err)
	}
	return rows, nil
}

// SeedCanonicalSymbols inserts the initial canonical_symbols dictionary.
func (r *Repo) SeedCanonicalSymbols(ctx context.Context, entries []struct {
	Canonical   string
	AssetClass  string
	BaseCCY     string
	QuoteCCY    string
	Description string
}) error {
	for _, e := range entries {
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO canonical_symbols (canonical, asset_class, base_ccy, quote_ccy, description)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (canonical) DO NOTHING
		`, e.Canonical, e.AssetClass, e.BaseCCY, e.QuoteCCY, e.Description)
		if err != nil {
			return fmt.Errorf("seed canonical %s: %w", e.Canonical, err)
		}
	}
	return nil
}
