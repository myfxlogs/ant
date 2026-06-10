import { useState, useCallback, useEffect, useRef } from 'react';
import { message } from 'antd';
import { useTranslation } from 'react-i18next';
import { pythonStrategyApi } from '@/client/pythonStrategy';
import { backtestRunsApi, type BacktestTrade } from '@/client/backtestRuns';
import {
  backendDirectivesToStrategyDirectives,
  PRESETS, DATE_PRESETS, dateFromPreset,
  TIMEFRAME_MAX_MONTHS,
} from './backtestParamHelpers';
import type { StrategyDirective, PresetKey } from './backtestParamHelpers';
import { useTuning } from './useTuning';
import { useGateEvaluation } from './useGateEvaluation';

export type { StrategyDirective, PresetKey };
export { PRESETS, DATE_PRESETS };

export type BacktestStatus = 'idle' | 'running' | 'completed' | 'error';

/** Chart-ready trade for klinecharts backtest overlay. */
export interface ChartTrade {
  side: string;
  openTime: number;
  openPrice: number;
  closeTime?: number;
  closePrice?: number;
  pnl?: number;
}

export interface BacktestMetrics {
  totalReturn?: number; annualReturn?: number; maxDrawdown?: number;
  sharpeRatio?: number; winRate?: number; totalTrades?: number;
  equityCurve?: Array<{ time: number; equity: number }>;
  trades?: Array<{ id: string; time: number; side: string; price: number; volume: number; pnl?: number }>;
}

export function useBacktestParams() {
  const { t } = useTranslation();
  const [submitting, setSubmitting] = useState(false);
  const [status, setStatus] = useState<BacktestStatus>('idle');
  const [metrics, setMetrics] = useState<BacktestMetrics | null>(null);
  const [executionAssumptions, setExecutionAssumptions] = useState<any>(null);
  const [errorMsg, setErrorMsg] = useState('');

  const [initialCapital, setInitialCapital] = useState<number>(10000);
  const [leverage, setLeverage] = useState<number>(1);
  const [commission, setCommission] = useState<number>(0.001);
  const [slippage, setSlippage] = useState<number>(0.0);
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const [datePreset, setDatePreset] = useState('3M');
  const [tradeDirection, setTradeDirection] = useState('both');
  const [strictMode, setStrictMode] = useState(true);
  const [paramsExpanded, setParamsExpanded] = useState(true);
  const [resultsExpanded, setResultsExpanded] = useState(false);
  const [strategyDirectives, setStrategyDirectives] = useState<StrategyDirective[]>([]);

  const [backtestRunId, setBacktestRunId] = useState('');
  const [chartTrades, setChartTrades] = useState<ChartTrade[]>([]);
  const backtestWatchRef = useRef<(() => void) | null>(null);

  // Sub-hooks: tuning and gate evaluation are separate functional domains.
  const tuning = useTuning(t);
  const gate = useGateEvaluation();

  // Cleanup all SSE/watch connections on unmount.
  useEffect(() => () => { backtestWatchRef.current?.(); }, []);

  useEffect(() => {
    if (status === 'completed') setResultsExpanded(true);
  }, [status]);

  const applyDatePreset = useCallback((preset: { key: string; months: number }) => {
    setDatePreset(preset.key);
    const dates = dateFromPreset(preset.months);
    setStartDate(dates.start); setEndDate(dates.end);
  }, []);

  const applyPreset = useCallback((key: PresetKey) => {
    const p = PRESETS[key];
    setCommission(p.commission);
    setSlippage(p.slippage);
  }, []);

  const updateStrategyDirectivesFromCode = useCallback((dirs: { key: string; value: string }[]) => {
    setStrategyDirectives(backendDirectivesToStrategyDirectives(dirs));
  }, []);

  const getTimeframeWarning = useCallback((timeframe: string, presetMonths: number): string | null => {
    const maxMonths = TIMEFRAME_MAX_MONTHS[timeframe];
    if (maxMonths && presetMonths > maxMonths) {
      return `${timeframe} timeframe recommends max ${maxMonths} month${maxMonths > 1 ? 's' : ''}`;
    }
    return null;
  }, []);

  const runBacktest = useCallback(async (params: {
    code: string; accountId: string; symbol: string; timeframe: string;
    templateId?: string;
  }) => {
    const { code, accountId, symbol, timeframe, templateId } = params;
    if (!code || !symbol) { message.warning(t('strategy.backtestParams.enterCodeAndSymbol')); return; }
    setSubmitting(true);
    try {
      const result = await pythonStrategyApi.startBacktestRun({
        code, accountId, symbol, timeframe, initialCapital,
        mode: 'KLINE_RANGE',
        from: startDate ? new Date(startDate) : undefined,
        to: endDate ? new Date(endDate) : undefined,
        templateId: templateId || undefined,
        executionConfig: {
          commission, slippage, leverage,
          tradeDirection: tradeDirection as 'long' | 'short' | 'both',
          strictMode,
        },
      });
      if (!result.runId) throw new Error('No run ID');
      setBacktestRunId(result.runId);
      setStatus('running');
      setExecutionAssumptions(null);
      // Stop previous watch before starting a new one.
      backtestWatchRef.current?.();
      const stopWatching = await pythonStrategyApi.watchBacktestRun(result.runId, (update: any) => {
        if (update.status === 'SUCCEEDED' || update.status === 'FAILED' || update.status === 'CANCELED') {
          setStatus(update.status === 'SUCCEEDED' ? 'completed' : 'error');
          setMetrics(update.metrics || null);
          setExecutionAssumptions(update.executionAssumptions || null);
          setErrorMsg(update.error || ''); stopWatching();
          backtestWatchRef.current = null;
          // Fetch trade details for chart overlay markers.
          if (update.status === 'SUCCEEDED') {
            backtestRunsApi.getTrades(result.runId).then((tr) => {
              setChartTrades(tr.trades.map((t: BacktestTrade) => ({
                side: t.side,
                openTime: t.open_ts,
                openPrice: t.open_price,
                closeTime: t.close_ts,
                closePrice: t.close_price,
                pnl: t.pnl,
              })));
            }).catch(() => {
              setChartTrades([]);
            });
          } else { setChartTrades([]); }
        } else { setMetrics(update.metrics || null); }
      });
      backtestWatchRef.current = stopWatching;
    } catch (e: any) {
      message.error(e?.message || t('strategy.backtestParams.backtestFailed'));
      setStatus('error'); setErrorMsg(e?.message || 'Unknown error');
    } finally { setSubmitting(false); }
  }, [initialCapital, commission, slippage, leverage, tradeDirection, strictMode, startDate, endDate, t]);

  // applyDefaults bulk-sets multiple params at once (for Settings → Load/Reset).
  const applyDefaults = useCallback((d: {
    commission: number; slippage: number; leverage: number;
    tradeDirection: string; strictMode: boolean;
  }) => {
    setCommission(d.commission);
    setSlippage(d.slippage);
    setLeverage(d.leverage);
    setTradeDirection(d.tradeDirection);
    setStrictMode(d.strictMode);
  }, []);

  return {
    submitting, status, metrics, executionAssumptions, errorMsg,
    initialCapital, setInitialCapital, leverage, setLeverage,
    commission, setCommission, slippage, setSlippage,
    startDate, setStartDate, endDate, setEndDate,
    datePreset, tradeDirection, setTradeDirection,
    strictMode, setStrictMode,
    paramsExpanded, setParamsExpanded,
    resultsExpanded, setResultsExpanded,
    strategyDirectives, updateStrategyDirectivesFromCode,
    applyDefaults, applyPreset, applyDatePreset, getTimeframeWarning,
    runBacktest,
    backtestRunId, chartTrades,
    // Tuning domain (delegated).
    tuning: {
      subTab: tuning.subTab, setSubTab: tuning.setSubTab,
      method: tuning.tuneMethod, setMethod: tuning.setTuneMethod,
      sweepDimensions: tuning.sweepDimensions, updateSweepFromCode: tuning.updateSweepFromCode,
      toggleDimension: tuning.toggleDimension,
      enabledDims: tuning.enabledSweepDims, cartesianSize: tuning.cartesianSize,
      running: tuning.tuningRunning, run: tuning.runTuning,
    },
    // Gate domain (delegated).
    gate: {
      loading: gate.gateLoading, gates: gate.gateGates,
      summary: gate.gateSummary, error: gate.gateError,
      run: () => gate.runGate(backtestRunId, () => tuning.setSubTab('gate')),
    },
  };
}
