package main

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/agent"
	internalai "alphaforge/internal/ai"
	"alphaforge/internal/analysis"
	"alphaforge/internal/config"
	"alphaforge/internal/connect/admin"
	"alphaforge/internal/connect/ai"
	assetanalysis "alphaforge/internal/connect/asset_analysis"
	"alphaforge/internal/connect/gateway"
	strategy "alphaforge/internal/connect/strategy"
	mktplace "alphaforge/internal/connect/marketplace"
	"alphaforge/internal/marketplace"
	"alphaforge/internal/pkg/secretbox"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
	systemai "alphaforge/internal/service/systemai"

	connectrpc "connectrpc.com/connect"
)

// aiServicesDeps holds AI-related dependencies created by setupAIServices.
type aiServicesDeps struct {
	aiSvc          *systemai.Service
	agentGateway   *agent.GatewayServer
	gateEvalServer *ai.GateEvalServer
}

// setupAIServices wires all AI-related services and returns the shared AI service
// and agent gateway for use by downstream handlers.
func setupAIServices(
	ctx context.Context,
	mux *http.ServeMux,
	pool *pgxpool.Pool,
	cfg *config.Config,
	userRepo *repository.UserRepository,
	marketDataRepo repository.MarketDataStore,
	platformSvc *service.PlatformService,
	mktplaceSvc *marketplace.Service,
	mktplaceHandler *mktplace.MarketplaceServer,
	quotaChecker *service.QuotaChecker,
	walletSvc *service.WalletService,
	convRepo *repository.AIConversationRepository,
	session *internalai.ConversationSession,
	backtestRunRepo *repository.BacktestRunRepository,
	log *zap.Logger,
	otelInterceptor connectrpc.Interceptor,
	authInterceptor connectrpc.Interceptor,
) aiServicesDeps {
	aiRepo := repository.NewSystemAIConfigRepository(pool)
	var aiBox *secretbox.Box
	if mk := cfg.AntMasterKey; mk != "" {
		aiBox = secretbox.New([]byte(mk))
	}
	aiSvc := systemai.NewService(aiRepo, aiBox)
	aiSvc.SetUserRepo(userRepo)
	aiSvc.SetCircuitBreakerDB(&pgxCB{p: pool})
	agentDefRepo := repository.NewAIAgentDefinitionRepository(pool)
	aiServer := ai.NewAIServer(aiSvc, convRepo, session, log)
	aiServer.SetAgentDefRepo(agentDefRepo)
	mux.Handle(antv1c.NewAIServiceHandler(aiServer, withSency(otelInterceptor, authInterceptor)))
	mux.Handle(antv1c.NewAgentDefinitionServiceHandler(aiServer, withSency(otelInterceptor, authInterceptor)))

	assetAnalyzer := analysis.NewAnalyzer(marketDataRepo, log)
	assetAnalysisServer := assetanalysis.NewAssetAnalysisServer(assetAnalyzer, aiSvc, platformSvc, log)
	mux.Handle(antv1c.NewAssetAnalysisServiceHandler(assetAnalysisServer, withSency(otelInterceptor, authInterceptor)))

	gatewayProviderRepo := repository.NewSystemAIProviderRepository(pool)
	gatewayModelRepo := repository.NewAIModelRepository(pool)
	gatewayTokenUsageRepo := repository.NewAITokenUsageRepository(pool)
	gatewayServer := gateway.NewAIGatewayServer(gatewayProviderRepo, gatewayModelRepo, gatewayTokenUsageRepo, walletSvc, aiBox, log)
	mux.Handle(antv1c.NewAIGatewayServiceHandler(gatewayServer, withSency(otelInterceptor, authInterceptor)))
	aiSvc.SetGatewayProviderRepo(gatewayProviderRepo)
	wireAIBilling(aiSvc, walletSvc, gatewayServer, gatewayModelRepo, quotaChecker, gatewayTokenUsageRepo)

	agentGateway := agent.NewGatewayServer(pool, marketDataRepo, aiSvc, log)
	mux.Handle(antv1c.NewAgentGatewayServiceHandler(agentGateway, withSency(otelInterceptor, authInterceptor)))

	mktplaceHandler.SetGenerator(agentGateway.Generator())
	if agentGateway.Generator() != nil {
		mktplaceSvc.SetOptimizer(agentGateway.Generator())
	}

	if pool != nil && agentGateway.HookEngine() != nil {
		if err := admin.LoadHookConfigsFromDB(ctx, pool, agentGateway.HookEngine()); err != nil {
			log.Warn("failed to load hook configs from DB", zap.Error(err))
		}
	}

	if ss := agentGateway.SettingsStore(); ss != nil {
		aiSvc.SetModelFilter(func(ctx context.Context, userID uuid.UUID, model string) bool {
			rs, err := ss.ResolveSettings(ctx, userID)
			if err != nil || !rs.Loaded {
				return true
			}
			if !rs.Managed.EnforceAllowedModels {
				return true
			}
			if len(rs.Managed.AllowedModels) == 0 {
				return false
			}
			for _, allowed := range rs.Managed.AllowedModels {
				if allowed == model {
					return true
				}
			}
			return false
		})
	}

	// Remaining AI service registrations.
	codeAssistServer := ai.NewCodeAssistServer(aiSvc, session, log)
	mux.Handle(antv1c.NewCodeAssistServiceHandler(codeAssistServer, withSency(otelInterceptor, authInterceptor)))
	systemAIServer := ai.NewSystemAIServer(aiSvc, log)
	mux.Handle(antv1c.NewSystemAIServiceHandler(systemAIServer, withSency(otelInterceptor, authInterceptor)))
	aiPrimaryServer := ai.NewAIPrimaryServer(aiSvc, log)
	mux.Handle(antv1c.NewAIPrimaryServiceHandler(aiPrimaryServer, withSency(otelInterceptor, authInterceptor)))
	backtestTradesServer := strategy.NewBacktestTradesServer(backtestRunRepo, log)
	mux.Handle(antv1c.NewBacktestTradesServiceHandler(backtestTradesServer, withSency(otelInterceptor, authInterceptor)))
	gateEvalServer := ai.NewGateEvalServer(backtestRunRepo, log)
	mux.Handle(antv1c.NewGateServiceHandler(gateEvalServer, withSency(otelInterceptor, authInterceptor)))
	strategyPlanServer := ai.NewStrategyPlanServer(aiSvc, backtestRunRepo, convRepo, marketDataRepo, log)
	strategyPlanServer.SetPoolAdapter(
		func(ctx context.Context, sql string, args ...any) error {
			_, e := pool.Exec(ctx, sql, args...)
			return e
		},
		func(ctx context.Context, sql string, args ...any) (string, error) {
			var v string
			row := pool.QueryRow(ctx, sql, args...)
			err := row.Scan(&v)
			return v, err
		},
	)
	mux.Handle(antv1c.NewStrategyPlanServiceHandler(strategyPlanServer, withSency(otelInterceptor, authInterceptor)))

	return aiServicesDeps{
		aiSvc:          aiSvc,
		agentGateway:   agentGateway,
		gateEvalServer: gateEvalServer,
	}
}
