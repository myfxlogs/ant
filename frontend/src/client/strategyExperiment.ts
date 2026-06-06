import { create } from '@bufbuild/protobuf';
import { strategyExperimentClient } from './connect';
import { SubmitStrategyExperimentRequestSchema } from '../gen/ant/v1/strategy_experiment_pb';
import type { StrategyExperiment, StrategyExperimentCandidate } from '../gen/ant/v1/strategy_experiment_pb';

export type { StrategyExperiment, StrategyExperimentCandidate };

type SubmitStrategyExperimentParams = {
  baseTemplateId: string;
  parameterSpace: Record<string, unknown>;
  searchMethod?: string;
  maxCandidates?: number;
  objective?: string;
  strategyCode?: string;
  symbol?: string;
  timeframe?: string;
  fromTsUnixMs?: bigint;
  toTsUnixMs?: bigint;
};

export const strategyExperimentApi = {
  submit: (params: SubmitStrategyExperimentParams) =>
    strategyExperimentClient.submitStrategyExperiment(
      create(SubmitStrategyExperimentRequestSchema, {
        baseTemplateId: params.baseTemplateId,
        parameterSpace: params.parameterSpace,
        searchMethod: params.searchMethod ?? 'grid',
        maxCandidates: params.maxCandidates ?? 12,
        objective: params.objective ?? 'balanced',
        idempotencyKey: `ui-${Date.now()}`,
        strategyCode: params.strategyCode ?? '',
        symbol: params.symbol ?? '',
        timeframe: params.timeframe ?? '',
        fromTsUnixMs: params.fromTsUnixMs ?? 0n,
        toTsUnixMs: params.toTsUnixMs ?? 0n,
      }),
    ),

  list: async () => {
    const res = await strategyExperimentClient.listStrategyExperiments({ limit: 50, offset: 0 });
    return res.experiments;
  },

  get: (experimentId: string) => strategyExperimentClient.getStrategyExperiment({ experimentId }),

  cancel: (experimentId: string) => strategyExperimentClient.cancelStrategyExperiment({ experimentId }),

  listCandidates: async (experimentId: string) => {
    const res = await strategyExperimentClient.listExperimentCandidates({ experimentId });
    return res.candidates;
  },

  promoteCandidateToDraft: (candidateId: string, name: string) =>
    strategyExperimentClient.promoteCandidateToDraft({ candidateId, name }),

  // SSE streaming — push-first architecture, replaces polling.
  // Returns { stream, abort } so callers can clean up on unmount.
  watchExperiment: (experimentId: string) => {
    const abortController = new AbortController();
    const stream = strategyExperimentClient.watchExperiment(
      { experimentId },
      { signal: abortController.signal },
    );
    return {
      stream,
      abort: () => abortController.abort(),
    };
  },
};
