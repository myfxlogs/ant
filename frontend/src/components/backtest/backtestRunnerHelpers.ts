import type { TFunction } from 'i18next';
import { message } from 'antd';
import { strategyRuntimeApi } from '@/client/strategyRuntime';
import { backtestRunsApi } from '@/client/backtestRuns';
import { BacktestRunStatus } from '@/gen/ant/v1/backtest_run_pb';
import {
  DEFAULTS_SAVED_KEY, DEFAULTS_LOADED_KEY, DEFAULTS_RESET_KEY,
  SETTINGS_SAVE_KEY, SETTINGS_LOAD_KEY, SETTINGS_RESET_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import {
  FACTORY_DEFAULTS, saveDefaults, removeDefaults, protoToMetrics,
  type StandardParams, type BacktestMetrics, type ChartTrade,
} from './backtestRunnerTypes';

export function buildSettingsItems(
  t: TFunction, standardParams: StandardParams,
  saved: Partial<StandardParams> | null, applyDefaults: (d: Partial<StandardParams>) => void,
) {
  return [
    { key: 'save', label: t(SETTINGS_SAVE_KEY), onClick: () => { saveDefaults(standardParams); message.success(t(DEFAULTS_SAVED_KEY)); } },
    ...(saved ? [{ key: 'load', label: t(SETTINGS_LOAD_KEY), onClick: () => { applyDefaults(saved); message.success(t(DEFAULTS_LOADED_KEY)); } }] : []),
    { key: 'reset', label: t(SETTINGS_RESET_KEY), onClick: () => { removeDefaults(); applyDefaults(FACTORY_DEFAULTS); message.success(t(DEFAULTS_RESET_KEY)); } },
  ];
}

export async function restoreLastRunFn(
  accountId: string, templateId: string | undefined,
  setMetrics: (m: BacktestMetrics | null) => void,
  setExecutionAssumptions: (a: import('@/gen/ant/v1/backtest_execution_config_pb').ExecutionAssumptions | null) => void,
  setRunId: (id: string) => void,
  setStatus: (s: 'idle' | 'running' | 'completed' | 'error' | 'degraded') => void,
  setChartTrades: React.Dispatch<React.SetStateAction<ChartTrade[]>>,
) {
  try {
    const resp = await strategyRuntimeApi.listBacktestRuns({ accountId: accountId || undefined, templateId: templateId || undefined, limit: 1, offset: 0 });
    const lastRun = resp.runs?.[0];
    if (!lastRun || lastRun.status !== BacktestRunStatus.SUCCEEDED) return;
    const runIdStr = lastRun.id;
    if (!runIdStr) return;
    const detail = await strategyRuntimeApi.getBacktestRun(runIdStr);
    if (detail.metrics) setMetrics(protoToMetrics(detail.metrics));
    if (detail.executionAssumptions) setExecutionAssumptions(detail.executionAssumptions);
    setRunId(runIdStr); setStatus('completed');
    const tr = await backtestRunsApi.getTrades(runIdStr);
    setChartTrades(tr.trades.map((t2) => ({
      side: t2.side, openTime: t2.open_ts, openPrice: t2.open_price,
      closeTime: t2.close_ts, closePrice: t2.close_price, pnl: t2.pnl, volume: t2.volume,
      ticket: t2.ticket, commission: t2.commission, reason: t2.reason,
    })));
  } catch { /* silent — restoration is best-effort */ }
}
