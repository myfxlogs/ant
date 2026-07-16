package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

type ShareToken struct {
	ID            uuid.UUID `db:"id"`
	UserID        uuid.UUID `db:"user_id"`
	AccountID     string    `db:"account_id"`
	Token         string    `db:"token"`
	Description   string    `db:"description"`
	ShowPositions bool      `db:"show_positions"`
	ExpiresAt     time.Time `db:"expires_at"`
	ViewCount     int       `db:"view_count"`
	MaxViews      *int      `db:"max_views"`
	CreatedAt     time.Time `db:"created_at"`
}

type ShareRepository struct {
	db *pgxpool.Pool
}

func NewShareRepository(db *pgxpool.Pool) *ShareRepository {
	return &ShareRepository{db: db}
}

func (r *ShareRepository) Create(ctx context.Context, t *ShareToken) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO share_tokens (user_id, account_id, token, description, show_positions, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		t.UserID, t.AccountID, t.Token, t.Description, t.ShowPositions, t.ExpiresAt)
	return err
}

func (r *ShareRepository) GetByToken(ctx context.Context, token string) (*ShareToken, error) {
	var t ShareToken
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, account_id, token, description, show_positions, expires_at, view_count, max_views, created_at
		 FROM share_tokens WHERE token=$1`, token).Scan(
		&t.ID, &t.UserID, &t.AccountID, &t.Token, &t.Description, &t.ShowPositions,
		&t.ExpiresAt, &t.ViewCount, &t.MaxViews, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *ShareRepository) ListAll(ctx context.Context, limit, offset int) ([]*ShareToken, int, error) {
	var total int
	r.db.QueryRow(ctx, `SELECT count(*) FROM share_tokens`).Scan(&total)
	if limit <= 0 { limit = 20 }
	if offset < 0 { offset = 0 }
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, account_id, token, description, show_positions, expires_at, view_count, max_views, created_at
		 FROM share_tokens ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*ShareToken
	for rows.Next() {
		var t ShareToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.AccountID, &t.Token, &t.Description, &t.ShowPositions,
			&t.ExpiresAt, &t.ViewCount, &t.MaxViews, &t.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, &t)
	}
	return out, total, rows.Err()
}

func (r *ShareRepository) UpdateShowPositions(ctx context.Context, userID uuid.UUID, token string, show bool) error {
	_, err := r.db.Exec(ctx, `UPDATE share_tokens SET show_positions=$1 WHERE user_id=$2 AND token=$3`, show, userID, token)
	return err
}

func (r *ShareRepository) GetPositionsSnapshot(ctx context.Context, token string) ([]*antv1.SharedPosition, error) {
	var data []byte
	var updatedAt time.Time
	err := r.db.QueryRow(ctx,
		`SELECT positions_snapshot, COALESCE(positions_snapshot_at, '1970-01-01') FROM share_tokens WHERE token=$1`, token,
	).Scan(&data, &updatedAt)
	if err != nil || data == nil {
		return nil, err
	}
	if time.Since(updatedAt) > 60*time.Second {
		return nil, nil
	}
	var snap antv1.SharedPositionList
	if err := proto.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	return snap.Positions, nil
}

func (r *ShareRepository) SetPositionsSnapshot(ctx context.Context, token string, positions []*antv1.SharedPosition) error {
	data, err := proto.Marshal(&antv1.SharedPositionList{Positions: positions})
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`UPDATE share_tokens SET positions_snapshot=$1, positions_snapshot_at=NOW() WHERE token=$2`, data, token)
	return err
}

func (r *ShareRepository) DeleteByUser(ctx context.Context, userID uuid.UUID, token string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM share_tokens WHERE user_id=$1 AND token=$2`, userID, token)
	return err
}

func (r *ShareRepository) IncrementView(ctx context.Context, token string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE share_tokens SET view_count = view_count + 1 WHERE token=$1`, token)
	return err
}

func (r *ShareRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*ShareToken, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, account_id, token, description, show_positions, expires_at, view_count, max_views, created_at
		 FROM share_tokens WHERE user_id=$1 ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ShareToken
	for rows.Next() {
		var t ShareToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.AccountID, &t.Token, &t.Description, &t.ShowPositions,
			&t.ExpiresAt, &t.ViewCount, &t.MaxViews, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}
