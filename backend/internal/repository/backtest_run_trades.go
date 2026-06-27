package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// BacktestRunTrade mirrors the backtest_run_trades table row.
type BacktestRunTrade struct {
	RunID      uuid.UUID
	Ticket     int64
	Side       string
	Volume     float64
	OpenTs     int64
	OpenPrice  float64
	CloseTs    int64
	ClosePrice float64
	PnL        float64
	Commission float64
	Reason     string
}

// BatchCreateTrades inserts backtest trades with (run_id, ticket) upsert.
func (r *BacktestRunRepository) BatchCreateTrades(ctx context.Context, trades []*BacktestRunTrade) error {
	batch := &pgx.Batch{}
	for _, t := range trades {
		batch.Queue(
			`INSERT INTO backtest_run_trades (run_id, ticket, side, volume, open_ts, open_price, close_ts, close_price, pnl, commission, reason)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
			 ON CONFLICT (run_id, ticket) DO UPDATE SET
			   side=$3, volume=$4, open_ts=$5, open_price=$6, close_ts=$7, close_price=$8, pnl=$9, commission=$10, reason=$11`,
			t.RunID, t.Ticket, t.Side, t.Volume, t.OpenTs, t.OpenPrice, t.CloseTs, t.ClosePrice, t.PnL, t.Commission, t.Reason,
		)
	}
	br := r.db.SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < len(trades); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("backtest run trade batch insert row %d: %w", i, err)
		}
	}
	return nil
}

// ListTradesByRunID returns all trades for a backtest run, ordered by open_ts.
func (r *BacktestRunRepository) ListTradesByRunID(ctx context.Context, runID uuid.UUID) ([]*BacktestRunTrade, error) {
	rows, err := r.db.Query(ctx,
		`SELECT run_id, ticket, side, volume, open_ts, open_price, close_ts, close_price, pnl, commission, reason
		 FROM backtest_run_trades WHERE run_id = $1 ORDER BY open_ts`, runID)
	if err != nil {
		return nil, fmt.Errorf("list backtest run trades: %w", err)
	}
	defer rows.Close()
	var out []*BacktestRunTrade
	for rows.Next() {
		t := &BacktestRunTrade{}
		if err := rows.Scan(&t.RunID, &t.Ticket, &t.Side, &t.Volume, &t.OpenTs, &t.OpenPrice, &t.CloseTs, &t.ClosePrice, &t.PnL, &t.Commission, &t.Reason); err != nil {
			return nil, fmt.Errorf("scan backtest run trade: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate backtest run trades: %w", err)
	}
	return out, nil
}
