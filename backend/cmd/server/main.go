package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/connect"
	"anttrader/internal/mdgateway"
	"anttrader/internal/marketplace"
	"anttrader/internal/mthub"
	"anttrader/internal/server"
	"anttrader/internal/service"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	// Connect to PostgreSQL
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		env("DB_USER", "ant"),
		env("DB_PASSWORD", "ant"),
		env("DB_HOST", "postgres"),
		env("DB_PORT", "5432"),
		env("DB_NAME", "ant"),
		env("DB_SSLMODE", "disable"),
	)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal("pg connect failed", zap.Error(err))
	}
	defer pool.Close()

	// Connect to ClickHouse
	chHost := env("CH_HOST", "clickhouse")
	chPort := env("CH_PORT", "9000")
	chUser := env("CH_USER", "default")
	chPass := env("CH_PASSWORD", "")
	chDB := env("CH_DATABASE", "ant")
	ch, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", chHost, chPort)},
		Auth: clickhouse.Auth{
			Database: chDB,
			Username: chUser,
			Password: chPass,
		},
	})
	if err != nil {
		log.Fatal("clickhouse connect failed", zap.Error(err))
	}
	defer ch.Close()
	if err := ch.Ping(context.Background()); err != nil {
		log.Fatal("clickhouse ping failed", zap.Error(err))
	}

	// Connect to NATS
	natsURL := env("NATS_URL", nats.DefaultURL)
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatal("nats connect failed", zap.Error(err))
	}
	defer nc.Close()

	// --- mdgateway pipeline (M10 runner) ---
	spillDir := env("SPILL_DIR", "/var/lib/ant/spill")
	pipelineCtx, pipelineCancel := context.WithCancel(context.Background())
	defer pipelineCancel()

	go func() {
		log.Info("mdgateway pipeline starting", zap.String("spill_dir", spillDir))
		if err := mdgateway.Run(pipelineCtx, mdgateway.RunnerDeps{
			Log:      log,
			PG:       pool,
			CH:       ch,
			NATSConn: nc,
			SpillDir: spillDir,
		}); err != nil {
			log.Error("mdgateway pipeline exited with error", zap.Error(err))
		}
	}()

	// Services
	platformSvc := service.NewPlatformService(pool)
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" { jwtSecret = "ant-dev-secret-change-in-production" }

	hub := mthub.NewHub()
	eventBroker := mthub.NewOrderEventBroker()
	mthubSvc := mthub.NewMtHubService(hub, eventBroker)

	mux := http.NewServeMux()

	// ConnectRPC handlers
	authServer := connect.NewAuthServer(pool, jwtSecret)
	mux.Handle(antv1c.NewAuthServiceHandler(authServer))

	mthubServer := connect.NewMtHubServer(mthubSvc)
	mux.Handle(antv1c.NewMtHubServiceHandler(mthubServer))

	accountServer := connect.NewAccountServer(platformSvc)
	mux.Handle(antv1c.NewAccountServiceHandler(accountServer))

	mktServer := connect.NewMarketServer(platformSvc)
	mux.Handle(antv1c.NewMarketServiceHandler(mktServer))

	mktplaceSvc := marketplace.New(pool)
	mktplaceHandler := connect.NewMarketplaceServer(mktplaceSvc)
	mux.Handle(antv1c.NewMarketplaceServiceHandler(mktplaceHandler))

	// Health check (includes CH + NATS)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if err := pool.Ping(context.Background()); err != nil {
			w.WriteHeader(503); w.Write([]byte("pg unreachable")); return
		}
		if err := ch.Ping(context.Background()); err != nil {
			w.WriteHeader(503); w.Write([]byte("ch unreachable")); return
		}
		if !nc.IsConnected() {
			w.WriteHeader(503); w.Write([]byte("nats disconnected")); return
		}
		w.Write([]byte("ant ok"))
	})

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	port := env("PORT", "8080")
	log.Info("ant v2 starting", zap.String("port", port), zap.String("ch", chHost), zap.String("nats", natsURL))

	go func() {
		<-ctx.Done()
		log.Info("shutting down")
		stop()
		pipelineCancel()
		time.Sleep(100 * time.Millisecond)
		nc.Close()
		ch.Close()
		pool.Close()
	}()

	if err := server.Run(ctx, mux, port, log); err != nil {
		log.Fatal("server failed", zap.Error(err))
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" { return v }
	return fallback
}
