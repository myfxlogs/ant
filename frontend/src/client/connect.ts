import { createClient } from "@connectrpc/connect";
import { AuthService } from "../gen/auth_pb";
import { AccountService } from "../gen/account_pb";
import { TradingService } from "../gen/trading_pb";
import { MarketService } from "../gen/market_pb";
import { StreamService } from "../gen/stream_pb";
import { StrategyService } from "../gen/strategy_pb";
import { AIService } from "../gen/ai_pb";
import { SystemAIService } from "../gen/system_ai_pb";
import { AIPrimaryService } from "../gen/ai_primary_pb";
import { CodeAssistService } from "../gen/code_assist_pb";
import { AdminUserService } from "../gen/admin_user_service_pb";
import { AdminAccountService } from "../gen/admin_account_service_pb";
import { AdminTradingService } from "../gen/admin_trading_service_pb";
import { AdminConfigService } from "../gen/admin_config_service_pb";
import { AdminLogService } from "../gen/admin_log_service_pb";
import { AdminSystemService } from "../gen/admin_system_service_pb";
import { AnalyticsService } from "../gen/analytics_pb";
import { LogService } from "../gen/log_pb";
import { PythonStrategyService } from "../gen/python_strategy_pb";
import { BacktestDatasetService } from "../gen/backtest_dataset_pb";
import { BacktestTradesService } from "../gen/backtest_trades_pb";
import { DebateV2Service } from "../gen/debate_v2_service_pb";
import { DebateV2StreamService } from "../gen/debate_v2_stream_pb";
import { ObjectiveScoreService } from "../gen/objective_score_pb";
import { IndicatorCatalogService } from "../gen/indicator_catalog_pb";
import { EconomicDataService } from "../gen/economic_data_pb";
import { ScheduleHealthService } from "../gen/schedule_health_pb";
import { StrategyExperimentService } from "../gen/strategy_experiment_pb";
import { MarketRegimeService } from "../gen/market_regime_pb";
import { StrategyAssetService } from "../gen/strategy_asset_pb";
import { JobService } from "../gen/job_pb";
import { streamTransport, transport } from "./transport";

export const authClient = createClient(AuthService, transport);
export const accountClient = createClient(AccountService, transport);
export const tradingClient = createClient(TradingService, transport);
export const marketClient = createClient(MarketService, transport);
export const streamClient = createClient(StreamService, streamTransport);
export const strategyClient = createClient(StrategyService, transport);
export const aiClient = createClient(AIService, transport);
export const systemAIClient = createClient(SystemAIService, transport);
export const aiPrimaryClient = createClient(AIPrimaryService, transport);
export const codeAssistClient = createClient(CodeAssistService, transport);
export const adminUserClient = createClient(AdminUserService, transport);
export const adminAccountClient = createClient(AdminAccountService, transport);
export const adminTradingClient = createClient(AdminTradingService, transport);
export const adminConfigClient = createClient(AdminConfigService, transport);
export const adminLogClient = createClient(AdminLogService, transport);
export const adminSystemClient = createClient(AdminSystemService, transport);
export const analyticsClient = createClient(AnalyticsService, transport);
export const pythonStrategyClient = createClient(
  PythonStrategyService,
  transport,
);
export const pythonStrategyStreamClient = createClient(
  PythonStrategyService,
  streamTransport,
);
export const backtestDatasetClient = createClient(
  BacktestDatasetService,
  transport,
);
export const backtestTradesClient = createClient(
  BacktestTradesService,
  transport,
);
export const debateV2Client = createClient(DebateV2Service, transport);
export const debateV2StreamClient = createClient(DebateV2StreamService, streamTransport);
export const objectiveScoreClient = createClient(
  ObjectiveScoreService,
  transport,
);
export const indicatorCatalogClient = createClient(
  IndicatorCatalogService,
  transport,
);
export const economicDataClient = createClient(EconomicDataService, transport);
export const logClient = createClient(LogService, transport);
export const scheduleHealthClient = createClient(
  ScheduleHealthService,
  transport,
);
export const strategyExperimentClient = createClient(
  StrategyExperimentService,
  transport,
);
export const marketRegimeClient = createClient(MarketRegimeService, transport);
export const strategyAssetClient = createClient(StrategyAssetService, transport);
export const jobClient = createClient(JobService, transport);
export const jobStreamClient = createClient(JobService, streamTransport);
