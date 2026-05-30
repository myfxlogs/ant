package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
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
	"anttrader/internal/mdgateway/chmigrate"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/model"
	"anttrader/internal/mthub"
	"anttrader/internal/notifier"
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
	"anttrader/internal/factor"

	connectrpc "connectrpc.com/connect"
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
	// --- mdgateway pipeline (M10 runner) ---
	tradeRecordRepo := repository.NewTradeRecordRepository(pool)
	accountSyncSvc := service.NewAccountSyncService(tradeRecordRepo, mthubSvc, log)

	spillDir := cfg.SpillDir
	pipelineCtx, pipelineCancel := context.WithCancel(context.Background())
	defer pipelineCancel()

	// B-2.3: Per-broker 3-level margin call detection.
	// Level 1 (预警): margin_level <= call_pct * 1.5 → SSE only
	// Level 2 (警告): margin_level <= call_pct → SSE + Email
	// Level 3 (危急): margin_level <= call_pct * 0.7 → SSE + Email (1min cooldown)
	var marginCallMu sync.Mutex
	marginCallLastSent := make(map[string]map[int]time.Time)
	// Per-account broker thresholds loaded from mt_accounts.
	marginCallThresholds := make(map[string]float64) // accountID → broker_margin_call_pct
	var emailNotifier *notifier.EmailNotifier             // set after creation; referenced by OnAccountProfit closure
	var platformAgg *risksvc.PlatformAggregator           // set after creation; referenced by OnOrderUpdate closure
	var snapshotMu sync.Mutex
	lastSnapshot := make(map[string]time.Time) // throttle: 1 snapshot/hour/account

	// Load per-account broker margin call thresholds (default 100.0 from migration 122).
	func() {
		rows, err := pool.Query(context.Background(), `SELECT id, broker_margin_call_pct FROM mt_accounts`)
		if err != nil {
			log.Warn("B-2.3: failed to load margin thresholds, using defaults", zap.Error(err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var aid string
			var pct float64
			if err := rows.Scan(&aid, &pct); err != nil {
				log.Warn("B-2.3: scan margin threshold failed", zap.Error(err))
				continue
			}
			marginCallThresholds[aid] = pct
		}
	}()

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
				// Write latest balance/equity to PG via AccountService.
				writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				userUID, _ := uuid.Parse(userID)
				if err := accountSvc.UpdateAccountMetrics(writeCtx, userUID, accountID, p.Balance, p.Equity, p.Credit, p.Margin, p.FreeMargin, p.MarginLevel); err != nil {
					log.Warn("OnAccountProfit: pg update failed", zap.String("account", accountID), zap.Error(err))
				}
				// Record hourly equity snapshot (throttled).
				func() {
					snapshotMu.Lock()
					last, exists := lastSnapshot[accountID]
					if exists && time.Since(last) < time.Hour {
						snapshotMu.Unlock()
						return
					}
					lastSnapshot[accountID] = time.Now()
					snapshotMu.Unlock()
					if err := accountSvc.RecordBalanceSnapshot(writeCtx, accountID, p.Balance, p.Equity, p.Margin, p.FreeMargin); err != nil {
						log.Debug("OnAccountProfit: snapshot insert failed", zap.String("account", accountID), zap.Error(err))
					}
				}()
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
				// B-2.3: 3-level margin call detection with per-broker thresholds.
				if p.MarginLevel > 0 {
					callPct := marginCallThresholds[accountID]
					if callPct <= 0 {
						callPct = 100.0
					}
					service.CheckMarginCall(accountID, userID, p.MarginLevel, p.Margin, p.Equity, callPct, &marginCallMu, marginCallLastSent, eventStore, emailNotifier)
				}
			},
			OnOrderUpdate: func(accountID, userID string, o *mdtick.OrderUpdate) {
				// Update PG with latest account metrics via AccountService.
				writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				userUID, _ := uuid.Parse(userID)
				if err := accountSvc.UpdateAccountMetrics(writeCtx, userUID, accountID, o.Balance, o.Equity, o.Credit, o.Margin, o.FreeMargin, o.MarginLevel); err != nil {
					log.Warn("OnOrderUpdate: pg update failed", zap.String("account", accountID), zap.Error(err))
				}

				// Publish profit update to AccountProfitBroker for SSE streaming.
				accountBroker.Publish(&mthub.AccountProfitEvent{
					AccountID: accountID, UserID: userID, Platform: o.Platform,
					Balance: o.Balance, Credit: o.Credit, Equity: o.Equity,
					Margin: o.Margin, FreeMargin: o.FreeMargin, MarginLevel: o.MarginLevel,
					Profit: o.Profit, ProfitPercent: o.ProfitPercent, Status: "connected", Timestamp: time.Now(),
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

				// B-1.2: Feed PlatformAggregator for risk pipeline Stage 3 (platform_limits).
				platformAgg.ClearAccount(accountID)
				for _, pos := range o.Positions {
					netVol := pos.Volume
					if pos.Type == "sell" {
						netVol = -netVol
					}
					platformAgg.UpdatePosition(accountID, &risksvc.AggregatorPosition{
						Canonical: pos.Symbol,
						NetVolume: netVol,
						Notional:  pos.Volume * pos.CurrentPrice * 100000,
						Margin:    0,
					})
				}

					// Auto-write closed orders to trade_records.
					if o.UpdateType == "close" && o.UpdateCloseTime > 0 {
						uid, err := uuid.Parse(accountID)
						if err == nil {
							rec := &model.TradeRecord{
								AccountID:    uid,
								Ticket:       o.UpdateTicket,
								Symbol:       o.UpdateSymbol,
								OrderType:    o.UpdateOrderType,
								Volume:       o.UpdateVolume,
								OpenPrice:    o.UpdateOpenPrice,
								ClosePrice:   o.UpdateClosePrice,
								Profit:       o.UpdateProfit,
								Swap:         o.UpdateSwap,
								Commission:   o.UpdateCommission,
								OpenTime:     time.Unix(o.UpdateOpenTime, 0),
								CloseTime:    time.Unix(o.UpdateCloseTime, 0),
								StopLoss:     o.UpdateSL,
								TakeProfit:   o.UpdateTP,
								OrderComment: o.UpdateComment,
								Platform:     o.Platform,
							}
							if err := tradeRecordRepo.Create(writeCtx, rec); err != nil {
								log.Warn("OnOrderUpdate: write closed trade failed", zap.String("account", accountID), zap.Int64("ticket", o.UpdateTicket), zap.Error(err))
							}
						}
					}
			},
			OnAccountDisconnect: func(accountID string) {
				accountSyncSvc.SyncAccountHistory(accountID)
				platformAgg.ClearAccount(accountID)
				// Update DB status so frontend doesn't keep showing stale "connected" state.
				writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := accountSvc.DisconnectAccountByID(writeCtx, accountID); err != nil {
					log.Warn("OnAccountDisconnect: failed to update account_status", zap.String("account", accountID), zap.Error(err))
				}
			},
			OnBrokerInfo: func(accountID, platform, broker string, info *mdtick.BrokerInfo) {
				accountSyncSvc.SyncAccountHistory(accountID)
				// H17: Trigger reconciliation on broker reconnect so ant-side state
				// stays consistent with broker-side reality (ADR-0013).
				if reconLoop != nil {
					reconLoop.ReconcileAccount(context.Background(), accountID)
				}
					// Publish initial position snapshot so frontend has data on first load.
					go func() {
						sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()
						orders, err := mthubSvc.OpenedOrders(sctx, accountID)
						if err != nil {
							log.Warn("OnBrokerInfo: OpenedOrders failed, initial position snapshot skipped",
								zap.String("account", accountID), zap.String("platform", platform), zap.Error(err))
							return
						}
						snapshot := &mthub.PositionSnapshot{AccountID: accountID, Positions: make([]mthub.PositionSnapshotItem, 0, len(orders))}
						for _, o := range orders {
							snapshot.Positions = append(snapshot.Positions, mthub.PositionSnapshotItem{
								Ticket: o.Ticket, Symbol: o.SymbolRaw, Type: service.MapSideToString(o.Side), Volume: o.Volume.InexactFloat64(),
								OpenPrice: o.OpenPrice.InexactFloat64(), Profit: o.Profit.InexactFloat64(),
								Swap: o.Swap.InexactFloat64(), Commission: o.Commission.InexactFloat64(), Comment: o.Comment,
							})
						}
						snapshotBroker.Publish(snapshot)
					}()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				pct := info.MarginCallPct
				stop := info.StopOutPct
				// Zero values mean "proto doesn't expose these yet" — keep the schema DEFAULTs.
				if pct > 0 || stop > 0 {
					if err := accountSvc.UpdateBrokerThresholds(ctx, accountID, pct, stop); err != nil {
						log.Warn("failed to persist broker margin info",
							zap.String("account", accountID), zap.Error(err))
					} else {
						log.Info("broker margin thresholds updated",
							zap.String("account", accountID),
							zap.Float64("margin_call_pct", pct),
							zap.Float64("stop_out_pct", stop))
					}
				}
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
	marketDataRepo := repository.NewMarketDataRepository(ch, log)

	authServer := user.NewAuthServer(userRepo, jwtSecret, log)
	authServer.SetInsecureCookies(true) // no TLS in Docker deployment
	mux.Handle(antv1c.NewAuthServiceHandler(authServer, connectrpc.WithInterceptors(rateLimitInterceptor, authInterceptor)))

	reconLoop = mthub.NewReconciliationLoop(hub, pool, rdb.Client(), log, reconcileGate)

	mthubServer := system.NewMtHubServer(mthubSvc, platformSvc, marketDataRepo, tradeRecordRepo, log)
	mux.Handle(antv1c.NewMtHubServiceHandler(mthubServer, connectrpc.WithInterceptors(authInterceptor)))

	searcher := brokersearch.New("", "")
	accountEventPub := mdgateway.NewAccountEventPublisher(js, log)
	mtTester := user.NewMTConnectionTester(cfg.MtapiToken, log)
	accountServer := user.NewAccountServer(accountSvc, searcher, accountEventPub, mtTester, log)
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
	backtestRunRepo := repository.NewBacktestRunRepository(pool)

	aiRepo := repository.NewSystemAIConfigRepository(pool)
	var aiBox *secretbox.Box
	if mk := cfg.AntMasterKey; mk != "" {
		aiBox = secretbox.New([]byte(mk))
	}
	aiSvc := systemai.NewService(aiRepo, aiBox)
	aiServer := ai.NewAIServer(aiSvc, convRepo, log)
	mux.Handle(antv1c.NewAIServiceHandler(aiServer, connectrpc.WithInterceptors(authInterceptor)))

	// Debate V2 service (multi-expert AI strategy generation).
	debateV2Svc := service.NewDebateV2Service(pool, jobRepo, log)
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
	pythonStrategyServer := strategy.NewPythonStrategyServer(backtestRunRepo, log)
	if cfg.StrategyServiceURL != "" {
		pythonClient := strategysvc.NewPythonClient(cfg.StrategyServiceURL)
		pythonStrategyServer.SetClient(pythonClient)
		strategyServer.SetClient(pythonClient) // S2.5: real RunBacktest
		log.Info("Python strategy client configured", zap.String("url", cfg.StrategyServiceURL))
	}
	mux.Handle(antv1c.NewPythonStrategyServiceHandler(pythonStrategyServer, connectrpc.WithInterceptors(authInterceptor)))
	codeAssistServer := ai.NewCodeAssistServer(aiSvc, log)
	mux.Handle(antv1c.NewCodeAssistServiceHandler(codeAssistServer, connectrpc.WithInterceptors(authInterceptor)))
	systemAIServer := ai.NewSystemAIServer(aiSvc, log)
	mux.Handle(antv1c.NewSystemAIServiceHandler(systemAIServer, connectrpc.WithInterceptors(authInterceptor)))
	aiPrimaryServer := ai.NewAIPrimaryServer(aiSvc, log)
	mux.Handle(antv1c.NewAIPrimaryServiceHandler(aiPrimaryServer, connectrpc.WithInterceptors(authInterceptor)))
	backtestTradesServer := strategy.NewBacktestTradesServer(backtestRunRepo, log)
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
			rows, err := pool.Query(context.Background(),
				`SELECT user_id, COALESCE(capability_tier, 0),
				        COALESCE(order_types_allowed, '{}'),
				        lot_per_order_max, daily_order_max, leverage_max,
				        COALESCE(symbol_whitelist, '{}'),
				        COALESCE(killswitch_enabled, false)
				 FROM user_risk_profiles`)
			if err != nil {
				log.Error("capability LoadFromPG query failed, using defaults", zap.Error(err))
			} else {
				if err := capStore.LoadFromPG(context.Background(), rows); err != nil {
					log.Error("capability LoadFromPG scan failed, using defaults", zap.Error(err))
				}
				log.Info("capability store loaded", zap.Int("users", capStore.Count()))
			}
		hardLimit := risksvc.NewHardLimitEvaluator(&risksvc.KycJurisdictionRule{Gate: jurisGate})
		platformAgg = risksvc.NewPlatformAggregator()
		platformAgg.StartRefreshLoop(5 * time.Second) // B-1.4: async recalculate on dirty
		platformLimits := risksvc.DefaultPlatformLimits()
		riskEngine := risksvc.NewEngine(
			&risksvc.MaxPosition{Max: 20},
			&risksvc.Margin{MinLevel: 1.5},
		)
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

		// D-1: Wire factor subscriber for bar-event DSL evaluation (M10-BASE-B6).
		// TODO(H16): factorSub is created and started but not yet connected to a
		// factor evaluation pipeline. Needs: (1) a factor registry that registers DSL
		// strategies, (2) a bar-stream subscription from the market data gateway,
		// (3) evaluation results fed back into the signal/order pipeline.
		factorSub := factor.NewSubscriber(factor.DefaultSubscriberConfig(), log)
		go factorSub.Start(context.Background())

	adminJurisdictionServer := admin.NewAdminJurisdictionServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminJurisdictionServiceHandler(adminJurisdictionServer, connectrpc.WithInterceptors(authInterceptor, adminInterceptor)))

	// --- SRE control plane ---
	sreKillSwitch := controlplane.NewKillSwitch()
	mthubSvc.SetKillSwitch(sreKillSwitch) // V3-R-5: PlaceOrder blocked when kill switch engaged
	sreBreakers := controlplane.NewBreakerRegistry(controlplane.DefaultBreakerConfig())
	sreCanary := controlplane.NewCanaryManager()
	emailNotifier = notifier.NewEmailNotifier(notifier.EmailConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		User:     cfg.SMTPUser,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
		To:       splitAndTrim(cfg.SMTPTo, ","),
	}, log)
	sreHandler := admin.NewSREHandler(sreKillSwitch, sreBreakers, sreCanary, platformSvc, emailNotifier, log)

	analyticsRepo := repository.NewAnalyticsRepository(pool)
	analyticsServer := system.NewAnalyticsServer(analyticsRepo, platformSvc, log)
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

