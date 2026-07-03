import { agentGatewayClient } from './connect';
import type { AgentGenerateStrategyChunk, StrategyPlan } from '../gen/ant/v1/agent_gateway_pb';
import { AgentGenerateStrategyRequestSchema } from '../gen/ant/v1/agent_gateway_pb';
import { create } from '@bufbuild/protobuf';

export interface AgentGenInput {
  message: string;
  symbol?: string;
  timeframe?: string;
  params?: Record<string, string>;
  planMode?: string;           // "plan" | "generate"
  planFeedback?: string;       // user modification feedback
  confirmedPlan?: StrategyPlan; // confirmed plan for code generation
  backtestConfig?: {
    symbol?: string;
    timeframe?: string;
    startDateMs?: bigint;
    endDateMs?: bigint;
    initialCapital?: string;
    commission?: string;
    slippage?: string;
    leverage?: string;
    strictMode?: boolean;
  };
}

export interface AgentGenCallbacks {
  onPhase: (phase: string) => void;
  onDelta: (delta: string) => void;
  onPythonSource: (code: string) => void;
  onCompileError: (err: string) => void;
  onBacktestError: (err: string) => void;
  onCoverageScore: (score: number) => void;
  onResult: (result: AgentGenerateStrategyChunk['result']) => void;
  onProfile: (profile: AgentGenerateStrategyChunk['profile']) => void;
  onAnalysis: (analysis: AgentGenerateStrategyChunk['analysis']) => void;
  onAttempts: (attempts: number) => void;
  onError: (error: string) => void;
  onPlan: (plan: StrategyPlan) => void;
}

export function agentGenerateStrategyStream(
  input: AgentGenInput,
  callbacks: AgentGenCallbacks,
): () => void {
  const abortController = new AbortController();

  (async () => {
    try {
      const req = create(AgentGenerateStrategyRequestSchema, {
        message: input.message,
        symbol: input.symbol || '',
        timeframe: input.timeframe || '',
        params: input.params || {},
        planMode: input.planMode || '',
        planFeedback: input.planFeedback || '',
        confirmedPlan: input.confirmedPlan,
        backtestConfig: input.backtestConfig ? {
          symbol: input.backtestConfig.symbol || '',
          timeframe: input.backtestConfig.timeframe || '',
          startDateMs: input.backtestConfig.startDateMs || 0n,
          endDateMs: input.backtestConfig.endDateMs || 0n,
          initialCapital: input.backtestConfig.initialCapital || '',
          commission: input.backtestConfig.commission || '',
          slippage: input.backtestConfig.slippage || '',
          leverage: input.backtestConfig.leverage || '',
          strictMode: input.backtestConfig.strictMode || false,
        } : undefined,
      });

      const stream = agentGatewayClient.generateStrategy(req, {
        signal: abortController.signal,
      });

      for await (const chunk of stream) {
        handleAgentChunk(chunk, callbacks);
      }
    } catch (e: unknown) {
      const s = String(e);
      if ((e as { name?: string })?.name === 'AbortError' || s.includes('canceled')) return;
      callbacks.onError(s);
    }
  })();

  return () => abortController.abort();
}

function handleAgentChunk(chunk: AgentGenerateStrategyChunk, cbs: AgentGenCallbacks): void {
  cbs.onPhase(chunk.phase);

  if (chunk.delta) {
    cbs.onDelta(chunk.delta);
  }
  if (chunk.pythonSource) {
    cbs.onPythonSource(chunk.pythonSource);
  }
  if (chunk.compileError) {
    cbs.onCompileError(chunk.compileError);
  }
  if (chunk.backtestError) {
    cbs.onBacktestError(chunk.backtestError);
  }
  if (chunk.coverageScore) {
    cbs.onCoverageScore(chunk.coverageScore);
  }
  if (chunk.result) {
    cbs.onResult(chunk.result);
  }
  if (chunk.profile) {
    cbs.onProfile(chunk.profile);
  }
  if (chunk.analysis) {
    cbs.onAnalysis(chunk.analysis);
  }
  if (chunk.attempts) {
    cbs.onAttempts(chunk.attempts);
  }
  if (chunk.error) {
    cbs.onError(chunk.error);
  }
  if (chunk.plan) {
    cbs.onPlan(chunk.plan);
  }
}
