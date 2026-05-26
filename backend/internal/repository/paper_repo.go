package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PaperAccount represents a virtual trading account for paper trading.
type PaperAccount struct {
	ID             string
	UserID         string
	Name           string
	InitialBalance float64
	CurrentBalance float64
	Equity         float64
	Currency       string
	CreatedAt      time.Time
	Archived       bool
}

// PaperOrder represents a simulated order.
type PaperOrder struct {
	ID              string
	PaperAccountID  string
	StrategyID      *string
	Symbol          string
	Side            string
	Volume          float64
	FillPrice       *float64
	SlippageBps     float64
	State           string
	CreatedAt       time.Time
	ClosedAt        *time.Time
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
func (r *PaperRepo) CreateAccount(ctx context.Context, userID, name string, initialBalance float64) (*PaperAccount, error) {
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
		INSERT INTO paper_orders (paper_account_id, strategy_id, symbol, side, volume, fill_price, slippage_bps, state)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, o.PaperAccountID, o.StrategyID, o.Symbol, o.Side, o.Volume, o.FillPrice, o.SlippageBps, o.State)
	return fmt.Errorf("create paper order: %w", err)
}

// ListOrders returns all paper orders for a paper account.
func (r *PaperRepo) ListOrders(ctx context.Context, paperAccountID string) ([]*PaperOrder, error) {
	rows, err := r.pg.Query(ctx, `
		SELECT id, paper_account_id, strategy_id, symbol, side, volume, fill_price, slippage_bps, state, created_at, closed_at
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
			&o.Volume, &o.FillPrice, &o.SlippageBps, &o.State, &o.CreatedAt, &o.ClosedAt); err != nil {
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
	return fmt.Errorf("create paper strategy: %w", err)
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
