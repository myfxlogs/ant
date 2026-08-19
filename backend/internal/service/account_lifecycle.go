package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/repository"
)

// ── Account Info ──

// GetAccountCredentials returns the credentials needed for MT password verification.
// Does NOT return the decrypted password — callers verify against the broker directly.
func (s *AccountService) GetAccountCredentials(ctx context.Context, userID uuid.UUID, id string) (*AccountCredentials, error) {
	pgID, err := stringToPgUUID(id)
	if err != nil {
		return nil, fmt.Errorf("service: get account credentials: invalid account id: %w", err)
	}
	row, err := s.queries.GetAccountCredentials(ctx, repository.GetAccountCredentialsParams{ID: pgID, UserID: uuidToPgUUID(userID)})
	if err != nil {
		return nil, fmt.Errorf("service: get account credentials: %w", err)
	}
	return &AccountCredentials{
		Login:         row.Login,
		Platform:      row.MtType,
		BrokerHost:    row.BrokerHost,
		BrokerCompany: row.BrokerCompany,
		AccountID:     id,
	}, nil
}

// AccountInfoUpdate holds parameters for updating MT account info.
type AccountInfoUpdate struct {
	Tx         pgx.Tx
	UserID     uuid.UUID
	ID         string
	Balance    decimal.Decimal
	Equity     decimal.Decimal
	Credit     decimal.Decimal
	Margin     decimal.Decimal
	FreeMargin decimal.Decimal
	Leverage   int64
	Currency   string
	IsInvestor bool
}

// UpdateAccountInfoTx updates balance/equity/margin/leverage/currency within a transaction.
// Does NOT touch account_status — that is owned by the gateway lifecycle.
func (s *AccountService) UpdateAccountInfoTx(ctx context.Context, p AccountInfoUpdate) error {
	_, err := p.Tx.Exec(ctx, `
		UPDATE mt_accounts SET
			balance = $3, equity = $4, credit = $5, margin = $6,
			free_margin = $7, leverage = $8, currency = $9,
			is_investor = $10, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1::uuid AND user_id = $2 AND deleted_at IS NULL
	`, p.ID, p.UserID, p.Balance, p.Equity, p.Credit, p.Margin, p.FreeMargin, p.Leverage, p.Currency, p.IsInvestor)
	if err != nil {
		return fmt.Errorf("service: update account info: %w", err)
	}
	return nil
}

// UpdateAccountInfo updates balance/equity/margin/leverage/currency after MT verification.
func (s *AccountService) UpdateAccountInfo(ctx context.Context, p AccountInfoUpdate) error {
	_, err := s.db.Exec(ctx, `
		UPDATE mt_accounts SET
			balance = $3, equity = $4, credit = $5, margin = $6,
			free_margin = $7, leverage = $8, currency = $9,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1::uuid AND user_id = $2 AND deleted_at IS NULL
	`, p.ID, p.UserID, p.Balance, p.Equity, p.Credit, p.Margin, p.FreeMargin, p.Leverage, p.Currency)
	if err != nil {
		return fmt.Errorf("service: update account info: %w", err)
	}
	return nil
}

// UpdateAccountMetrics updates runtime balance/equity/margin metrics from broker callbacks.
func (s *AccountService) UpdateAccountMetrics(ctx context.Context, userID uuid.UUID, id string, balance, equity, credit, margin, freeMargin, marginLevel decimal.Decimal) error {
	pgID, err := stringToPgUUID(id)
	if err != nil {
		return fmt.Errorf("service: update account metrics: invalid account id: %w", err)
	}
	balN, err := decimalToPgNumeric(balance)
	if err != nil {
		return fmt.Errorf("service: update account metrics: balance: %w", err)
	}
	eqN, err := decimalToPgNumeric(equity)
	if err != nil {
		return fmt.Errorf("service: update account metrics: equity: %w", err)
	}
	crN, err := decimalToPgNumeric(credit)
	if err != nil {
		return fmt.Errorf("service: update account metrics: credit: %w", err)
	}
	marginN, err := decimalToPgNumeric(margin)
	if err != nil {
		return fmt.Errorf("service: update account metrics: margin: %w", err)
	}
	fmN, err := decimalToPgNumeric(freeMargin)
	if err != nil {
		return fmt.Errorf("service: update account metrics: free_margin: %w", err)
	}
	mlN, err := decimalToPgNumeric(marginLevel)
	if err != nil {
		return fmt.Errorf("service: update account metrics: margin_level: %w", err)
	}
	return s.queries.UpdateAccountMetrics(ctx, repository.UpdateAccountMetricsParams{
		ID:          pgID,
		UserID:      uuidToPgUUID(userID),
		Balance:     balN,
		Equity:      eqN,
		Credit:      crN,
		Margin:      marginN,
		FreeMargin:  fmN,
		MarginLevel: mlN,
	})
}

// UpdateBrokerThresholds updates broker margin_call/stop_out thresholds on an account.
// This is a system callback (called from the gateway pipeline where only accountID
// is available). Do not expose via ConnectRPC without adding a user_id check.
func (s *AccountService) UpdateBrokerThresholds(ctx context.Context, id string, marginCallPct, stopOutPct decimal.Decimal) error {
	_, err := s.db.Exec(ctx,
		`UPDATE mt_accounts SET broker_margin_call_pct=$1, broker_stop_out_pct=$2,
		 updated_at=CURRENT_TIMESTAMP WHERE id=$3 AND deleted_at IS NULL`, marginCallPct, stopOutPct, id)
	if err != nil {
		return fmt.Errorf("service: update broker thresholds: %w", err)
	}
	return nil
}

// LogAudit inserts an account audit event. Non-blocking — failures are logged but not returned.
func (s *AccountService) LogAudit(ctx context.Context, accountID, userID uuid.UUID, action, detail string) {
	if _, err := s.db.Exec(ctx,
		`INSERT INTO account_audit_log (account_id, user_id, action, detail) VALUES ($1, $2, $3, $4)`,
		accountID, userID, action, detail); err != nil && s.log != nil {
		s.log.Warn("LogAudit: insert failed", zap.String("action", action), zap.Error(err))
	}
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

// DisconnectAccountByID marks the given account as disconnected without a user_id check.
// Intended for gateway lifecycle callbacks where only the account ID is known.
func (s *AccountService) DisconnectAccountByID(ctx context.Context, accountID string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE mt_accounts SET account_status = 'disconnected', updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid AND deleted_at IS NULL`,
		accountID)
	if err != nil {
		return fmt.Errorf("service: disconnect account: %w", err)
	}
	return nil
}

// SetStatus updates the account_status for a user-owned account.
func (s *AccountService) SetStatus(ctx context.Context, userID uuid.UUID, id string, status AccountStatus) error {
	pgID, err := stringToPgUUID(id)
	if err != nil {
		return fmt.Errorf("service: set status: invalid account id: %w", err)
	}
	_, err = s.db.Exec(ctx,
		`UPDATE mt_accounts SET account_status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2::uuid AND user_id = $3 AND deleted_at IS NULL`,
		string(status), pgID, uuidToPgUUID(userID))
	if err != nil {
		return fmt.Errorf("service: set status: %w", err)
	}
	s.InvalidateSummaryCache(userID.String())
	return nil
}

// CleanupOldSnapshots purges balance snapshot records older than the retention window.
func (s *AccountService) CleanupOldSnapshots(ctx context.Context, log *zap.Logger) error {
	tag, err := s.db.Exec(ctx,
		`DELETE FROM account_balance_history WHERE recorded_at < NOW() - INTERVAL '90 days'`)
	if err != nil {
		return fmt.Errorf("service: cleanup old snapshots: %w", err)
	}
	if log != nil {
		log.Info("cleaned up old balance snapshots", zap.Int64("deleted", tag.RowsAffected()))
	}
	return nil
}
