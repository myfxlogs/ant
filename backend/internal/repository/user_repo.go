package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"anttrader/internal/model"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (email, password_hash, nickname, avatar, role, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		user.Email,
		user.PasswordHash,
		user.Nickname,
		user.Avatar,
		user.Role,
		user.Status,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user := &model.User{}
	query := `
		SELECT id, email, password_hash, nickname, avatar, role, status, 
			   last_login_at, created_at, updated_at
		FROM users WHERE id = $1
	`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Nickname,
		&user.Avatar,
		&user.Role,
		&user.Status,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	query := `
		SELECT id, email, password_hash, nickname, avatar, role, status, 
			   last_login_at, created_at, updated_at
		FROM users WHERE email = $1
	`
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Nickname,
		&user.Avatar,
		&user.Role,
		&user.Status,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users 
		SET nickname = $2, avatar = $3, status = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, query,
		user.ID,
		user.Nickname,
		user.Avatar,
		user.Status,
	).Scan(&user.UpdatedAt)
}

func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET last_login_at = $2 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, time.Now())
	return err
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, passwordHash)
	return err
}

// GetAIPrimary 读取用户在 /ai/settings 选定的「默认主模型」。
// 自 061 迁移引入；空字符串表示未设置（调用方应回落到 pickEnabledSystemAIRow）。
func (r *UserRepository) GetAIPrimary(ctx context.Context, id uuid.UUID) (providerID, model string, err error) {
	const q = `SELECT ai_primary_provider_id, ai_primary_model FROM users WHERE id = $1`
	err = r.db.QueryRow(ctx, q, id).Scan(&providerID, &model)
	return
}

// SetAIPrimary 写入用户的「默认主模型」选择。空字符串 = 清除。
func (r *UserRepository) SetAIPrimary(ctx context.Context, id uuid.UUID, providerID, model string) error {
	const q = `UPDATE users SET ai_primary_provider_id = $2, ai_primary_model = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, providerID, model)
	return err
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	err := r.db.QueryRow(ctx, query, email).Scan(&exists)
	return exists, err
}
