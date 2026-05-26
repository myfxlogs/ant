// Command etl-mt-accounts encrypts plaintext password and mt_token
// columns from mt_accounts into password_encrypted and mtapi_token_encrypted
// using the secrets vault. Must run after migration 098 and before
// M7.1 mdgateway reads the v2 view.
package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"anttrader/internal/config"
	"anttrader/internal/secrets"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	cfg := config.Load()
	if cfg.AntMasterKey == "" {
		log.Fatal("ANT_MASTER_KEY not set")
	}

	vault, err := secrets.New(cfg.AntMasterKey, 1)
	if err != nil {
		log.Fatal("vault init failed", zap.Error(err))
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal("pg connect failed", zap.Error(err))
	}
	defer pool.Close()

	ctx := context.Background()

	rows, err := pool.Query(ctx,
		"SELECT id, password, mt_token FROM mt_accounts WHERE (password IS NOT NULL AND password != '' AND password_encrypted IS NULL) OR (mt_token IS NOT NULL AND mt_token != '' AND mtapi_token_encrypted IS NULL)")
	if err != nil {
		log.Fatal("query failed", zap.Error(err))
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var id, password, mtToken string
		if err := rows.Scan(&id, &password, &mtToken); err != nil {
			log.Error("scan failed", zap.Error(err))
			continue
		}

		if password != "" {
			enc, err := vault.Encrypt(ctx, secrets.PurposeMTPassword, []byte(password))
			if err != nil {
				log.Error("encrypt password failed", zap.String("id", id), zap.Error(err))
				continue
			}
			if _, err := pool.Exec(ctx,
				"UPDATE mt_accounts SET password_encrypted = $1 WHERE id = $2", enc, id); err != nil {
				log.Error("update password_encrypted failed", zap.String("id", id), zap.Error(err))
				continue
			}
		}

		if mtToken != "" {
			enc, err := vault.Encrypt(ctx, secrets.PurposeMTAPIToken, []byte(mtToken))
			if err != nil {
				log.Error("encrypt token failed", zap.String("id", id), zap.Error(err))
				continue
			}
			if _, err := pool.Exec(ctx,
				"UPDATE mt_accounts SET mtapi_token_encrypted = $1 WHERE id = $2", enc, id); err != nil {
				log.Error("update mtapi_token_encrypted failed", zap.String("id", id), zap.Error(err))
				continue
			}
		}
		count++
	}
	log.Info("etl-mt-accounts: complete", zap.Int("rows", count))
}
