import { strategyExperimentClient } from './connect';
import type { StrategyExperiment, StrategyExperimentCandidate } from '../gen/ant/v1/strategy_experiment_pb';

export type { StrategyExperiment, StrategyExperimentCandidate };

type SubmitStrategyExperimentParams = {
  baseTemplateId: string;
  parameterSpace: Record<string, unknown>;
  searchMethod?: string;
  maxCandidates?: number;
  objective?: string;
};

export const strategyExperimentApi = {
  submit: (params: SubmitStrategyExperimentParams) =>
    strategyExperimentClient.submitStrategyExperiment({
      baseTemplateId: params.baseTemplateId,
      parameterSpace: params.parameterSpace,
      searchMethod: params.searchMethod ?? 'grid',
      maxCandidates: params.maxCandidates ?? 12,
      objective: params.objective ?? 'balanced',
      idempotencyKey: `ui-${Date.now()}`,
    }),

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
};
