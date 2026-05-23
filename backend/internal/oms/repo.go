// Package oms — order repository.
package oms

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Repo handles order persistence.
type Repo struct {
	db *sqlx.DB
}

// NewRepo creates an OMS order repository.
func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

// Insert creates a new order record.
func (r *Repo) Insert(ctx context.Context, o *Order) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO orders (id, mt_account_id, platform, ticket, symbol, broker_symbol_raw,
			order_type, volume, price, stop_loss, take_profit, state)
		VALUES (:id, :mt_account_id, :platform, :ticket, :symbol, :broker_symbol_raw,
			:order_type, :volume, :price, :stop_loss, :take_profit, :state)
	`, o)
	if err != nil {
		return fmt.Errorf("oms insert order: %w", err)
	}
	return nil
}

// UpdateState updates the order state (with transition validation).
func (r *Repo) UpdateState(ctx context.Context, orderID string, newState OrderState) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE orders SET state = $1, updated_at = now() WHERE id = $2`,
		string(newState), orderID)
	if err != nil {
		return fmt.Errorf("oms update state %s: %w", orderID, err)
	}
	return nil
}

// FindByID retrieves an order by its UUID.
func (r *Repo) FindByID(ctx context.Context, id string) (*Order, error) {
	var o Order
	err := r.db.GetContext(ctx, &o,
		`SELECT id, mt_account_id, platform, ticket, symbol, broker_symbol_raw,
			order_type, volume, price, stop_loss, take_profit, state
		 FROM orders WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("oms find order %s: %w", id, err)
	}
	return &o, nil
}

// FindByTicket retrieves an order by broker ticket + account.
func (r *Repo) FindByTicket(ctx context.Context, accountID, ticket string) (*Order, error) {
	var o Order
	err := r.db.GetContext(ctx, &o,
		`SELECT id, mt_account_id, platform, ticket, symbol, broker_symbol_raw,
			order_type, volume, price, stop_loss, take_profit, state
		 FROM orders WHERE mt_account_id = $1 AND ticket = $2`, accountID, ticket)
	if err != nil {
		return nil, fmt.Errorf("oms find order by ticket %s/%s: %w", accountID, ticket, err)
	}
	return &o, nil
}
