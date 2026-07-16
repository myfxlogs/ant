import { strategyPlanClient } from './connect';
import type { AnalyzePlanChunk, ExecutePlanChunk, ConversateChunk, ToolCall, ToolResult, BacktestMetricsMsg } from '../gen/ant/v1/strategy_execution_pb';

export interface PlanCallbacks {
  onDelta: (delta: string) => void;
  onPlan: (plan: string) => void;
  onError: (error: string) => void;
  onDone: () => void;
}

export interface ExecuteCallbacks {
  onPhase: (phase: string) => void;
  onDelta: (delta: string) => void;
  onCode: (code: string) => void;
  onPreviousCode: (code: string) => void;
  onAnalysis?: (text: string) => void;
  onToolCall: (call: ToolCall) => void;
  onToolResult: (result: ToolResult) => void;
  onError: (error: string) => void;
  onDone: () => void;
}

export interface ConversateCallbacks {
  onDelta: (delta: string) => void;
  onPlan: (plan: string) => void;
  onCode: (code: string) => void;
  onPreviousCode: (code: string) => void;
  onToolCall: (call: ToolCall) => void;
  onToolResult: (result: ToolResult) => void;
  onError: (error: string) => void;
  onDone: () => void;
}

export function conversate(
  input: { message: string; conversationId?: string; symbol?: string; timeframe?: string; plan?: string; currentCode?: string; backtestMetrics?: BacktestMetricsMsg },
  callbacks: ConversateCallbacks,
): () => void {
  const abort = new AbortController();
  (async () => {
    try {
      const stream = strategyPlanClient.conversate({
        message: input.message, conversationId: input.conversationId || '',
        symbol: input.symbol || '', timeframe: input.timeframe || '',
        plan: input.plan || '', currentCode: input.currentCode || '',
        backtestMetrics: input.backtestMetrics,
      }, { signal: abort.signal });
      for await (const chunk of stream) {
        if (chunk.delta) callbacks.onDelta(chunk.delta);
        if (chunk.plan) callbacks.onPlan(chunk.plan);
        if (chunk.code) callbacks.onCode(chunk.code);
        if (chunk.previousCode) callbacks.onPreviousCode(chunk.previousCode);
        if (chunk.toolCall) callbacks.onToolCall(chunk.toolCall);
        if (chunk.toolResult) callbacks.onToolResult(chunk.toolResult);
        if (chunk.error) callbacks.onError(chunk.error);
      }
      callbacks.onDone();
    } catch (e: unknown) {
      const s = String(e);
      if ((e as { name?: string })?.name === 'AbortError' || s.includes('canceled')) return;
      callbacks.onError(s);
    }
  })();
  return () => abort.abort();
}

export function analyzePlan(
  input: { message: string; conversationId?: string; symbol?: string; timeframe?: string },
  callbacks: PlanCallbacks,
): () => void {
  const abort = new AbortController();
  (async () => {
    try {
      const stream = strategyPlanClient.analyzePlan({
        message: input.message,
        conversationId: input.conversationId || '',
        symbol: input.symbol || '',
        timeframe: input.timeframe || '',
      }, { signal: abort.signal });
      for await (const chunk of stream) {
        if (chunk.delta) callbacks.onDelta(chunk.delta);
        if (chunk.plan) callbacks.onPlan(chunk.plan);
        if (chunk.error) callbacks.onError(chunk.error);
      }
      callbacks.onDone();
    } catch (e: unknown) {
      const s = String(e);
      if ((e as { name?: string })?.name === 'AbortError' || s.includes('canceled')) return;
      callbacks.onError(s);
    }
  })();
  return () => abort.abort();
}

export function diagnosePlan(
  input: { plan: string; conversationId?: string; feedbackMessage: string; currentCode: string; backtestMetrics?: BacktestMetricsMsg },
  callbacks: PlanCallbacks,
): () => void {
  const abort = new AbortController();
  (async () => {
    try {
      const stream = strategyPlanClient.diagnose({
        plan: input.plan, conversationId: input.conversationId || '',
        feedbackMessage: input.feedbackMessage, currentCode: input.currentCode,
        backtestMetrics: input.backtestMetrics,
      }, { signal: abort.signal });
      for await (const chunk of stream) {
        if (chunk.delta) callbacks.onDelta(chunk.delta);
        if (chunk.plan) callbacks.onPlan(chunk.plan);
        if (chunk.error) callbacks.onError(chunk.error);
      }
      callbacks.onDone();
    } catch (e: unknown) {
      const s = String(e);
      if ((e as { name?: string })?.name === 'AbortError' || s.includes('canceled')) return;
      callbacks.onError(s);
    }
  })();
  return () => abort.abort();
}

export function executePlan(
  input: { plan: string; conversationId?: string; symbol?: string; timeframe?: string; previousCode?: string; feedbackMessage?: string; backtestMetrics?: BacktestMetricsMsg },
  callbacks: ExecuteCallbacks,
): () => void {
  const abort = new AbortController();
  (async () => {
    try {
      const stream = strategyPlanClient.executePlan({
        plan: input.plan,
        conversationId: input.conversationId || '',
        symbol: input.symbol || '',
        timeframe: input.timeframe || '',
        previousCode: input.previousCode || '',
        feedbackMessage: input.feedbackMessage || '',
        backtestMetrics: input.backtestMetrics,
      }, { signal: abort.signal });
      for await (const chunk of stream) {
        handleExecuteChunk(chunk, callbacks);
      }
      callbacks.onDone();
    } catch (e: unknown) {
      const s = String(e);
      if ((e as { name?: string })?.name === 'AbortError' || s.includes('canceled')) return;
      callbacks.onError(s);
    }
  })();
  return () => abort.abort();
}

function handleExecuteChunk(chunk: ExecutePlanChunk, cbs: ExecuteCallbacks): void {
  cbs.onPhase(chunk.phase);
  if (chunk.delta) cbs.onDelta(chunk.delta);
  if (chunk.code) cbs.onCode(chunk.code);
  if (chunk.previousCode) cbs.onPreviousCode(chunk.previousCode);
  if (chunk.toolCall) cbs.onToolCall(chunk.toolCall);
  if (chunk.toolResult) cbs.onToolResult(chunk.toolResult);
  if (chunk.analysis) cbs.onAnalysis?.(chunk.analysis);
  if (chunk.error) cbs.onError(chunk.error);
}
