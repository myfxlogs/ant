import type { PartialMessage } from '@bufbuild/protobuf';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import { strategyClient } from './connect';
import type { BacktestMetrics } from '../gen/ant/v1/common_pb';
import type { TemplateParameter, TemplateI18n } from '../gen/ant/v1/strategy_template_entity_pb';

export type { StrategyTemplate, TemplateParameter, TemplateI18n } from '../gen/ant/v1/strategy_template_entity_pb';
export type { StrategyCard } from '../gen/ant/v1/strategy_template_entity_pb';
export type { StrategySchedule, ScheduleConfig } from '../gen/ant/v1/strategy_schedule_entity_pb';
export type { StrategySignal } from '../gen/ant/v1/strategy_signal_messages_pb';
export type { BacktestMetrics };

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

  getTemplate: async (id: string) => {
    return await strategyClient.getTemplate({ id });
  },

  createTemplate: async (params: {
    name: string;
    description: string;
    code: string;
    parameters?: PartialMessage<TemplateParameter>[];
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
    parameters?: PartialMessage<TemplateParameter>[];
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
    parameters?: PartialMessage<TemplateParameter>[];
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

import { strategyRuntimeClient } from './connect';

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

  importStrategy: async (params: {
    sourceCode: string;
    sourceName: string;
    sourceLang?: string;
    workspaceId?: string;
  }) => {
    return await strategyRuntimeClient.importStrategy({
      sourceCode: params.sourceCode,
      sourceName: params.sourceName,
      sourceLang: params.sourceLang || 'mql4',
      workspaceId: params.workspaceId,
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
  }) => {
    return await strategyRuntimeClient.startStrategy({
      accountId: params.accountId,
      strategyCode: params.strategyCode,
      symbol: params.symbol,
      timeframe: params.timeframe,
      mode: params.mode || 'paper',
      params: params.params || {},
      extraSymbols: params.extraSymbols ?? [],
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

// ── Strategy version history ────────────────────────────────────────

export const strategyVersionApi = {
  list: async (strategyId: string, limit = 50, offset = 0) => {
    const r = await strategyRuntimeClient.listStrategyVersions({ strategyId, limit, offset });
    return { versions: r.versions, total: r.total };
  },

  get: async (strategyId: string, versionNumber: number) => {
    const r = await strategyRuntimeClient.getStrategyVersion({ strategyId, versionNumber });
    return { version: r.version, sourceCode: r.sourceCode };
  },

  rollback: async (strategyId: string, versionNumber: number) => {
    const r = await strategyRuntimeClient.rollbackStrategyVersion({ strategyId, versionNumber });
    return { newVersion: r.newVersion, restoredSourceCode: r.restoredSourceCode };
  },

  diff: async (strategyId: string, fromVersion: number, toVersion: number) => {
    const r = await strategyRuntimeClient.diffStrategyVersions({ strategyId, fromVersion, toVersion });
    return {
      fromVersion: r.fromVersion, fromSourceCode: r.fromSourceCode,
      toVersion: r.toVersion, toSourceCode: r.toSourceCode,
    };
  },

  updateCode: async (strategyId: string, sourceCode: string, changeSummary: string) => {
    const r = await strategyRuntimeClient.updateStrategyCode({ strategyId, sourceCode, changeSummary });
    return { newVersion: r.newVersion };
  },
};
