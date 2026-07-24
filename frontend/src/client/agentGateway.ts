import { agentGatewayClient } from "./connect";
import { create } from "@bufbuild/protobuf";
import { SubmitStrategyRequestSchema, AgentBacktestConfigSchema, SubmitMode } from "../gen/ant/v1/agent_gateway_pb";

export async function submitStrategy(input: {
  sourceCode: string;
  language: string;
  symbol: string;
  timeframe: string;
  startDateMs?: number;
  endDateMs?: number;
}) {
  const now = Date.now();
  const btCfg = create(AgentBacktestConfigSchema, {
    symbol: input.symbol,
    timeframe: input.timeframe,
    startDateMs: BigInt(input.startDateMs || now - 365 * 24 * 60 * 60 * 1000),
    endDateMs: BigInt(input.endDateMs || now),
    initialCapital: "10000",
    commission: "0.0003",
    slippage: "0.00001",
    leverage: "100",
    strictMode: true,
  });

  const req = create(SubmitStrategyRequestSchema, {
    sourceCode: input.sourceCode,
    language: input.language,
    params: {},
    backtestConfig: btCfg,
    mode: SubmitMode.SYNC,
  });

  const resp = await agentGatewayClient.submitStrategy(req);
  return resp;
}
