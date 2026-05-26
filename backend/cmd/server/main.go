package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"


	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/connect/admin"
	"anttrader/internal/connect/ai"
	mktplace "anttrader/internal/connect/marketplace"
	"anttrader/internal/connect/strategy"
	"anttrader/internal/connect/system"
	"anttrader/internal/connect/user"
	"anttrader/internal/interceptor"
	"anttrader/internal/marketplace"
	"anttrader/internal/mdgateway"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/mthub"
	"anttrader/internal/pkg/secretbox"
	"anttrader/internal/repository"
	"anttrader/internal/risksvc"
	"anttrader/internal/secrets"
	"anttrader/internal/server"
	"anttrader/internal/service"
	systemai "anttrader/internal/service/systemai"
	antredis "anttrader/internal/storage/redis"
	"anttrader/internal/config"

	connectrpc "connectrpc.com/connect"
)

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
		log.Warn("ANT_MASTER_KEY not set — account passwords will NOT be decrypted; gateways will fail to connect")
	}

	// Services
	platformSvc := service.NewPlatformService(pool)
	jwtSecret := cfg.JWTSecret
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	authInterceptor := interceptor.NewAuthInterceptor(jwtSecret, nil)

	hub := mthub.NewHub()
	eventBroker := mthub.NewOrderEventBroker()
	accountBroker := mthub.NewAccountProfitBroker()
	snapshotBroker := mthub.NewPositionSnapshotBroker()
	idemGuard := mthub.NewIdempotencyGuard(rdb.Client())
	reconcileGate := mthub.NewReconcileGate()
	js, err := nc.JetStream()
	if err != nil {
		log.Fatal("nats jetstream failed", zap.Error(err))
	}
	eventStore := mthub.NewTradeEventStore(js)
	mthubSvc := mthub.NewMtHubService(hub, eventBroker, accountBroker, snapshotBroker, idemGuard, reconcileGate, eventStore)
	// --- mdgateway pipeline (M10 runner) ---
	spillDir := cfg.SpillDir
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
			Secrets:  secClient,
			Hub:      hub,
			OnAccountProfit: func(accountID, userID string, p *mdtick.ProfitUpdate) {
				// Write latest balance/equity to PG.
				writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if _, err := pool.Exec(writeCtx,
					`UPDATE mt_accounts SET balance=$1, equity=$2, credit=$3, margin=$4, free_margin=$5, margin_level=$6, account_status='connected', updated_at=CURRENT_TIMESTAMP WHERE id=$7::uuid`,
					p.Balance, p.Equity, p.Credit, p.Margin, p.FreeMargin, p.MarginLevel, accountID); err != nil {
					log.Warn("OnAccountProfit: pg update failed", zap.String("account", accountID), zap.Error(err))
				}
				// Publish to mthub for real-time SSE streaming.
				mthubSvc.PublishAccountProfit(&mthub.AccountProfitEvent{
					AccountID: accountID, UserID: userID, Platform: p.Platform,
					Balance: p.Balance, Credit: p.Credit, Equity: p.Equity,
					Margin: p.Margin, FreeMargin: p.FreeMargin, MarginLevel: p.MarginLevel,
					Profit: p.Profit, ProfitPercent: p.ProfitPercent,
					Status: "connected", Timestamp: time.Now(),
					Positions: func() []mthub.AccountProfitPosition {
						out := make([]mthub.AccountProfitPosition, 0, len(p.Positions))
						for _, pos := range p.Positions {
							out = append(out, mthub.AccountProfitPosition{
								Ticket: pos.Ticket, Symbol: pos.Symbol,
								Profit: pos.Profit, Volume: pos.Volume,
								CurrentPrice: pos.CurrentPrice,
							})
						}
						return out
					}(),
				})
			},
			OnOrderUpdate: func(accountID, userID string, o *mdtick.OrderUpdate) {
				// Update PG with latest account metrics.
				writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if _, err := pool.Exec(writeCtx,
					"UPDATE mt_accounts SET balance=$1, equity=$2, credit=$3, margin=$4, free_margin=$5, margin_level=$6, account_status='connected', updated_at=CURRENT_TIMESTAMP WHERE id=$7::uuid",
					o.Balance, o.Equity, o.Credit, o.Margin, o.FreeMargin, o.MarginLevel, accountID); err != nil {
					log.Warn("OnOrderUpdate: pg update failed", zap.String("account", accountID), zap.Error(err))
				}

				// Publish profit update to AccountProfitBroker for SSE streaming.
				accountBroker.Publish(&mthub.AccountProfitEvent{
					AccountID: accountID, UserID: userID, Platform: o.Platform,
					Balance: o.Balance, Credit: o.Credit, Equity: o.Equity,
					Margin: o.Margin, FreeMargin: o.FreeMargin, MarginLevel: o.MarginLevel,
					Profit: o.Profit, Status: "connected", Timestamp: time.Now(),
					Positions: func() []mthub.AccountProfitPosition {
						out := make([]mthub.AccountProfitPosition, 0, len(o.Positions))
						for _, pos := range o.Positions {
							out = append(out, mthub.AccountProfitPosition{
								Ticket: pos.Ticket, Symbol: pos.Symbol,
								Profit: pos.Profit, Volume: pos.Volume,
								CurrentPrice: pos.CurrentPrice,
							})
						}
						return out
					}(),
				})

				// Publish FULL position snapshot — ticket, symbol, lots, prices,
				// profit all in one event (matching proto OpenedOrders model).
				snapshot := &mthub.PositionSnapshot{
					AccountID:   accountID,
					UserID:      userID,
					Platform:    o.Platform,
					Balance:     o.Balance,
					Credit:      o.Credit,
					Equity:      o.Equity,
					Margin:      o.Margin,
					FreeMargin:  o.FreeMargin,
					MarginLevel: o.MarginLevel,
					Profit:      o.Profit,
					Positions:   make([]mthub.PositionSnapshotItem, 0, len(o.Positions)),
				}
				for _, pos := range o.Positions {
					snapshot.Positions = append(snapshot.Positions, mthub.PositionSnapshotItem{
						Ticket:       pos.Ticket,
						Symbol:       pos.Symbol,
						Type:         pos.Type,
						Volume:       pos.Volume,
						OpenPrice:    pos.OpenPrice,
						CurrentPrice: pos.CurrentPrice,
						StopLoss:     pos.StopLoss,
						TakeProfit:   pos.TakeProfit,
						Profit:       pos.Profit,
						Swap:         pos.Swap,
						Commission:   pos.Commission,
						Comment:      pos.Comment,
						OpenTime:     pos.OpenTime,
					})
				}
				snapshotBroker.Publish(snapshot)
			},
		}); err != nil {
			log.Error("mdgateway pipeline exited with error", zap.Error(err))
		}
	}()

	mux := http.NewServeMux()

	// ConnectRPC handlers
	// Repositories for handler→service→repository layering (P1-2).
	userRepo := repository.NewUserRepository(pool)
	convRepo := repository.NewAIConversationRepository(pool)
	jobRepo := repository.NewJobRepository(pool)
	schedHealthRepo := repository.NewScheduleHealthRepository(pool)
	marketDataRepo := repository.NewMarketDataRepository(ch)

	authServer := user.NewAuthServer(userRepo, jwtSecret, log)
	mux.Handle(antv1c.NewAuthServiceHandler(authServer, connectrpc.WithInterceptors(authInterceptor)))

	reconLoop := mthub.NewReconciliationLoop(hub, pool, rdb.Client(), log, reconcileGate)

	mthubServer := system.NewMtHubServer(mthubSvc, platformSvc, log)
	mux.Handle(antv1c.NewMtHubServiceHandler(mthubServer, connectrpc.WithInterceptors(authInterceptor)))

	accountServer := user.NewAccountServer(platformSvc, log)
	mux.Handle(antv1c.NewAccountServiceHandler(accountServer, connectrpc.WithInterceptors(authInterceptor)))

	mktServer := mktplace.NewMarketServer(platformSvc, marketDataRepo, nc, log)
	mux.Handle(antv1c.NewMarketServiceHandler(mktServer, connectrpc.WithInterceptors(authInterceptor)))

	mktplaceSvc := marketplace.New(pool)
	mktplaceHandler := mktplace.NewMarketplaceServer(mktplaceSvc, log)
	mux.Handle(antv1c.NewMarketplaceServiceHandler(mktplaceHandler, connectrpc.WithInterceptors(authInterceptor)))

	logRepo := repository.NewLogRepository(pool)
	logSvc := service.NewLogService(logRepo)
	strategyExperimentRepo := repository.NewStrategyExperimentRepository(pool)
	strategyAssetRepo := repository.NewStrategyAssetRepository(pool)

	aiRepo := repository.NewSystemAIConfigRepository(pool)
	var aiBox *secretbox.Box
	if mk := cfg.AntMasterKey; mk != "" {
		aiBox = secretbox.New([]byte(mk))
	}
	aiSvc := systemai.NewService(aiRepo, aiBox)
	aiServer := ai.NewAIServer(aiSvc, convRepo, log)
	mux.Handle(antv1c.NewAIServiceHandler(aiServer, connectrpc.WithInterceptors(authInterceptor)))

	streamServer := system.NewStreamServer(mthubSvc, platformSvc, log)
	mux.Handle(antv1c.NewStreamServiceHandler(streamServer, connectrpc.WithInterceptors(authInterceptor)))

	strategySvc := service.NewStrategySvc(pool)
	strategyServer := strategy.NewStrategyServer(strategySvc, log)
	mux.Handle(antv1c.NewStrategyServiceHandler(strategyServer, connectrpc.WithInterceptors(authInterceptor)))

	// Mock/stub handlers — return mock data for services not yet connected to real backends.
	// Real: SystemAI, AIPrimary, Job, ScheduleHealth
	// Mock: PythonStrategy, CodeAssist, BacktestTrades, EconomicData
	// Stub: DebateV2, DebateV2Stream
	pythonStrategyServer := strategy.NewPythonStrategyServer(log)
	mux.Handle(antv1c.NewPythonStrategyServiceHandler(pythonStrategyServer, connectrpc.WithInterceptors(authInterceptor)))
	codeAssistServer := ai.NewCodeAssistServer(log)
	mux.Handle(antv1c.NewCodeAssistServiceHandler(codeAssistServer, connectrpc.WithInterceptors(authInterceptor)))
	systemAIServer := ai.NewSystemAIServer(aiSvc, log)
	mux.Handle(antv1c.NewSystemAIServiceHandler(systemAIServer, connectrpc.WithInterceptors(authInterceptor)))
	aiPrimaryServer := ai.NewAIPrimaryServer(aiSvc, log)
	mux.Handle(antv1c.NewAIPrimaryServiceHandler(aiPrimaryServer, connectrpc.WithInterceptors(authInterceptor)))
	backtestTradesServer := strategy.NewBacktestTradesServer(log)
	mux.Handle(antv1c.NewBacktestTradesServiceHandler(backtestTradesServer, connectrpc.WithInterceptors(authInterceptor)))
	economicDataServer := system.NewEconomicDataServer(log)
	mux.Handle(antv1c.NewEconomicDataServiceHandler(economicDataServer, connectrpc.WithInterceptors(authInterceptor)))
	jobServer := system.NewJobServer(jobRepo, log)
	mux.Handle(antv1c.NewJobServiceHandler(jobServer, connectrpc.WithInterceptors(authInterceptor)))
	logServiceServer := system.NewLogServiceServer(logSvc, log)
	mux.Handle(antv1c.NewLogServiceHandler(logServiceServer, connectrpc.WithInterceptors(authInterceptor)))
	adminRepo := repository.NewAdminRepository(pool)
	adminTradingServer := admin.NewAdminTradingServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminTradingServiceHandler(adminTradingServer, connectrpc.WithInterceptors(authInterceptor)))
	adminConfigServer := admin.NewAdminConfigServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminConfigServiceHandler(adminConfigServer, connectrpc.WithInterceptors(authInterceptor)))
	adminLogServer := admin.NewAdminLogServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminLogServiceHandler(adminLogServer, connectrpc.WithInterceptors(authInterceptor)))
	adminAccountServer := admin.NewAdminAccountServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminAccountServiceHandler(adminAccountServer, connectrpc.WithInterceptors(authInterceptor)))
	adminUserServer := admin.NewAdminUserServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminUserServiceHandler(adminUserServer, connectrpc.WithInterceptors(authInterceptor)))
	adminSystemServer := admin.NewAdminSystemServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminSystemServiceHandler(adminSystemServer, connectrpc.WithInterceptors(authInterceptor)))

	// --- M11-16 Jurisdictional Gate ---
	jurisStore := risksvc.NewPgJurisdictionStore(pool)
	geoipResolver := risksvc.NewMaxMindGeoIPResolver(cfg.GeoIPDBPath)
	jurisGate := &risksvc.JurisdictionGate{
		Store:               jurisStore,
		GeoIP:               geoipResolver,
		RequireKYC:           cfg.RequireKYC,
		RequireDisclaimer:    cfg.RequireDisclaimer,
		RequireQuestionnaire: cfg.RequireQuestionnaire,
	}
	_ = jurisGate // wired into SignalPipeline at runtime via risksvc.KycJurisdictionRule

	adminJurisdictionServer := admin.NewAdminJurisdictionServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminJurisdictionServiceHandler(adminJurisdictionServer, connectrpc.WithInterceptors(authInterceptor)))

	analyticsRepo := repository.NewAnalyticsRepository(pool)
	analyticsServer := system.NewAnalyticsServer(analyticsRepo, log)
	mux.Handle(antv1c.NewAnalyticsServiceHandler(analyticsServer, connectrpc.WithInterceptors(authInterceptor)))

	marketRegimeRepo := repository.NewMarketRegimeRepository(pool)
	marketRegimeServer := mktplace.NewMarketRegimeServer(marketRegimeRepo, log)
	mux.Handle(antv1c.NewMarketRegimeServiceHandler(marketRegimeServer, connectrpc.WithInterceptors(authInterceptor)))

	strategyExperimentServer := strategy.NewStrategyExperimentServer(strategyExperimentRepo, log)
	mux.Handle(antv1c.NewStrategyExperimentServiceHandler(strategyExperimentServer, connectrpc.WithInterceptors(authInterceptor)))
	strategyAssetServer := strategy.NewStrategyAssetServer(strategyAssetRepo, log)
	mux.Handle(antv1c.NewStrategyAssetServiceHandler(strategyAssetServer, connectrpc.WithInterceptors(authInterceptor)))
	scheduleHealthServer := system.NewScheduleHealthServer(schedHealthRepo, log)
	mux.Handle(antv1c.NewScheduleHealthServiceHandler(scheduleHealthServer, connectrpc.WithInterceptors(authInterceptor)))
	indicatorCatalogServer := mktplace.NewIndicatorCatalogServer(log)
	mux.Handle(antv1c.NewIndicatorCatalogServiceHandler(indicatorCatalogServer, connectrpc.WithInterceptors(authInterceptor)))

	// Catch-all: returns CodeUnimplemented for unknown routes (frontend silently swallows).
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":"unimplemented"}`))
	})

	// Prometheus /metrics endpoint (M10 ADR-0010 §2.4).
	mux.Handle("/metrics", mdgateway.MetricsHandler())

	// Health check (includes CH + NATS + Redis)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if err := pool.Ping(context.Background()); err != nil {
			w.WriteHeader(503)
			w.Write([]byte("pg unreachable"))
			return
		}
		if err := ch.Ping(context.Background()); err != nil {
			w.WriteHeader(503)
			w.Write([]byte("ch unreachable"))
			return
		}
		if !nc.IsConnected() {
			w.WriteHeader(503)
			w.Write([]byte("nats disconnected"))
			return
		}
		if err := rdb.Ping(context.Background()); err != nil {
			w.WriteHeader(503)
			w.Write([]byte("redis unreachable"))
			return
		}
		w.Write([]byte("ant ok"))
	})

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
