import { notification, message } from 'antd';
import type { TFunction } from 'i18next';
import type { BacktestRunUpdate, MarketplaceQualityPreview } from '@/gen/ant/v1/backtest_run_query_pb';
import type { GateEvaluationUpdate, GateResult } from '@/gen/ant/v1/ai_gate_pb';
import { isTerminalRun, isSucceededRun } from '@/pages/strategy/StrategyTemplatePage.utils';
import { backtestRunsApi, type BacktestTrade } from '@/client/backtestRuns';
import { BACKTEST_COMPLETED_KEY, BACKTEST_ERROR_KEY, BACKTEST_DEGRADED_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { TOTAL_RETURN_KEY } from '@/gen/ant/v1/i18n/strategy_backtest_keys';
import { BACKTEST_FAILED_KEY } from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import type { BacktestBlindSpot } from '@/gen/ant/v1/backtest_run_query_pb';
import { BacktestRunStatus } from '@/gen/ant/v1/backtest_run_pb';
import type { BacktestMetrics, ChartTrade } from './backtestRunnerTypes';
import { protoToMetrics } from './backtestRunnerTypes';

export type BacktestBlindSpotItem = { id: string; description: string; severity: string; category: string; location: string };

interface WatchCallbacks {
  setFixDepth: (n: number) => void;
  setGateResults: React.Dispatch<React.SetStateAction<GateResult[]>>;
  setGateUpdate: (u: GateEvaluationUpdate | null) => void;
  setQualityPreview: (q: MarketplaceQualityPreview | null) => void;
  setStatus: (s: 'idle' | 'running' | 'completed' | 'error' | 'degraded') => void;
  setMetrics: (m: BacktestMetrics | null) => void;
  setExecutionAssumptions: (a: import('@/gen/ant/v1/backtest_execution_config_pb').ExecutionAssumptions | null) => void;
  setErrorMsg: (e: string) => void;
  setChartTrades: React.Dispatch<React.SetStateAction<ChartTrade[]>>;
  setBlindSpots: React.Dispatch<React.SetStateAction<BacktestBlindSpotItem[]>>;
  stopWatching: () => void;
}

export function handleBacktestUpdate(
  update: BacktestRunUpdate,
  runId: string,
  t: TFunction,
  cb: WatchCallbacks,
): void {
  const run = update.run;
  if (run?.fixDepth) cb.setFixDepth(run.fixDepth);
  if (update.gateUpdate?.gate) cb.setGateResults(prev => [...prev, update.gateUpdate!.gate!]);
  if (update.gateUpdate?.completed) cb.setGateUpdate(update.gateUpdate);
  if (update.qualityPreview) cb.setQualityPreview(update.qualityPreview);
  if (update.blindSpots && update.blindSpots.length > 0) {
    cb.setBlindSpots(update.blindSpots.map((b: BacktestBlindSpot) => ({ id: b.id, description: b.description, severity: b.severity, category: b.category, location: b.location })));
  }
  if (run && isTerminalRun(run)) {
    handleTerminalRun(update, run, runId, t, cb);
  } else {
    cb.setMetrics(protoToMetrics(update.metrics));
  }
}

function handleTerminalRun(update: BacktestRunUpdate, run: NonNullable<BacktestRunUpdate['run']>, runId: string, t: TFunction, cb: WatchCallbacks): void {
  const ok = isSucceededRun(run);
  const isDegraded = run.status === BacktestRunStatus.DEGRADED;
  if (isDegraded) {
    cb.setStatus('degraded');
  } else {
    cb.setStatus(ok ? 'completed' : 'error');
  }
  cb.setMetrics(protoToMetrics(update.metrics));
  cb.setExecutionAssumptions(update.executionAssumptions ?? null);
  cb.setErrorMsg(update.run?.error ?? '');
  cb.stopWatching();
  if (isDegraded) {
    notification.warning({ message: t(BACKTEST_DEGRADED_KEY), description: '', placement: 'bottomRight', duration: 6 });
    backtestRunsApi.getTrades(runId).then((tr) => {
      cb.setChartTrades(tr.trades.map((t: BacktestTrade) => ({
        side: t.side, openTime: t.open_ts, openPrice: t.open_price,
        closeTime: t.close_ts, closePrice: t.close_price, pnl: t.pnl, volume: t.volume,
        ticket: t.ticket, commission: t.commission, reason: t.reason,
      })));
    }).catch(() => cb.setChartTrades([]));
  } else if (ok) {
    const m = protoToMetrics(update.metrics);
    notification.success({ message: t(BACKTEST_COMPLETED_KEY), description: t(TOTAL_RETURN_KEY) + ': ' + ((m?.totalReturn ?? 0) * 100).toFixed(2) + '%', placement: 'bottomRight', duration: 4 });
    backtestRunsApi.getTrades(runId).then((tr) => {
      cb.setChartTrades(tr.trades.map((t: BacktestTrade) => ({
        side: t.side, openTime: t.open_ts, openPrice: t.open_price,
        closeTime: t.close_ts, closePrice: t.close_price, pnl: t.pnl, volume: t.volume,
        ticket: t.ticket, commission: t.commission, reason: t.reason,
      })));
    }).catch(() => cb.setChartTrades([]));
  } else {
    cb.setChartTrades([]);
    notification.error({ message: t(BACKTEST_ERROR_KEY), description: update.run?.error || '', placement: 'bottomRight', duration: 6 });
  }
}

export function handleBacktestError(e: unknown, t: TFunction): { status: 'error'; msg: string } {
  const msg = e instanceof Error ? e.message : String(e);
  message.error(msg || t(BACKTEST_FAILED_KEY));
  return { status: 'error', msg: msg || 'Unknown error' };
}
