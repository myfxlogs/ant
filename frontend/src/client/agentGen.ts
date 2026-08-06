import { agentGatewayClient } from './connect';
import type { AgentGenerateStrategyChunk, StrategyPlan, BacktestRunSummary } from '../gen/ant/v1/agent_gateway_pb';
import { AgentGenerateStrategyRequestSchema } from '../gen/ant/v1/agent_gateway_pb';
import { create } from '@bufbuild/protobuf';
import i18n from '@/i18n';

export interface BacktestSummary {
  totalReturn?: number;
  maxDrawdown?: number;
  sharpeRatio?: number;
  winRate?: number;
  totalTrades?: number;
  templateName?: string;
  startedAt?: string;
}

export interface AgentGenInput {
  message: string;
  symbol?: string;
  timeframe?: string;
  params?: Record<string, string>;
  planMode?: string;           // "plan" | "generate"
  planFeedback?: string;       // user modification feedback
  confirmedPlan?: StrategyPlan; // confirmed plan for code generation
  conversationId?: string;     // multi-turn conversation session ID
  accountId?: string;          // selected MT account for workspace context
  currentCode?: string;        // user's current strategy code for AI modification
  lastBacktest?: BacktestSummary; // latest backtest metrics for AI context
  recentBacktests?: BacktestSummary[]; // recent backtest runs for AI context
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
  onReasoning: (reasoning: string) => void;
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
  onDone?: () => void;
}

function buildAgentRequest(input: AgentGenInput) {
  return create(AgentGenerateStrategyRequestSchema, {
    message: input.message,
    symbol: input.symbol || '',
    timeframe: input.timeframe || '',
    params: input.params || {},
    planMode: input.planMode || '',
    planFeedback: input.planFeedback || '',
    confirmedPlan: input.confirmedPlan,
    conversationId: input.conversationId || '',
    accountId: input.accountId || '',
    currentCode: input.currentCode || '',
    locale: i18n.language || 'en',
    backtestConfig: input.backtestConfig ? buildBacktestConfig(input.backtestConfig) : undefined,
    lastBacktest: (input.lastBacktest as BacktestRunSummary | undefined) || undefined,
    recentBacktests: (input.recentBacktests as BacktestRunSummary[] | undefined) || undefined,
  });
}

function buildBacktestConfig(bc: NonNullable<AgentGenInput['backtestConfig']>) {
  return {
    symbol: bc.symbol || '',
    timeframe: bc.timeframe || '',
    startDateMs: bc.startDateMs || 0n,
    endDateMs: bc.endDateMs || 0n,
    initialCapital: bc.initialCapital || '',
    commission: bc.commission || '',
    slippage: bc.slippage || '',
    leverage: bc.leverage || '',
    strictMode: bc.strictMode || false,
  };
}

export function agentGenerateStrategyStream(
  input: AgentGenInput,
  callbacks: AgentGenCallbacks,
): () => void {
  const abortController = new AbortController();

  (async () => {
    try {
      const req = buildAgentRequest(input);
      const stream = agentGatewayClient.generateStrategy(req, {
        signal: abortController.signal,
      });

      for await (const chunk of stream) {
        handleAgentChunk(chunk, callbacks);
      }
      callbacks.onDone?.();
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
  const dispatch: Array<[unknown, ((v: never) => void) | undefined]> = [
    [chunk.delta, cbs.onDelta as ((v: never) => void) | undefined],
    [chunk.reasoning, cbs.onReasoning as ((v: never) => void) | undefined],
    [chunk.pythonSource, cbs.onPythonSource as ((v: never) => void) | undefined],
    [chunk.compileError, cbs.onCompileError as ((v: never) => void) | undefined],
    [chunk.backtestError, cbs.onBacktestError as ((v: never) => void) | undefined],
    [chunk.coverageScore, cbs.onCoverageScore as ((v: never) => void) | undefined],
    [chunk.result, cbs.onResult as ((v: never) => void) | undefined],
    [chunk.profile, cbs.onProfile as ((v: never) => void) | undefined],
    [chunk.analysis, cbs.onAnalysis as ((v: never) => void) | undefined],
    [chunk.attempts, cbs.onAttempts as ((v: never) => void) | undefined],
    [chunk.error, cbs.onError as ((v: never) => void) | undefined],
    [chunk.plan, cbs.onPlan as ((v: never) => void) | undefined],
  ];
  for (const [val, fn] of dispatch) {
    if (val && fn) fn(val as never);
  }
}
