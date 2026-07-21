package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/secrets"
)

// GetDecryptedPassword returns the decrypted MT password for internal use (mdgateway connection).
// The plaintext is never logged or persisted.
func (s *AccountService) GetDecryptedPassword(ctx context.Context, accountID string) (string, error) {
	if s.sec == nil {
		return "", fmt.Errorf("service: get decrypted password: secrets client not configured")
	}
	var encPwd []byte
	err := s.db.QueryRow(ctx,
		`SELECT password_encrypted FROM mt_accounts WHERE id = $1::uuid AND deleted_at IS NULL`,
		accountID).Scan(&encPwd)
	if err != nil {
		return "", fmt.Errorf("service: get decrypted password: %w", err)
	}
	if encPwd == nil {
		return "", fmt.Errorf("service: get decrypted password: no encrypted password stored")
	}
	plain, err := s.sec.Decrypt(ctx, secrets.PurposeMTPassword, encPwd)
	if err != nil {
		return "", fmt.Errorf("service: get decrypted password: decrypt: %w", err)
	}
	return string(plain), nil
}

// BackfillPlaintextCredentials encrypts plaintext passwords for existing accounts.
// Called once on startup to migrate legacy data before dropping plaintext columns.
// No-ops if the plaintext password column has already been dropped.
func (s *AccountService) BackfillPlaintextCredentials(ctx context.Context) (int, error) {
	if s.sec == nil {
		return 0, fmt.Errorf("service: backfill: secrets client not configured")
	}
	// Check if the password column still exists before querying it.
	var hasCol bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = 'mt_accounts' AND column_name = 'password')`).Scan(&hasCol)
	if err != nil {
		return 0, fmt.Errorf("service: backfill: check column: %w", err)
	}
	if !hasCol {
		if s.log != nil {
			s.log.Info("service: plaintext credential backfill skipped (password column already dropped)")
		}
		return 0, nil
	}
	if s.log != nil {
		s.log.Info("service: starting plaintext credential backfill")
	}
	rows, err := s.db.Query(ctx,
		`SELECT id::text, password FROM mt_accounts
		 WHERE password_encrypted IS NULL AND password IS NOT NULL AND password <> '' AND deleted_at IS NULL`)
	if err != nil {
		return 0, fmt.Errorf("service: backfill: query: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, plaintext string
		if err := rows.Scan(&id, &plaintext); err != nil {
			if s.log != nil {
				s.log.Warn("service: backfill: scan failed", zap.Error(err))
			}
			continue
		}
		enc, err := s.sec.Encrypt(ctx, secrets.PurposeMTPassword, []byte(plaintext))
		if err != nil {
			if s.log != nil {
				s.log.Error("service: backfill: encrypt failed", zap.String("account", id), zap.Error(err))
			}
			continue
		}
		_, err = s.db.Exec(ctx,
			`UPDATE mt_accounts SET password_encrypted = $2 WHERE id = $1::uuid AND password_encrypted IS NULL`,
			id, enc)
		if err != nil {
			if s.log != nil {
				s.log.Error("service: backfill: update failed", zap.String("account", id), zap.Error(err))
			}
			continue
		}
		count++
	}
	if s.log != nil {
		s.log.Info("service: plaintext credential backfill complete", zap.Int("migrated", count))
	}
	return count, rows.Err()
}

// GetUserAccountIDs returns all account IDs belonging to a user.
func (s *AccountService) GetUserAccountIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.Query(ctx, "SELECT id FROM mt_accounts WHERE user_id = $1 AND deleted_at IS NULL", userID)
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

// UpdateSummaryCache updates the in-memory account summary cache for a user.
func (s *AccountService) UpdateSummaryCache(userID, accountID string, balance, equity decimal.Decimal, status string) {
	s.summaryMu.Lock()
	defer s.summaryMu.Unlock()
	entry, ok := s.summaryCache[userID]
	if !ok {
		entry = &userSummaryCacheEntry{accounts: make(map[string]accountSummaryItem)}
		s.summaryCache[userID] = entry
	}
	entry.accounts[accountID] = accountSummaryItem{balance: balance, equity: equity, status: status}
	entry.summary = computeUserAccountsSummary(entry.accounts)
}

// InvalidateSummaryCache drops the cached summary for a user.
func (s *AccountService) InvalidateSummaryCache(userID string) {
	s.summaryMu.Lock()
	defer s.summaryMu.Unlock()
	delete(s.summaryCache, userID)
}

// computeUserAccountsSummary aggregates per-account items into a summary.
func computeUserAccountsSummary(accounts map[string]accountSummaryItem) UserAccountsSummary {
	var totalBalance, totalEquity decimal.Decimal
	var connected int32
	for _, a := range accounts {
		totalBalance = totalBalance.Add(a.balance)
		totalEquity = totalEquity.Add(a.equity)
		if a.status == string(StatusConnected) {
			connected++
		}
	}
	return UserAccountsSummary{
		TotalBalance:   totalBalance,
		TotalEquity:    totalEquity,
		TotalProfit:    totalEquity.Sub(totalBalance),
		AccountCount:   int32(len(accounts)),
		ConnectedCount: connected,
	}
}

// GetUserAccountsSummary returns the aggregated summary for a user, using the cache when available.
func (s *AccountService) GetUserAccountsSummary(ctx context.Context, userID string) (*UserAccountsSummary, error) {
	s.summaryMu.RLock()
	entry, ok := s.summaryCache[userID]
	s.summaryMu.RUnlock()
	if ok {
		return &entry.summary, nil
	}
	var totalBalance, totalEquity pgtype.Numeric
	var connected, total int32
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(balance),0), COALESCE(SUM(equity),0),
		       COUNT(*) FILTER (WHERE account_status = 'connected'), COUNT(*)
		FROM mt_accounts
		WHERE user_id = $1::uuid AND deleted_at IS NULL
	`, userID).Scan(&totalBalance, &totalEquity, &connected, &total)
	if err != nil {
		return nil, fmt.Errorf("service: get user accounts summary: %w", err)
	}
	bal := pgNumericToDecimal(totalBalance)
	eq := pgNumericToDecimal(totalEquity)
	summary := UserAccountsSummary{
		TotalBalance:   bal,
		TotalEquity:    eq,
		TotalProfit:    eq.Sub(bal),
		AccountCount:   total,
		ConnectedCount: connected,
	}
	s.summaryMu.Lock()
	s.summaryCache[userID] = &userSummaryCacheEntry{
		accounts: make(map[string]accountSummaryItem),
		summary:  summary,
	}
	s.summaryMu.Unlock()
	return &summary, nil
}

// GetUserAccountSnapshots returns the current state of all MT accounts for a user.
func (s *AccountService) GetUserAccountSnapshots(ctx context.Context, userID string) ([]AccountSnapshot, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, account_status, balance, equity, credit, margin, free_margin, margin_level
		FROM mt_accounts
		WHERE user_id = $1::uuid AND deleted_at IS NULL
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("service: get account snapshots: %w", err)
	}
	defer rows.Close()
	var out []AccountSnapshot
	for rows.Next() {
		var id, status string
		var bal, eq, cr, mar, fm, ml pgtype.Numeric
		if err := rows.Scan(&id, &status, &bal, &eq, &cr, &mar, &fm, &ml); err != nil {
			return nil, fmt.Errorf("service: scan account snapshot: %w", err)
		}
		out = append(out, AccountSnapshot{
			ID:          id,
			Status:      status,
			Balance:     pgNumericToDecimal(bal),
			Equity:      pgNumericToDecimal(eq),
			Credit:      pgNumericToDecimal(cr),
			Margin:      pgNumericToDecimal(mar),
			FreeMargin:  pgNumericToDecimal(fm),
			MarginLevel: pgNumericToDecimal(ml),
		})
	}
	return out, rows.Err()
}

// RecordBalanceSnapshot persists a balance sample to the time-series table, throttled per account.
func (s *AccountService) RecordBalanceSnapshot(ctx context.Context, accountID, userID string, balance, equity, margin, freeMargin decimal.Decimal) error {
	const minInterval = 5 * time.Second
	s.snapshotThrottleMu.Lock()
	last, ok := s.snapshotThrottle[accountID]
	if ok && time.Since(last) < minInterval {
		s.snapshotThrottleMu.Unlock()
		return nil
	}
	s.snapshotThrottle[accountID] = time.Now()
	s.snapshotThrottleMu.Unlock()
	_, err := s.db.Exec(ctx, `
		INSERT INTO account_balance_snapshots (account_id, user_id, balance, equity, margin, free_margin)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)
	`, accountID, userID, balance, equity, margin, freeMargin)
	if err != nil {
		return fmt.Errorf("service: record balance snapshot: %w", err)
	}
	return nil
}
