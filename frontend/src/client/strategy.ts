import { timestampDate } from '@bufbuild/protobuf/wkt';
import { strategyClient } from './connect';
import type { StrategyTemplate, TemplateParameter, TemplateI18n } from '../gen/ant/v1/strategy_template_entity_pb';

export type { StrategyTemplate, TemplateParameter, TemplateI18n } from '../gen/ant/v1/strategy_template_entity_pb';
export type { StrategyCard } from '../gen/ant/v1/strategy_template_entity_pb';

export interface RunBacktestResult {
  success: boolean;
  metrics?: { totalReturn: number; annualReturn: number; sharpeRatio: number; maxDrawdown: number; winRate: number; profitFactor: number; totalTrades: number; averageProfit: number; averageLoss: number };
  riskScore?: number;
  riskLevel?: string;
  riskReasons?: string[];
  riskWarnings?: string[];
  isReliable?: boolean;
  error?: string;
}

export interface ExecuteSignalResult {
  ticket: bigint;
  symbol: string;
  type: string;
  volume: number;
  price: number;
  executedAt?: Date;
}

export const strategyApi = {
  watchSchedules: (signal?: AbortSignal) => strategyClient.watchSchedules({}, { signal }),
  listTemplates: async () => {
    const response = await strategyClient.listTemplates({});
    return response.templates;
  },

  listStrategyCards: async (params?: { filter?: string; sort?: string; search?: string; limit?: number; offset?: number }) => {
    const response = await strategyClient.listStrategyCards({
      filter: params?.filter ?? '',
      sort: params?.sort ?? '',
      search: params?.search ?? '',
      limit: params?.limit ?? 0,
      offset: params?.offset ?? 0,
    });
    return { cards: response.cards, total: response.total };
  },

  getTemplate: async (id: string): Promise<StrategyTemplate> => {
    return await strategyClient.getTemplate({ id });
  },

  createTemplate: async (params: {
    name: string;
    description: string;
    code: string;
    parameters?: TemplateParameter[];
    isPublic?: boolean;
    tags?: string[];
    i18n?: TemplateI18n;
  }) => {
    return await strategyClient.createTemplate({
      name: params.name,
      description: params.description,
      code: params.code,
      parameters: params.parameters || [],
      isPublic: params.isPublic || false,
      tags: params.tags || [],
      i18n: params.i18n,
    });
  },

  updateTemplate: async (params: {
    id: string;
    name?: string;
    description?: string;
    code?: string;
    parameters?: TemplateParameter[];
    isPublic?: boolean;
    tags?: string[];
    i18n?: TemplateI18n;
  }) => {
    return await strategyClient.updateTemplate({
      id: params.id,
      name: params.name,
      description: params.description,
      code: params.code,
      parameters: params.parameters,
      isPublic: params.isPublic,
      tags: params.tags,
      i18n: params.i18n,
    });
  },

  deleteTemplate: async (id: string) => {
    await strategyClient.deleteTemplate({ id });
  },

  createTemplateDraft: async (params: { name: string }) => {
    return await strategyClient.createTemplateDraft({ name: params.name });
  },

  updateTemplateDraft: async (params: {
    id: string;
    name?: string;
    description?: string;
    code?: string;
    parameters?: TemplateParameter[];
    tags?: string[];
  }) => {
    return await strategyClient.updateTemplateDraft({
      id: params.id,
      name: params.name,
      description: params.description,
      code: params.code,
      parameters: params.parameters || [],
      tags: params.tags || [],
    });
  },

  publishTemplateDraft: async (id: string) => {
    return await strategyClient.publishTemplateDraft({ id });
  },

  cancelTemplateDraft: async (id: string) => {
    await strategyClient.cancelTemplateDraft({ id });
  },

  listSchedules: async () => {
    const response = await strategyClient.listSchedules({});
    return response.schedules;
  },

  getSchedule: async (id: string) => {
    return await strategyClient.getSchedule({ id });
  },

  listSignals: async (accountId?: string, status?: string) => {
    const response = await strategyClient.listSignals({
      accountId: accountId || '',
      status: status || '',
    });
    return response.signals;
  },

  executeSignal: async (signalId: string): Promise<ExecuteSignalResult> => {
    const response = await strategyClient.executeSignal({ signalId });
    return {
      ticket: response.ticket,
      symbol: response.symbol,
      type: response.type,
      volume: Number(response.volume) || 0,
      price: Number(response.price) || 0,
      executedAt: response.executedAt ? timestampDate(response.executedAt) : undefined,
    };
  },

  confirmSignal: async (signalId: string) => {
    await strategyClient.confirmSignal({ signalId });
  },

  cancelSignal: async (signalId: string) => {
    await strategyClient.cancelSignal({ signalId });
  },
};

// ── Strategy Import (via StrategyRuntimeService) ────────────────────

import { strategyRuntimeClient, strategyRuntimeStreamClient } from './connect';

export const strategyImportApi = {
  analyzeCode: async (params: {
    sourceCode: string;
    sourceName: string;
    sourceLang?: string;
  }) => {
    return await strategyRuntimeClient.analyzeImportCode({
      sourceCode: params.sourceCode,
      sourceName: params.sourceName,
      sourceLang: params.sourceLang || 'mql4',
    });
  },

  getImportedStrategy: async (strategyId: string) => {
    return await strategyRuntimeClient.getImportedStrategy({ strategyId });
  },
};

// ── Strategy Run lifecycle (Phase 1) ─────────────────────────────────

export const strategyRunsApi = {
  listRuns: async (params?: { accountId?: string; limit?: number; offset?: number }) => {
    const r = await strategyRuntimeClient.listStrategyRuns({
      accountId: params?.accountId || '',
      limit: params?.limit ?? 50,
      offset: params?.offset ?? 0,
    });
    return r.runs;
  },

  getRun: async (runId: string) => {
    return await strategyRuntimeClient.getStrategyRun({ runId });
  },
};

// ── Active strategy monitoring + control (Phase 2) ──────────────────

export const strategyActiveApi = {
  start: async (params: {
    accountId: string;
    strategyCode: string;
    symbol: string;
    timeframe: string;
    mode?: string;
    params?: Record<string, string>;
    extraSymbols?: string[];
    strategyId?: string;
  }) => {
    return await strategyRuntimeClient.startStrategy({
      accountId: params.accountId,
      strategyCode: params.strategyCode,
      symbol: params.symbol,
      timeframe: params.timeframe,
      mode: params.mode || 'paper',
      params: params.params || {},
      extraSymbols: params.extraSymbols ?? [],
      strategyId: params.strategyId ?? '',
    });
  },

  listActive: async (accountId?: string) => {
    const r = await strategyRuntimeClient.listActiveStrategies({
      accountId: accountId || '',
    });
    return r.strategies;
  },

  getActive: async (runId: string) => {
    const r = await strategyRuntimeClient.getActiveStrategy({ runId });
    return r.strategy;
  },

  stop: async (runId: string) => {
    return await strategyRuntimeClient.stopStrategy({ runId });
  },

  watchSignals: (runId: string, signal?: AbortSignal) =>
    strategyRuntimeStreamClient.watchStrategySignals({ runId }, { signal }),

  watchActive: (accountId: string, signal?: AbortSignal) =>
    strategyRuntimeStreamClient.watchActiveStrategies({ accountId }, { signal }),
};

export { strategyVersionApi } from './strategy_version';
