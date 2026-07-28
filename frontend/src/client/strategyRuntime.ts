import { strategyRuntimeClient, strategyRuntimeStreamClient } from './connect';
import { create } from '@bufbuild/protobuf';
import type { Timestamp } from '@bufbuild/protobuf/wkt';
import { BacktestRunMode } from '../gen/ant/v1/backtest_run_pb';
import { TradeDirection } from '../gen/ant/v1/backtest_execution_config_pb';
import {
  StartBacktestRunRequestSchema,
} from '../gen/ant/v1/backtest_run_start_pb';
import {
  GetBacktestRunRequestSchema,
  ListBacktestRunsRequestSchema,
  WatchBacktestRunRequestSchema,
  type BacktestRunUpdate,
} from '../gen/ant/v1/backtest_run_query_pb';
import {
  CancelBacktestRunRequestSchema,
  DeleteBacktestRunRequestSchema,
  DeleteBacktestRunsRequestSchema,
} from '../gen/ant/v1/backtest_run_control_pb';
import type { ExecuteStrategyResponse, ValidateStrategyResponse, BacktestStrategyResponse, GetStrategyTemplatesResponse } from '../gen/ant/v1/strategy_runtime_pb';

export interface ExecuteStrategyResult {
  success: boolean;
  signal?: any;
  logs: string[];
  error: string;
}

export interface ValidateStrategyResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
}

export interface BacktestResult {
  success: boolean;
  metrics?: any;
  equityCurve: number[];
  error: string;
}

export interface StrategyTemplateInfo {
  name: string;
  description: string;
  code: string;
}

const strategyRuntimeService = strategyRuntimeClient;
const strategyRuntimeStreamService = strategyRuntimeStreamClient;

function toTimestamp(d?: Date): Timestamp | undefined {
  if (!d) return undefined;
  const ms = d.getTime();
  const seconds = Math.floor(ms / 1000);
  const nanos = (ms % 1000) * 1_000_000;
  return { seconds: BigInt(seconds), nanos } as unknown as Timestamp;
}

export const strategyRuntimeApi = {
  execute: async (params: {
    code: string;
    accountId: string;
    symbol: string;
    timeframe?: string
  }): Promise<ExecuteStrategyResult> => {
    const msg = await strategyRuntimeService.execute({
      code: params.code,
      accountId: params.accountId,
      symbol: params.symbol,
      timeframe: params.timeframe || '',
    }) as ExecuteStrategyResponse;
    return {
      success: msg.success,
      signal: msg.signal,
      logs: msg.logs || [],
      error: msg.error,
    };
  },

  backtest: async (params: {
    code: string;
    accountId: string;
    symbol: string;
    timeframe: string;
    initialCapital?: number
  }): Promise<BacktestResult> => {
    const msg = await strategyRuntimeService.backtest(
      {
        code: params.code,
        accountId: params.accountId,
        symbol: params.symbol,
        timeframe: params.timeframe,
        initialCapital: params.initialCapital || 10000,
      },
      {
        timeoutMs: 300_000,
      },
    ) as BacktestStrategyResponse;
    return {
      success: msg.success,
      metrics: msg.metrics,
      equityCurve: msg.equityCurve || [],
      error: msg.error,
    };
  },

  getTemplates: async (): Promise<StrategyTemplateInfo[]> => {
    const msg = await strategyRuntimeService.getTemplates({}) as GetStrategyTemplatesResponse;
    return (msg.templates || []).map((t) => ({
      name: t.name,
      description: t.description,
      code: t.code,
    }));
  },

  validate: async (code: string): Promise<ValidateStrategyResult> => {
    const msg = await strategyRuntimeService.validate({ code }) as ValidateStrategyResponse;
    return {
      valid: msg.valid || false,
      errors: msg.errors || [],
      warnings: msg.warnings || [],
    };
  },

  startBacktestRun: async (params: {
    code: string;
    accountId: string;
    symbol: string;
    timeframe: string;
    initialCapital?: number;
    mode: 'KLINE_RANGE' | 'DATASET';
    from?: Date;
    to?: Date;
    datasetId?: string;
    templateId?: string;
    templateDraftId?: string;
    extraSymbols?: string[];
    strategyId?: string;
    autoGate?: boolean;
    executionConfig?: {
      commission: number;
      slippage: number;
      leverage: number;
      tradeDirection: 'long' | 'short' | 'both';
      strictMode: boolean;
    };
  }): Promise<{ runId: string }> => {
    const msg = create(StartBacktestRunRequestSchema, {
      code: params.code,
      accountId: params.accountId,
      symbol: params.symbol,
      timeframe: params.timeframe,
      initialCapital: String(params.initialCapital ?? 10000),
      mode:
        params.mode === 'DATASET'
          ? BacktestRunMode.DATASET
          : BacktestRunMode.KLINE_RANGE,
      from: params.mode === 'KLINE_RANGE' ? toTimestamp(params.from) : undefined,
      to: params.mode === 'KLINE_RANGE' ? toTimestamp(params.to) : undefined,
      datasetId: params.mode === 'DATASET' ? params.datasetId : undefined,
      templateId: params.templateId,
      templateDraftId: params.templateDraftId,
      extraSymbols: (params.extraSymbols ?? []).filter((s) => !!s && s !== params.symbol),
      strategyId: params.strategyId,
      autoGate: params.autoGate ?? false,
      executionConfig: params.executionConfig ? {
        commission: params.executionConfig.commission,
        slippage: params.executionConfig.slippage,
        leverage: params.executionConfig.leverage,
        tradeDirection:
          params.executionConfig.tradeDirection === 'long' ? TradeDirection.LONG
          : params.executionConfig.tradeDirection === 'short' ? TradeDirection.SHORT
          : TradeDirection.BOTH,
        strictMode: params.executionConfig.strictMode,
      } : undefined,
    });
    const resp = await strategyRuntimeService.startBacktestRun(msg);
    return { runId: resp.runId };
  },

  getBacktestRun: async (runId: string) => {
    const msg = create(GetBacktestRunRequestSchema, { runId });
    return (await strategyRuntimeService.getBacktestRun(msg));
  },

  listBacktestRuns: async (params: { accountId?: string; templateId?: string; limit?: number; offset?: number }) => {
    const msg = create(ListBacktestRunsRequestSchema, {
      accountId: params.accountId,
      templateId: params.templateId,
      limit: params.limit ?? 50,
      offset: params.offset ?? 0,
    });
    return (await strategyRuntimeService.listBacktestRuns(msg));
  },

  cancelBacktestRun: async (runId: string) => {
    const msg = create(CancelBacktestRunRequestSchema, { runId });
    return (await strategyRuntimeService.cancelBacktestRun(msg));
  },

  deleteBacktestRun: async (runId: string) => {
    const msg = create(DeleteBacktestRunRequestSchema, { runId });
    return (await strategyRuntimeService.deleteBacktestRun(msg));
  },

  deleteBacktestRuns: async (runIds: string[]) => {
    const msg = create(DeleteBacktestRunsRequestSchema, { runIds });
    return (await strategyRuntimeService.deleteBacktestRuns(msg));
  },

  watchBacktestRun: (runId: string, onUpdate: (u: BacktestRunUpdate) => void, onError?: (e: unknown) => void) => {
    const abortController = new AbortController();
    (async () => {
      try {
        const msg = create(WatchBacktestRunRequestSchema, { runId });
        const stream = strategyRuntimeStreamService.watchBacktestRun(msg, { signal: abortController.signal });
        for await (const u of stream) {
          onUpdate(u);
        }
      } catch (e) {
        const errorStr = String(e);
        if ((e as { name?: string })?.name === 'AbortError' || errorStr.includes('canceled') || errorStr.includes('aborted')) {
          return;
        }
        onError?.(e);
      }
    })();
    return () => abortController.abort();
  },
};
