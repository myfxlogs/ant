// Command migrate-ch applies ClickHouse schema migrations for ant v2.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"go.uber.org/zap"

	"anttrader/internal/mdgateway/chmigrate"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	host := env("CH_HOST", "127.0.0.1")
	port := env("CH_PORT", "9000")
	user := env("CH_USER", "default")
	password := env("CH_PASSWORD", "clickhouse")
	database := env("CH_DATABASE", "ant")

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", host, port)},
		Auth: clickhouse.Auth{
			Database: database,
			Username: user,
			Password: password,
		},
		DialTimeout: 10 * time.Second,
	})
	if err != nil {
		log.Fatal("connect failed", zap.Error(err))
	}
	defer conn.Close()

	if err := chmigrate.Run(context.Background(), conn, log); err != nil {
		log.Fatal("migration failed", zap.Error(err))
	}
	log.Info("migrate-ch: complete")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
