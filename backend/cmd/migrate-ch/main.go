// Command migrate-ch applies ClickHouse schema migrations for ant v2.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"go.uber.org/zap"

	"anttrader/internal/config"
	"anttrader/internal/mdgateway/chmigrate"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	cfg := config.Load()

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", cfg.CHHost, cfg.CHPort)},
		Auth: clickhouse.Auth{
			Database: cfg.CHDatabase,
			Username: cfg.CHUser,
			Password: cfg.CHPassword,
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
