// Internal SRE endpoints — exempt from ConnectRPC requirement (operational necessity).
// Kill switches, circuit breakers, and canary configs need plain HTTP (curl-able)
// to remain functional in degraded states. Auth cookie endpoints are HTTP-native.
package main

import (
	"context"
	"net/http"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/config"
	"anttrader/internal/ai"
	"anttrader/internal/connect/admin"
	mktplace "anttrader/internal/connect/marketplace"
	"anttrader/internal/connect/strategy"
	"anttrader/internal/connect/system"
	"anttrader/internal/connect/user"
	"anttrader/internal/controlplane"
	"anttrader/internal/interceptor"
	"anttrader/internal/mdgateway"
	"anttrader/internal/mthub"
	"anttrader/internal/notifier"
	"anttrader/internal/pglisten"
	"anttrader/internal/repository"
	"anttrader/internal/service"
	systemai "anttrader/internal/service/systemai"
	antredis "anttrader/internal/storage/redis"

	connectrpc "connectrpc.com/connect"
)

func registerSREHandlers(
	mux *http.ServeMux,
	log *zap.Logger,
	pool *pgxpool.Pool,
	ch clickhouse.Conn,
	nc *nats.Conn,
	rdb *antredis.Client,
	cfg *config.Config,
	authInterceptor *interceptor.AuthInterceptor,
	platformSvc *service.PlatformService,
	mthubSvc *mthub.MtHubService,
	authServer *user.AuthServer,
	strategyExperimentRepo *repository.StrategyExperimentRepository,
	strategyAssetRepo *repository.StrategyAssetRepository,
	schedHealthRepo *repository.ScheduleHealthRepository,
	analyticsCache *service.AnalyticsCache,
	aiSvc *systemai.Service,
) (*notifier.EmailNotifier, func()) {
	// --- SRE control plane ---
	sreKillSwitch := controlplane.NewKillSwitch()
	mthubSvc.SetKillSwitch(sreKillSwitch) // V3-R-5: PlaceOrder blocked when kill switch engaged
	sreBreakers := controlplane.NewBreakerRegistry(controlplane.DefaultBreakerConfig())
	sreCanary := controlplane.NewCanaryManager()
	emailNotifier := notifier.NewEmailNotifier(notifier.EmailConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		User:     cfg.SMTPUser,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
		To:       splitAndTrim(cfg.SMTPTo, ","),
	}, log)
	sreHandler := admin.NewSREHandler(sreKillSwitch, sreBreakers, sreCanary, platformSvc, emailNotifier, log)
	// ConnectRPC handler (proto binary) -- replaces REST endpoints below
	mux.Handle(antv1c.NewAdminSREServiceHandler(sreHandler, connectrpc.WithInterceptors(authInterceptor)))

	analyticsRepo := repository.NewAnalyticsRepository(pool)
	analyticsServer := system.NewAnalyticsServer(analyticsRepo, platformSvc, analyticsCache, log)
	mux.Handle(antv1c.NewAnalyticsServiceHandler(analyticsServer, connectrpc.WithInterceptors(authInterceptor)))

	marketDataRepo := repository.NewMarketDataRepository(ch, log)
	marketRegimeRepo := repository.NewMarketRegimeRepository(pool)
	marketRegimeServer := mktplace.NewMarketRegimeServer(marketRegimeRepo, marketDataRepo, log)
	mux.Handle(antv1c.NewMarketRegimeServiceHandler(marketRegimeServer, connectrpc.WithInterceptors(authInterceptor)))

	strategyExperimentServer := strategy.NewStrategyExperimentServer(strategyExperimentRepo, log)
	strategyExperimentServer.SetPgListen(pglisten.New(pool, log))
	mux.Handle(antv1c.NewStrategyExperimentServiceHandler(strategyExperimentServer, connectrpc.WithInterceptors(authInterceptor)))
	backtestRunRepo := repository.NewBacktestRunRepository(pool)
	experimentWorker := strategy.NewExperimentWorker(strategyExperimentRepo, backtestRunRepo, marketDataRepo, log)
	if aiSvc != nil {
		experimentWorker.SetAIService(aiSvc)
	}
	experimentWorker.Start(context.Background())

	// AI reflection loop: validates historical predictions → recalibrates confidence.
	calRepo := ai.NewCalibrationRepository(pool)
	calSvc := ai.NewCalibrationService(calRepo)
	reflectionWorker := ai.NewReflectionWorker(calSvc, ch, log)
	reflectionWorker.Start(context.Background())
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

	workerCleanup := func() {
		experimentWorker.Stop()
		reflectionWorker.Stop()
	}
	return emailNotifier, workerCleanup
}
