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
)

// adminHandlerDeps holds dependencies for registerAdminHandlers.
type adminHandlerDeps struct {
	Mux              *http.ServeMux
	Pool             *pgxpool.Pool
	Log              *zap.Logger
	WalletSvc        *service.WalletService
	AccountNumberSvc *usersvc.AccountNumberService
	StrategySvc      *service.StrategySvc
	SettingsStore    *agent.SettingsStore
	HookEngine       *agent.HookEngine
	AccountEventPub  *mdgateway.AccountEventPublisher
	RDB              *antredis.Client
	NC               *nats.Conn
	Interceptors     interceptorSet
}

// registerAdminHandlers wires all admin ConnectRPC handlers.
func registerAdminHandlers(d adminHandlerDeps) {
	mux := d.Mux
	pool := d.Pool
	log := d.Log
	ic := d.Interceptors

	adminRepo := repository.NewAdminRepository(pool)
	passwordResetRepo := repository.NewPasswordResetRepo(pool)

	adminTradingServer := admin.NewAdminTradingServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminTradingServiceHandler(adminTradingServer, withSency(ic.otel, ic.auth, ic.admin)))

	adminConfigServer := admin.NewAdminConfigServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminConfigServiceHandler(adminConfigServer, withSency(ic.otel, ic.auth, ic.admin)))

	adminLogServer := admin.NewAdminLogServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminLogServiceHandler(adminLogServer, withSency(ic.otel, ic.auth, ic.admin)))

	adminAccountServer := admin.NewAdminAccountServer(adminRepo, log).WithPublisher(d.AccountEventPub)
	mux.Handle(antv1c.NewAdminAccountServiceHandler(adminAccountServer, withSency(ic.otel, ic.auth, ic.admin)))

	deletionSvc := service.NewUserDeletionService(adminRepo, log)
	adminUserServer := admin.NewAdminUserServer(adminRepo, passwordResetRepo, d.WalletSvc, d.AccountNumberSvc, deletionSvc, log)
	mux.Handle(antv1c.NewAdminUserServiceHandler(adminUserServer, withSency(ic.otel, ic.auth, ic.admin)))

	adminSystemServer := admin.NewAdminSystemServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminSystemServiceHandler(adminSystemServer, withSency(ic.otel, ic.auth, ic.admin)))

	adminStrategyServer := admin.NewAdminStrategyServer(d.StrategySvc, log)
	mux.Handle(antv1c.NewAdminStrategyServiceHandler(adminStrategyServer, withSency(ic.otel, ic.auth, ic.admin)))

	adminJurisdictionServer := admin.NewAdminJurisdictionServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminJurisdictionServiceHandler(adminJurisdictionServer, withSency(ic.otel, ic.auth, ic.admin)))

	// P3.5: Admin billing service (subscription + revenue + wallet transactions)
	adminBillingServer := admin.NewAdminBillingServer(adminRepo, log)
	mux.Handle(antv1c.NewAdminBillingServiceHandler(adminBillingServer, withSency(ic.otel, ic.auth, ic.admin)))

	// ADR-0025 §5.4 + §8: Agent settings + hooks management.
	if d.SettingsStore != nil {
		adminAgentSettingsServer := admin.NewAdminAgentSettingsServer(d.SettingsStore, log)
		mux.Handle(antv1c.NewAdminAgentSettingsServiceHandler(adminAgentSettingsServer, withSency(ic.otel, ic.auth, ic.admin)))
	}
	if pool != nil {
		agentHooksServer := admin.NewAgentHooksServer(pool, d.HookEngine, log)
		mux.Handle(antv1c.NewAgentHooksServiceHandler(agentHooksServer, withSency(ic.otel, ic.auth, ic.admin)))
	}

	// Admin monitor — real-time SSE system metrics
	adminMonitorServer := admin.NewAdminMonitorServer(pool, d.RDB, d.NC, log)
	mux.Handle(antv1c.NewAdminMonitorServiceHandler(adminMonitorServer, withSency(ic.otel, ic.auth, ic.admin)))
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
