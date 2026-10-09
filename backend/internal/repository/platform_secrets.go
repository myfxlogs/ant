package repository

// platform_secrets.go —— ENV-TO-PG-1：D 档秘密件密文 KV 仓储（ADR-0031）。
// 本层只存取密文（BYTEA），不触加解密——调用方持 secrets.Client 负责封装/解封。
// 明文永不落库、永不进日志/错误串；错误只带 key 名。

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// InsertPlatformSecretIfAbsent seed-once 写入（ON CONFLICT DO NOTHING，仅缺键种入）。
func (r *AdminRepository) InsertPlatformSecretIfAbsent(ctx context.Context, key string, valueEnc []byte) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO platform_secrets (key, value_enc) VALUES ($1, $2) ON CONFLICT (key) DO NOTHING`,
		key, valueEnc)
	if err != nil {
		return fmt.Errorf("seed platform secret %s: %w", key, err)
	}
	return nil
}

// LoadPlatformSecrets 全表 key→密文。解密由调用方做（持 KEK 的 secrets.Client）。
func (r *AdminRepository) LoadPlatformSecrets(ctx context.Context) (map[string][]byte, error) {
	rows, err := r.db.Query(ctx, `SELECT key, value_enc FROM platform_secrets`)
	if err != nil {
		return nil, fmt.Errorf("load platform secrets: %w", err)
	}
	defer rows.Close()
	out := make(map[string][]byte)
	for rows.Next() {
		var k string
		var enc []byte
		if err := rows.Scan(&k, &enc); err != nil {
			return nil, fmt.Errorf("scan platform secret: %w", err)
		}
		out[k] = enc
	}
	return out, rows.Err()
}

// PlatformSecretExists 单键存在判定（seed 幂等性测试/运维核查用）。
func (r *AdminRepository) PlatformSecretExists(ctx context.Context, key string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM platform_secrets WHERE key = $1)`, key).Scan(&exists)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return exists, nil
}
