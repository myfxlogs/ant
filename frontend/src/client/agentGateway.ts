import { agentGatewayClient } from "./connect";
import type { SubmitStrategyRequest, AgentBacktestConfig } from "../gen/ant/v1/agent_gateway_pb";
import { SubmitMode } from "../gen/ant/v1/agent_gateway_pb";

export interface SubmitStrategyInput {
  sourceCode: string;
  language: string;
  params?: Record<string, string>;
  backtestConfig: {
    symbol: string;
    timeframe: string;
    startDateMs: number;
    endDateMs: number;
    initialCapital?: string;
    commission?: string;
    slippage?: string;
    leverage?: string;
    strictMode?: boolean;
  };
}

export async function submitStrategy(input: SubmitStrategyInput) {
  const btCfg = new AgentBacktestConfig({
    symbol: input.backtestConfig.symbol,
    timeframe: input.backtestConfig.timeframe,
    startDateMs: input.backtestConfig.startDateMs,
    endDateMs: input.backtestConfig.endDateMs,
    initialCapital: input.backtestConfig.initialCapital || "10000",
    commission: input.backtestConfig.commission || "0.0003",
    slippage: input.backtestConfig.slippage || "0.00001",
    leverage: input.backtestConfig.leverage || "100",
    strictMode: input.backtestConfig.strictMode ?? true,
  });

  const req = new SubmitStrategyRequest({
    sourceCode: input.sourceCode,
    language: input.language,
    params: input.params || {},
    backtestConfig: btCfg,
    mode: SubmitMode.SUBMIT_SYNC,
  });

  const resp = await agentGatewayClient.submitStrategy(req);
  return resp;
}
