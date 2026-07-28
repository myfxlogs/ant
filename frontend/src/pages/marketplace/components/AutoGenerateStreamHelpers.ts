import type { Dispatch, SetStateAction } from 'react';

export type Stage = 'idle' | 'generating' | 'compiling' | 'backtesting' | 'evaluating' | 'publishing' | 'completed' | 'failed';

interface StreamHandlers {
  setStage: Dispatch<SetStateAction<Stage>>;
  setProgress: Dispatch<SetStateAction<number>>;
  setDelta: Dispatch<SetStateAction<string>>;
  setErrorStage: Dispatch<SetStateAction<string>>;
  setErrorDetail: Dispatch<SetStateAction<string>>;
  setRetryable: Dispatch<SetStateAction<boolean>>;
  setResult: Dispatch<SetStateAction<{ strategyId: string; publishId: string; backtest: unknown } | null>>;
  setViolations: Dispatch<SetStateAction<Array<{ metric: string; actual: string | number; threshold: string | number }>>>;
}

export function resetGenerationState(h: StreamHandlers) {
  h.setStage('generating');
  h.setProgress(0);
  h.setDelta('');
  h.setErrorStage('');
  h.setErrorDetail('');
  h.setRetryable(false);
  h.setResult(null);
  h.setViolations([]);
}

export async function processStreamEvent(
  ev: { stage?: string; progress?: number; delta?: string; message?: string; errorStage?: string; errorDetail?: string; retryable?: boolean; strategyId?: string; publishId?: string; backtest?: unknown; violations?: Array<{ metric: string; actual: string | number; threshold: string | number }> },
  h: StreamHandlers,
) {
  const s = (ev.stage || 'generating') as Stage;
  h.setStage(s);
  if (ev.progress) h.setProgress(ev.progress);
  if (ev.delta) h.setDelta(prev => prev + ev.delta);
  if (ev.message) h.setDelta(prev => prev + ev.message + '\n');

  if (s === 'failed') {
    h.setErrorStage(ev.errorStage || '');
    h.setErrorDetail(ev.errorDetail || '');
    h.setRetryable(ev.retryable ?? false);
  } else if (s === 'completed') {
    if (ev.strategyId || ev.publishId) {
      h.setResult({ strategyId: ev.strategyId || '', publishId: ev.publishId || '', backtest: ev.backtest });
    }
    if (ev.violations && ev.violations.length > 0) {
      h.setViolations(ev.violations);
    }
  }
}

export function handleStreamError(e: unknown, h: StreamHandlers) {
  if (e instanceof Error && e.name === 'AbortError') return;
  h.setStage('failed');
  h.setErrorStage('generating');
  h.setErrorDetail(e instanceof Error ? e.message : String(e));
  h.setRetryable(true);
}
