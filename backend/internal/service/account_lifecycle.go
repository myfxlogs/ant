package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"anttrader/internal/repository"
)

// ConnectAccount marks an account as ready for connection (#36: now sets status to 'connecting').
func (s *AccountService) ConnectAccount(ctx context.Context, userID uuid.UUID, accountID string) error {
	tag, err := s.db.Exec(ctx,
		"UPDATE mt_accounts SET account_status = 'connecting', updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid AND user_id = $2",
		accountID, userID)
	if err != nil {
		return fmt.Errorf("service: connect account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("account not found: %s", accountID)
	}
	return nil
}

// DisconnectAccount sets the account status to disconnected.
func (s *AccountService) DisconnectAccount(ctx context.Context, userID uuid.UUID, id string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE mt_accounts SET account_status = 'disconnected', updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid AND user_id = $2
	`, id, userID)
	if err != nil {
		return fmt.Errorf("service: disconnect account: %w", err)
	}
	s.InvalidateSummaryCache(userID.String())
	return nil
}

// DisconnectAccountByID sets the account status to disconnected by account ID only
// (for system callbacks where userID is not available).
func (s *AccountService) DisconnectAccountByID(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE mt_accounts SET account_status = 'disconnected', updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid
	`, id)
	if err != nil {
		return fmt.Errorf("service: disconnect account by id: %w", err)
	}
	return nil
}

// ReconnectAccount sets the account status to connecting for re-connection.
func (s *AccountService) ReconnectAccount(ctx context.Context, userID uuid.UUID, id string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE mt_accounts SET account_status = 'connecting', updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid AND user_id = $2
	`, id, userID)
	if err != nil {
		return fmt.Errorf("service: reconnect account: %w", err)
	}
	return nil
}

// MarkAccountNeedsRebind marks an account as needing rebind instead of deleting it (#4).
func (s *AccountService) MarkAccountNeedsRebind(ctx context.Context, userID uuid.UUID, id string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE mt_accounts SET account_status = 'needs_rebind', updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid AND user_id = $2`,
		id, userID)
	if err != nil {
		return fmt.Errorf("service: mark account needs_rebind: %w", err)
	}
	return nil
}

// GetAccountCredentials returns the credentials needed for MT password verification.
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
		Login:      row.Login,
		Platform:   row.MtType,
		BrokerHost: row.BrokerHost,
	}, nil
}

// UpdateAccountInfoTx updates balance/equity/margin/leverage/currency within a transaction.
// Does NOT touch account_status — that is owned by the gateway lifecycle (startGatewayForAccount / OnAccountDisconnect).
func (s *AccountService) UpdateAccountInfoTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, id string, balance, equity, credit, margin, freeMargin float64, leverage int64, currency string, isInvestor bool) error {
	_, err := tx.Exec(ctx, `
		UPDATE mt_accounts SET
			balance = $3,
			equity  = $4,
			credit  = $5,
			margin  = $6,
			free_margin = $7,
			leverage = $8,
			currency = $9,
			is_investor = $10,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1::uuid AND user_id = $2
	`, id, userID, balance, equity, credit, margin, freeMargin, leverage, currency, isInvestor)
	if err != nil {
		return fmt.Errorf("service: update account info: %w", err)
	}
	return nil
}

// UpdateAccountInfo updates balance/equity/margin/leverage/currency after MT verification.
// Does NOT touch account_status — that is owned by the gateway lifecycle (startGatewayForAccount / OnAccountDisconnect).
func (s *AccountService) UpdateAccountInfo(ctx context.Context, userID uuid.UUID, id string, balance, equity, credit, margin, freeMargin float64, leverage int64, currency string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE mt_accounts SET
			balance = $3,
			equity  = $4,
			credit  = $5,
			margin  = $6,
			free_margin = $7,
			leverage = $8,
			currency = $9,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1::uuid AND user_id = $2
	`, id, userID, balance, equity, credit, margin, freeMargin, leverage, currency)
	if err != nil {
		return fmt.Errorf("service: update account info: %w", err)
	}
	return nil
}

// UpdateAccountMetrics updates runtime balance/equity/margin metrics from broker callbacks.
// Unlike UpdateAccountInfo, this does not overwrite leverage or currency.
func (s *AccountService) UpdateAccountMetrics(ctx context.Context, userID uuid.UUID, id string, balance, equity, credit, margin, freeMargin, marginLevel float64) error {
	pgID, err := stringToPgUUID(id)
	if err != nil {
		return fmt.Errorf("service: update account metrics: invalid account id: %w", err)
	}
	balN, err := float64ToPgNumeric(balance)
	if err != nil {
		return fmt.Errorf("service: update account metrics: balance: %w", err)
	}
	eqN, err := float64ToPgNumeric(equity)
	if err != nil {
		return fmt.Errorf("service: update account metrics: equity: %w", err)
	}
	crN, err := float64ToPgNumeric(credit)
	if err != nil {
		return fmt.Errorf("service: update account metrics: credit: %w", err)
	}
	marginN, err := float64ToPgNumeric(margin)
	if err != nil {
		return fmt.Errorf("service: update account metrics: margin: %w", err)
	}
	fmN, err := float64ToPgNumeric(freeMargin)
	if err != nil {
		return fmt.Errorf("service: update account metrics: free_margin: %w", err)
	}
	mlN, err := float64ToPgNumeric(marginLevel)
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
func (s *AccountService) UpdateBrokerThresholds(ctx context.Context, id string, marginCallPct, stopOutPct float64) error {
	_, err := s.db.Exec(ctx,
		`UPDATE mt_accounts SET broker_margin_call_pct=$1, broker_stop_out_pct=$2,
		 updated_at=CURRENT_TIMESTAMP WHERE id=$3`, marginCallPct, stopOutPct, id)
	if err != nil {
		return fmt.Errorf("service: update broker thresholds: %w", err)
	}
	return nil
}

// UpdateTradingPassword updates the trading password for an account (#38: old password verification).
func (s *AccountService) UpdateTradingPassword(ctx context.Context, userID uuid.UUID, id, oldPassword, newPassword string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE mt_accounts SET password = $4, updated_at = CURRENT_TIMESTAMP WHERE id = $1::uuid AND user_id = $2 AND password = $3
	`, id, userID, oldPassword, newPassword)
	if err != nil {
		return fmt.Errorf("service: update trading password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountPasswordMismatch
	}
	return nil
}

// CleanupOldSnapshots deletes account_balance_history rows older than retention
// (default 90 days) and trade_records older than 2 years — keeps disk bounded.
// Designed to run as a daily background job.
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
