import { useState, useCallback, useEffect, useRef, useMemo } from 'react';
import { message, notification } from 'antd';
import { useTranslation } from 'react-i18next';
import {
  BACKTEST_FAILED_KEY, ENTER_CODE_AND_SYMBOL_KEY,
  DEFAULTS_SAVED_KEY, DEFAULTS_LOADED_KEY, DEFAULTS_RESET_KEY,
  SETTINGS_SAVE_KEY, SETTINGS_LOAD_KEY, SETTINGS_RESET_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import { BACKTEST_COMPLETED_KEY, BACKTEST_ERROR_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { TOTAL_RETURN_KEY } from '@/gen/ant/v1/i18n/strategy_backtest_keys';
import { strategyRuntimeApi } from '@/client/strategyRuntime';
import { backtestRunsApi, type BacktestTrade } from '@/client/backtestRuns';
import type { BacktestRunUpdate, MarketplaceQualityPreview } from '@/gen/ant/v1/backtest_run_query_pb';
import type { GateEvaluationUpdate, GateResult } from '@/gen/ant/v1/ai_gate_pb';
import { isTerminalRun, isSucceededRun } from '@/pages/strategy/StrategyTemplatePage.utils';
import {
  backendDirectivesToStrategyDirectives,
  PRESETS, DATE_PRESETS, dateFromPreset,
  type StrategyDirective, type PresetKey,
} from '@/pages/strategy/hooks/backtestParamHelpers';
import { useTuning } from '@/pages/strategy/hooks/useTuning';
import { useGateEvaluation } from '@/pages/strategy/hooks/useGateEvaluation';
import {
  type BacktestStatus, type ChartTrade, type BacktestMetrics,
  type ExtractedParam, type StandardParams, type BacktestRunnerInputs,
  FACTORY_DEFAULTS, loadSavedDefaults, saveDefaults, removeDefaults,
  getTimeframeWarning, protoToMetrics,
} from './backtestRunnerTypes';

export type { StrategyDirective, PresetKey };
export { PRESETS, DATE_PRESETS };
export type { SweepDimension, TuneMethod, BacktestSubTab } from '@/pages/strategy/hooks/useTuning';
export { OPTIMIZER_INFO } from '@/pages/strategy/hooks/useTuning';
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
  const [strategyParamsModalOpen, setStrategyParamsModalOpen] = useState(false);

  // ── Internal ──────────────────────────────────────────────────────────
  const [runId, setRunId] = useState('');
  const [fixDepth, setFixDepth] = useState(0);
  const [gateUpdate, setGateUpdate] = useState<GateEvaluationUpdate | null>(null);
  const [gateResults, setGateResults] = useState<GateResult[]>([]);
  const [qualityPreview, setQualityPreview] = useState<MarketplaceQualityPreview | null>(null);
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
  }, []);

  // ── Settings menu ─────────────────────────────────────────────────────

  const settingsItems = useMemo(() => [
    {
      key: 'save', label: t(SETTINGS_SAVE_KEY),
      onClick: () => { saveDefaults(standardParams); message.success(t(DEFAULTS_SAVED_KEY)); },
    },
    ...(saved ? [{
      key: 'load', label: t(SETTINGS_LOAD_KEY),
      onClick: () => { applyDefaults(saved); message.success(t(DEFAULTS_LOADED_KEY)); },
    }] : []),
    {
      key: 'reset', label: t(SETTINGS_RESET_KEY),
      onClick: () => { removeDefaults(); applyDefaults(FACTORY_DEFAULTS); message.success(t(DEFAULTS_RESET_KEY)); },
    },
  ], [t, standardParams, saved, applyDefaults]);

  // ── Run backtest ──────────────────────────────────────────────────────

  const run = useCallback(async (inputs: BacktestRunnerInputs) => {
    const { strategyCode, accountId, symbol, timeframe, templateId, strategyId } = inputs;
    if (!strategyCode || !symbol) { message.warning(t(ENTER_CODE_AND_SYMBOL_KEY)); return; }
    setSubmitting(true);
    setActiveTab('results');
    try {
      const result = await strategyRuntimeApi.startBacktestRun({
        code: strategyCode, accountId, symbol, timeframe, initialCapital,
        mode: 'KLINE_RANGE',
        from: startDate ? new Date(startDate) : undefined,
        to: endDate ? new Date(endDate) : undefined,
        templateId: templateId || undefined,
        strategyId: strategyId || undefined,
        autoGate: true,
        executionConfig: {
          commission, slippage, leverage,
          tradeDirection: tradeDirection as 'long' | 'short' | 'both',
          strictMode,
        },
      });
      if (!result.runId) throw new Error('No run ID');
      setRunId(result.runId);
      setStatus('running');
      setExecutionAssumptions(null);
      setGateUpdate(null);
      setGateResults([]);
      setQualityPreview(null);
      setFixDepth(0);
      watchRef.current?.();
      const stopWatching = await strategyRuntimeApi.watchBacktestRun(result.runId, (update: BacktestRunUpdate) => {
        const run = update.run;
        if (run?.fixDepth) setFixDepth(run.fixDepth);
        if (update.gateUpdate?.gate) setGateResults(prev => [...prev, update.gateUpdate!.gate!]);
        if (update.gateUpdate?.completed) setGateUpdate(update.gateUpdate);
        if (update.qualityPreview) setQualityPreview(update.qualityPreview);
        if (run && isTerminalRun(run)) {
          const ok = isSucceededRun(run);
          setStatus(ok ? 'completed' : 'error');
          setMetrics(protoToMetrics(update.metrics));
          setExecutionAssumptions(update.executionAssumptions ?? null);
          setErrorMsg(update.run?.error ?? ''); stopWatching();
          watchRef.current = null;
          if (ok) {
            const m = protoToMetrics(update.metrics);
            notification.success({ message: t(BACKTEST_COMPLETED_KEY), description: t(TOTAL_RETURN_KEY) + ': ' + ((m?.totalReturn ?? 0) * 100).toFixed(2) + '%', placement: 'bottomRight', duration: 4 });
            backtestRunsApi.getTrades(result.runId).then((tr) => {
              setChartTrades(tr.trades.map((t: BacktestTrade) => ({
                side: t.side,
                openTime: t.open_ts, openPrice: t.open_price,
                closeTime: t.close_ts, closePrice: t.close_price,
                pnl: t.pnl, volume: t.volume,
              })));
            }).catch(() => setChartTrades([]));
          } else { setChartTrades([]); }
          if (!ok) {
            notification.error({ message: t(BACKTEST_ERROR_KEY), description: update.run?.error || '', placement: 'bottomRight', duration: 6 });
          }
        } else { setMetrics(protoToMetrics(update.metrics)); }
      });
      watchRef.current = stopWatching;
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      message.error(msg || t(BACKTEST_FAILED_KEY));
      setStatus('error'); setErrorMsg(msg || 'Unknown error');
    } finally { setSubmitting(false); }
  }, [initialCapital, commission, slippage, leverage, tradeDirection, strictMode, startDate, endDate, t]);

  // ── Return ────────────────────────────────────────────────────────────

  const cancelRun = useCallback(async () => {
    if (!runId) return;
    try {
      await strategyRuntimeApi.cancelBacktestRun(runId);
      watchRef.current?.();
      watchRef.current = null;
      setStatus('idle');
      setRunId('');
      message.info(t('strategy.backtest.canceled', { defaultValue: 'Backtest canceled' }));
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      message.error(msg || t('strategy.backtest.cancelFailed', { defaultValue: 'Cancel failed' }));
    }
  }, [runId, t]);

  return {
    // Standard params
    initialCapital, setInitialCapital, leverage, setLeverage,
    lotSize, setLotSize, commission, setCommission, slippage, setSlippage,
    tradeDirection, setTradeDirection, strictMode, setStrictMode,
    standardParams, applyDefaults, applyPreset,
    // Date
    startDate, setStartDate, endDate, setEndDate,
    datePreset, applyDatePreset, getTimeframeWarning,
    // Strategy params
    extractedParams, strategyParamValues, setParam,
    updateExtractedParams, updateDirectivesFromCode,
    // Run
    run, submitting, status, metrics, executionAssumptions, errorMsg,
    runId, fixDepth, chartTrades, resetStatus,
    cancelRun,
    gateUpdate, gateResults, qualityPreview,
    // Directives
    strategyDirectives,
    // UI
    activeTab, setActiveTab, panelHeight, setPanelHeight, dragging, setDragging,
    strategyParamsModalOpen, setStrategyParamsModalOpen,
    // Settings
    settingsItems,
    // Delegated
    tuning, gate,
    // Validation wiring
    handleValidationResult,
  };
}
