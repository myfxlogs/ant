package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"alphaforge/internal/repository"
	"alphaforge/internal/secrets"
)

// ── CRUD ──

// ListAccounts returns all accounts belonging to the given user.
func (s *AccountService) ListAccounts(ctx context.Context, userID uuid.UUID) ([]AccountDTO, error) {
	rows, err := s.queries.ListAccounts(ctx, uuidToPgUUID(userID))
	if err != nil {
		return nil, err
	}
	out := make([]AccountDTO, len(rows))
	for i, r := range rows {
		out[i] = mtAccountToDTO(r)
	}
	return out, nil
}

// GetAccount returns a single account by ID.
func (s *AccountService) GetAccount(ctx context.Context, userID uuid.UUID, accountID string) (*AccountDTO, error) {
	pgID, err := stringToPgUUID(accountID)
	if err != nil {
		return nil, fmt.Errorf("service: get account: invalid account id: %w", err)
	}
	row, err := s.queries.GetAccount(ctx, repository.GetAccountParams{ID: pgID, UserID: uuidToPgUUID(userID)})
	if err != nil {
		return nil, err
	}
	a := mtAccountToDTO(row)
	return &a, nil
}

// BeginTx starts a new database transaction.
func (s *AccountService) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return s.db.Begin(ctx)
}

// CreateAccountTx inserts a new MT account row within a transaction.
// The password is encrypted via secrets.Client before storage.
func (s *AccountService) CreateAccountTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, login, password, mtType, brokerCompany, brokerServer, brokerHost string) (string, error) {
	if s.sec == nil {
		return "", fmt.Errorf("service: create account: secrets client not configured")
	}
	encPwd, err := s.sec.Encrypt(ctx, secrets.PurposeMTPassword, []byte(password))
	if err != nil {
		return "", fmt.Errorf("service: create account: encrypt password: %w", err)
	}
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO mt_accounts (user_id, login, password_encrypted, mt_type, broker_company, broker_server, broker_host, account_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text
	`, userID, login, encPwd, mtType, brokerCompany, brokerServer, brokerHost, string(StatusConnecting)).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return "", ErrAccountAlreadyBound
		}
		return "", fmt.Errorf("service: create account: %w", err)
	}
	return id, nil
}

// CreateAccount inserts a new MT account row and returns the generated ID.
func (s *AccountService) CreateAccount(ctx context.Context, userID uuid.UUID, login, password, mtType, brokerCompany, brokerServer, brokerHost string) (string, error) {
	tx, err := s.BeginTx(ctx)
	if err != nil {
		return "", fmt.Errorf("service: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	id, err := s.CreateAccountTx(ctx, tx, userID, login, password, mtType, brokerCompany, brokerServer, brokerHost)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("service: commit tx: %w", err)
	}
	s.InvalidateSummaryCache(userID.String())
	return id, nil
}

// UpdateAccount updates broker fields for an account.
func (s *AccountService) UpdateAccount(ctx context.Context, userID uuid.UUID, id, brokerCompany, brokerServer, brokerHost string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE mt_accounts SET
			broker_company = COALESCE(NULLIF($3, ''), broker_company),
			broker_server  = COALESCE(NULLIF($4, ''), broker_server),
			broker_host    = COALESCE(NULLIF($5, ''), broker_host),
			updated_at     = CURRENT_TIMESTAMP
		WHERE id = $1::uuid AND user_id = $2 AND deleted_at IS NULL
	`, id, userID, brokerCompany, brokerServer, brokerHost)
	if err != nil {
		return fmt.Errorf("service: update account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// DeleteAccount soft-deletes an MT account by setting deleted_at.
func (s *AccountService) DeleteAccount(ctx context.Context, userID uuid.UUID, id string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE mt_accounts SET deleted_at = NOW(), account_status = 'disconnected', password_encrypted = NULL WHERE id = $1::uuid AND user_id = $2 AND deleted_at IS NULL`,
		id, userID)
	if err != nil {
		return fmt.Errorf("service: delete account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	s.InvalidateSummaryCache(userID.String())
	return nil
}
