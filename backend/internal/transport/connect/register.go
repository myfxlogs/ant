package connect

import (
	"net/http"

	connectrpc "connectrpc.com/connect"
	"connectrpc.com/grpcreflect"

	v1connect "anttrader/gen/proto/antraderconnect"
)

type Services struct {
	Auth               v1connect.AuthServiceHandler
	Account            v1connect.AccountServiceHandler
	Trading            v1connect.TradingServiceHandler
	Market             v1connect.MarketServiceHandler
	Stream             v1connect.StreamServiceHandler
	Strategy           v1connect.StrategyServiceHandler
	AI                 v1connect.AIServiceHandler
	AdminUser          v1connect.AdminUserServiceHandler
	AdminAccount       v1connect.AdminAccountServiceHandler
	AdminTrading       v1connect.AdminTradingServiceHandler
	AdminConfig        v1connect.AdminConfigServiceHandler
	AdminLog           v1connect.AdminLogServiceHandler
	AdminSystem        v1connect.AdminSystemServiceHandler
	Analytics          v1connect.AnalyticsServiceHandler
	PythonStrategy     v1connect.PythonStrategyServiceHandler
	BacktestDataset    v1connect.BacktestDatasetServiceHandler
	BacktestTrades     v1connect.BacktestTradesServiceHandler
	DebateV2           v1connect.DebateV2ServiceHandler
	DebateV2Stream     v1connect.DebateV2StreamServiceHandler
	AIPrimary          v1connect.AIPrimaryServiceHandler
	CodeAssist         v1connect.CodeAssistServiceHandler
	AutoTrading        v1connect.AutoTradingServiceHandler
	Log                v1connect.LogServiceHandler
	ScheduleHealth     v1connect.ScheduleHealthServiceHandler
	SystemAI           v1connect.SystemAIServiceHandler
	ObjectiveScore     v1connect.ObjectiveScoreServiceHandler
	IndicatorCatalog   v1connect.IndicatorCatalogServiceHandler
	EconomicData       v1connect.EconomicDataServiceHandler
	Agent              v1connect.AgentServiceHandler
	Job                v1connect.JobServiceHandler
	StrategyExperiment v1connect.StrategyExperimentServiceHandler
	StrategyAsset      v1connect.StrategyAssetServiceHandler
	MarketRegime       v1connect.MarketRegimeServiceHandler
}

func Register(mux *http.ServeMux, services Services, interceptor connectrpc.Interceptor) {
	if mux == nil {
		return
	}
	opts := []connectrpc.HandlerOption{connectrpc.WithInterceptors(interceptor)}
	mux.Handle(v1connect.NewAuthServiceHandler(services.Auth, opts...))
	mux.Handle(v1connect.NewAccountServiceHandler(services.Account, opts...))
	mux.Handle(v1connect.NewTradingServiceHandler(services.Trading, opts...))
	mux.Handle(v1connect.NewMarketServiceHandler(services.Market, opts...))
	mux.Handle(v1connect.NewStreamServiceHandler(services.Stream, opts...))
	mux.Handle(v1connect.NewStrategyServiceHandler(services.Strategy, opts...))
	mux.Handle(v1connect.NewAIServiceHandler(services.AI, opts...))
	mux.Handle(v1connect.NewAdminUserServiceHandler(services.AdminUser, opts...))
	mux.Handle(v1connect.NewAdminAccountServiceHandler(services.AdminAccount, opts...))
	mux.Handle(v1connect.NewAdminTradingServiceHandler(services.AdminTrading, opts...))
	mux.Handle(v1connect.NewAdminConfigServiceHandler(services.AdminConfig, opts...))
	mux.Handle(v1connect.NewAdminLogServiceHandler(services.AdminLog, opts...))
	mux.Handle(v1connect.NewAdminSystemServiceHandler(services.AdminSystem, opts...))
	mux.Handle(v1connect.NewAnalyticsServiceHandler(services.Analytics, opts...))
	mux.Handle(v1connect.NewPythonStrategyServiceHandler(services.PythonStrategy, opts...))
	mux.Handle(v1connect.NewBacktestDatasetServiceHandler(services.BacktestDataset, opts...))
	mux.Handle(v1connect.NewBacktestTradesServiceHandler(services.BacktestTrades, opts...))
	mux.Handle(v1connect.NewDebateV2ServiceHandler(services.DebateV2, opts...))
	mux.Handle(v1connect.NewDebateV2StreamServiceHandler(services.DebateV2Stream, opts...))
	mux.Handle(v1connect.NewAIPrimaryServiceHandler(services.AIPrimary, opts...))
	mux.Handle(v1connect.NewCodeAssistServiceHandler(services.CodeAssist, opts...))
	mux.Handle(v1connect.NewAutoTradingServiceHandler(services.AutoTrading, opts...))
	mux.Handle(v1connect.NewLogServiceHandler(services.Log, opts...))
	mux.Handle(v1connect.NewScheduleHealthServiceHandler(services.ScheduleHealth, opts...))
	mux.Handle(v1connect.NewSystemAIServiceHandler(services.SystemAI, opts...))
	mux.Handle(v1connect.NewObjectiveScoreServiceHandler(services.ObjectiveScore, opts...))
	mux.Handle(v1connect.NewIndicatorCatalogServiceHandler(services.IndicatorCatalog, opts...))
	mux.Handle(v1connect.NewEconomicDataServiceHandler(services.EconomicData, opts...))
	mux.Handle(v1connect.NewAgentServiceHandler(services.Agent, opts...))
	mux.Handle(v1connect.NewJobServiceHandler(services.Job, opts...))
	mux.Handle(v1connect.NewStrategyExperimentServiceHandler(services.StrategyExperiment, opts...))
	mux.Handle(v1connect.NewStrategyAssetServiceHandler(services.StrategyAsset, opts...))
	mux.Handle(v1connect.NewMarketRegimeServiceHandler(services.MarketRegime, opts...))
	mux.Handle(grpcreflect.NewHandlerV1(grpcreflect.NewStaticReflector(serviceNames()...)))
}

func serviceNames() []string {
	return []string{
		v1connect.AuthServiceName,
		v1connect.AccountServiceName,
		v1connect.TradingServiceName,
		v1connect.MarketServiceName,
		v1connect.StreamServiceName,
		v1connect.StrategyServiceName,
		v1connect.AIServiceName,
		v1connect.AdminUserServiceName,
		v1connect.AdminAccountServiceName,
		v1connect.AdminTradingServiceName,
		v1connect.AdminConfigServiceName,
		v1connect.AdminLogServiceName,
		v1connect.AdminSystemServiceName,
		v1connect.AnalyticsServiceName,
		v1connect.PythonStrategyServiceName,
		v1connect.BacktestDatasetServiceName,
		v1connect.BacktestTradesServiceName,
		v1connect.DebateV2ServiceName,
		v1connect.AIPrimaryServiceName,
		v1connect.CodeAssistServiceName,
		v1connect.AutoTradingServiceName,
		v1connect.LogServiceName,
		v1connect.ScheduleHealthServiceName,
		v1connect.SystemAIServiceName,
		v1connect.ObjectiveScoreServiceName,
		v1connect.IndicatorCatalogServiceName,
		v1connect.EconomicDataServiceName,
		v1connect.AgentServiceName,
		v1connect.JobServiceName,
		v1connect.StrategyExperimentServiceName,
		v1connect.StrategyAssetServiceName,
		v1connect.MarketRegimeServiceName,
	}
}
