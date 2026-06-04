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

export const gateApi = {
  /** Run the 6-gate pipeline against a completed backtest run. Returns an abort function. */
  runEvaluation: (
    params: {
      backtestRunId: string; expression?: string; numAttempts?: number;
      paperDays?: number; paperNetPnl?: number; paperNetReturn?: number;
      backtestNetReturn?: number; paperTradeCount?: number;
    },
    callbacks: GateStreamCallbacks,
  ): (() => void) => {
    const abortController = new AbortController();
    (async () => {
      try {
        const msg = create(RunGateEvaluationRequestSchema, {
          backtestRunId: params.backtestRunId,
          expression: params.expression ?? '',
          numAttempts: params.numAttempts ?? 1,
          paperDays: params.paperDays ?? 0,
          paperNetPnl: params.paperNetPnl ?? 0,
          paperNetReturn: params.paperNetReturn ?? 0,
          backtestNetReturn: params.backtestNetReturn ?? 0,
          paperTradeCount: params.paperTradeCount ?? 0,
        });
        const stream = gateClient.runEvaluation(msg, { signal: abortController.signal });
        for await (const u of stream) {
          if (u.gate) {
            callbacks.onGate(u.gate);
          }
          if (u.completed) {
            callbacks.onCompleted(u.completed);
          }
        }
      } catch (e: unknown) {
        const s = String(e);
        if ((e as { name?: string })?.name === 'AbortError' || s.includes('canceled') || s.includes('aborted')) {
          return;
        }
        callbacks.onError?.(e);
      }
    })();
    return () => abortController.abort();
  },
};
