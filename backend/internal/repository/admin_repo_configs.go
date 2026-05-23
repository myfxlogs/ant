package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"anttrader/internal/model"
)

func (r *AdminRepository) GetConfig(ctx context.Context, key string) (*model.SystemConfig, error) {
	query := `SELECT key, value, description, enabled, created_at, updated_at FROM system_config WHERE key = $1`
	config := &model.SystemConfig{}
	err := r.db.QueryRow(ctx, query, key).Scan(
		&config.Key, &config.Value, &config.Description, &config.Enabled, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrConfigNotFound
		}
		return nil, err
	}
	return config, nil
}

func (r *AdminRepository) ListConfigs(ctx context.Context) ([]*model.SystemConfig, error) {
	query := `SELECT key, value, description, enabled, created_at, updated_at FROM system_config ORDER BY key`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*model.SystemConfig
	for rows.Next() {
		var c model.SystemConfig
		err := rows.Scan(&c.Key, &c.Value, &c.Description, &c.Enabled, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		configs = append(configs, &c)
	}
	return configs, nil
}

func (r *AdminRepository) SetConfig(ctx context.Context, key, value, description string) error {
	query := `
		INSERT INTO system_config (key, value, description, enabled, updated_at)
		VALUES ($1, $2, $3, TRUE, CURRENT_TIMESTAMP)
		ON CONFLICT (key) DO UPDATE SET value = $2, description = $3, updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.Exec(ctx, query, key, value, description)
	return err
}

func (r *AdminRepository) SetConfigEnabled(ctx context.Context, key string, enabled bool) error {
	query := `UPDATE system_config SET enabled = $2, updated_at = CURRENT_TIMESTAMP WHERE key = $1`
	result, err := r.db.Exec(ctx, query, key, enabled)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrConfigNotFound
	}
	return nil
}
