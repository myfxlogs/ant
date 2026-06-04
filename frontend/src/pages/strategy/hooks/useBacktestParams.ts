import { useState, useCallback, useMemo, useEffect, useRef } from 'react';
import { message } from 'antd';
import { pythonStrategyApi } from '@/client/pythonStrategy';
import { gateApi } from '@/client/gate';
import type { GateResult, GatePipelineSummary } from '@/gen/ant/v1/ai_gate_pb';

export type BacktestStatus = 'idle' | 'running' | 'completed' | 'error';
export type BacktestSubTab = 'results' | 'tuning' | 'gate';

export interface SweepDimension {
  key: string; label: string; source: 'code' | 'risk';
  enabled: boolean; values: number[];
}

export interface BacktestMetrics {
  totalReturn?: number; annualReturn?: number; maxDrawdown?: number;
  sharpeRatio?: number; winRate?: number; totalTrades?: number;
  equityCurve?: Array<{ time: number; equity: number }>;
  trades?: Array<{ id: string; time: number; side: string; price: number; volume: number; pnl?: number }>;
}

const DEFAULT_SWEEP_DIMS: SweepDimension[] = [
  { key: 'length', label: 'Period / Length', source: 'code', enabled: true, values: [10, 14, 20, 30, 50, 100] },
  { key: 'mult', label: 'Multiplier', source: 'code', enabled: true, values: [1.5, 2.0, 2.5, 3.0] },
  { key: 'stopLoss', label: 'Stop Loss %', source: 'risk', enabled: false, values: [2, 5, 8, 10, 15] },
  { key: 'takeProfit', label: 'Take Profit %', source: 'risk', enabled: false, values: [3, 5, 10, 15, 20] },
  { key: 'maxPositions', label: 'Max Positions', source: 'risk', enabled: false, values: [1, 3, 5, 10] },
];

function dateFromPreset(months: number): { start: string; end: string } {
  const end = new Date(); const start = new Date();
  start.setMonth(start.getMonth() - months);
  return { start: start.toISOString().slice(0, 10), end: end.toISOString().slice(0, 10) };
}

export const DATE_PRESETS = [
  { key: '1M', label: '1M', months: 1 },
  { key: '3M', label: '3M', months: 3 },
  { key: '6M', label: '6M', months: 6 },
  { key: '1Y', label: '1Y', months: 12 },
];

export function useBacktestParams() {
  const [submitting, setSubmitting] = useState(false);
  const [status, setStatus] = useState<BacktestStatus>('idle');
  const [metrics, setMetrics] = useState<BacktestMetrics | null>(null);
  const [errorMsg, setErrorMsg] = useState('');

  const [initialCapital, setInitialCapital] = useState<number>(10000);
  const [leverage, setLeverage] = useState<number>(1);
  const [commission, setCommission] = useState<number>(0.001);
  const [slippage, setSlippage] = useState<number>(0.0005);
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const [datePreset, setDatePreset] = useState('3M');
  const [tradeDirection, setTradeDirection] = useState('both');
  const [highPrecision, setHighPrecision] = useState(false);
  const [paramsExpanded, setParamsExpanded] = useState(true);
  const [resultsExpanded, setResultsExpanded] = useState(false);

  // Auto-expand results when backtest completes
  useEffect(() => {
    if (status === 'completed') setResultsExpanded(true);
  }, [status]);

  const applyDatePreset = useCallback((preset: { key: string; months: number }) => {
    setDatePreset(preset.key);
    const dates = dateFromPreset(preset.months);
    setStartDate(dates.start); setEndDate(dates.end);
  }, []);

  const runBacktest = useCallback(async (params: {
    code: string; accountId: string; symbol: string; timeframe: string;
  }) => {
    const { code, accountId, symbol, timeframe } = params;
    if (!code || !symbol) return;
    setSubmitting(true);
    try {
      const result = await pythonStrategyApi.startBacktestRun({
        code, accountId, symbol, timeframe, initialCapital,
      });
      if (!result.runId) throw new Error('No run ID');
      setBacktestRunId(result.runId);
      setStatus('running'); setGateGates([]); setGateSummary(null); setGateError('');
      const stopWatching = await pythonStrategyApi.watchBacktestRun(result.runId, (update: any) => {
        if (update.status === 'SUCCEEDED' || update.status === 'FAILED' || update.status === 'CANCELED') {
          setStatus(update.status === 'SUCCEEDED' ? 'completed' : 'error');
          setMetrics(update.metrics || null); setErrorMsg(update.error || ''); stopWatching();
        } else { setMetrics(update.metrics || null); }
      });
    } catch (e: any) {
      message.error(e?.message || 'Backtest failed');
      setStatus('error'); setErrorMsg(e?.message || 'Unknown error');
    } finally { setSubmitting(false); }
  }, [initialCapital]);

  // Smart Tuning
  const [subTab, setSubTab] = useState<BacktestSubTab>('results');
  const [tuneMethod, setTuneMethod] = useState<'grid' | 'random'>('grid');
  const [sweepDimensions, setSweepDimensions] = useState<SweepDimension[]>(DEFAULT_SWEEP_DIMS);
  const [tuningRunning, setTuningRunning] = useState(false);
  const [backtestRunId, setBacktestRunId] = useState('');

  // Gate evaluation
  const [gateLoading, setGateLoading] = useState(false);
  const [gateGates, setGateGates] = useState<GateResult[]>([]);
  const [gateSummary, setGateSummary] = useState<GatePipelineSummary | null>(null);
  const [gateError, setGateError] = useState('');
  const gateStopRef = useRef<(() => void) | null>(null);

  const enabledSweepDims = useMemo(() => sweepDimensions.filter(d => d.enabled), [sweepDimensions]);
  const cartesianSize = useMemo(() => enabledSweepDims.reduce((acc, d) => acc * d.values.length, 1), [enabledSweepDims]);

  const toggleDimension = useCallback((key: string) => {
    setSweepDimensions(prev => prev.map(d => d.key === key ? { ...d, enabled: !d.enabled } : d));
  }, []);

  const runTuning = useCallback(async () => {
    setTuningRunning(true);
    try {
      // Map enabled sweep dimensions to parameter space
      const paramSpace: Record<string, number[]> = {};
      const enabled = sweepDimensions.filter(d => d.enabled);
      for (const dim of enabled) {
        if (dim.values && dim.values.length > 0) paramSpace[dim.key] = dim.values as number[];
        else paramSpace[dim.key] = [0.01, 0.02, 0.03, 0.05, 0.10];
      }
      // Create experiment via API (templateId is optional; code-based tuning uses empty string)
      const { strategyExperimentApi } = await import('@/client/strategyExperiment');
      await strategyExperimentApi.submit({
        baseTemplateId: '',
        parameterSpace: paramSpace as Record<string, unknown>,
        searchMethod: tuneMethod === 'grid' ? 'grid' : 'random',
        maxCandidates: Math.min(cartesianSize || 24, 48),
        objective: 'balanced',
      });
      // TODO: poll for experiment completion and display candidates
    } finally {
      setTuningRunning(false);
    }
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

  return {
    submitting, status, metrics, errorMsg,
    initialCapital, setInitialCapital, leverage, setLeverage,
    commission, setCommission, slippage, setSlippage,
    startDate, setStartDate, endDate, setEndDate,
    datePreset, tradeDirection, setTradeDirection, highPrecision, setHighPrecision,
    paramsExpanded, setParamsExpanded,
    resultsExpanded, setResultsExpanded,
    applyDatePreset,
    runBacktest,
    subTab, setSubTab, tuneMethod, setTuneMethod,
    sweepDimensions, toggleDimension, enabledSweepDims, cartesianSize,
    tuningRunning, runTuning,
    backtestRunId, gateLoading, gateGates, gateSummary, gateError, runGate,
  };
}
