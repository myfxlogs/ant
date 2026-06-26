import type { PartialMessage } from '@bufbuild/protobuf';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import { strategyClient } from './connect';
import type { BacktestMetrics } from '../gen/ant/v1/common_pb';
import type { TemplateParameter } from '../gen/ant/v1/strategy_template_entity_pb';

export type { StrategyTemplate, TemplateParameter } from '../gen/ant/v1/strategy_template_entity_pb';
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
    i18n?: string;
  }) => {
    return await strategyClient.createTemplate({
      name: params.name,
      description: params.description,
      code: params.code,
      parameters: params.parameters || [],
      isPublic: params.isPublic || false,
      tags: params.tags || [],
      i18n: params.i18n || '',
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
    i18n?: string;
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
      volume: response.volume,
      price: response.price,
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

// ── Strategy Import (via PythonStrategyService) ────────────────────

import { pythonStrategyClient } from './connect';

export const strategyImportApi = {
  analyzeCode: async (params: {
    sourceCode: string;
    sourceName: string;
    sourceLang?: string;
  }) => {
    return await pythonStrategyClient.analyzeImportCode({
      sourceCode: params.sourceCode,
      sourceName: params.sourceName,
      sourceLang: params.sourceLang || 'mql4',
    });
  },

  generateCode: async (params: {
    sourceCode: string;
    sourceName: string;
    sourceLang?: string;
  }) => {
    return await pythonStrategyClient.generateImportCode({
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
    return await pythonStrategyClient.importStrategy({
      sourceCode: params.sourceCode,
      sourceName: params.sourceName,
      sourceLang: params.sourceLang || 'mql4',
      workspaceId: params.workspaceId,
    });
  },
};
