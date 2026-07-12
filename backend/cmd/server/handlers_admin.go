package main

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/agent"
	"alphaforge/internal/connect/admin"
	"alphaforge/internal/mdgateway"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
	usersvc "alphaforge/internal/service/user"
	antredis "alphaforge/internal/storage/redis"

	connectrpc "connectrpc.com/connect"
)

// registerAdminHandlers wires all admin ConnectRPC handlers.
func registerAdminHandlers(
	mux *http.ServeMux,
	pool *pgxpool.Pool,
	log *zap.Logger,
	walletSvc *service.WalletService,
	accountNumberSvc *usersvc.AccountNumberService,
	strategySvc *service.StrategySvc,
	settingsStore *agent.SettingsStore,
	hookEngine *agent.HookEngine,
	accountEventPub *mdgateway.AccountEventPublisher,
	rdb *antredis.Client,
	nc *nats.Conn,
	otelInterceptor, authInterceptor, adminInterceptor connectrpc.Interceptor,
) {
	adminRepo := repository.NewAdminRepository(pool)
	passwordResetRepo := repository.NewPasswordResetRepo(pool)

	adminTradingServer := admin.NewAdminTradingServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminTradingServiceHandler(adminTradingServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor, adminInterceptor)))

	adminConfigServer := admin.NewAdminConfigServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminConfigServiceHandler(adminConfigServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor, adminInterceptor)))

	adminLogServer := admin.NewAdminLogServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminLogServiceHandler(adminLogServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor, adminInterceptor)))

	adminAccountServer := admin.NewAdminAccountServer(adminRepo, log).WithPublisher(accountEventPub)
	mux.Handle(antv1c.NewAdminAccountServiceHandler(adminAccountServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor, adminInterceptor)))

	deletionSvc := service.NewUserDeletionService(adminRepo, log)
	adminUserServer := admin.NewAdminUserServer(adminRepo, passwordResetRepo, walletSvc, accountNumberSvc, deletionSvc, log)
	mux.Handle(antv1c.NewAdminUserServiceHandler(adminUserServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor, adminInterceptor)))

	adminSystemServer := admin.NewAdminSystemServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminSystemServiceHandler(adminSystemServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor, adminInterceptor)))

	adminStrategyServer := admin.NewAdminStrategyServer(strategySvc, log)
	mux.Handle(antv1c.NewAdminStrategyServiceHandler(adminStrategyServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor, adminInterceptor)))

	adminJurisdictionServer := admin.NewAdminJurisdictionServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminJurisdictionServiceHandler(adminJurisdictionServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor, adminInterceptor)))

	// P3.5: Admin billing service (subscription + revenue + wallet transactions)
	adminBillingServer := admin.NewAdminBillingServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminBillingServiceHandler(adminBillingServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor, adminInterceptor)))

	// ADR-0025 §5.4 + §8: Agent settings + hooks management.
	if settingsStore != nil {
		adminAgentSettingsServer := admin.NewAdminAgentSettingsServer(settingsStore, log)
		mux.Handle(antv1c.NewAdminAgentSettingsServiceHandler(adminAgentSettingsServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor, adminInterceptor)))
	}
	if pool != nil {
		agentHooksServer := admin.NewAgentHooksServer(pool, hookEngine, log)
		mux.Handle(antv1c.NewAgentHooksServiceHandler(agentHooksServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor, adminInterceptor)))
	}

	// Admin monitor — real-time SSE system metrics
	adminMonitorServer := admin.NewAdminMonitorServer(pool, rdb, nc, log)
	mux.Handle(antv1c.NewAdminMonitorServiceHandler(adminMonitorServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor, adminInterceptor)))
}

// startHardDeleteCleanup periodically hard-deletes expired soft-deleted users (30-day retention).
func startHardDeleteCleanup(ctx context.Context, pool *pgxpool.Pool, log *zap.Logger) {
	cleanupRepo := repository.NewAdminRepository(pool)
	doCleanup := func() {
		cleanCtx, ccl := context.WithTimeout(ctx, 5*time.Minute)
		deleted, err := cleanupRepo.HardDeleteExpiredUsers(cleanCtx, 30)
		if err != nil {
			log.Warn("hard-delete expired users failed", zap.Error(err))
		} else if deleted > 0 {
			log.Info("hard-deleted expired users", zap.Int64("count", deleted))
		}
		ccl()
	}
	// Run immediately on startup.
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
}
