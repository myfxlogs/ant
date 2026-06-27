// SRE and platform handlers — ConnectRPC for all business logic.
// Auth cookie endpoints are HTTP-native (httpOnly cookie requirement).
package main

import (
	"context"
	"net/http"

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
	userRepo *repository.UserRepository,
	mux *http.ServeMux,
	log *zap.Logger,
	pool *pgxpool.Pool,
	store repository.MarketDataStore,
	nc *nats.Conn,
	rdb *antredis.Client,
	cfg *config.Config,
	authInterceptor *interceptor.AuthInterceptor,
	otelInterceptor connectrpc.Interceptor,
	platformSvc *service.PlatformService,
	mthubSvc *mthub.MtHubService,
	authServer *user.AuthServer,
	strategyExperimentRepo *repository.StrategyExperimentRepository,
	strategyAssetRepo *repository.StrategyAssetRepository,
	schedHealthRepo *repository.ScheduleHealthRepository,
	analyticsCache *service.AnalyticsCache,
	aiSvc *systemai.Service,
	backtestRunRepo *repository.BacktestRunRepository,
	pgListen *pglisten.Listener,
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
	// AdminSRE — ConnectRPC (proto binary)
	mux.Handle(antv1c.NewAdminSREServiceHandler(sreHandler, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	analyticsRepo := repository.NewAnalyticsRepository(pool)
	analyticsServer := system.NewAnalyticsServer(analyticsRepo, platformSvc, analyticsCache, aiSvc, log)
	mux.Handle(antv1c.NewAnalyticsServiceHandler(analyticsServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	marketRegimeRepo := repository.NewMarketRegimeRepository(pool)
	marketRegimeServer := mktplace.NewMarketRegimeServer(marketRegimeRepo, store, platformSvc, log)
	mux.Handle(antv1c.NewMarketRegimeServiceHandler(marketRegimeServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

	strategyExperimentServer := strategy.NewStrategyExperimentServer(strategyExperimentRepo, log)
	strategyExperimentServer.SetPgListen(pgListen)
	mux.Handle(antv1c.NewStrategyExperimentServiceHandler(strategyExperimentServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	experimentWorker := strategy.NewExperimentWorker(strategyExperimentRepo, backtestRunRepo, store, log)
	experimentWorker.SetPgListen(pgListen)
	if aiSvc != nil {
		experimentWorker.SetAIService(aiSvc)
	}
	experimentWorker.Start(context.Background())

	// AI reflection loop: validates historical predictions → recalibrates confidence.
	calRepo := ai.NewCalibrationRepository(pool)
	calSvc := ai.NewCalibrationService(calRepo)
	reflectionWorker := ai.NewReflectionWorker(calSvc, store, log)
	reflectionWorker.Start(context.Background())
	strategyAssetServer := strategy.NewStrategyAssetServer(strategyAssetRepo, userRepo, log)
	mux.Handle(antv1c.NewStrategyAssetServiceHandler(strategyAssetServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	scheduleHealthServer := system.NewScheduleHealthServer(schedHealthRepo, log)
	mux.Handle(antv1c.NewScheduleHealthServiceHandler(scheduleHealthServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
	indicatorCatalogServer := mktplace.NewIndicatorCatalogServer(log)
	mux.Handle(antv1c.NewIndicatorCatalogServiceHandler(indicatorCatalogServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))

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

	// /readyz — lightweight readiness probe (K8s standard). Does NOT check DB dependencies;
	// unlike /healthz, returning 503 here tells K8s to stop routing traffic without killing the pod.
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ready"))
	})

	workerCleanup := func() {
		experimentWorker.Stop()
		reflectionWorker.Stop()
	}
	return emailNotifier, workerCleanup
}
