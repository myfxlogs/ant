package repository

import (
	"context"
	"fmt"
)

// ReconcileRepository provides queries for deposit reconciliation.
type ReconcileRepository struct {
	db DBTX
}

func NewReconcileRepository(db DBTX) *ReconcileRepository {
	return &ReconcileRepository{db: db}
}

// SumConfirmedDeposits returns the total amount of all CONFIRMED deposits.
func (r *ReconcileRepository) SumConfirmedDeposits(ctx context.Context) (string, error) {
	var total string
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount)::text, '0') FROM deposits WHERE status = 'CONFIRMED'`,
	).Scan(&total)
	if err != nil {
		return "", fmt.Errorf("reconcile repo: sum deposits: %w", err)
	}
	return total, nil
}

// SumDepositCredits returns the total amount credited to wallets via deposit transactions.
func (r *ReconcileRepository) SumDepositCredits(ctx context.Context) (string, error) {
	var total string
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount)::text, '0') FROM wallet_transactions WHERE tx_type = 'deposit'`,
	).Scan(&total)
	if err != nil {
		return "", fmt.Errorf("reconcile repo: sum deposit credits: %w", err)
	}
	return total, nil
}

// SumSweptAmounts returns the total amount of all DONE sweep_logs.
// Used by chain reconcile to compute expected on-chain balance: deposits - swept.
func (r *ReconcileRepository) SumSweptAmounts(ctx context.Context) (string, error) {
	var total string
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount)::text, '0') FROM sweep_logs WHERE status = 'DONE'`,
	).Scan(&total)
	if err != nil {
		return "", fmt.Errorf("reconcile repo: sum swept: %w", err)
	}
	return total, nil
}
