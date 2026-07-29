import { gateClient } from './connect';
import { create } from '@bufbuild/protobuf';
import {
  RunGateEvaluationRequestSchema,
  type GateEvaluationUpdate,
} from '../gen/ant/v1/ai_gate_pb';

export interface GateStreamCallbacks {
  onGate: (gate: NonNullable<GateEvaluationUpdate['gate']>) => void;
  onCompleted: (summary: NonNullable<GateEvaluationUpdate['completed']>) => void;
  onError?: (err: unknown) => void;
}

type GateParams = {
  backtestRunId: string; expression?: string; numAttempts?: number;
  paperDays?: number; paperNetPnl?: string; paperNetReturn?: number;
  backtestNetReturn?: number; paperTradeCount?: number;
};

function buildGateRequest(params: GateParams) {
  return create(RunGateEvaluationRequestSchema, {
    backtestRunId: params.backtestRunId,
    expression: params.expression ?? '',
    numAttempts: params.numAttempts ?? 1,
    paperDays: params.paperDays ?? 0,
    paperNetPnl: params.paperNetPnl ?? '',
    paperNetReturn: params.paperNetReturn ?? 0,
    backtestNetReturn: params.backtestNetReturn ?? 0,
    paperTradeCount: params.paperTradeCount ?? 0,
  });
}

export const gateApi = {
  /** Run the 6-gate pipeline against a completed backtest run. Returns an abort function. */
  runEvaluation: (
    params: GateParams,
    callbacks: GateStreamCallbacks,
  ): (() => void) => {
    const abortController = new AbortController();
    (async () => {
      try {
        const msg = buildGateRequest(params);
        const stream = gateClient.runEvaluation(msg, { signal: abortController.signal });
        for await (const u of stream) {
          if (u.gate) callbacks.onGate(u.gate);
          if (u.completed) callbacks.onCompleted(u.completed);
        }
      } catch (e: unknown) {
        const s = String(e);
        if ((e as { name?: string })?.name === 'AbortError' || s.includes('canceled') || s.includes('aborted')) return;
        callbacks.onError?.(e);
      }
    })();
    return () => abortController.abort();
  },
};
