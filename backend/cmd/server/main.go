package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"anttrader/internal/config"
	"anttrader/internal/interceptor"
	"anttrader/internal/mdgateway/chmigrate"
	"anttrader/internal/mthub"
	"anttrader/internal/notifier"
	"anttrader/internal/repository"
	"anttrader/internal/risksvc"
	"anttrader/internal/secrets"
	"anttrader/internal/server"
	"anttrader/internal/service"
	antredis "anttrader/internal/storage/redis"
)

func splitAndTrim(s, sep string) []string {
	var out []string
	for _, part := range strings.Split(s, sep) {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatal("invalid config", zap.Error(err))
	}

	// Connect to PostgreSQL
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBSSLMode,
	)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal("pg connect failed", zap.Error(err))
	}
	defer pool.Close()

	// Connect to ClickHouse
	chHost := cfg.CHHost
	chPort := cfg.CHPort
	chUser := cfg.CHUser
	chPass := cfg.CHPassword
	chDB := cfg.CHDatabase
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
	if err := chmigrate.Run(context.Background(), ch, log); err != nil {
		log.Fatal("chmigrate failed", zap.Error(err))
	}

	// Connect to NATS
	natsURL := cfg.NATSURL
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatal("nats connect failed", zap.Error(err))
	}
	defer nc.Close()

	// Connect to Redis
	redisCfg := antredis.Config{
		Host:         cfg.RedisHost,
		Port:         6379,
		Password:     cfg.RedisPassword,
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 3,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
	}
	if p := cfg.RedisPort; p != "" {
		fmt.Sscanf(p, "%d", &redisCfg.Port)
	}
	rdb, err := antredis.Connect(context.Background(), redisCfg)
	if err != nil {
		log.Fatal("redis connect failed", zap.Error(err))
	}
	defer rdb.Close()

	// --- Secrets client (decrypts account passwords and mtapi tokens) ---
	var secClient secrets.Client
	if mk := cfg.AntMasterKey; mk != "" {
		var err error
		secClient, err = secrets.New(mk, 1)
		if err != nil {
			log.Fatal("secrets: cannot create client from ANT_MASTER_KEY", zap.Error(err))
		}
		log.Info("secrets: client initialized")
	} else {
		log.Fatal("ANT_MASTER_KEY is required — generate one with: go run cmd/ant-vault/main.go")
	}

	// Services
	accountSvc := service.NewAccountService(pool)
	accountSvc.SetLogger(log)
	platformSvc := service.NewPlatformService(pool, accountSvc)
	platformSvc.SetLogger(log)
	jwtSecret := cfg.JWTSecret
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	authInterceptor := interceptor.NewAuthInterceptor(jwtSecret, nil)
	adminInterceptor := interceptor.NewAdminInterceptor(platformSvc, log)
	rateLimitInterceptor := interceptor.NewRateLimitInterceptor(cfg.RateLimitLoginPerMinute, cfg.RateLimitEnabled)

	hub := mthub.NewHub()
	eventBroker := mthub.NewOrderEventBroker()
	accountBroker := mthub.NewAccountProfitBroker()
	snapshotBroker := mthub.NewPositionSnapshotBroker()
	idemGuard := mthub.NewIdempotencyGuard(rdb.Client())
	reconcileGate := mthub.NewReconcileGate()
	var reconLoop *mthub.ReconciliationLoop // H17: declared early so OnBrokerInfo callback can trigger reconciliation
	js, err := nc.JetStream()
	if err != nil {
		log.Fatal("nats jetstream failed", zap.Error(err))
	}
	eventStore := mthub.NewTradeEventStore(js)
	mthubSvc := mthub.NewMtHubService(hub, eventBroker, accountBroker, snapshotBroker, idemGuard, reconcileGate, eventStore)
	// --- Analytics cache ---
	analyticsCache := service.NewAnalyticsCache(rdb.Client(), log)

	// --- mdgateway pipeline (M10 runner) ---
	tradeRecordRepo := repository.NewTradeRecordRepository(pool)
	accountSyncSvc := service.NewAccountSyncService(tradeRecordRepo, mthubSvc, analyticsCache, log)

	spillDir := cfg.SpillDir
	pipelineCtx, pipelineCancel := context.WithCancel(context.Background())
	defer pipelineCancel()

	var emailNotifier *notifier.EmailNotifier             // set after creation; referenced by OnAccountProfit closure
	var platformAgg *risksvc.PlatformAggregator           // set after creation; referenced by OnOrderUpdate closure

	go startMdGatewayPipeline(pipelineCtx, log, pool, ch, nc, spillDir, secClient, hub, accountSvc, mthubSvc, accountSyncSvc, tradeRecordRepo, snapshotBroker, accountBroker, eventStore, &emailNotifier, &platformAgg, &reconLoop)

	mux := http.NewServeMux()
	reconLoop, emailNotifier, platformAgg = registerHandlers(mux, log, pool, ch, nc, rdb, cfg, jwtSecret, accountSvc, platformSvc, authInterceptor, adminInterceptor, rateLimitInterceptor, mthubSvc, hub, tradeRecordRepo, js, eventStore, reconcileGate, analyticsCache)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start reconciliation loop (cancelled on shutdown)
	go reconLoop.Start(ctx)

	port := cfg.Port
	log.Info("ant v2 starting", zap.String("port", port), zap.String("ch", chHost), zap.String("nats", natsURL))

	go func() {
		<-ctx.Done()
		log.Info("shutting down")
		pipelineCancel()
	}()

	if err := server.Run(ctx, mux, port, log); err != nil {
		log.Fatal("server failed", zap.Error(err))
	}


}

