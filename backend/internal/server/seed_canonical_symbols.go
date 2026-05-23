package server

import (
	"context"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"anttrader/internal/symbol"
)

// seedCanonicalSymbols ensures the canonical_symbols table is populated
// with the ~50 mainstream symbols on startup. Idempotent (ON CONFLICT DO NOTHING).
func seedCanonicalSymbols(ctx context.Context, db *sqlx.DB) {
	if db == nil {
		return
	}
	entries := symbol.SeedCanonicals()
	for _, e := range entries {
		_, err := db.ExecContext(ctx, `
			INSERT INTO canonical_symbols (canonical, asset_class, base_ccy, quote_ccy, description)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (canonical) DO NOTHING
		`, e.Canonical, e.AssetClass, e.BaseCCY, e.QuoteCCY, e.Description)
		if err != nil {
			zap.L().Warn("canonical symbol seed failed", zap.String("symbol", e.Canonical), zap.Error(err))
		}
	}
	zap.L().Info("canonical symbols seeded", zap.Int("count", len(entries)))
}
