package main

import (
	"context"
	"net/http"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
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

// aiServicesParams holds parameters for setupAIServices.
type aiServicesParams struct {
	Ctx              context.Context
	Mux              *http.ServeMux
	Pool             *pgxpool.Pool
	Cfg              *config.Config
	UserRepo         *repository.UserRepository
	MarketDataRepo   repository.MarketDataStore
	PlatformSvc      *service.PlatformService
	MktplaceSvc      *marketplace.Service
	MktplaceHandler  *mktplace.MarketplaceServer
	QuotaChecker     *service.QuotaChecker
	WalletSvc        *service.WalletService
	ConvRepo         *repository.AIConversationRepository
	Session          *internalai.ConversationSession
	BacktestRunRepo  *repository.BacktestRunRepository
	Log              *zap.Logger
	OtelInterceptor  connectrpc.Interceptor
	AuthInterceptor  connectrpc.Interceptor
}

// setupAIServices wires all AI-related services and returns the shared AI service
// and agent gateway for use by downstream handlers.
func setupAIServices(p aiServicesParams) aiServicesDeps {
	ctx := p.Ctx
	mux := p.Mux
	pool := p.Pool
	cfg := p.Cfg
	log := p.Log
	aiRepo := repository.NewSystemAIConfigRepository(pool)
	var aiBox *secretbox.Box
	if mk := cfg.AntMasterKey; mk != "" {
		aiBox = secretbox.New([]byte(mk))
	}
	aiSvc := systemai.NewService(aiRepo, aiBox)
	aiSvc.SetUserRepo(p.UserRepo)
	aiSvc.SetLogger(log)
	aiSvc.SetCircuitBreakerDB(&pgxCB{p: pool})
	agentDefRepo := repository.NewAIAgentDefinitionRepository(pool)
	aiServer := ai.NewAIServer(aiSvc, p.ConvRepo, p.Session, log)
	aiServer.SetAgentDefRepo(agentDefRepo)
	aiServer.SetFeedbackRepo(repository.NewSessionFeedbackRepository(pool))
	mux.Handle(antv1c.NewAIServiceHandler(aiServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
	mux.Handle(antv1c.NewAgentDefinitionServiceHandler(aiServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))

	assetAnalyzer := analysis.NewAnalyzer(p.MarketDataRepo, log)
	assetAnalysisServer := assetanalysis.NewAssetAnalysisServer(assetAnalyzer, aiSvc, p.PlatformSvc, log)
	mux.Handle(antv1c.NewAssetAnalysisServiceHandler(assetAnalysisServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))

	gatewayProviderRepo := repository.NewSystemAIProviderRepository(pool)
	gatewayModelRepo := repository.NewAIModelRepository(pool)
	gatewayTokenUsageRepo := repository.NewAITokenUsageRepository(pool)
	gatewayServer := gateway.NewAIGatewayServer(gatewayProviderRepo, gatewayModelRepo, gatewayTokenUsageRepo, p.WalletSvc, aiBox, log)
	mux.Handle(antv1c.NewAIGatewayServiceHandler(gatewayServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
	aiSvc.SetGatewayProviderRepo(gatewayProviderRepo)

	// Phase 1.1: per-user daily quota + platform-wide daily cost circuit breaker.
	dailyQuotaCfg := service.DailyQuotaConfig{
		MaxSessionsPerDay: envInt("AI_DAILY_MAX_SESSIONS", 5),
		MaxTokensPerDay:   envInt("AI_DAILY_MAX_TOKENS", 200_000),
	}
	dailyQuota := service.NewDailyQuotaChecker(gatewayTokenUsageRepo, dailyQuotaCfg, log)
	costThreshold := decimal.NewFromInt(int64(envInt("AI_DAILY_COST_LIMIT_USD", 50)))
	if v := os.Getenv("AI_DAILY_COST_LIMIT_USD"); v != "" {
		if parsed, err := decimal.NewFromString(v); err == nil && parsed.IsPositive() {
			costThreshold = parsed
		}
	}
	costBreaker := service.NewPlatformCostBreaker(gatewayTokenUsageRepo, costThreshold, log)

	// Wire runtime config from agent_managed_settings (admin UI adjustable without restart).
	settingsStore := agent.NewSettingsStore(pool)
	dailyQuota.SetManagedSettingProvider(settingsStore)
	costBreaker.SetManagedSettingProvider(settingsStore)

	// BYO-only degradation: when cost breaker trips, system-paid Gateway fallback is blocked,
	// but BYO-key users continue to work.
	aiSvc.SetCostBreaker(costBreaker)

	wireAIBilling(aiSvc, p.WalletSvc, gatewayServer, gatewayModelRepo, p.QuotaChecker, gatewayTokenUsageRepo, dailyQuota)

	creditSvc := wireCreditBilling(p, pool, gatewayModelRepo, mux, log)
	agentGateway := wireAgentGateway(p, pool, aiSvc, creditSvc, mux, log, ctx)

	// Remaining AI service registrations.
	codeAssistServer := ai.NewCodeAssistServer(aiSvc, p.Session, log)
	mux.Handle(antv1c.NewCodeAssistServiceHandler(codeAssistServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
	systemAIServer := ai.NewSystemAIServer(aiSvc, log)
	mux.Handle(antv1c.NewSystemAIServiceHandler(systemAIServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
	aiPrimaryServer := ai.NewAIPrimaryServer(aiSvc, log)
	mux.Handle(antv1c.NewAIPrimaryServiceHandler(aiPrimaryServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
	backtestTradesServer := strategy.NewBacktestTradesServer(p.BacktestRunRepo, log)
	mux.Handle(antv1c.NewBacktestTradesServiceHandler(backtestTradesServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
	gateEvalServer := ai.NewGateEvalServer(p.BacktestRunRepo, log)
	mux.Handle(antv1c.NewGateServiceHandler(gateEvalServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
	strategyPlanServer := ai.NewStrategyPlanServer(aiSvc, p.BacktestRunRepo, p.ConvRepo, p.MarketDataRepo, log)
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
	mux.Handle(antv1c.NewStrategyPlanServiceHandler(strategyPlanServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))

	return aiServicesDeps{
		aiSvc:          aiSvc,
		agentGateway:   agentGateway,
		gateEvalServer: gateEvalServer,
	}
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func wireCreditBilling(p aiServicesParams, pool *pgxpool.Pool, gatewayModelRepo *repository.AIModelRepository, mux *http.ServeMux, log *zap.Logger) *service.CreditService {
	creditRepo := repository.NewCreditRepository(pool)
	creditSvc := service.NewCreditService(creditRepo, gatewayModelRepo, log)
	creditServer := ai.NewCreditServer(creditSvc, creditRepo, log)
	mux.Handle(antv1c.NewCreditServiceHandler(creditServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
	adminCreditServer := ai.NewAdminCreditServer(creditRepo, log)
	mux.Handle(antv1c.NewAdminCreditServiceHandler(adminCreditServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
	return creditSvc
}

func wireAgentGateway(p aiServicesParams, pool *pgxpool.Pool, aiSvc *systemai.Service, creditSvc *service.CreditService, mux *http.ServeMux, log *zap.Logger, ctx context.Context) *agent.GatewayServer {
	agentGateway := agent.NewGatewayServer(pool, p.MarketDataRepo, aiSvc, log)
	agentGateway.Generator().SetCreditSvc(creditSvc)
	mux.Handle(antv1c.NewAgentGatewayServiceHandler(agentGateway, withSency(p.OtelInterceptor, p.AuthInterceptor)))

	p.MktplaceHandler.SetGenerator(agentGateway.Generator())
	if agentGateway.Generator() != nil {
		p.MktplaceSvc.SetOptimizer(agentGateway.Generator())
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

	return agentGateway
}
