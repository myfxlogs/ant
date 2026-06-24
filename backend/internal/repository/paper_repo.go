package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// PaperAccount represents a virtual trading account for paper trading.
// Monetary fields use decimal.Decimal (NUMERIC in DB) per platform data precision rules.
type PaperAccount struct {
	ID             string
	UserID         string
	Name           string
	InitialBalance decimal.Decimal
	CurrentBalance decimal.Decimal
	Equity         decimal.Decimal
	Currency       string
	CreatedAt      time.Time
	Archived       bool
}

// PaperOrder represents a simulated order.
// Monetary fields use decimal.Decimal (NUMERIC in DB) per platform data precision rules.
type PaperOrder struct {
	ID             string
	PaperAccountID string
	StrategyID     *string
	Symbol         string
	Side           string
	Volume         decimal.Decimal
	FillPrice      decimal.Decimal
	StopLoss       decimal.Decimal
	TakeProfit     decimal.Decimal
	SlippageBps    int32
	PnL            decimal.Decimal
	State          string
	CreatedAt      time.Time
	ClosedAt       *time.Time
}

// PaperStrategy represents a user's paper (simulated) strategy.
type PaperStrategy struct {
	ID              string
	UserID          string
	Name            string
	Description     *string
	Category        *string
	DSLCode         string
	BacktestMetrics map[string]interface{}
	PromotedAt      *time.Time
	CreatedAt       time.Time
	Archived        bool
}

// PaperRepo manages paper trading data in PostgreSQL.
type PaperRepo struct {
	pg *pgxpool.Pool
}

// NewPaperRepo creates a PaperRepo backed by the given pool.
func NewPaperRepo(pg *pgxpool.Pool) *PaperRepo {
	return &PaperRepo{pg: pg}
}

// CreateAccount inserts a new paper account.
func (r *PaperRepo) CreateAccount(ctx context.Context, userID, name string, initialBalance decimal.Decimal) (*PaperAccount, error) {
	a := &PaperAccount{}
	err := r.pg.QueryRow(ctx, `
		INSERT INTO paper_accounts (user_id, name, initial_balance, current_balance, equity)
		VALUES ($1, $2, $3, $3, $3)
		RETURNING id, user_id, name, initial_balance, current_balance, equity, currency, created_at, archived
	`, userID, name, initialBalance).Scan(
		&a.ID, &a.UserID, &a.Name,
		&a.InitialBalance, &a.CurrentBalance, &a.Equity,
		&a.Currency, &a.CreatedAt, &a.Archived,
	)
	if err != nil {
		return nil, err
	}
	return a, nil
}

// GetAccount returns a single paper account by ID.
func (r *PaperRepo) GetAccount(ctx context.Context, accountID string) (*PaperAccount, error) {
	var a PaperAccount
	err := r.pg.QueryRow(ctx, `
		SELECT id, user_id, name, initial_balance, current_balance, equity, currency, created_at, archived
		FROM paper_accounts WHERE id = $1 AND archived = false
	`, accountID).Scan(&a.ID, &a.UserID, &a.Name, &a.InitialBalance, &a.CurrentBalance, &a.Equity, &a.Currency, &a.CreatedAt, &a.Archived)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListAccounts returns all non-archived paper accounts for a user.
func (r *PaperRepo) ListAccounts(ctx context.Context, userID string) ([]*PaperAccount, error) {
	rows, err := r.pg.Query(ctx, `
		SELECT id, user_id, name, initial_balance, current_balance, equity, currency, created_at, archived
		FROM paper_accounts
		WHERE user_id = $1 AND archived = false
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPaperAccounts(rows)
}

// CreateOrder inserts a paper order.
func (r *PaperRepo) CreateOrder(ctx context.Context, o *PaperOrder) error {
	_, err := r.pg.Exec(ctx, `
		INSERT INTO paper_orders (paper_account_id, strategy_id, symbol, side, volume, fill_price, stop_loss, take_profit, slippage_bps, state, pnl)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, o.PaperAccountID, o.StrategyID, o.Symbol, o.Side, o.Volume, o.FillPrice, o.StopLoss, o.TakeProfit, o.SlippageBps, o.State, o.PnL)
	if err != nil {
		return fmt.Errorf("create paper order: %w", err)
	}
	return nil
}

// GetOrder returns a paper order by ID.
func (r *PaperRepo) GetOrder(ctx context.Context, orderID string) (*PaperOrder, error) {
	o := &PaperOrder{}
	err := r.pg.QueryRow(ctx, `
		SELECT id, paper_account_id, strategy_id, symbol, side, volume, fill_price,
		       COALESCE(stop_loss, 0), COALESCE(take_profit, 0),
		       slippage_bps, state, pnl, created_at, closed_at
		FROM paper_orders WHERE id = $1
	`, orderID).Scan(&o.ID, &o.PaperAccountID, &o.StrategyID, &o.Symbol, &o.Side,
		&o.Volume, &o.FillPrice, &o.StopLoss, &o.TakeProfit,
		&o.SlippageBps, &o.State, &o.PnL, &o.CreatedAt, &o.ClosedAt)
	if err != nil {
		return nil, fmt.Errorf("get paper order: %w", err)
	}
	return o, nil
}

// UpdateOrder updates stop_loss, take_profit, and state of a paper order.
func (r *PaperRepo) UpdateOrder(ctx context.Context, o *PaperOrder) error {
	_, err := r.pg.Exec(ctx, `
		UPDATE paper_orders SET stop_loss = $2, take_profit = $3, state = $4, closed_at = $5
		WHERE id = $1
	`, o.ID, o.StopLoss, o.TakeProfit, o.State, o.ClosedAt)
	return err
}

// FindOpenOrder returns the most recent open order for a paper account by symbol.
func (r *PaperRepo) FindOpenOrder(ctx context.Context, paperAccountID, symbol string) (*PaperOrder, error) {
	o := &PaperOrder{}
	err := r.pg.QueryRow(ctx, `
		SELECT id, paper_account_id, strategy_id, symbol, side, volume, fill_price,
		       COALESCE(stop_loss, 0), COALESCE(take_profit, 0),
		       slippage_bps, state, pnl, created_at, closed_at
		FROM paper_orders
		WHERE paper_account_id = $1 AND symbol = $2 AND state = 'open'
		ORDER BY created_at DESC LIMIT 1
	`, paperAccountID, symbol).Scan(&o.ID, &o.PaperAccountID, &o.StrategyID, &o.Symbol, &o.Side,
		&o.Volume, &o.FillPrice, &o.StopLoss, &o.TakeProfit,
		&o.SlippageBps, &o.State, &o.PnL, &o.CreatedAt, &o.ClosedAt)
	if err != nil {
		return nil, fmt.Errorf("find open paper order: %w", err)
	}
	return o, nil
}

// UpdateOrderState updates the state, close price, and PnL of a paper order.
func (r *PaperRepo) UpdateOrderState(ctx context.Context, orderID string, state string, closePrice, pnl decimal.Decimal, closedAt time.Time) error {
	_, err := r.pg.Exec(ctx, `
		UPDATE paper_orders SET state = $2, close_price = $3, pnl = $4, closed_at = $5
		WHERE id = $1
	`, orderID, state, closePrice, pnl, closedAt)
	return err
}

// UpdateAccountBalance updates the current balance and equity of a paper account.
func (r *PaperRepo) UpdateAccountBalance(ctx context.Context, accountID string, balance, equity decimal.Decimal) error {
	_, err := r.pg.Exec(ctx, `
		UPDATE paper_accounts SET current_balance = $2, equity = $3 WHERE id = $1
	`, accountID, balance, equity)
	return err
}

// ListOrders returns all paper orders for a paper account.
func (r *PaperRepo) ListOrders(ctx context.Context, paperAccountID string) ([]*PaperOrder, error) {
	rows, err := r.pg.Query(ctx, `
		SELECT id, paper_account_id, strategy_id, symbol, side, volume, fill_price, slippage_bps, state, pnl, created_at, closed_at
		FROM paper_orders
		WHERE paper_account_id = $1
		ORDER BY created_at DESC
	`, paperAccountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*PaperOrder
	for rows.Next() {
		o := &PaperOrder{}
		if err := rows.Scan(&o.ID, &o.PaperAccountID, &o.StrategyID, &o.Symbol, &o.Side,
			&o.Volume, &o.FillPrice, &o.SlippageBps, &o.State, &o.PnL, &o.CreatedAt, &o.ClosedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// CreateStrategy inserts a paper strategy.
func (r *PaperRepo) CreateStrategy(ctx context.Context, s *PaperStrategy) error {
	_, err := r.pg.Exec(ctx, `
		INSERT INTO paper_strategies (user_id, name, description, category, dsl_code, backtest_metrics)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, s.UserID, s.Name, s.Description, s.Category, s.DSLCode, s.BacktestMetrics)
	if err != nil {
		return fmt.Errorf("create paper strategy: %w", err)
	}
	return nil
}

// ListStrategies returns all non-archived paper strategies for a user.
func (r *PaperRepo) ListStrategies(ctx context.Context, userID string) ([]*PaperStrategy, error) {
	rows, err := r.pg.Query(ctx, `
		SELECT id, user_id, name, description, category, dsl_code, backtest_metrics, promoted_at, created_at, archived
		FROM paper_strategies
		WHERE user_id = $1 AND archived = false
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*PaperStrategy
	for rows.Next() {
		s := &PaperStrategy{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.Description, &s.Category,
			&s.DSLCode, &s.BacktestMetrics, &s.PromotedAt, &s.CreatedAt, &s.Archived); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func scanPaperAccounts(rows pgx.Rows) ([]*PaperAccount, error) {
	var out []*PaperAccount
	for rows.Next() {
		a := &PaperAccount{}
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.InitialBalance, &a.CurrentBalance, &a.Equity, &a.Currency, &a.CreatedAt, &a.Archived); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
