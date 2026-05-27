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
	"anttrader/internal/controlplane"
	"anttrader/internal/costsvc"
	"anttrader/internal/interceptor"
	"anttrader/internal/marketplace"
	"anttrader/internal/mdgateway"
	"anttrader/internal/mdgateway/adapter/brokersearch"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/mthub"
	"anttrader/internal/pkg/secretbox"
	"anttrader/internal/repository"
	"anttrader/internal/risksvc"
	"anttrader/internal/secrets"
	"anttrader/internal/server"
	"anttrader/internal/service"
	systemai "anttrader/internal/service/systemai"
	"anttrader/internal/strategysvc"
	"anttrader/internal/usermgr"
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
	adminInterceptor := interceptor.NewAdminInterceptor(platformSvc, log)
	rateLimitInterceptor := interceptor.NewRateLimitInterceptor(cfg.RateLimitLoginPerMinute, cfg.RateLimitEnabled)

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
	mux.Handle(antv1c.NewAuthServiceHandler(authServer, connectrpc.WithInterceptors(rateLimitInterceptor, authInterceptor)))

	reconLoop := mthub.NewReconciliationLoop(hub, pool, rdb.Client(), log, reconcileGate)

	mthubServer := system.NewMtHubServer(mthubSvc, platformSvc, log)
	mux.Handle(antv1c.NewMtHubServiceHandler(mthubServer, connectrpc.WithInterceptors(authInterceptor)))

	accountServer := user.NewAccountServer(platformSvc, log)
	// S2.2: wire mtapi broker searcher for real broker discovery.
	accountServer.SetSearcher(brokersearch.New("", ""))
	// S2.4: wire NATS account event publisher for Connect/Disconnect/Reconnect.
	accountEventPub := mdgateway.NewAccountEventPublisher(js, log)
	accountServer.SetPublisher(accountEventPub)
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

	// Debate V2 service (multi-expert AI strategy generation).
	debateV2Svc := service.NewDebateV2Service(pool, jobRepo)
	debateV2Server := ai.NewDebateV2Server(debateV2Svc, log)
	mux.Handle(antv1c.NewDebateV2ServiceHandler(debateV2Server, connectrpc.WithInterceptors(authInterceptor)))

	gateProgressServer := ai.NewGateProgressServer(log)

	streamServer := system.NewStreamServer(mthubSvc, platformSvc, log)
	mux.Handle(antv1c.NewStreamServiceHandler(streamServer, connectrpc.WithInterceptors(authInterceptor)))

	strategySvc := service.NewStrategySvc(pool)
	strategyServer := strategy.NewStrategyServer(strategySvc, log)
	mux.Handle(antv1c.NewStrategyServiceHandler(strategyServer, connectrpc.WithInterceptors(authInterceptor)))

	// Mock/stub handlers — return mock data for services not yet connected to real backends.
	// Real: SystemAI, AIPrimary, Job, ScheduleHealth, DebateV2
	// Mock: PythonStrategy, CodeAssist, BacktestTrades, EconomicData
	pythonStrategyServer := strategy.NewPythonStrategyServer(log)
	if cfg.StrategyServiceURL != "" {
		pythonClient := strategysvc.NewPythonClient(cfg.StrategyServiceURL)
		pythonStrategyServer.SetClient(pythonClient)
		strategyServer.SetClient(pythonClient) // S2.5: real RunBacktest
		log.Info("Python strategy client configured", zap.String("url", cfg.StrategyServiceURL))
	}
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
	passwordResetRepo := repository.NewPasswordResetRepo(pool)
	adminTradingServer := admin.NewAdminTradingServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminTradingServiceHandler(adminTradingServer, connectrpc.WithInterceptors(authInterceptor, adminInterceptor)))
	adminConfigServer := admin.NewAdminConfigServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminConfigServiceHandler(adminConfigServer, connectrpc.WithInterceptors(authInterceptor, adminInterceptor)))
	adminLogServer := admin.NewAdminLogServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminLogServiceHandler(adminLogServer, connectrpc.WithInterceptors(authInterceptor, adminInterceptor)))
	adminAccountServer := admin.NewAdminAccountServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminAccountServiceHandler(adminAccountServer, connectrpc.WithInterceptors(authInterceptor, adminInterceptor)))
	adminUserServer := admin.NewAdminUserServer(adminRepo, passwordResetRepo, log)
	mux.Handle(antv1c.NewAdminUserServiceHandler(adminUserServer, connectrpc.WithInterceptors(authInterceptor, adminInterceptor)))
	adminSystemServer := admin.NewAdminSystemServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminSystemServiceHandler(adminSystemServer, connectrpc.WithInterceptors(authInterceptor, adminInterceptor)))

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
	// S1.1: Wire SignalPipeline for pre-trade risk checks (capability → hardlimit → platform → engine → sizer).
		capStore := risksvc.NewCapabilityStore()
		// NOTE: LoadFromPG skipped — user_risk_profiles schema doesn't yet match
		// CapabilityStore column expectations (093 migration vs capability.go struct).
		// Default Tier0 (view-only) applies until schema alignment is complete.
		hardLimit := risksvc.NewHardLimitEvaluator(&risksvc.KycJurisdictionRule{Gate: jurisGate})
		platformAgg := risksvc.NewPlatformAggregator()
		platformLimits := risksvc.DefaultPlatformLimits()
		riskEngine := risksvc.NewEngine()
		sizer := &risksvc.VolTargetSizer{RiskBudgetPct: 0.01}
		allocator := &risksvc.ProRataAllocator{}

		pipeline := risksvc.NewSignalPipeline(risksvc.PipelineConfig{
			CapStore:  capStore,
			HardLimit: hardLimit,
			Platform:  platformAgg,
			Limits:    platformLimits,
			Engine:    riskEngine,
			Sizer:     sizer,
			Allocator: allocator,
		})
		mthubSvc.SetRiskPipeline(pipeline)

		// Account state provider queries PG for balance/equity/margin per account.
		mthubSvc.SetAccountStateProvider(func(ctx context.Context, accountID string) (*mthub.AccountState, error) {
			var state mthub.AccountState
			var positions int64
			err := pool.QueryRow(ctx,
				`SELECT balance, equity, free_margin, COALESCE(margin, 0)::float8,
				        COALESCE((SELECT count(*) FROM positions WHERE mt_account_id = $1), 0)::int
				 FROM mt_accounts WHERE id = $1::uuid`,
				accountID,
			).Scan(&state.Balance, &state.Equity, &state.FreeMargin, &state.Margin, &positions)
			if err != nil {
				return nil, err
			}
			state.Positions = int(positions)
			return &state, nil
		})

		// S1.3: Wire per-user rate limiter (10 orders/sec/user, 100 signals/sec/user).
		limiter := usermgr.NewUserLimiter(usermgr.DefaultConfig())
		mthubSvc.SetUserLimiter(limiter)

		// S1.3: Wire pre-trade cost estimator (EURUSD default model).
		eurtusdModel := &costsvc.CostModel{
			Symbol:           "EURUSD",
			SpreadPips:       1.0,
			PipSize:          0.0001,
			PipValue:         10.0,
			CommissionPerLot: 7.0,
		}
		estimator := &costsvc.StaticEstimator{Model: eurtusdModel}
		mthubSvc.SetCostEstimator(estimator)

		// S1.2: OMS state writer for order lifecycle tracking.
		omsWriter := mthub.NewOmsWriter(pool, eventStore)
		mthubSvc.SetOmsWriter(omsWriter)

	adminJurisdictionServer := admin.NewAdminJurisdictionServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminJurisdictionServiceHandler(adminJurisdictionServer, connectrpc.WithInterceptors(authInterceptor, adminInterceptor)))

	// --- SRE control plane ---
	sreKillSwitch := controlplane.NewKillSwitch()
	sreBreakers := controlplane.NewBreakerRegistry(controlplane.DefaultBreakerConfig())
	sreCanary := controlplane.NewCanaryManager()
	sreHandler := admin.NewSREHandler(sreKillSwitch, sreBreakers, sreCanary, platformSvc, log)

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

	// Catch-all: return 404 for unknown routes.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"code":"not_found"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ant-v2"}`))
	})

	// Auth cookie endpoints — refresh token via httpOnly cookie.
	mux.HandleFunc("/api/auth/refresh", authServer.HandleTokenRefresh)
	mux.HandleFunc("/api/auth/logout", authServer.HandleLogout)

	// SSE endpoints for Debate V2 async jobs.
	mux.HandleFunc("/sse/debate-v2/advance-jobs/", func(w http.ResponseWriter, r *http.Request) {
		debateV2Server.HandleDebateV2AdvanceJobSSE(w, r, authInterceptor)
	})
	mux.HandleFunc("/sse/debate-v2/chat-jobs/", func(w http.ResponseWriter, r *http.Request) {
		debateV2Server.HandleDebateV2ChatJobSSE(w, r, authInterceptor)
	})

	// SSE endpoint for AI gate pipeline progress.
	mux.HandleFunc("/sse/ai/gate-progress", func(w http.ResponseWriter, r *http.Request) {
		gateProgressServer.HandleGateProgressSSE(w, r, authInterceptor)
	})

	// SRE control plane HTTP endpoints.
	mux.HandleFunc("/api/admin/sre/killswitch/status", func(w http.ResponseWriter, r *http.Request) {
		sreHandler.HandleKillSwitchStatus(w, r, authInterceptor)
	})
	mux.HandleFunc("/api/admin/sre/killswitch/engage", func(w http.ResponseWriter, r *http.Request) {
		sreHandler.HandleKillSwitchEngage(w, r, authInterceptor)
	})
	mux.HandleFunc("/api/admin/sre/killswitch/disengage", func(w http.ResponseWriter, r *http.Request) {
		sreHandler.HandleKillSwitchDisengage(w, r, authInterceptor)
	})
	mux.HandleFunc("/api/admin/sre/breakers", func(w http.ResponseWriter, r *http.Request) {
		sreHandler.HandleBreakersList(w, r, authInterceptor)
	})
	mux.HandleFunc("/api/admin/sre/breakers/reset", func(w http.ResponseWriter, r *http.Request) {
		sreHandler.HandleBreakerReset(w, r, authInterceptor)
	})
	mux.HandleFunc("/api/admin/sre/canary", func(w http.ResponseWriter, r *http.Request) {
		sreHandler.HandleCanaryList(w, r, authInterceptor)
	})
	mux.HandleFunc("/api/admin/sre/canary/set", func(w http.ResponseWriter, r *http.Request) {
		sreHandler.HandleCanarySet(w, r, authInterceptor)
	})
	mux.HandleFunc("/api/admin/sre/canary/delete", func(w http.ResponseWriter, r *http.Request) {
		sreHandler.HandleCanaryDelete(w, r, authInterceptor)
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
