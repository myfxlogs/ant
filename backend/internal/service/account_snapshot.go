package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"anttrader/internal/repository"
)

// RecordBalanceSnapshot inserts a throttled equity/balance snapshot row.
// Writes at most once per hour per account to bound disk growth
// (unthrottled profit updates can fire every few seconds).
func (s *AccountService) RecordBalanceSnapshot(ctx context.Context, id string, userID string, balance, equity, margin, freeMargin float64) error {
	s.snapshotThrottleMu.Lock()
	last, ok := s.snapshotThrottle[id]
	// Sweep stale entries (> 2h idle) every ~100 calls to bound map growth.
	if len(s.snapshotThrottle)%100 == 0 {
		for k, v := range s.snapshotThrottle {
			if time.Since(v) > 2*time.Hour {
				delete(s.snapshotThrottle, k)
			}
		}
	}
	if ok && time.Since(last) < time.Hour {
		s.snapshotThrottleMu.Unlock()
		return nil // throttled
	}
	// Mark before releasing lock so concurrent callers see a fresh timestamp.
	s.snapshotThrottle[id] = time.Now()
	s.snapshotThrottleMu.Unlock()

	_, err := s.db.Exec(ctx,
		`INSERT INTO account_balance_history (account_id, user_id, balance, equity, margin, free_margin, recorded_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		id, userID, balance, equity, margin, freeMargin)
	if err != nil {
		return fmt.Errorf("service: record balance snapshot: %w", err)
	}

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
			ID:      pgUUIDToString(r.ID),
			Status:  r.AccountStatus,
			Balance: pgNumericToFloat64Ignore(r.Balance),
			Equity:  pgNumericToFloat64Ignore(r.Equity),
			Credit:  pgNumericToFloat64Ignore(r.Credit),
			Margin:  pgNumericToFloat64Ignore(r.Margin),
			FreeMargin:  pgNumericToFloat64Ignore(r.FreeMargin),
			MarginLevel: pgNumericToFloat64Ignore(r.MarginLevel),
		}
	}
	return out, nil
}

// UpdateSummaryCache incrementally updates the in-memory summary for a user
// from a profit event. Called from the pipeline on every account profit update.
func (s *AccountService) UpdateSummaryCache(userID, accountID string, balance, equity float64, status string) {
	s.summaryMu.Lock()
	defer s.summaryMu.Unlock()

	entry, ok := s.summaryCache[userID]
	if !ok {
		return // cold: first GetUserAccountsSummary will seed from DB
	}

	prev, existed := entry.accounts[accountID]
	entry.accounts[accountID] = accountSummaryItem{balance: balance, equity: equity, status: status}

	if !existed {
		// New account added while cache was warm — recompute aggregates.
		entry.summary.AccountCount++
		entry.summary.TotalBalance += balance
		entry.summary.TotalEquity += equity
		entry.summary.TotalProfit += equity - balance
		if status == "connected" {
			entry.summary.ConnectedCount++
		}
	} else {
		// Existing account — update aggregate deltas.
		entry.summary.TotalBalance += balance - prev.balance
		entry.summary.TotalEquity += equity - prev.equity
		entry.summary.TotalProfit += (equity - balance) - (prev.equity - prev.balance)
		if prev.status != "connected" && status == "connected" {
			entry.summary.ConnectedCount++
		} else if prev.status == "connected" && status != "connected" {
			entry.summary.ConnectedCount--
		}
	}
}

// InvalidateSummaryCache removes the cached summary for a user, forcing the next
// GetUserAccountsSummary call to re-seed from DB. Call on account create/delete/disconnect.
func (s *AccountService) InvalidateSummaryCache(userID string) {
	s.summaryMu.Lock()
	delete(s.summaryCache, userID)
	s.summaryMu.Unlock()
}

// GetUserAccountsSummary computes an aggregated summary of a user's accounts (#30).
// Uses an in-memory cache seeded from DB on first access and kept fresh by
// UpdateSummaryCache on every profit event.
func (s *AccountService) GetUserAccountsSummary(ctx context.Context, userID string) (*UserAccountsSummary, error) {
	// Fast path: read from cache.
	s.summaryMu.RLock()
	if entry, ok := s.summaryCache[userID]; ok {
		cpy := entry.summary // stack copy under read lock
		s.summaryMu.RUnlock()
		return &cpy, nil
	}
	s.summaryMu.RUnlock()

	// Slow path: seed cache from DB under write lock with double-check.
	s.summaryMu.Lock()
	defer s.summaryMu.Unlock()

	if entry, ok := s.summaryCache[userID]; ok {
		cpy := entry.summary
		return &cpy, nil
	}

	rows, err := s.db.Query(ctx,
		"SELECT id::text, balance, equity, account_status FROM mt_accounts WHERE user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entry := &userSummaryCacheEntry{
		accounts: make(map[string]accountSummaryItem),
	}
	for rows.Next() {
		var id string
		var balance, equity sql.NullFloat64
		var status string
		if err := rows.Scan(&id, &balance, &equity, &status); err != nil {
			return nil, fmt.Errorf("service: get user accounts summary: scan row: %w", err)
		}
		entry.summary.TotalBalance += balance.Float64
		entry.summary.TotalEquity += equity.Float64
		entry.summary.TotalProfit += equity.Float64 - balance.Float64
		entry.summary.AccountCount++
		if status == "connected" {
			entry.summary.ConnectedCount++
		}
		entry.accounts[id] = accountSummaryItem{
			balance: balance.Float64, equity: equity.Float64, status: status,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	s.summaryCache[userID] = entry
	cpy := entry.summary
	return &cpy, nil
}
