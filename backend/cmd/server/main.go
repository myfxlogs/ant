package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/connect"
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

	// Marketplace handler
	// Account
	accountServer := connect.NewAccountServer(platformSvc)
	mux.Handle(antv1c.NewAccountServiceHandler(accountServer))

	mktServer := connect.NewMarketServer(platformSvc)
	mux.Handle(antv1c.NewMarketServiceHandler(mktServer))

	// Marketplace
	mktplaceSvc := marketplace.New(pool)
	mktplaceHandler := connect.NewMarketplaceServer(mktplaceSvc)
	mux.Handle(antv1c.NewMarketplaceServiceHandler(mktplaceHandler))

	// Health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if err := pool.Ping(context.Background()); err != nil {
			w.WriteHeader(503)
			w.Write([]byte("pg unreachable"))
			return
		}
		w.Write([]byte("ant ok"))
	})

	port := env("PORT", "8080")
	log.Info("ant v2 starting", zap.String("port", port))
	if err := server.Run(context.Background(), mux, port, log); err != nil {
		log.Fatal("server failed", zap.Error(err))
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" { return v }
	return fallback
}
