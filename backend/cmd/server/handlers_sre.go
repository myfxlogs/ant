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
	"anttrader/internal/connect/admin"
	"anttrader/internal/connect/ai"
	mktplace "anttrader/internal/connect/marketplace"
	"anttrader/internal/connect/strategy"
	"anttrader/internal/connect/system"
	"anttrader/internal/connect/user"
	"anttrader/internal/controlplane"
	"anttrader/internal/interceptor"
	"anttrader/internal/mdgateway"
	"anttrader/internal/mthub"
	"anttrader/internal/notifier"
	"anttrader/internal/repository"
	"anttrader/internal/service"
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
	debateV2Server *ai.DebateV2Server,
	gateProgressServer *ai.GateProgressServer,
	strategyExperimentRepo *repository.StrategyExperimentRepository,
	strategyAssetRepo *repository.StrategyAssetRepository,
	schedHealthRepo *repository.ScheduleHealthRepository,
) *notifier.EmailNotifier {
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

	return emailNotifier
}
