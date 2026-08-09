// SRE and platform handlers — ConnectRPC for all business logic.
// Auth cookie endpoints are HTTP-native (httpOnly cookie requirement).
package main

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/ai"
	"alphaforge/internal/config"
	"alphaforge/internal/connect/admin"
	mktplace "alphaforge/internal/connect/marketplace"
	"alphaforge/internal/connect/strategy"
	"alphaforge/internal/connect/system"
	"alphaforge/internal/connect/user"
	"alphaforge/internal/controlplane"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/mdgateway"
	"alphaforge/internal/mthub"
	"alphaforge/internal/notifier"
	"alphaforge/internal/pglisten"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
	systemai "alphaforge/internal/service/systemai"
	antredis "alphaforge/internal/storage/redis"

	connectrpc "connectrpc.com/connect"
)

// sreHandlerParams holds parameters for registerSREHandlers.
type sreHandlerParams struct {
	UserRepo               *repository.UserRepository
	Mux                    *http.ServeMux
	Log                    *zap.Logger
	Pool                   *pgxpool.Pool
	Store                  repository.MarketDataStore
	NC                     *nats.Conn
	RDB                    *antredis.Client
	Cfg                    *config.Config
	AuthInterceptor        *interceptor.AuthInterceptor
	OtelInterceptor        connectrpc.Interceptor
	PlatformSvc            *service.PlatformService
	MthubSvc               *mthub.MtHubService
	AuthServer             *user.AuthServer
	StrategyExperimentRepo *repository.StrategyExperimentRepository
	StrategyAssetRepo      *repository.StrategyAssetRepository
	SchedHealthRepo        *repository.ScheduleHealthRepository
	AnalyticsCache         *service.AnalyticsCache
	AISvc                  *systemai.Service
	PgListen               *pglisten.Listener
	EmailNotifier          *notifier.EmailNotifier
	StrategyExecServer     *strategy.StrategyExecutionServer
}

func registerSREHandlers(p sreHandlerParams) (*notifier.EmailNotifier, func()) {
	mux := p.Mux
	log := p.Log
	pool := p.Pool
	store := p.Store
	nc := p.NC
	rdb := p.RDB
	// --- SRE control plane ---
	sreKillSwitch := controlplane.NewKillSwitch()
	p.MthubSvc.SetKillSwitch(sreKillSwitch) // V3-R-5: PlaceOrder blocked when kill switch engaged
	sreBreakers := controlplane.NewBreakerRegistry(controlplane.DefaultBreakerConfig())
	sreCanary := controlplane.NewCanaryManager()
	sreHandler := admin.NewSREHandler(sreKillSwitch, sreBreakers, sreCanary, p.PlatformSvc, p.EmailNotifier, log)
	// AdminSRE — ConnectRPC (proto binary)
	mux.Handle(antv1c.NewAdminSREServiceHandler(sreHandler, withSency(p.OtelInterceptor, p.AuthInterceptor)))

	analyticsRepo := repository.NewAnalyticsRepository(pool)
	analyticsServer := system.NewAnalyticsServer(analyticsRepo, p.PlatformSvc, p.AnalyticsCache, p.AISvc, log)
	mux.Handle(antv1c.NewAnalyticsServiceHandler(analyticsServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))

	marketRegimeRepo := repository.NewMarketRegimeRepository(pool)
	marketRegimeServer := mktplace.NewMarketRegimeServer(marketRegimeRepo, store, p.PlatformSvc, log)
	mux.Handle(antv1c.NewMarketRegimeServiceHandler(marketRegimeServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))

	strategyExperimentServer := strategy.NewStrategyExperimentServer(p.StrategyExperimentRepo, log)
	strategyExperimentServer.SetPgListen(p.PgListen)
	mux.Handle(antv1c.NewStrategyExperimentServiceHandler(strategyExperimentServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
	experimentWorker := strategy.NewExperimentWorker(p.StrategyExperimentRepo, store, log)
	experimentWorker.SetPgListen(p.PgListen)
	if p.StrategyExecServer != nil {
		experimentWorker.SetExecutor(p.StrategyExecServer)
	}
	if p.AISvc != nil {
		experimentWorker.SetAIService(p.AISvc)
	}
	experimentWorker.Start(context.Background())

	// AI reflection loop: validates historical predictions → recalibrates confidence.
	calRepo := ai.NewCalibrationRepository(pool)
	calSvc := ai.NewCalibrationService(calRepo)
	reflectionWorker := ai.NewReflectionWorker(calSvc, store, log)
	reflectionWorker.Start(context.Background())
	strategyAssetServer := strategy.NewStrategyAssetServer(p.StrategyAssetRepo, p.UserRepo, log)
	mux.Handle(antv1c.NewStrategyAssetServiceHandler(strategyAssetServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
	scheduleHealthServer := system.NewScheduleHealthServer(p.SchedHealthRepo, log)
	mux.Handle(antv1c.NewScheduleHealthServiceHandler(scheduleHealthServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
	indicatorCatalogServer := mktplace.NewIndicatorCatalogServer(log)
	mux.Handle(antv1c.NewIndicatorCatalogServiceHandler(indicatorCatalogServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))

	// Catch-all: return 404 for unknown routes.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"not_found"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ant-v2"}`))
	})

	// Prometheus /metrics endpoint (M10 ADR-0010 §2.4).
	mux.Handle("/metrics", mdgateway.MetricsHandler())

	// Health check (includes CH + NATS + Redis)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if err := pool.Ping(context.Background()); err != nil {
			w.WriteHeader(503)
			_, _ = w.Write([]byte("pg unreachable"))
			return
		}
		if !nc.IsConnected() {
			w.WriteHeader(503)
			_, _ = w.Write([]byte("nats disconnected"))
			return
		}
		if err := rdb.Ping(context.Background()); err != nil {
			w.WriteHeader(503)
			_, _ = w.Write([]byte("redis unreachable"))
			return
		}
		_, _ = w.Write([]byte("ant ok"))
	})

	// /readyz — lightweight readiness probe (K8s standard). Does NOT check DB dependencies;
	// unlike /healthz, returning 503 here tells K8s to stop routing traffic without killing the pod.
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ready"))
	})

	workerCleanup := func() {
		experimentWorker.Stop()
		reflectionWorker.Stop()
	}
	return p.EmailNotifier, workerCleanup
}
