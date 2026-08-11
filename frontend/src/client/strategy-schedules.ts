import type { PartialMessage } from '@bufbuild/protobuf';
import { create } from '@bufbuild/protobuf';
import { strategyClient } from './connect';
import { ScheduleConfigSchema, type ScheduleConfig } from '../gen/ant/v1/strategy_schedule_entity_pb';
import { strategyApi } from './strategy';
import type { RunBacktestResult } from './strategy';

function toBigInt(v: unknown): bigint {
  if (typeof v === 'bigint') return v;
  if (typeof v === 'number' && Number.isFinite(v)) return BigInt(Math.floor(v));
  if (typeof v === 'string' && v.trim() !== '') {
    try { return BigInt(v); } catch { return 0n; }
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

export const strategyScheduleApi = {
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
    accountId?: string;
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
      accountId: params.accountId,
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
      initialCapital: String(params.initialCapital || 10000),
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
};

export const strategyTemplateApi = {
  list: strategyApi.listTemplates,
  get: strategyApi.getTemplate,
  create: strategyApi.createTemplate,
  update: strategyApi.updateTemplate,
  delete: strategyApi.deleteTemplate,
};

export const strategyScheduleV2Api = {
  watch: (signal?: AbortSignal) => strategyClient.watchSchedules({}, { signal }),
  list: strategyApi.listSchedules,
  get: strategyApi.getSchedule,
  create: strategyApi.createSchedule,
  update: strategyApi.updateSchedule,
  delete: strategyApi.deleteSchedule,
  toggle: strategyApi.toggleSchedule,
  runBacktest: strategyApi.runBacktest,
};
