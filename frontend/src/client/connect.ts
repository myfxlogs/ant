import { createClient, ConnectError, Code } from "@connectrpc/connect";
import { AuthService } from "../gen/ant/v1/auth_pb";
import { AccountService } from "../gen/ant/v1/account_pb";
import { MarketService } from "../gen/ant/v1/market_service_pb";
import { StreamService } from "../gen/ant/v1/stream_pb";
import { StrategyService } from "../gen/ant/v1/strategy_pb";
import { AIService } from "../gen/ant/v1/ai_pb";
import { SystemAIService } from "../gen/ant/v1/system_ai_pb";
import { AIPrimaryService } from "../gen/ant/v1/ai_primary_pb";
import { CodeAssistService } from "../gen/ant/v1/code_assist_pb";
import { LogService } from "../gen/ant/v1/log_pb";
import { PythonStrategyService } from "../gen/ant/v1/python_strategy_pb";
import { BacktestTradesService } from "../gen/ant/v1/backtest_trades_pb";
import { MtHubService } from "../gen/ant/v1/mthub_service_pb";
import { DebateV2Service } from "../gen/ant/v1/debate_v2_service_pb";
import { DebateV2StreamService } from "../gen/ant/v1/debate_v2_stream_pb";
import { EconomicDataService } from "../gen/ant/v1/economic_data_pb";
import { StrategyExperimentService } from "../gen/ant/v1/strategy_experiment_pb";
import { StrategyAssetService } from "../gen/ant/v1/strategy_asset_pb";
import { AdminTradingService } from "../gen/ant/v1/admin_trading_pb";
import { AdminConfigService } from "../gen/ant/v1/admin_config_pb";
import { AdminLogService } from "../gen/ant/v1/admin_log_pb";
import { AdminAccountService } from "../gen/ant/v1/admin_account_pb";
import { AdminUserService } from "../gen/ant/v1/admin_user_pb";
import { AdminSystemService } from "../gen/ant/v1/admin_system_pb";
import { AnalyticsService } from "../gen/ant/v1/analytics_pb";
import { MarketRegimeService } from "../gen/ant/v1/market_regime_pb";
import { MarketplaceService } from "../gen/ant/v1/marketplace_service_pb";
import { JobService } from "../gen/ant/v1/job_pb";
import { ScheduleHealthService } from "../gen/ant/v1/schedule_health_pb";
import { streamTransport, transport } from "./transport";

// Returns a Proxy client that throws for every method call.
// Used for services whose backend handlers are not yet implemented.
// Stubs must NEVER silently succeed — that hides missing backend implementations.
function createStubClient(): any {
  return new Proxy({}, {
    get(_target, methodName: string) {
      if (methodName === "then") return undefined;
      return async () => {
        throw new Error(
          `[stub] ${String(methodName)}() — backend not implemented. ` +
          `This is a stub client; the real backend handler is missing.`
        );
      };
    },
  });
}

export const authClient = createClient(AuthService, transport);
export const accountClient = createClient(AccountService, transport);
export const tradingClient = createClient(MtHubService, transport);
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
export const backtestTradesClient = createClient(
  BacktestTradesService,
  transport,
);
export const debateV2Client = createClient(DebateV2Service, transport);
export const debateV2StreamClient = createClient(DebateV2StreamService, streamTransport);
export const scheduleHealthClient = createClient(ScheduleHealthService, transport);
export const economicDataClient = createClient(EconomicDataService, transport);
export const logClient = createClient(LogService, transport);
export const strategyExperimentClient = createClient(
  StrategyExperimentService,
  transport,
);
export const marketRegimeClient = createClient(MarketRegimeService, transport);
export const marketplaceClient = createClient(MarketplaceService, transport);
export const strategyAssetClient = createClient(StrategyAssetService, transport);
export const jobClient = createClient(JobService, transport);
export const jobStreamClient = createClient(JobService, streamTransport);
