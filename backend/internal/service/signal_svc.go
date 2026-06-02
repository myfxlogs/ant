package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SignalRow struct {
	ID         uuid.UUID
	AccountID  uuid.UUID
	Symbol     string
	SignalType string
	Volume     float64
	Price      float64
	StopLoss   float64
	TakeProfit float64
	Reason     string
	Status     string
	ExecutedAt *time.Time
	Ticket     int64
	Profit     float64
	CreatedAt  time.Time
}

func (s *StrategySvc) ListSignals(ctx context.Context, userID, accountID uuid.UUID, status string) ([]SignalRow, error) {
	var rows pgx.Rows
	var err error
	signalsCols := `SELECT s.id, s.account_id, s.symbol, s.signal_type, s.volume, s.price, s.stop_loss, s.take_profit, s.reason, s.status, s.executed_at, s.ticket, s.profit, s.created_at FROM strategy_signals s JOIN mt_accounts a ON s.account_id = a.id`
	if accountID == uuid.Nil && status == "" {
		rows, err = s.pg.Query(ctx, signalsCols+` WHERE a.user_id = $1 ORDER BY s.created_at DESC LIMIT 100`, userID)
	} else if status == "" {
		rows, err = s.pg.Query(ctx, signalsCols+` WHERE s.account_id = $1 AND a.user_id = $2 ORDER BY s.created_at DESC LIMIT 100`, accountID, userID)
	} else if accountID == uuid.Nil {
		rows, err = s.pg.Query(ctx, signalsCols+` WHERE s.status = $1 AND a.user_id = $2 ORDER BY s.created_at DESC LIMIT 100`, status, userID)
	} else {
		rows, err = s.pg.Query(ctx, signalsCols+` WHERE s.account_id = $1 AND s.status = $2 AND a.user_id = $3 ORDER BY s.created_at DESC LIMIT 100`, accountID, status, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("list signals: %w", err)
	}
	defer rows.Close()
	return scanSignalRows(rows)
}

func (s *StrategySvc) GetSignal(ctx context.Context, id, userID uuid.UUID) (*SignalRow, error) {
	var r SignalRow
	err := s.pg.QueryRow(ctx,
		`SELECT s.id, s.account_id, s.symbol, s.signal_type, s.volume, s.price, s.stop_loss, s.take_profit, s.reason, s.status, s.executed_at, s.ticket, s.profit, s.created_at
		 FROM strategy_signals s JOIN mt_accounts a ON s.account_id = a.id
		 WHERE s.id = $1 AND a.user_id = $2`, id, userID,
	).Scan(&r.ID, &r.AccountID, &r.Symbol, &r.SignalType, &r.Volume, &r.Price, &r.StopLoss, &r.TakeProfit, &r.Reason, &r.Status, &r.ExecutedAt, &r.Ticket, &r.Profit, &r.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSignalNotFound
		}
		return nil, fmt.Errorf("GetSignal: %w", err)
	}
	return &r, nil
}

func (s *StrategySvc) ExecuteSignal(ctx context.Context, signalID, userID uuid.UUID) (*SignalRow, error) {
	now := time.Now()
	tag, err := s.pg.Exec(ctx,
		`UPDATE strategy_signals SET status='executed', executed_at=$2
		 WHERE id=$1 AND status='pending'
		 AND account_id IN (SELECT id FROM mt_accounts WHERE user_id = $3)`, signalID, now, userID)
	if err != nil {
		return nil, fmt.Errorf("ExecuteSignal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrSignalNotFound
	}
	return s.GetSignal(ctx, signalID, userID)
}

func (s *StrategySvc) ConfirmSignal(ctx context.Context, signalID, userID uuid.UUID) error {
	tag, err := s.pg.Exec(ctx,
		`UPDATE strategy_signals SET status='confirmed'
		 WHERE id=$1 AND status='pending'
		 AND account_id IN (SELECT id FROM mt_accounts WHERE user_id = $2)`, signalID, userID)
	if err != nil {
		return fmt.Errorf("ConfirmSignal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSignalNotFound
	}
	return nil
}

func (s *StrategySvc) CancelSignal(ctx context.Context, signalID, userID uuid.UUID) error {
	tag, err := s.pg.Exec(ctx,
		`UPDATE strategy_signals SET status='cancelled'
		 WHERE id=$1 AND status IN ('pending','confirmed')
		 AND account_id IN (SELECT id FROM mt_accounts WHERE user_id = $2)`, signalID, userID)
	if err != nil {
		return fmt.Errorf("CancelSignal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSignalNotFound
	}
	return nil
}

func scanSignalRows(rows pgx.Rows) ([]SignalRow, error) {
	var out []SignalRow
	for rows.Next() {
		var r SignalRow
		err := rows.Scan(&r.ID, &r.AccountID, &r.Symbol, &r.SignalType, &r.Volume, &r.Price, &r.StopLoss, &r.TakeProfit, &r.Reason, &r.Status, &r.ExecutedAt, &r.Ticket, &r.Profit, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan signal row: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
