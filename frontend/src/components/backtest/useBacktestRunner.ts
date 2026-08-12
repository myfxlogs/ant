import { useState, useCallback, useEffect, useRef, useMemo } from 'react';
import { message } from 'antd';
import { useTranslation } from 'react-i18next';
import { ENTER_CODE_AND_SYMBOL_KEY } from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import { strategyRuntimeApi } from '@/client/strategyRuntime';
import { trackFunnelEvent, FunnelEvents } from '@/utils/analytics';
import type { BacktestRunUpdate, MarketplaceQualityPreview } from '@/gen/ant/v1/backtest_run_query_pb';
import type { GateEvaluationUpdate, GateResult } from '@/gen/ant/v1/ai_gate_pb';
import {
  backendDirectivesToStrategyDirectives,
  PRESETS, DATE_PRESETS, dateFromPreset,
  type StrategyDirective, type PresetKey,
} from '@/pages/strategy/hooks/backtestParamHelpers';
import { useTuning } from '@/pages/strategy/hooks/useTuning';
import { useGateEvaluation } from '@/pages/strategy/hooks/useGateEvaluation';
import {
  type BacktestStatus, type ChartTrade, type BacktestMetrics, type ExtractedParam, type StandardParams, type BacktestRunnerInputs,
  FACTORY_DEFAULTS, loadSavedDefaults, getTimeframeWarning,
} from './backtestRunnerTypes';
import { handleBacktestUpdate, handleBacktestError, type BacktestBlindSpotItem } from './backtestRunnerWatch';
import { buildSettingsItems, restoreLastRunFn } from './backtestRunnerHelpers';
import { backtestRunsApi } from '@/client/backtestRuns';
import { protoToMetrics } from './backtestRunnerTypes';

export type { StrategyDirective, PresetKey };
export { PRESETS, DATE_PRESETS };
export type { SweepDimension, TuneMethod } from '@/pages/strategy/hooks/useTuning';
export type { BacktestStatus, ChartTrade, BacktestMetrics, ExtractedParam, StandardParams, BacktestRunnerInputs };
export { FACTORY_DEFAULTS };

export function useBacktestRunner() {
  const { t } = useTranslation();

  // ── Standard params ──────────────────────────────────────────────────
  const [initialCapital, setInitialCapital] = useState(FACTORY_DEFAULTS.initialCapital);
  const [leverage, setLeverage] = useState(FACTORY_DEFAULTS.leverage);
  const [lotSize, setLotSize] = useState(FACTORY_DEFAULTS.lotSize);
  const [commission, setCommission] = useState(FACTORY_DEFAULTS.commission);
  const [slippage, setSlippage] = useState(FACTORY_DEFAULTS.slippage);
  const [tradeDirection, setTradeDirection] = useState(FACTORY_DEFAULTS.tradeDirection);
  const [strictMode, setStrictMode] = useState(FACTORY_DEFAULTS.strictMode);

  // ── Date range ────────────────────────────────────────────────────────
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const [datePreset, setDatePreset] = useState('3M');

  // ── Strategy params ───────────────────────────────────────────────────
  const [extractedParams, setExtractedParams] = useState<ExtractedParam[]>([]);
  const [strategyParamValues, setStrategyParamValues] = useState<Record<string, string>>({});

  // ── Run state ─────────────────────────────────────────────────────────
  const [submitting, setSubmitting] = useState(false);
  const [status, setStatus] = useState<BacktestStatus>('idle');
  const [metrics, setMetrics] = useState<BacktestMetrics | null>(null);
  const [executionAssumptions, setExecutionAssumptions] = useState<import('@/gen/ant/v1/backtest_execution_config_pb').ExecutionAssumptions | null>(null);
  const [errorMsg, setErrorMsg] = useState('');

  // ── Chart trades ──────────────────────────────────────────────────────
  const [chartTrades, setChartTrades] = useState<ChartTrade[]>([]);

  // ── Directives ────────────────────────────────────────────────────────
  const [strategyDirectives, setStrategyDirectives] = useState<StrategyDirective[]>([]);

  // ── UI state ──────────────────────────────────────────────────────────
  const [activeTab, setActiveTab] = useState<string>('results');
  const [panelHeight, setPanelHeight] = useState(280);
  const [dragging, setDragging] = useState(false);
  const [userResized, setUserResized] = useState(false);
  const [strategyParamsModalOpen, setStrategyParamsModalOpen] = useState(false);

  // ── Internal ──────────────────────────────────────────────────────────
  const [runId, setRunId] = useState('');
  const [runMeta, setRunMeta] = useState<{ symbol?: string; timeframe?: string; createdAt?: string; name?: string } | null>(null);
  const [fixDepth, setFixDepth] = useState(0);
  const [gateUpdate, setGateUpdate] = useState<GateEvaluationUpdate | null>(null);
  const [gateResults, setGateResults] = useState<GateResult[]>([]);
  const [qualityPreview, setQualityPreview] = useState<MarketplaceQualityPreview | null>(null);
  const [blindSpots, setBlindSpots] = useState<BacktestBlindSpotItem[]>([]);
  const watchRef = useRef<(() => void) | null>(null);

  // Sub-hooks
  const tuning = useTuning(t);
  const gate = useGateEvaluation();

  // Cleanup on unmount.
  useEffect(() => () => { watchRef.current?.(); }, []);

  // ── Derived ───────────────────────────────────────────────────────────
  const standardParams: StandardParams = useMemo(() => ({
    initialCapital, leverage, lotSize, commission, slippage, tradeDirection, strictMode,
  }), [initialCapital, leverage, lotSize, commission, slippage, tradeDirection, strictMode]);

  const saved = useMemo(() => loadSavedDefaults(), []);

  // ── Actions ───────────────────────────────────────────────────────────

  const updateExtractedParams = useCallback((pj: string | ExtractedParam[] | null) => {
    if (!pj) return;
    try {
      const list: ExtractedParam[] = typeof pj === 'string' ? JSON.parse(pj) : pj;
      setExtractedParams(list || []);
      // Initialise strategy param values from defaults.
      const vals: Record<string, string> = {};
      for (const p of (list || [])) vals[p.name] = p.default;
      setStrategyParamValues(prev => ({ ...vals, ...prev }));
    } catch { /* ignore parse errors */ }
  }, []);

  const updateDirectivesFromCode = useCallback((dirs: { key: string; value: string }[]) => {
    setStrategyDirectives(backendDirectivesToStrategyDirectives(dirs));
  }, []);

  const handleValidationResult = useCallback((result: import('@/client/codeAssist').ValidateExtendedResult) => {
    if (result.sweepDimensions?.length > 0) tuning.updateSweepFromCode(result.sweepDimensions);
    if (result.strategyDirectives?.length > 0) updateDirectivesFromCode(result.strategyDirectives);
    updateExtractedParams(
      result.parameterEntries?.map((e: { name: string; type: string; default: string; label?: string }) => ({ name: e.name, type: e.type, default: e.default, label: e.label || '' })) || null
    );
  }, [tuning, updateDirectivesFromCode, updateExtractedParams]);

  const setParam = useCallback((name: string, value: string) => {
    setStrategyParamValues(prev => ({ ...prev, [name]: value }));
  }, []);

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

  const applyDefaults = useCallback((d: Partial<StandardParams>) => {
    if (d.commission !== undefined) setCommission(d.commission);
    if (d.slippage !== undefined) setSlippage(d.slippage);
    if (d.leverage !== undefined) setLeverage(d.leverage);
    if (d.tradeDirection !== undefined) setTradeDirection(d.tradeDirection);
    if (d.strictMode !== undefined) setStrictMode(d.strictMode);
  }, []);

  const resetStatus = useCallback(() => {
    setStatus('idle'); setErrorMsg(''); setMetrics(null);
    setExecutionAssumptions(null); setChartTrades([]);
    setRunId(''); setGateUpdate(null); setGateResults([]); setQualityPreview(null); setFixDepth(0);
    setBlindSpots([]);
    setUserResized(false);
  }, []);

  // ── Settings menu ─────────────────────────────────────────────────────

  const settingsItems = useMemo(() => buildSettingsItems(t, standardParams, saved, applyDefaults), [t, standardParams, saved, applyDefaults]);

  // ── Run backtest ──────────────────────────────────────────────────────

  const run = useCallback(async (
    inputs: BacktestRunnerInputs,
    overrides?: {
      params?: Record<string, string>;
      executionConfig?: {
        commission: number;
        slippage: number;
        leverage: number;
        tradeDirection: string;
        strictMode: boolean;
        signalTiming?: 'next_bar_open' | 'same_bar_close';
        fillRule?: 'bar_close' | 'market' | 'limit';
        simulationMode?: 'KLINE_RANGE' | 'OHLC_PATH';
      };
    },
  ) => {
    const { strategyCode, accountId, symbol, timeframe, templateId, strategyId } = inputs;
    if (!strategyCode || !symbol) { message.warning(t(ENTER_CODE_AND_SYMBOL_KEY)); return; }
    setSubmitting(true);
    setActiveTab('results');
    try {
      trackFunnelEvent(FunnelEvents.FIRST_BACKTEST);
      const paramValues = overrides?.params ?? strategyParamValues;
      const cfg = overrides?.executionConfig
        ? { commission: overrides.executionConfig.commission, slippage: overrides.executionConfig.slippage, leverage: overrides.executionConfig.leverage, tradeDirection: overrides.executionConfig.tradeDirection as 'long' | 'short' | 'both', strictMode: overrides.executionConfig.strictMode, signalTiming: overrides.executionConfig.signalTiming, fillRule: overrides.executionConfig.fillRule, simulationMode: overrides.executionConfig.simulationMode }
        : { commission, slippage, leverage, tradeDirection: tradeDirection as 'long' | 'short' | 'both', strictMode };
      const effectiveStartDate = inputs.startDate ?? startDate;
      const effectiveEndDate = inputs.endDate ?? endDate;
      const result = await strategyRuntimeApi.startBacktestRun({
        code: strategyCode, accountId, symbol, timeframe, initialCapital,
        mode: 'KLINE_RANGE',
        from: effectiveStartDate ? new Date(effectiveStartDate) : undefined,
        to: effectiveEndDate ? new Date(effectiveEndDate) : undefined,
        templateId: templateId || undefined,
        strategyId: strategyId || undefined,
        autoGate: true,
        parameterOverrides: paramValues,
        executionConfig: cfg,
      });
      if (!result.runId) throw new Error('No run ID');
      setRunId(result.runId);
      setStatus('running');
      setExecutionAssumptions(null);
      setGateUpdate(null);
      setGateResults([]);
      setQualityPreview(null);
      setBlindSpots([]);
      setFixDepth(0);
      watchRef.current?.();
      const stopWatching = await strategyRuntimeApi.watchBacktestRun(result.runId, (update: BacktestRunUpdate) => {
        handleBacktestUpdate(update, result.runId, t, {
          setFixDepth, setGateResults, setGateUpdate, setQualityPreview,
          setStatus, setMetrics, setExecutionAssumptions, setErrorMsg, setChartTrades,
          setBlindSpots,
          stopWatching: () => { stopWatching(); watchRef.current = null; },
        });
      });
      watchRef.current = stopWatching;
    } catch (e: unknown) {
      const { status, msg } = handleBacktestError(e, t);
      setStatus(status); setErrorMsg(msg);
    } finally { setSubmitting(false); }
  }, [initialCapital, commission, slippage, leverage, tradeDirection, strictMode, startDate, endDate, strategyParamValues, t]);

  // ── Return ────────────────────────────────────────────────────────────

  const cancelRun = useCallback(async () => {
    if (!runId) return;
    try {
      await strategyRuntimeApi.cancelBacktestRun(runId);
      watchRef.current?.(); watchRef.current = null;
      setStatus('idle'); setRunId('');
      message.info(t('strategy.backtest.canceled', { defaultValue: 'Backtest canceled' }));
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t('strategy.backtest.cancelFailed', { defaultValue: 'Cancel failed' }));
    }
  }, [runId, t]);

  const restoreLastRun = useCallback((accountId: string, templateId?: string) =>
    restoreLastRunFn(accountId, templateId, setMetrics, setExecutionAssumptions, setRunId, setStatus, setChartTrades), []);

  const loadRunById = useCallback(async (id: string, onCodeLoaded?: (code: string) => void) => {
    try {
      watchRef.current?.(); watchRef.current = null;
      const detail = await strategyRuntimeApi.getBacktestRun(id);
      const run = detail.run;
      if (run) {
        setRunMeta({
          symbol: run.symbol,
          timeframe: run.timeframe,
          createdAt: run.createdAt ? new Date(Number(run.createdAt.seconds) * 1000).toISOString() : undefined,
          name: run.name ?? undefined,
        });
      }
      if (detail.strategyCode && onCodeLoaded) {
        onCodeLoaded(detail.strategyCode);
      }
      if (detail.metrics) setMetrics(protoToMetrics(detail.metrics));
      if (detail.executionAssumptions) setExecutionAssumptions(detail.executionAssumptions);
      setRunId(id);
      setStatus('completed');
      const tr = await backtestRunsApi.getTrades(id);
      setChartTrades(tr.trades.map((t2) => ({
        side: t2.side, openTime: t2.open_ts, openPrice: t2.open_price,
        closeTime: t2.close_ts, closePrice: t2.close_price, pnl: t2.pnl, volume: t2.volume,
        ticket: t2.ticket, commission: t2.commission, reason: t2.reason,
      })));
    } catch { /* silent */ }
  }, []);

  return {
    initialCapital, setInitialCapital, leverage, setLeverage, lotSize, setLotSize,
    commission, setCommission, slippage, setSlippage, tradeDirection, setTradeDirection, strictMode, setStrictMode,
    standardParams, applyDefaults, applyPreset,
    startDate, setStartDate, endDate, setEndDate, datePreset, applyDatePreset, getTimeframeWarning,
    extractedParams, strategyParamValues, setParam, updateExtractedParams, updateDirectivesFromCode,
    run, submitting, status, metrics, executionAssumptions, errorMsg,
    runId, runMeta, fixDepth, chartTrades, blindSpots, resetStatus, cancelRun, restoreLastRun, loadRunById,
    gateUpdate, gateResults, qualityPreview, strategyDirectives,
    activeTab, setActiveTab, panelHeight, setPanelHeight, dragging, setDragging, userResized, setUserResized,
    strategyParamsModalOpen, setStrategyParamsModalOpen, settingsItems, tuning, gate, handleValidationResult,
  };
}
