import { pythonStrategyClient, pythonStrategyStreamClient } from './connect';
import { create } from '@bufbuild/protobuf';
import type { Timestamp } from '@bufbuild/protobuf/wkt';
import { BacktestRunMode } from '../gen/backtest_run_pb';
import {
  StartBacktestRunRequestSchema,
} from '../gen/backtest_run_start_pb';
import {
  GetBacktestRunRequestSchema,
  ListBacktestRunsRequestSchema,
  WatchBacktestRunRequestSchema,
} from '../gen/backtest_run_query_pb';
import {
  CancelBacktestRunRequestSchema,
  DeleteBacktestRunRequestSchema,
} from '../gen/backtest_run_control_pb';

export type { StrategySignal, BacktestMetrics } from '../gen/api_pb';

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

export interface PythonTemplate {
  name: string;
  description: string;
  code: string;
}

const pythonStrategyService = pythonStrategyClient;
const pythonStrategyStreamService = pythonStrategyStreamClient;

function toTimestamp(d?: Date): Timestamp | undefined {
  if (!d) return undefined;
  const ms = d.getTime();
  const seconds = Math.floor(ms / 1000);
  const nanos = (ms % 1000) * 1_000_000;
  return { seconds: BigInt(seconds), nanos } as unknown as Timestamp;
}

export const pythonStrategyApi = {
  execute: async (params: { 
    code: string; 
    accountId: string; 
    symbol: string; 
    timeframe?: string 
  }): Promise<ExecuteStrategyResult> => {
    const response: any = await pythonStrategyService.execute({
      code: params.code,
      accountId: params.accountId,
      symbol: params.symbol,
      timeframe: params.timeframe || '',
    });
    return {
      success: response.success,
      signal: response.signal,
      logs: response.logs || [],
      error: response.error,
    };
  },

  backtest: async (params: { 
    code: string; 
    accountId: string; 
    symbol: string; 
    timeframe: string; 
    initialCapital?: number 
  }): Promise<BacktestResult> => {
    const response: any = await pythonStrategyService.backtest(
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
    );
    return {
      success: response.success,
      metrics: response.metrics,
      equityCurve: response.equityCurve || [],
      error: response.error,
    };
  },

  getTemplates: async (): Promise<PythonTemplate[]> => {
    const response: any = await pythonStrategyService.getTemplates({});
    return (response.templates || []).map((t: any) => ({
      name: t.name,
      description: t.description,
      code: t.code,
    }));
  },

  validate: async (code: string): Promise<ValidateStrategyResult> => {
    const response: any = await pythonStrategyService.validate({ code });
    return {
      valid: response.valid || false,
      errors: response.errors || [],
      warnings: response.warnings || [],
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
		// Phase B2: secondary symbols (same timeframe/account) whose K-lines
		// are fetched and exposed to the strategy as features.
		extraSymbols?: string[];
	}): Promise<{ runId: string }> => {
		const msg = create(StartBacktestRunRequestSchema, {
			code: params.code,
			accountId: params.accountId,
			symbol: params.symbol,
			timeframe: params.timeframe,
			initialCapital: params.initialCapital ?? 10000,
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
		});
		const resp: any = await pythonStrategyService.startBacktestRun(msg as any);
		return { runId: resp.runId };
	},

	getBacktestRun: async (runId: string) => {
		const msg = create(GetBacktestRunRequestSchema, { runId });
		return (await pythonStrategyService.getBacktestRun(msg as any)) as any;
	},

	listBacktestRuns: async (params: { accountId?: string; limit?: number; offset?: number }) => {
		const msg = create(ListBacktestRunsRequestSchema, {
			accountId: params.accountId,
			limit: params.limit ?? 50,
			offset: params.offset ?? 0,
		});
		return (await pythonStrategyService.listBacktestRuns(msg as any)) as any;
	},

	cancelBacktestRun: async (runId: string) => {
		const msg = create(CancelBacktestRunRequestSchema, { runId });
		return (await pythonStrategyService.cancelBacktestRun(msg as any)) as any;
	},

	deleteBacktestRun: async (runId: string) => {
		const msg = create(DeleteBacktestRunRequestSchema, { runId });
		return (await pythonStrategyService.deleteBacktestRun(msg as any)) as any;
	},

	watchBacktestRun: (runId: string, onUpdate: (u: any) => void, onError?: (e: any) => void) => {
		const abortController = new AbortController();
		(async () => {
			try {
				const msg = create(WatchBacktestRunRequestSchema, { runId });
				const stream: any = pythonStrategyStreamService.watchBacktestRun(msg as any, { signal: abortController.signal });
				for await (const u of stream) {
					onUpdate(u);
				}
			} catch (e) {
				const errorStr = String(e);
				if ((e as any)?.name === 'AbortError' || errorStr.includes('canceled') || errorStr.includes('aborted')) {
					return;
				}
				onError?.(e);
			}
		})();
		return () => abortController.abort();
	},
};
