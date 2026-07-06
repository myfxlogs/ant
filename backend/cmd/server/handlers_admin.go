package main

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/agent"
	"anttrader/internal/connect/admin"
	"anttrader/internal/repository"
	"anttrader/internal/service"
	usersvc "anttrader/internal/service/user"

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

	adminAccountServer := admin.NewAdminAccountServer(adminRepo, log)
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

	// ADR-0025 §5.4 + §8: Agent settings + hooks management.
	if settingsStore != nil {
		adminAgentSettingsServer := admin.NewAdminAgentSettingsServer(settingsStore, log)
		mux.Handle(antv1c.NewAdminAgentSettingsServiceHandler(adminAgentSettingsServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor, adminInterceptor)))
	}
	if pool != nil {
		agentHooksServer := admin.NewAgentHooksServer(pool, hookEngine, log)
		mux.Handle(antv1c.NewAgentHooksServiceHandler(agentHooksServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor, adminInterceptor)))
	}
}
