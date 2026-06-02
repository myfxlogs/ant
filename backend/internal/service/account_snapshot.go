package service

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"anttrader/internal/repository"
)

// snapshotThrottle prevents writing more than one balance snapshot per account per hour.
var (
	snapshotThrottleMu sync.Mutex
	snapshotThrottle   = map[string]time.Time{}
)

// RecordBalanceSnapshot inserts a throttled equity/balance snapshot row.
// Writes at most once per hour per account to bound disk growth
// (unthrottled profit updates can fire every few seconds).
func (s *AccountService) RecordBalanceSnapshot(ctx context.Context, id string, balance, equity, margin, freeMargin float64) error {
	snapshotThrottleMu.Lock()
	last, ok := snapshotThrottle[id]
	// Sweep stale entries (> 2h idle) every ~100 calls to bound map growth.
	if len(snapshotThrottle)%100 == 0 {
		for k, v := range snapshotThrottle {
			if time.Since(v) > 2*time.Hour {
				delete(snapshotThrottle, k)
			}
		}
	}
	snapshotThrottleMu.Unlock()
	if ok && time.Since(last) < time.Hour {
		return nil // throttled
	}

	_, err := s.db.Exec(ctx,
		`INSERT INTO account_balance_history (account_id, balance, equity, margin, free_margin, recorded_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())`,
		id, balance, equity, margin, freeMargin)
	if err != nil {
		return fmt.Errorf("service: record balance snapshot: %w", err)
	}

	snapshotThrottleMu.Lock()
	snapshotThrottle[id] = time.Now()
	snapshotThrottleMu.Unlock()
	return nil
}

// UserOwnsAccount checks if an account belongs to the given user.
func (s *AccountService) UserOwnsAccount(ctx context.Context, userID, accountID string) (bool, error) {
	pgUserID, err := stringToPgUUID(userID)
	if err != nil {
		return false, fmt.Errorf("service: user owns account: invalid user id: %w", err)
	}
	pgAccountID, err := stringToPgUUID(accountID)
	if err != nil {
		return false, fmt.Errorf("service: user owns account: invalid account id: %w", err)
	}
	return s.queries.UserOwnsAccount(ctx, repository.UserOwnsAccountParams{ID: pgAccountID, UserID: pgUserID})
}

// GetUserAccountIDs returns all account IDs belonging to a user.
func (s *AccountService) GetUserAccountIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.Query(ctx, "SELECT id FROM mt_accounts WHERE user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetUserAccountSnapshots returns current state for all of a user's MT accounts.
func (s *AccountService) GetUserAccountSnapshots(ctx context.Context, userID string) ([]AccountSnapshot, error) {
	pgID, err := stringToPgUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("service: get account snapshots: invalid user id: %w", err)
	}
	rows, err := s.queries.GetAccountSnapshots(ctx, pgID)
	if err != nil {
		return nil, err
	}
	out := make([]AccountSnapshot, len(rows))
	for i, r := range rows {
		out[i] = AccountSnapshot{
			ID:          pgUUIDToString(r.ID),
			Status:      r.AccountStatus,
			Balance:     pgNumericToFloat64(r.Balance),
			Equity:      pgNumericToFloat64(r.Equity),
			Credit:      pgNumericToFloat64(r.Credit),
			Margin:      pgNumericToFloat64(r.Margin),
			FreeMargin:  pgNumericToFloat64(r.FreeMargin),
			MarginLevel: pgNumericToFloat64(r.MarginLevel),
		}
	}
	return out, nil
}

// GetUserAccountsSummary computes an aggregated summary of a user's accounts (#30).
func (s *AccountService) GetUserAccountsSummary(ctx context.Context, userID string) (*UserAccountsSummary, error) {
	rows, err := s.db.Query(ctx,
		"SELECT balance, equity, account_status FROM mt_accounts WHERE user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sum UserAccountsSummary
	for rows.Next() {
		var balance, equity sql.NullFloat64
		var status string
		if err := rows.Scan(&balance, &equity, &status); err != nil {
			if s.log != nil {
				s.log.Warn("GetUserAccountsSummary: scan error, skipping row", zap.Error(err))
			}
			continue
		}
		sum.TotalBalance += balance.Float64
		sum.TotalEquity += equity.Float64
		sum.TotalProfit += equity.Float64 - balance.Float64
		sum.AccountCount++
		if status == "connected" {
			sum.ConnectedCount++
		}
	}
	return &sum, rows.Err()
}
