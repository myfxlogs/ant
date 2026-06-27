package main

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	internalai "anttrader/internal/ai"
	"anttrader/internal/analysis"
	"anttrader/internal/config"
	"anttrader/internal/connect/admin"
	"anttrader/internal/connect/ai"
	algo "anttrader/internal/connect/algo"
	assetanalysis "anttrader/internal/connect/asset_analysis"
	"anttrader/internal/connect/gateway"
	"anttrader/internal/connect/autotrading"
	mktplace "anttrader/internal/connect/marketplace"
	"anttrader/internal/connect/notification"
	"anttrader/internal/connect/strategy"
	"anttrader/internal/usermgr"
	"anttrader/internal/connect/system"
	"anttrader/internal/connect/user"
	paperhdr "anttrader/internal/connect/paper"
	papereng "anttrader/internal/paper"
	"anttrader/internal/interceptor"
	"anttrader/internal/marketplace"
	"anttrader/internal/mdgateway"
	"anttrader/internal/mdgateway/adapter"
	"anttrader/internal/mdgateway/adapter/brokersearch"
	"anttrader/internal/mthub"
	notifpubsub "anttrader/internal/notification"
	"anttrader/internal/notifier"
	"anttrader/internal/pglisten"
	"anttrader/internal/pkg/secretbox"
	"anttrader/internal/repository"
	"anttrader/internal/risk"
	"anttrader/internal/risksvc"
	"anttrader/internal/service"
	systemai "anttrader/internal/service/systemai"
	antredis "anttrader/internal/storage/redis"

	connectrpc "connectrpc.com/connect"
)

func registerHandlers(
	ctx context.Context,
	mux *http.ServeMux,
	log *zap.Logger,
	pool *pgxpool.Pool,
	store repository.MarketDataStore,
	nc *nats.Conn,
	rdb *antredis.Client,
	cfg *config.Config,
	jwtSecret string,
	accountSvc *service.AccountService,
	platformSvc *service.PlatformService,
	authInterceptor *interceptor.AuthInterceptor,
	adminInterceptor *interceptor.AdminInterceptor,
	rateLimitInterceptor *interceptor.RateLimitInterceptor,
	otelInterceptor connectrpc.Interceptor,
	mthubSvc *mthub.MtHubService,
	hub *mthub.Hub,
	tradeRecordRepo *repository.TradeRecordRepository,
	js nats.JetStreamContext,
	eventStore *mthub.TradeEventStore,
	reconcileGate *mthub.ReconcileGate,
	analyticsCache *service.AnalyticsCache,
	brokerReg *adapter.BrokerRegistry,
) (*mthub.ReconciliationLoop, *notifier.EmailNotifier, *risksvc.PlatformAggregator, *notifpubsub.Sender, *strategy.ScheduleEngine, func()) {

	// ConnectRPC handlers
	// Repositories for handler→service→repository layering (P1-2).
	userRepo := repository.NewUserRepository(pool)
	convRepo := repository.NewAIConversationRepository(pool)
	session := internalai.NewConversationSession(convRepo)
	templatesRepo := repository.NewAIStrategyTemplatesRepository(pool)
	jobRepo := repository.NewJobRepository(pool)
	schedHealthRepo := repository.NewScheduleHealthRepository(pool)
	marketDataRepo := store
	walletRepo := repository.NewWalletRepository(pool)
	walletSvc := service.NewWalletService(walletRepo, pool, log)
	accountNumberSvc := service.NewAccountNumberService(pool)
	registrationSvc := service.NewRegistrationService(userRepo, accountNumberSvc, walletSvc, log)

	authServer := user.NewAuthServer(userRepo, jwtSecret, log)
	authServer.SetInsecureCookies(true) // no TLS in Docker deployment
	authServer.WithRegistration(registrationSvc)
	mux.Handle(antv1c.NewAuthServiceHandler(authServer, connectrpc.WithInterceptors(otelInterceptor,rateLimitInterceptor, authInterceptor)))

	walletServer := user.NewWalletServer(walletSvc, platformSvc, log)
	mux.Handle(antv1c.NewWalletServiceHandler(walletServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	reconLoop := mthub.NewReconciliationLoop(hub, pool, rdb.Client(), log, reconcileGate)

	mthubServer := system.NewMtHubServer(mthubSvc, platformSvc, marketDataRepo, tradeRecordRepo, log)
	mux.Handle(antv1c.NewMtHubServiceHandler(mthubServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	searcher := brokersearch.New("", "")
	accountEventPub := mdgateway.NewAccountEventPublisher(js, log)
	mtTester := user.NewMTConnectionTester(cfg.MtapiToken, log)
	accountServer := user.NewAccountServer(accountSvc, searcher, accountEventPub, mtTester, log).
		WithSessionWaiter(hub)
	mux.Handle(antv1c.NewAccountServiceHandler(accountServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	mktServer := mktplace.NewMarketServer(platformSvc, marketDataRepo, nc, log)
	mux.Handle(antv1c.NewMarketServiceHandler(mktServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	mktplaceSvc := marketplace.New(pool, log)
	mktplaceHandler := mktplace.NewMarketplaceServer(mktplaceSvc, platformSvc, log)
	mux.Handle(antv1c.NewMarketplaceServiceHandler(mktplaceHandler, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	mktplaceSvc.StartRenewalLoop(ctx, log) // daily subscription renewal (background goroutine)

	// M12-A2: Execution Algo handler (TWAP/VWAP/POV/Shortfall).
	// brokerReg is created in main.go before the pipeline starts; gateways register
	// via adapter.RegisterDefaults inside the mdgateway runner after connection.
	algoServer := algo.NewExecutionAlgoServer(brokerReg, log)
	mux.Handle(antv1c.NewExecutionAlgoServiceHandler(algoServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	logRepo := repository.NewLogRepository(pool)
	logSvc := service.NewLogService(logRepo)
	strategyExperimentRepo := repository.NewStrategyExperimentRepository(pool)
	strategyAssetRepo := repository.NewStrategyAssetRepository(pool)
	backtestRunRepo := repository.NewBacktestRunRepository(pool)

	aiRepo := repository.NewSystemAIConfigRepository(pool)
	var aiBox *secretbox.Box
	if mk := cfg.AntMasterKey; mk != "" {
		aiBox = secretbox.New([]byte(mk))
	}
	aiSvc := systemai.NewService(aiRepo, aiBox)
	aiSvc.SetUserRepo(userRepo)
	aiSvc.SetCircuitBreakerDB(&pgxCB{p: pool}) // persistent, shared across instances
	agentDefRepo := repository.NewAIAgentDefinitionRepository(pool)
	aiServer := ai.NewAIServer(aiSvc, convRepo, session, log)
	aiServer.SetAgentDefRepo(agentDefRepo)
	mux.Handle(antv1c.NewAIServiceHandler(aiServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	// Agent definition CRUD (no proto RPC yet — raw HTTP).
	mux.Handle(antv1c.NewAgentDefinitionServiceHandler(aiServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	// P3: AI Asset Analysis — MTF outlook, S/R levels, volatility, AI recommendation.
	assetAnalyzer := analysis.NewAnalyzer(marketDataRepo, log)
	assetAnalysisServer := assetanalysis.NewAssetAnalysisServer(assetAnalyzer, aiSvc, platformSvc, log)
	mux.Handle(antv1c.NewAssetAnalysisServiceHandler(assetAnalysisServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	// Share performance: generate expiring public links for trading results.
	shareRepo := repository.NewShareRepository(pool)
	
	analyticsRepo := repository.NewAnalyticsRepository(pool)
	shareServer := user.NewShareServer(shareRepo, tradeRecordRepo, analyticsRepo, userRepo, mthubSvc, pool, jwtSecret, log)
	mux.Handle(antv1c.NewShareServiceHandler(shareServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor)))
		mux.HandleFunc("/api/shares/create", shareServer.HandleCreateShareTokenREST)
		mux.HandleFunc("/api/shares/update", shareServer.HandleUpdateShareToken)
		mux.HandleFunc("/api/share/performance", shareServer.HandleGetSharedPerformanceJSON)
		mux.HandleFunc("/api/shares/delete", shareServer.HandleDeleteShareToken)
		mux.HandleFunc("/api/shares/list", shareServer.HandleListShareTokens)
		mux.HandleFunc("/api/admin/shares/list", func(w http.ResponseWriter, r *http.Request) {
			uid, err := authInterceptor.UserIDFromHTTP(r)
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			ok, _ := platformSvc.IsAdmin(r.Context(), uid)
			if !ok {
				http.Error(w, `{"error":"admin required"}`, http.StatusForbidden)
				return
			}
			shareServer.HandleListAllShareTokens(w, r)
		})

	// AI Gateway: platform-operated AI model relay with token billing.
	gatewayProviderRepo := repository.NewSystemAIProviderRepository(pool)
	gatewayModelRepo := repository.NewAIModelRepository(pool)
	gatewayTokenUsageRepo := repository.NewAITokenUsageRepository(pool)
	gatewayServer := gateway.NewAIGatewayServer(gatewayProviderRepo, gatewayModelRepo, gatewayTokenUsageRepo, walletSvc, aiBox, log)
	mux.Handle(antv1c.NewAIGatewayServiceHandler(gatewayServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	// Wire AI Gateway fallback: if user has no own API key, use system providers.
	aiSvc.SetGatewayProviderRepo(gatewayProviderRepo)

	// Wire wallet pre-check: block AI calls when balance is insufficient.
	aiMinBalance := 1.0
	if v := os.Getenv("AI_MIN_BALANCE"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed >= 0 {
			aiMinBalance = parsed
		}
	}
	aiSvc.SetWalletChecker(func(ctx context.Context, userID uuid.UUID) error {
		w, err := walletSvc.GetOrCreateWallet(ctx, userID)
		if err != nil {
			return nil // don't block on wallet lookup errors
		}
		bal, _ := strconv.ParseFloat(w.Balance, 64)
		if bal < aiMinBalance {
			return systemai.ErrInsufficientBalance
		}
		return nil
	})

	// Wire post-call billing: after a successful AI call, deduct cost before returning the result.
	// If deduction fails (insufficient balance), the error propagates to the user and they see a
	// friendly "insufficient balance" message. No free AI.
	aiSvc.SetPostCallBiller(func(ctx context.Context, userID uuid.UUID, providerID, modelName string, inputTokens, outputTokens int) error {
		cost := "0"
		if m, err := gatewayModelRepo.GetByProviderAndModel(ctx, providerID, modelName); err == nil && m != nil {
			ip, _ := strconv.ParseFloat(m.PricePer1MInput, 64)
			op, _ := strconv.ParseFloat(m.PricePer1MOutput, 64)
			inCost := float64(inputTokens) / 1_000_000.0 * ip
			outCost := float64(outputTokens) / 1_000_000.0 * op
			cost = strconv.FormatFloat(inCost+outCost, 'f', 8, 64)
		}
		if err := gatewayServer.RecordTokenUsage(ctx, userID, "system", providerID, modelName, "chat", inputTokens, outputTokens, cost); err != nil {
			// Map wallet insufficient balance to the AI-level sentinel so handlers
			// can detect it with errors.Is(err, systemai.ErrInsufficientBalance).
			if strings.Contains(err.Error(), "insufficient balance") {
				return systemai.ErrInsufficientBalance
			}
			return err
		}
		return nil
	})

	// Wire token billing: all ChatCompletion calls automatically record usage through this hook.
	aiSvc.SetTokenRecorder(func(ctx context.Context, r systemai.TokenRecord) {
		// Compute cost from model pricing (per-1M-token rates).
		cost := "0"
		if m, err := gatewayModelRepo.GetByProviderAndModel(ctx, r.ProviderID, r.Model); err == nil && m != nil {
			ip, _ := strconv.ParseFloat(m.PricePer1MInput, 64)
			op, _ := strconv.ParseFloat(m.PricePer1MOutput, 64)
			inCost := float64(r.InputTokens) / 1_000_000.0 * ip
			outCost := float64(r.OutputTokens) / 1_000_000.0 * op
			cost = strconv.FormatFloat(inCost+outCost, 'f', 8, 64)
		}
		// Usage tracking only (billing is handled by postCallBiller).
		_ = gatewayServer.RecordTokenUsage(ctx, r.UserID, "user", r.ProviderID, r.Model, r.Feature, r.InputTokens, r.OutputTokens, cost)
	})

	streamServer := system.NewStreamServer(mthubSvc, platformSvc, log)
	streamServer.SetMarketDataRepo(marketDataRepo)
	mux.Handle(antv1c.NewStreamServiceHandler(streamServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	strategySvc := service.NewStrategySvc(pool)
	strategyServer := strategy.NewStrategyServer(strategySvc, log)
	strategyServer.SetCodeAccessChecker(mktplaceSvc) // marketplace code protection
	pgListen := pglisten.New(pool, log)
	strategyServer.SetPgListen(pgListen)
	mktplaceHandler.SetPgListen(pgListen) // marketplace SSE streaming
	mux.Handle(antv1c.NewStrategyServiceHandler(strategyServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	// Paper trading + notification deps created early — both needed by strategy execution config.
	paperRepo := repository.NewPaperRepo(pool)
	paperEngine := papereng.New(paperRepo, mthubSvc, log)
	notifSub := notifpubsub.NewSubscriber()
	notifRepo := repository.NewNotificationRepository(pool)
	notifSender := notifpubsub.NewSender(notifRepo, notifSub, log)

	jurisGate, capStore, platformAgg := initRiskPipeline(pool, log, mthubSvc, hub, eventStore, cfg)
	strategyExecServer := configureStrategyExecution(backtestRunRepo, marketDataRepo, mthubSvc, hub,
		paperEngine, notifSender, aiSvc, pgListen, jurisGate, capStore, cfg, log)
	mux.Handle(antv1c.NewStrategyRuntimeServiceHandler(strategyExecServer,
		connectrpc.WithInterceptors(otelInterceptor, authInterceptor)))

	paperHandler := paperhdr.NewHandler(paperRepo, paperEngine, strategyExecServer, log,
		func(ctx context.Context, userID string) string {
			var mt4ID string
			err := pool.QueryRow(ctx,
				`SELECT id::text FROM mt_accounts
				 WHERE user_id = $1::uuid AND is_disabled = false
				 ORDER BY created_at LIMIT 1`,
				userID).Scan(&mt4ID)
			if err != nil {
				return ""
			}
			return mt4ID
		})
	mux.Handle(antv1c.NewPaperTradingServiceHandler(paperHandler,
		connectrpc.WithInterceptors(otelInterceptor, authInterceptor)))
	codeAssistServer := ai.NewCodeAssistServer(aiSvc, session, log)
	// CodeAssist uses LLM-only code analysis and generation.
	mux.Handle(antv1c.NewCodeAssistServiceHandler(codeAssistServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	systemAIServer := ai.NewSystemAIServer(aiSvc, log)
	mux.Handle(antv1c.NewSystemAIServiceHandler(systemAIServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	aiPrimaryServer := ai.NewAIPrimaryServer(aiSvc, log)
	mux.Handle(antv1c.NewAIPrimaryServiceHandler(aiPrimaryServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	backtestTradesServer := strategy.NewBacktestTradesServer(backtestRunRepo, log)
	mux.Handle(antv1c.NewBacktestTradesServiceHandler(backtestTradesServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	gateEvalServer := ai.NewGateEvalServer(backtestRunRepo, log)
	mux.Handle(antv1c.NewGateServiceHandler(gateEvalServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	strategyGenServer := ai.NewStrategyGenServer(aiSvc, templatesRepo, convRepo, backtestRunRepo, log)
	mux.Handle(antv1c.NewStrategyGenerationServiceHandler(strategyGenServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	// Claude Code style: separated plan → execute pipeline
	strategyPlanServer := ai.NewStrategyPlanServer(aiSvc, templatesRepo, backtestRunRepo, convRepo, marketDataRepo, log)
	strategyPlanServer.SetPoolAdapter(
		func(ctx context.Context, sql string, args ...any) error { _, e := pool.Exec(ctx, sql, args...); return e },
		func(ctx context.Context, sql string, args ...any) (string, error) { var v string; row := pool.QueryRow(ctx, sql, args...); err := row.Scan(&v); return v, err },
	)
	mux.Handle(antv1c.NewStrategyPlanServiceHandler(strategyPlanServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor)))

	economicDataServer := system.NewEconomicDataServer(log)
	mux.Handle(antv1c.NewEconomicDataServiceHandler(economicDataServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	jobServer := system.NewJobServer(jobRepo, log)
	jobServer.SetPgListen(pgListen)
	mux.Handle(antv1c.NewJobServiceHandler(jobServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	logServiceServer := system.NewLogServiceServer(logSvc, log)
	mux.Handle(antv1c.NewLogServiceHandler(logServiceServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	notifServer := notification.NewNotificationServer(notifRepo, notifSub, log)
	mux.Handle(antv1c.NewNotificationServiceHandler(notifServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	gateEvalServer.SetNotificationSender(notifSender)

		gate := risk.NewDefaultGate()
		gate.SetKillSwitch(func() bool { return cfg.RiskGateKillSwitch })
		gate.SetAutotradeEnabled(func(uid string) bool { return cfg.RiskGateAutotradeEnabled })

	adminRepo := repository.NewAdminRepository(pool)
	passwordResetRepo := repository.NewPasswordResetRepo(pool)
	adminTradingServer := admin.NewAdminTradingServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminTradingServiceHandler(adminTradingServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
	adminConfigServer := admin.NewAdminConfigServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminConfigServiceHandler(adminConfigServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
	adminLogServer := admin.NewAdminLogServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminLogServiceHandler(adminLogServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
	adminAccountServer := admin.NewAdminAccountServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminAccountServiceHandler(adminAccountServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
	deletionSvc := service.NewUserDeletionService(adminRepo, log)
		adminUserServer := admin.NewAdminUserServer(adminRepo, passwordResetRepo, walletSvc, accountNumberSvc, deletionSvc, log)
	mux.Handle(antv1c.NewAdminUserServiceHandler(adminUserServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
	adminSystemServer := admin.NewAdminSystemServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminSystemServiceHandler(adminSystemServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
		adminStrategyServer := admin.NewAdminStrategyServer(strategySvc, log)
		mux.Handle(antv1c.NewAdminStrategyServiceHandler(adminStrategyServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))

	// S1.1-S1.3: Wire SignalPipeline, rate limiter, cost estimator, OMS writer.

	// AutoTradingService handler — leverages existing pipeline + repositories.
	autoTradingRepo := repository.NewAutoTradingRepository(pool)
	autoTradingServer := autotrading.NewAutoTradingServer(autoTradingRepo, nil, log)

	// Wire per-user risk config into the Gate (frontend settings → Gate evaluation).
	// Tries RiskConfig (account-level) first, falls back to GlobalSettings (user-level).
	strategyExecServer.AddGateRule(&risk.UserRiskConfigRule{Store: func(ctx context.Context, accountID string) (*risk.UserRiskConfig, error) {
		aid, err := uuid.Parse(accountID)
		if err != nil {
			return nil, nil
		}
		// 1. Try account-level RiskConfig.
		if rc, err := autoTradingRepo.GetRiskConfigByAccountID(ctx, aid); err == nil && rc != nil {
			return &risk.UserRiskConfig{
				MaxLotSize: rc.MaxLotSize, MaxPositions: rc.MaxPositions,
				MaxDailyLoss: rc.MaxDailyLoss.InexactFloat64(),
				MaxDrawdownPercent: rc.MaxDrawdownPercent,
				MaxRiskPercent: rc.MaxRiskPercent,
			}, nil
		}
		// 2. Fall back to user-level GlobalSettings.
		uid, _ := uuid.Parse(usermgr.GetUserID(ctx))
		if uid != uuid.Nil {
			if gs, err := autoTradingRepo.GetGlobalSettingsByUserID(ctx, uid); err == nil && gs != nil {
				return &risk.UserRiskConfig{
					MaxLotSize: gs.MaxLotSize, MaxPositions: int(gs.MaxPositions),
					MaxDailyLoss: gs.MaxDailyLoss.InexactFloat64(),
					MaxDrawdownPercent: gs.MaxDrawdownPercent,
					MaxRiskPercent: gs.MaxRiskPercent,
				}, nil
			}
		}
		return nil, nil // no config = no restriction
	}})
	mux.Handle(antv1c.NewAutoTradingServiceHandler(autoTradingServer,
		connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	// Schedule execution engine — timer-driven loop that dispatches due schedules
	// to RunLiveStrategy (bar stream → Go-native executor → signal → OMS).
	scheduleRepo := repository.NewStrategyScheduleRepository(pool)
	scheduleEngine := strategy.NewScheduleEngine(scheduleRepo, templatesRepo,
		strategyExecServer,
		func(userID uuid.UUID) bool {
			settings, err := autoTradingRepo.GetGlobalSettingsByUserID(context.Background(), userID)
			if err != nil {
				return true // default to enabled if settings not found
			}
			return settings.AutoTradeEnabled
		},
		log)
	strategyServer.SetEngine(scheduleEngine)

	// Factor subscriber activation prerequisites (M10-BASE-B6):
	// When ready to wire, create and start:
	//   factorSub := factor.NewSubscriber(factor.DefaultSubscriberConfig(), log)
	//   go factorSub.Start(pipelineCtx)
	// Required before activation:
	//   (1) Factor registry that registers DSL strategies
	//   (2) Bar-stream subscription from mdgateway → factorSub.Push()
	//   (3) Evaluation results → signal/order pipeline

	adminJurisdictionServer := admin.NewAdminJurisdictionServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminJurisdictionServiceHandler(adminJurisdictionServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))

	emailNotifier, workerCleanup := registerSREHandlers(
		userRepo, mux, log, pool, store, nc, rdb, cfg,
		authInterceptor, otelInterceptor, platformSvc, mthubSvc,
		authServer,
		strategyExperimentRepo, strategyAssetRepo, schedHealthRepo,
		analyticsCache,
		aiSvc,
		backtestRunRepo,
		pgListen,
	)

	// Daily hard-delete of expired soft-deleted users (30-day retention).
	// After 30 days, CASCADE/SET NULL FKs from migrations 149/150 take effect,
	// permanently removing all user-owned data.
	go func() {
		// Run immediately on startup so recently-expired users are cleaned
		// without waiting 24h for the first tick.
		doCleanup := func() {
			cleanCtx, ccl := context.WithTimeout(ctx, 5*time.Minute)
			deleted, err := adminRepo.HardDeleteExpiredUsers(cleanCtx, 30)
			if err != nil {
				log.Warn("hard-delete expired users failed", zap.Error(err))
			} else if deleted > 0 {
				log.Info("hard-deleted expired users", zap.Int64("count", deleted))
			}
			ccl()
		}
		doCleanup()

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				doCleanup()
			}
		}
	}()

	return reconLoop, emailNotifier, platformAgg, notifSender, scheduleEngine, workerCleanup
}

// configureStrategyExecution creates the StrategyExecutionServer with all dependencies
// wired in — bar source, paper engine, notification sender, gate, and auto-gate
// callback. Returns the fully configured server ready for handler registration.
func configureStrategyExecution(
	backtestRunRepo *repository.BacktestRunRepository,
	marketDataRepo repository.MarketDataStore,
	mthubSvc *mthub.MtHubService,
	hub *mthub.Hub,
	paperEngine *papereng.PaperEngine,
	notifSender *notifpubsub.Sender,
	aiSvc *systemai.Service,
	pgListen *pglisten.Listener,
	jurisGate *risksvc.JurisdictionGate,
	capStore *risksvc.CapabilityStore,
	cfg *config.Config,
	log *zap.Logger,
) *strategy.StrategyExecutionServer {
	srv := strategy.NewStrategyExecutionServer(backtestRunRepo, log)
	srv.SetPgListen(pgListen)
	srv.SetMarketDataRepo(marketDataRepo)
	srv.SetBarSource(strategy.NewLiveSource(mthubSvc, marketDataRepo))
	srv.SetMtHub(mthubSvc)
	srv.SetGoExecutor(strategy.NewGoExecutor(".", log))
	srv.StartBacktestWorker(context.Background())
	srv.SetPaperEngine(paperEngine)
	srv.SetNotificationSender(notifSender)

	// D6-A: risk Gate — system rules only. User trading constraints are opt-in.
	gate := risk.NewGateWithSystemRules()
	gate.SetKillSwitch(func() bool { return cfg.RiskGateKillSwitch })
	gate.SetAutotradeEnabled(func(uid string) bool { return cfg.RiskGateAutotradeEnabled })

	// Wire KYC/Jurisdiction (legal compliance — non-optional when configured).
	if jurisGate != nil {
		gate.AddRule(&risk.KycJurisdictionGateRule{
			Gate:      jurisGate,
			UserIDFn:  usermgr.GetUserID,
			ClientIPFn: func(ctx context.Context) string { return "" },
		})
	}
	// Wire capability tier (per-user trading limits from DB).
	if capStore != nil {
		gate.AddRule(risk.NewCapabilityTierRule(capStore))
	}
	srv.SetGate(gate)       // live_runner startup guard only (gate runs in mthub now)
	mthubSvc.SetGate(gate)   // D6-A single chokepoint: all orders through mthub
	// T3.2b: Inject AccountStateProvider for live trading.
	// Uses the MT4 gateway's FetchOpenedOrders to derive equity/balance/margin.
	srv.SetAccountProvider(strategy.NewMTAccountStateProvider(hub, log))
	log.Info("D6-A: risk.Gate + AccountStateProvider injected into StrategyExecutionServer")

	// Auto-gate: runs gate evaluation after every backtest completion.
	// On failure, spawns async auto-fix (LLM code repair → new backtest).
	onBacktestComplete := func(ctx context.Context, run *repository.BacktestRun) {
		dailyReturns := internalai.EquityCurveToDailyReturns(run.ProtoResponse)
		if len(dailyReturns) < 10 {
			return
		}
		input := internalai.PipelineInput{
			DailyReturns: dailyReturns,
			NumAttempts:  1,
		}
		result := internalai.Pipeline(input)
		ai.SendGateNotification(ctx, notifSender, run.UserID, run, result)
		if result.Passed {
			return
		}
		go autoFixCode(context.Background(), run, result, aiSvc, backtestRunRepo, notifSender, log)
	}
	srv.SetOnBacktestComplete(onBacktestComplete)
	return srv
}
