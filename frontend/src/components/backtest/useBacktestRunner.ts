import { useState, useCallback, useEffect, useRef, useMemo } from 'react';
import { message } from 'antd';
import { useTranslation } from 'react-i18next';
import {
  BACKTEST_FAILED_KEY, ENTER_CODE_AND_SYMBOL_KEY,
  DEFAULTS_SAVED_KEY, DEFAULTS_LOADED_KEY, DEFAULTS_RESET_KEY,
  SETTINGS_SAVE_KEY, SETTINGS_LOAD_KEY, SETTINGS_RESET_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import { pythonStrategyApi } from '@/client/pythonStrategy';
import { backtestRunsApi, type BacktestTrade } from '@/client/backtestRuns';
import type { BacktestRunUpdate } from '@/gen/ant/v1/backtest_run_query_pb';
import { isTerminalRun, isSucceededRun } from '@/pages/strategy/StrategyTemplatePage.utils';
import {
  backendDirectivesToStrategyDirectives,
  PRESETS, DATE_PRESETS, dateFromPreset,
  TIMEFRAME_MAX_MONTHS, type StrategyDirective, type PresetKey,
} from '@/pages/strategy/hooks/backtestParamHelpers';
import { useTuning } from '@/pages/strategy/hooks/useTuning';
import { useGateEvaluation } from '@/pages/strategy/hooks/useGateEvaluation';

export type { StrategyDirective, PresetKey };
export { PRESETS, DATE_PRESETS };
export type { SweepDimension, TuneMethod, BacktestSubTab } from '@/pages/strategy/hooks/useTuning';
export { OPTIMIZER_INFO } from '@/pages/strategy/hooks/useTuning';

export type BacktestStatus = 'idle' | 'running' | 'completed' | 'error';

export interface ChartTrade {
  side: string;
  openTime: number;
  openPrice: number;
  closeTime?: number;
  closePrice?: number;
  pnl?: number;
  volume?: number;
}

export interface BacktestMetrics {
  totalReturn?: number; annualReturn?: number; maxDrawdown?: number;
  sharpeRatio?: number; winRate?: number; totalTrades?: number;
  profitFactor?: number;
  winningTrades?: number; losingTrades?: number;
  averageProfit?: number; averageLoss?: number;
}

export interface ExtractedParam {
  name: string; type: string; default: string; label: string;
}

export interface StandardParams {
  initialCapital: number;
  leverage: number;
  lotSize: number;
  commission: number;
  slippage: number;
  tradeDirection: string;
  strictMode: boolean;
}

export const FACTORY_DEFAULTS: StandardParams = {
  initialCapital: 10000, leverage: 1, lotSize: 0.01,
  commission: 0.001, slippage: 0.0,
  tradeDirection: 'both', strictMode: true,
};

const DEFAULTS_KEY = 'ant_backtest_defaults';

function loadSavedDefaults(): StandardParams | null {
  try { const raw = localStorage.getItem(DEFAULTS_KEY); return raw ? JSON.parse(raw) : null; }
  catch { return null; }
}

function saveDefaults(vals: StandardParams) {
  try { localStorage.setItem(DEFAULTS_KEY, JSON.stringify(vals)); } catch { /* quota */ }
}

function removeDefaults() {
  try { localStorage.removeItem(DEFAULTS_KEY); } catch { /* ignore */ }
}

export interface BacktestRunnerInputs {
  strategyCode: string;
  accountId: string;
  symbol: string;
  timeframe: string;
  templateId?: string;
}

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
  const [executionAssumptions, setExecutionAssumptions] = useState<any>(null);
  const [errorMsg, setErrorMsg] = useState('');

  // ── Chart trades ──────────────────────────────────────────────────────
  const [chartTrades, setChartTrades] = useState<ChartTrade[]>([]);

  // ── Directives ────────────────────────────────────────────────────────
  const [strategyDirectives, setStrategyDirectives] = useState<StrategyDirective[]>([]);

  // ── UI state ──────────────────────────────────────────────────────────
  const [activeTab, setActiveTab] = useState<string>('params');
  const [panelHeight, setPanelHeight] = useState(280);
  const [dragging, setDragging] = useState(false);

  // ── Internal ──────────────────────────────────────────────────────────
  const [runId, setRunId] = useState('');
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

  const handleValidationResult = useCallback((result: any) => {
    if (result.sweepDimensions?.length > 0) tuning.updateSweepFromCode(result.sweepDimensions);
    if (result.strategyDirectives?.length > 0) updateDirectivesFromCode(result.strategyDirectives);
    updateExtractedParams((result as any).parametersJson || (result as any).parameters_json);
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

  const getTimeframeWarning = useCallback((timeframe: string, presetMonths: number): string | null => {
    const maxMonths = TIMEFRAME_MAX_MONTHS[timeframe];
    if (maxMonths && presetMonths > maxMonths) {
      return `${timeframe} timeframe recommends max ${maxMonths} month${maxMonths > 1 ? 's' : ''}`;
    }
    return null;
  }, []);

  const resetStatus = useCallback(() => {
    setStatus('idle'); setErrorMsg(''); setMetrics(null);
    setExecutionAssumptions(null); setChartTrades([]);
    setRunId('');
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
    const { strategyCode, accountId, symbol, timeframe, templateId } = inputs;
    if (!strategyCode || !symbol) { message.warning(t(ENTER_CODE_AND_SYMBOL_KEY)); return; }
    setSubmitting(true);
    setActiveTab('results');
    try {
      const result = await pythonStrategyApi.startBacktestRun({
        code: strategyCode, accountId, symbol, timeframe, initialCapital,
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
      setRunId(result.runId);
      setStatus('running');
      setExecutionAssumptions(null);
      watchRef.current?.();
      const stopWatching = await pythonStrategyApi.watchBacktestRun(result.runId, (update: BacktestRunUpdate) => {
        const run = update.run;
        if (run && isTerminalRun(run)) {
          const ok = isSucceededRun(run);
          setStatus(ok ? 'completed' : 'error');
          setMetrics(update.metrics ?? null);
          setExecutionAssumptions(update.executionAssumptions ?? null);
          setErrorMsg(update.run?.error ?? ''); stopWatching();
          watchRef.current = null;
          if (ok) {
            backtestRunsApi.getTrades(result.runId).then((tr) => {
              setChartTrades(tr.trades.map((t: BacktestTrade) => ({
                side: t.side,
                openTime: t.open_ts, openPrice: t.open_price,
                closeTime: t.close_ts, closePrice: t.close_price,
                pnl: t.pnl, volume: t.volume,
              })));
            }).catch(() => setChartTrades([]));
          } else { setChartTrades([]); }
        } else { setMetrics(update.metrics || null); }
      });
      watchRef.current = stopWatching;
    } catch (e: any) {
      message.error(e?.message || t(BACKTEST_FAILED_KEY));
      setStatus('error'); setErrorMsg(e?.message || 'Unknown error');
    } finally { setSubmitting(false); }
  }, [initialCapital, commission, slippage, leverage, lotSize, tradeDirection, strictMode, startDate, endDate, t]);

  // ── Return ────────────────────────────────────────────────────────────

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
    runId, chartTrades, resetStatus,
    // Directives
    strategyDirectives,
    // UI
    activeTab, setActiveTab, panelHeight, setPanelHeight, dragging, setDragging,
    // Settings
    settingsItems,
    // Delegated
    tuning, gate,
    // Validation wiring
    handleValidationResult,
  };
}
