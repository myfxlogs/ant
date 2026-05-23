import type { PartialMessage } from '@bufbuild/protobuf';
import { create } from '@bufbuild/protobuf';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import { strategyClient } from './connect';
import type { BacktestMetrics } from '../gen/common_pb';
import { ScheduleConfigSchema, type ScheduleConfig } from '../gen/strategy_schedule_entity_pb';
import type { TemplateParameter } from '../gen/strategy_template_entity_pb';

export type { StrategyTemplate, TemplateParameter } from '../gen/strategy_template_entity_pb';
export type { StrategySchedule, ScheduleConfig } from '../gen/strategy_schedule_entity_pb';
export type { StrategySignal } from '../gen/strategy_signal_messages_pb';
export type { BacktestMetrics };

export interface RunBacktestResult {
  success: boolean;
  metrics?: BacktestMetrics;
  riskScore: number;
  riskLevel: string;
  riskReasons: string[];
  riskWarnings: string[];
  isReliable: boolean;
  error: string;
}

export interface ExecuteSignalResult {
  ticket: bigint;
  symbol: string;
  type: string;
  volume: number;
  price: number;
  executedAt?: Date;
}

function toBigInt(v: unknown): bigint {
  if (typeof v === 'bigint') return v;
  if (typeof v === 'number' && Number.isFinite(v)) return BigInt(Math.floor(v));
  if (typeof v === 'string' && v.trim() !== '') {
    try {
      return BigInt(v);
    } catch {
      return 0n;
    }
  }
  return 0n;
}

function normalizeScheduleConfig(cfg: PartialMessage<ScheduleConfig> | undefined): ScheduleConfig {
  if (!cfg) {
    return create(ScheduleConfigSchema, {
      cronExpression: '',
      intervalMs: 0n,
      eventTrigger: '',
      triggerMode: '',
      stableOverrideIntervalMs: 0n,
      hfCooldownMs: 0n,
    });
  }
  return create(ScheduleConfigSchema, {
    cronExpression: String(cfg.cronExpression ?? ''),
    intervalMs: toBigInt(cfg.intervalMs),
    eventTrigger: String(cfg.eventTrigger ?? ''),
    triggerMode: String(cfg.triggerMode ?? ''),
    stableOverrideIntervalMs: toBigInt(cfg.stableOverrideIntervalMs),
    hfCooldownMs: toBigInt(cfg.hfCooldownMs),
  });
}

export const strategyApi = {
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
  }) => {
    return await strategyClient.createTemplate({
      name: params.name,
      description: params.description,
      code: params.code,
      parameters: params.parameters || [],
      isPublic: params.isPublic || false,
      tags: params.tags || [],
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
  }) => {
    return await strategyClient.updateTemplate({
      id: params.id,
      name: params.name,
      description: params.description,
      code: params.code,
      parameters: params.parameters,
      isPublic: params.isPublic,
      tags: params.tags,
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

  createSchedule: async (params: {
    templateId: string;
    accountId: string;
    name: string;
    symbol: string;
    timeframe: string;
    parameters?: Record<string, string>;
    scheduleType: string;
    scheduleConfig?: PartialMessage<ScheduleConfig>;
  }) => {
    const scheduleConfig = normalizeScheduleConfig(params.scheduleConfig);
    return await strategyClient.createSchedule({
      templateId: params.templateId,
      accountId: params.accountId,
      name: params.name,
      symbol: params.symbol,
      timeframe: params.timeframe,
      parameters: params.parameters || {},
      scheduleType: params.scheduleType,
      scheduleConfig,
    });
  },

  updateSchedule: async (params: {
    id: string;
    name?: string;
    symbol?: string;
    timeframe?: string;
    parameters?: Record<string, string>;
    scheduleType?: string;
    scheduleConfig?: PartialMessage<ScheduleConfig>;
  }) => {
    const scheduleConfig = params.scheduleConfig ? normalizeScheduleConfig(params.scheduleConfig) : undefined;
    return await strategyClient.updateSchedule({
      id: params.id,
      name: params.name,
      symbol: params.symbol,
      timeframe: params.timeframe,
      parameters: params.parameters,
      scheduleType: params.scheduleType,
      scheduleConfig,
    });
  },

  deleteSchedule: async (id: string) => {
    await strategyClient.deleteSchedule({ id });
  },

  toggleSchedule: async (id: string, active: boolean) => {
    return await strategyClient.toggleSchedule({ id, active });
  },

  runBacktest: async (params: {
    templateId: string;
    accountId: string;
    symbol: string;
    timeframe: string;
    parameters?: Record<string, string>;
    initialCapital?: number;
  }): Promise<RunBacktestResult> => {
    const response = await strategyClient.runBacktest({
      templateId: params.templateId,
      accountId: params.accountId,
      symbol: params.symbol,
      timeframe: params.timeframe,
      parameters: params.parameters || {},
      initialCapital: params.initialCapital || 10000,
    });
    return {
      success: response.success,
      metrics: response.metrics,
      riskScore: response.riskScore,
      riskLevel: response.riskLevel,
      riskReasons: response.riskReasons,
      riskWarnings: response.riskWarnings,
      isReliable: response.isReliable,
      error: response.error,
    };
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

export const strategyTemplateApi = {
  list: strategyApi.listTemplates,
  get: strategyApi.getTemplate,
  create: strategyApi.createTemplate,
  update: strategyApi.updateTemplate,
  delete: strategyApi.deleteTemplate,
};

export const strategyScheduleV2Api = {
  list: strategyApi.listSchedules,
  get: strategyApi.getSchedule,
  create: strategyApi.createSchedule,
  update: strategyApi.updateSchedule,
  delete: strategyApi.deleteSchedule,
  toggle: strategyApi.toggleSchedule,
  runBacktest: strategyApi.runBacktest,
};

export interface CreateTemplateRequest {
  name: string;
  description: string;
  code: string;
  parameters?: PartialMessage<TemplateParameter>[];
  isPublic?: boolean;
  tags?: string[];
}
