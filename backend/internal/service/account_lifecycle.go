package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// ── Status Lifecycle ──

// SetStatus updates the account_status column and invalidates summary cache on disconnect.
func (s *AccountService) SetStatus(ctx context.Context, userID uuid.UUID, id string, status AccountStatus) error {
	tag, err := s.db.Exec(ctx,
		"UPDATE mt_accounts SET account_status = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid AND user_id = $2",
		id, userID, string(status))
	if err != nil {
		return fmt.Errorf("service: set status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("account not found: %s", id)
	}
	if status == StatusDisconnected {
		s.InvalidateSummaryCache(userID.String())
	}
	return nil
}

// DisconnectAccountByID sets the account status to disconnected by account ID only
// (for system callbacks where userID is not available).
func (s *AccountService) DisconnectAccountByID(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx,
		"UPDATE mt_accounts SET account_status = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid",
		id, string(StatusDisconnected))
	if err != nil {
		return fmt.Errorf("service: disconnect account by id: %w", err)
	}
	return nil
}

// ── Password ──

// UpdateTradingPassword updates the trading password for an account.
func (s *AccountService) UpdateTradingPassword(ctx context.Context, userID uuid.UUID, id, oldPassword, newPassword string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE mt_accounts SET password = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1::uuid AND user_id = $2 AND password = $3
	`, id, userID, oldPassword, newPassword)
	if err != nil {
		return fmt.Errorf("service: update trading password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountPasswordMismatch
	}
	return nil
}

// ── Cleanup ──

// CleanupOldSnapshots deletes account_balance_history rows older than retention
// (default 90 days) and trade_records older than 2 years — keeps disk bounded.
func (s *AccountService) CleanupOldSnapshots(ctx context.Context, log *zap.Logger) {
	if _, err := s.db.Exec(ctx,
		`DELETE FROM account_balance_history WHERE recorded_at < NOW() - INTERVAL '90 days'`); err != nil {
		log.Warn("cleanup: account_balance_history", zap.Error(err))
	}
	if _, err := s.db.Exec(ctx,
		`DELETE FROM trade_records WHERE close_time < NOW() - INTERVAL '2 years'`); err != nil {
		log.Warn("cleanup: trade_records", zap.Error(err))
	}
}

// ── Snapshots / Cache / Summary ──

// RecordBalanceSnapshot inserts a throttled equity/balance snapshot row.
// Writes at most once per hour per account to bound disk growth.
func (s *AccountService) RecordBalanceSnapshot(ctx context.Context, id, userID string, balance, equity, margin, freeMargin decimal.Decimal) error {
	s.snapshotThrottleMu.Lock()
	last, ok := s.snapshotThrottle[id]
	if len(s.snapshotThrottle)%100 == 0 {
		for k, v := range s.snapshotThrottle {
			if time.Since(v) > 2*time.Hour {
				delete(s.snapshotThrottle, k)
			}
		}
	}
	if ok && time.Since(last) < time.Hour {
		s.snapshotThrottleMu.Unlock()
		return nil
	}
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
			Balance:     pgNumericToDecimal(r.Balance),
			Equity:      pgNumericToDecimal(r.Equity),
			Credit:      pgNumericToDecimal(r.Credit),
			Margin:      pgNumericToDecimal(r.Margin),
			FreeMargin:  pgNumericToDecimal(r.FreeMargin),
			MarginLevel: pgNumericToDecimal(r.MarginLevel),
		}
	}
	return out, nil
}

// UpdateSummaryCache incrementally updates the in-memory summary for a user
// from a profit event.
func (s *AccountService) UpdateSummaryCache(userID, accountID string, balance, equity decimal.Decimal, status string) {
	s.summaryMu.Lock()
	defer s.summaryMu.Unlock()

	entry, ok := s.summaryCache[userID]
	if !ok {
		return
	}

	prev, existed := entry.accounts[accountID]
	entry.accounts[accountID] = accountSummaryItem{balance: balance, equity: equity, status: status}

	if !existed {
		entry.summary.AccountCount++
		entry.summary.TotalBalance = entry.summary.TotalBalance.Add(balance)
		entry.summary.TotalEquity = entry.summary.TotalEquity.Add(equity)
		entry.summary.TotalProfit = entry.summary.TotalProfit.Add(equity.Sub(balance))
		if status == "connected" {
			entry.summary.ConnectedCount++
		}
	} else {
		entry.summary.TotalBalance = entry.summary.TotalBalance.Add(balance.Sub(prev.balance))
		entry.summary.TotalEquity = entry.summary.TotalEquity.Add(equity.Sub(prev.equity))
		entry.summary.TotalProfit = entry.summary.TotalProfit.Add(equity.Sub(balance).Sub(prev.equity.Sub(prev.balance)))
		if prev.status != "connected" && status == "connected" {
			entry.summary.ConnectedCount++
		} else if prev.status == "connected" && status != "connected" {
			entry.summary.ConnectedCount--
		}
	}
}

// InvalidateSummaryCache removes the cached summary for a user, forcing the next
// GetUserAccountsSummary call to re-seed from DB.
func (s *AccountService) InvalidateSummaryCache(userID string) {
	s.summaryMu.Lock()
	delete(s.summaryCache, userID)
	s.summaryMu.Unlock()
}

// GetUserAccountsSummary computes an aggregated summary of a user's accounts.
func (s *AccountService) GetUserAccountsSummary(ctx context.Context, userID string) (*UserAccountsSummary, error) {
	s.summaryMu.RLock()
	if entry, ok := s.summaryCache[userID]; ok {
		cpy := entry.summary
		s.summaryMu.RUnlock()
		return &cpy, nil
	}
	s.summaryMu.RUnlock()

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
		var balance, equity pgtype.Numeric
		var status string
		if err := rows.Scan(&id, &balance, &equity, &status); err != nil {
			return nil, fmt.Errorf("service: get user accounts summary: scan row: %w", err)
		}
		bal := pgNumericToDecimal(balance)
		eq := pgNumericToDecimal(equity)
		entry.summary.TotalBalance = entry.summary.TotalBalance.Add(bal)
		entry.summary.TotalEquity = entry.summary.TotalEquity.Add(eq)
		entry.summary.TotalProfit = entry.summary.TotalProfit.Add(eq.Sub(bal))
		entry.summary.AccountCount++
		if status == "connected" {
			entry.summary.ConnectedCount++
		}
		entry.accounts[id] = accountSummaryItem{
			balance: bal, equity: eq, status: status,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	s.summaryCache[userID] = entry
	cpy := entry.summary
	return &cpy, nil
}

// ErrPlatformPasswordMismatch is returned when the platform login password is incorrect.
var ErrPlatformPasswordMismatch = errors.New("platform password does not match")

// VerifyUserPassword checks the user's platform login password against the stored hash.
func (s *AccountService) VerifyUserPassword(ctx context.Context, userID uuid.UUID, password string) error {
	var hash string
	err := s.db.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash)
	if err != nil {
		return fmt.Errorf("service: verify user password: %w", err)
	}
	if !VerifyPassword(hash, password) {
		return ErrPlatformPasswordMismatch
	}
	return nil
}
