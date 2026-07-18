package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"alphaforge/internal/model"
)

func (r *AdminRepository) GetConfig(ctx context.Context, key string) (*model.SystemConfig, error) {
	query := `SELECT key, value, description, enabled, admin_visible, value_type, created_at, updated_at FROM system_config WHERE key = $1`
	config := &model.SystemConfig{}
	err := r.db.QueryRow(ctx, query, key).Scan(
		&config.Key, &config.Value, &config.Description, &config.Enabled, &config.AdminVisible, &config.ValueType, &config.CreatedAt, &config.UpdatedAt,
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
	query := `SELECT key, value, description, enabled, admin_visible, value_type, created_at, updated_at FROM system_config WHERE admin_visible = TRUE ORDER BY key`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*model.SystemConfig
	for rows.Next() {
		var c model.SystemConfig
		err := rows.Scan(&c.Key, &c.Value, &c.Description, &c.Enabled, &c.AdminVisible, &c.ValueType, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, err
		}
		configs = append(configs, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return configs, nil
}

func (r *AdminRepository) SetConfig(ctx context.Context, key, value, description string) error {
	query := `
		INSERT INTO system_config (key, value, description, enabled, admin_visible, value_type, updated_at)
		VALUES ($1, $2, $3, TRUE, TRUE, 'text', CURRENT_TIMESTAMP)
		ON CONFLICT (key) DO UPDATE SET value = $2, description = $3, updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.Exec(ctx, query, key, value, description)
	if err != nil {
		return fmt.Errorf("set config: %w", err)
	}
	return nil
}

func (r *AdminRepository) SetConfigEnabled(ctx context.Context, key string, enabled bool) error {
	query := `UPDATE system_config SET enabled = $2, updated_at = CURRENT_TIMESTAMP WHERE key = $1`
	result, err := r.db.Exec(ctx, query, key, enabled)
	if err != nil {
		return fmt.Errorf("set config enabled: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrConfigNotFound
	}
	return nil
}

func (r *AdminRepository) ConfigKeyExists(ctx context.Context, key string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM system_config WHERE key = $1 AND admin_visible = TRUE)`,
		key,
	).Scan(&exists)
	return exists, err
}

// SetConfigValue updates the value of an existing config key.
func (r *AdminRepository) SetConfigValue(ctx context.Context, key, value string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE system_config SET value = $2, updated_at = NOW() WHERE key = $1`, key, value)
	if err != nil {
		return fmt.Errorf("set config value: %w", err)
	}
	return nil
}

// GetHotWalletKey returns the encrypted private key for the active hot wallet
// from the wallet_secrets table. The caller is responsible for decrypting it.
func (r *AdminRepository) GetHotWalletKey(ctx context.Context) ([]byte, error) {
	var encryptedData []byte
	err := r.db.QueryRow(ctx,
		`SELECT encrypted_data FROM wallet_secrets WHERE purpose = 'hot-wallet' AND is_active = true LIMIT 1`,
	).Scan(&encryptedData)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("admin repo: no active hot wallet key found in wallet_secrets")
		}
		return nil, fmt.Errorf("admin repo: get hot wallet key: %w", err)
	}
	return encryptedData, nil
}
