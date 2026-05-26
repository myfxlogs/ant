package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"anttrader/internal/mdgateway"
	"anttrader/internal/secrets"
)

func main() {
	log, _ := zap.NewDevelopment()
	defer log.Sync()

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		getenv("DB_USER", "ant"),
		getenv("DB_PASSWORD", "ant"),
		getenv("DB_HOST", "localhost"),
		getenv("DB_PORT", "5432"),
		getenv("DB_NAME", "ant"),
		getenv("DB_SSLMODE", "disable"),
	)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal("pg connect failed", zap.Error(err))
	}
	defer pool.Close()
	log.Info("PG connected")

	ch, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s",
			getenv("CH_HOST", "localhost"),
			getenv("CH_PORT", "9000"),
		)},
		Auth: clickhouse.Auth{
			Database: getenv("CH_DATABASE", "ant"),
			Username: getenv("CH_USER", "default"),
			Password: getenv("CH_PASSWORD", ""),
		},
	})
	if err != nil {
		log.Fatal("clickhouse connect failed", zap.Error(err))
	}
	defer ch.Close()
	if err := ch.Ping(context.Background()); err != nil {
		log.Fatal("clickhouse ping failed", zap.Error(err))
	}
	log.Info("CH connected")

	nc, err := nats.Connect(getenv("NATS_URL", nats.DefaultURL))
	if err != nil {
		log.Fatal("nats connect failed", zap.Error(err))
	}
	defer nc.Close()
	log.Info("NATS connected")

	var secClient secrets.Client
	if mk := os.Getenv("ANT_MASTER_KEY"); mk != "" {
		secClient, err = secrets.New(mk, 1)
		if err != nil {
			log.Fatal("secrets init failed", zap.Error(err))
		}
		log.Info("secrets client initialized")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Info("starting mdgateway pipeline...")
	go func() {
		if err := mdgateway.Run(ctx, mdgateway.RunnerDeps{
			Log:      log,
			PG:       pool,
			CH:       ch,
			NATSConn: nc,
			SpillDir: "/tmp/ant-spill",
			Secrets:  secClient,
		}); err != nil {
			log.Error("pipeline error", zap.Error(err))
		}
	}()

	// Run for 2 minutes then exit.
	log.Info("pipeline running, will exit in 120s...")
	timer := time.NewTimer(120 * time.Second)
	select {
	case <-ctx.Done():
		log.Info("signal received, shutting down")
	case <-timer.C:
		log.Info("timer expired, shutting down")
	}
	cancel()
	time.Sleep(500 * time.Millisecond)
	log.Info("pipeline test complete")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
