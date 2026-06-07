import { useState, useCallback, useMemo, useEffect, useRef } from 'react';
import { message } from 'antd';
import { pythonStrategyApi } from '@/client/pythonStrategy';
import { gateApi } from '@/client/gate';
import { backtestRunsApi, type BacktestTrade } from '@/client/backtestRuns';
import type { GateResult, GatePipelineSummary } from '@/gen/ant/v1/ai_gate_pb';
import {
  backendDirectivesToStrategyDirectives, backendSweepToSweepDimensions,
  PRESETS, DATE_PRESETS, DEFAULT_SWEEP_DIMS, dateFromPreset,
  TIMEFRAME_MAX_MONTHS, OPTIMIZER_INFO,
} from './backtestParamHelpers';
import type {
  StrategyDirective, SweepDimension, PresetKey, TuneMethod,
} from './backtestParamHelpers';

export type { StrategyDirective, SweepDimension, PresetKey, TuneMethod };
export { PRESETS, DATE_PRESETS, OPTIMIZER_INFO };

export type BacktestStatus = 'idle' | 'running' | 'completed' | 'error';
export type BacktestSubTab = 'results' | 'tuning' | 'gate';

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
  }) => {
    const { code, accountId, symbol, timeframe } = params;
    if (!code || !symbol) { message.warning('Please enter strategy code and select a symbol'); return; }
    setSubmitting(true);
    try {
      const result = await pythonStrategyApi.startBacktestRun({
        code, accountId, symbol, timeframe, initialCapital,
        mode: 'KLINE_RANGE',
        from: startDate ? new Date(startDate) : undefined,
        to: endDate ? new Date(endDate) : undefined,
        executionConfig: {
          commission, slippage, leverage,
          tradeDirection: tradeDirection as 'long' | 'short' | 'both',
          strictMode,
        },
      });
      if (!result.runId) throw new Error('No run ID');
      setBacktestRunId(result.runId);
      setStatus('running'); setGateGates([]); setGateSummary(null); setGateError('');
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
            }).catch((e: unknown) => {
              console.warn('Failed to fetch backtest trades for chart markers:', e);
              setChartTrades([]);
            });
          } else { setChartTrades([]); }
        } else { setMetrics(update.metrics || null); }
      });
      backtestWatchRef.current = stopWatching;
    } catch (e: any) {
      message.error(e?.message || 'Backtest failed');
      setStatus('error'); setErrorMsg(e?.message || 'Unknown error');
    } finally { setSubmitting(false); }
  }, [initialCapital, commission, slippage, leverage, tradeDirection, strictMode, startDate, endDate]);

  // Smart Tuning.
  const [subTab, setSubTab] = useState<BacktestSubTab>('results');
  const [tuneMethod, setTuneMethod] = useState<TuneMethod>('grid');
  const [sweepDimensions, setSweepDimensions] = useState<SweepDimension[]>(DEFAULT_SWEEP_DIMS);

  const updateSweepFromCode = useCallback((dims: { key: string; type: string; default: number; min: number; max: number; step: number; hasRange: boolean }[]) => {
    const extracted = backendSweepToSweepDimensions(dims);
    if (extracted.length > 0) { setSweepDimensions(extracted); }
  }, []);
  const [tuningRunning, setTuningRunning] = useState(false);
  const [backtestRunId, setBacktestRunId] = useState('');
  const [chartTrades, setChartTrades] = useState<ChartTrade[]>([]);

  // Gate evaluation.
  const [gateLoading, setGateLoading] = useState(false);
  const [gateGates, setGateGates] = useState<GateResult[]>([]);
  const [gateSummary, setGateSummary] = useState<GatePipelineSummary | null>(null);
  const [gateError, setGateError] = useState('');
  const gateStopRef = useRef<(() => void) | null>(null);
  const backtestWatchRef = useRef<(() => void) | null>(null);

  // Cleanup all SSE/watch connections on unmount.
  useEffect(() => () => {
    gateStopRef.current?.();
    backtestWatchRef.current?.();
  }, []);

  const enabledSweepDims = useMemo(() => sweepDimensions.filter(d => d.enabled), [sweepDimensions]);
  const cartesianSize = useMemo(() => enabledSweepDims.reduce((acc, d) => acc * d.values.length, 1), [enabledSweepDims]);

  const toggleDimension = useCallback((key: string) => {
    setSweepDimensions(prev => prev.map(d => d.key === key ? { ...d, enabled: !d.enabled } : d));
  }, []);

  const runTuning = useCallback(async (params: {
    code: string; symbol: string; timeframe: string;
    startDate: string; endDate: string;
  }): Promise<string> => {
    setTuningRunning(true);
    try {
      const paramSpace: Record<string, number[]> = {};
      const enabled = sweepDimensions.filter(d => d.enabled);
      for (const dim of enabled) {
        if (dim.values && dim.values.length > 0) paramSpace[dim.key] = dim.values as number[];
        else paramSpace[dim.key] = [0.01, 0.02, 0.03, 0.05, 0.10];
      }
      const { strategyExperimentApi } = await import('@/client/strategyExperiment');
      const fromMs = params.startDate ? new Date(params.startDate).getTime() : 0;
      const toMs = params.endDate ? new Date(params.endDate).getTime() : 0;
      const result = await strategyExperimentApi.submit({
        baseTemplateId: '',
        parameterSpace: paramSpace as Record<string, unknown>,
        searchMethod: tuneMethod,
        maxCandidates: Math.min(cartesianSize || 24, 48),
        objective: 'balanced',
        strategyCode: params.code || '',
        symbol: params.symbol || '',
        timeframe: params.timeframe || '',
        fromTsUnixMs: BigInt(fromMs),
        toTsUnixMs: BigInt(toMs),
      });
      message.success('Smart Tuning started');
      return result.experiment?.id || result.jobId || '';
    } catch (e: any) {
      message.error(e?.message || 'Tuning failed');
      return '';
    } finally { setTuningRunning(false); }
  }, [sweepDimensions, tuneMethod, cartesianSize]);

  const runGate = useCallback(() => {
    if (!backtestRunId) return;
    gateStopRef.current?.();
    setGateLoading(true); setGateGates([]); setGateSummary(null); setGateError('');
    setSubTab('gate');
    const stop = gateApi.runEvaluation(
      { backtestRunId },
      {
        onGate: (g) => setGateGates(prev => [...prev, g]),
        onCompleted: (s) => { setGateSummary(s); setGateLoading(false); },
        onError: (e) => { setGateError(String(e?.message ?? e ?? 'Unknown error')); setGateLoading(false); },
      },
    );
    gateStopRef.current = stop;
  }, [backtestRunId]);

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
    subTab, setSubTab, tuneMethod, setTuneMethod,
    sweepDimensions, updateSweepFromCode, toggleDimension, enabledSweepDims, cartesianSize,
    tuningRunning, runTuning,
    backtestRunId, chartTrades, gateLoading, gateGates, gateSummary, gateError, runGate,
  };
}
