package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"alphaforge/internal/model"
)

// DepositRepository manages deposit_requests persistence.
type DepositRepository struct {
	db DBTX
}

func NewDepositRepository(db DBTX) *DepositRepository {
	return &DepositRepository{db: db}
}

func (r *DepositRepository) Create(ctx context.Context, req *model.DepositRequest) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO deposit_requests (id, user_id, amount, amount_usd, tx_hash, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		req.ID, req.UserID, req.Amount, req.AmountUSD, req.TxHash, req.Status)
	return err
}

func (r *DepositRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.DepositRequest, error) {
	var d model.DepositRequest
	err := r.db.QueryRow(ctx,
		`SELECT dr.id, dr.user_id, dr.amount, dr.amount_usd, dr.tx_hash, dr.status,
		        dr.reviewer_id, dr.review_note, dr.reviewed_at, dr.wallet_tx_id,
		        dr.created_at, dr.updated_at, u.email as user_email
		 FROM deposit_requests dr
		 LEFT JOIN users u ON u.id = dr.user_id
		 WHERE dr.id = $1`, id).Scan(
		&d.ID, &d.UserID, &d.Amount, &d.AmountUSD, &d.TxHash, &d.Status,
		&d.ReviewerID, &d.ReviewNote, &d.ReviewedAt, &d.WalletTxID,
		&d.CreatedAt, &d.UpdatedAt, &d.UserEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDepositNotFound
	}
	return &d, err
}

func (r *DepositRepository) ListByUser(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]model.DepositRequest, int64, error) {
	var total int64
	err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM deposit_requests WHERE user_id = $1`, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}
	offset := (page - 1) * pageSize
	rows, err := r.db.Query(ctx,
		`SELECT dr.id, dr.user_id, dr.amount, dr.amount_usd, dr.tx_hash, dr.status,
		        dr.reviewer_id, dr.review_note, dr.reviewed_at, dr.wallet_tx_id,
		        dr.created_at, dr.updated_at, NULL as user_email
		 FROM deposit_requests dr
		 WHERE dr.user_id = $1
		 ORDER BY dr.created_at DESC
		 LIMIT $2 OFFSET $3`, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.DepositRequest
	for rows.Next() {
		var d model.DepositRequest
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.Amount, &d.AmountUSD, &d.TxHash, &d.Status,
			&d.ReviewerID, &d.ReviewNote, &d.ReviewedAt, &d.WalletTxID,
			&d.CreatedAt, &d.UpdatedAt, &d.UserEmail); err != nil {
			return nil, 0, err
		}
		out = append(out, d)
	}
	return out, total, rows.Err()
}

func (r *DepositRepository) ListAll(ctx context.Context, page, pageSize int, status string) ([]model.DepositRequest, int64, error) {
	var total int64
	if status != "" {
		err := r.db.QueryRow(ctx,
			`SELECT count(*) FROM deposit_requests WHERE status = $1`, status).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
	} else {
		err := r.db.QueryRow(ctx, `SELECT count(*) FROM deposit_requests`).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
	}
	if total == 0 {
		return nil, 0, nil
	}
	offset := (page - 1) * pageSize
	var rows pgx.Rows
	var err error
	if status != "" {
		rows, err = r.db.Query(ctx,
			`SELECT dr.id, dr.user_id, dr.amount, dr.amount_usd, dr.tx_hash, dr.status,
			        dr.reviewer_id, dr.review_note, dr.reviewed_at, dr.wallet_tx_id,
			        dr.created_at, dr.updated_at, u.email as user_email
			 FROM deposit_requests dr
			 LEFT JOIN users u ON u.id = dr.user_id
			 WHERE dr.status = $1
			 ORDER BY dr.created_at DESC
			 LIMIT $2 OFFSET $3`, status, pageSize, offset)
	} else {
		rows, err = r.db.Query(ctx,
			`SELECT dr.id, dr.user_id, dr.amount, dr.amount_usd, dr.tx_hash, dr.status,
			        dr.reviewer_id, dr.review_note, dr.reviewed_at, dr.wallet_tx_id,
			        dr.created_at, dr.updated_at, u.email as user_email
			 FROM deposit_requests dr
			 LEFT JOIN users u ON u.id = dr.user_id
			 ORDER BY dr.created_at DESC
			 LIMIT $1 OFFSET $2`, pageSize, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.DepositRequest
	for rows.Next() {
		var d model.DepositRequest
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.Amount, &d.AmountUSD, &d.TxHash, &d.Status,
			&d.ReviewerID, &d.ReviewNote, &d.ReviewedAt, &d.WalletTxID,
			&d.CreatedAt, &d.UpdatedAt, &d.UserEmail); err != nil {
			return nil, 0, err
		}
		out = append(out, d)
	}
	return out, total, rows.Err()
}

// UpdateStatusTx updates a deposit request's status within a transaction.
// Returns ErrDepositAlreadyProcessed if the deposit is not in PENDING status.
func (r *DepositRepository) UpdateStatusTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string, reviewerID uuid.UUID, reviewNote string, walletTxID *uuid.UUID) error {
	tag, err := tx.Exec(ctx,
		`UPDATE deposit_requests
		 SET status = $2, reviewer_id = $3, review_note = $4, wallet_tx_id = $5,
		     reviewed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $1 AND status = 'PENDING'`,
		id, status, reviewerID, reviewNote, walletTxID)
	if err != nil {
		return fmt.Errorf("update deposit status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDepositAlreadyProcessed
	}
	return nil
}

var (
	ErrDepositNotFound          = errors.New("deposit request not found")
	ErrDepositAlreadyProcessed  = errors.New("deposit request already processed")
)
