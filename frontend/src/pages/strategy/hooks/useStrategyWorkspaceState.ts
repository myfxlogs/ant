import { useState, useCallback, useEffect, useMemo, useRef } from 'react';
import { useStrategyCode } from './useStrategyCode';
import { useBacktestRunner } from '@/components/backtest/useBacktestRunner';
import type { SweepDimension, BacktestMetrics, StrategyDirective, PresetKey, BacktestSubTab } from '@/components/backtest/useBacktestRunner';
import { useQuickTradeData } from './useQuickTradeData';
import type { QuickTradePosition, RecentTrade } from './useQuickTradeData';
import { useAIWorkflow } from './useAIWorkflow';
import { useHistoryState } from './useHistoryState';
import { useAccountSlice } from './useAccountSlice';
import { useTemplateSlice } from './useTemplateSlice';
import { useLayoutSlice } from './useLayoutSlice';
import { useWorkspaceEffects } from './useWorkspaceEffects';
import { useWorkspaceStore } from '@/stores/workspaceStore';

export type { SweepDimension, BacktestMetrics, StrategyDirective, PresetKey, BacktestSubTab, QuickTradePosition, RecentTrade };
export { DATE_PRESETS } from '@/components/backtest/useBacktestRunner';
export type BacktestStatus = 'idle' | 'running' | 'completed' | 'error';

export function useStrategyWorkspaceState() {
  const layout = useLayoutSlice();
  const account = useAccountSlice();
  const btCtx = useBacktestRunner();
  const onValidateResult = useCallback(
    (result: import('@/client/codeAssist').ValidateExtendedResult) => { btCtx.handleValidationResult(result); },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- btCtx.handleValidationResult is stable  | REF: rd.md#part-0.2-hooks-deps
    [btCtx.handleValidationResult],
  );
  const codeCtx = useStrategyCode({ onValidateResult });
  const templates = useTemplateSlice({
    handleLoadTemplate: codeCtx.handleLoadTemplate, validateCode: codeCtx.validateCode, updateExtractedParams: btCtx.updateExtractedParams,
  });
  const handleRunBacktest = useCallback((overrides?: {
    params?: Record<string, string>;
    executionConfig?: {
      commission: number;
      slippage: number;
      leverage: number;
      tradeDirection: string;
      strictMode: boolean;
    };
  }) => {
    btCtx.run({ strategyCode: codeCtx.code, symbol: account.symbol, accountId: account.accountId, timeframe: account.timeframe, templateId: templates.selectedId || undefined, strategyId: codeCtx.strategyId }, overrides);
  }, [codeCtx.code, codeCtx.strategyId, account, templates.selectedId, btCtx]);
  const handleRunTuning = useCallback(async (): Promise<string> => btCtx.tuning.runTuning({ code: codeCtx.code, symbol: account.symbol, timeframe: account.timeframe, startDate: btCtx.startDate, endDate: btCtx.endDate, templateId: templates.selectedId || undefined, strategyName: btCtx.runMeta?.name || codeCtx.loadedTemplate?.name || '', backtestRunId: btCtx.runId || '' }), [codeCtx.code, codeCtx.loadedTemplate, account, btCtx, templates.selectedId]);
  const qt = useQuickTradeData(account.accountId, account.symbol);
  const [btCollapsed, setBtCollapsed] = useState(false);
  const [autoExpandHistory, setAutoExpandHistory] = useState(false);
  const history = useHistoryState(account.accountId);
  const setCenterTab = useWorkspaceStore(s => s.setCenterTab);
  const ai = useAIWorkflow(codeCtx, btCtx.metrics, () => setCenterTab('chat'));

  // Refresh backtest history list when a run completes or errors.
  const prevStatusRef = useRef(btCtx.status);
  useEffect(() => {
    const prev = prevStatusRef.current;
    prevStatusRef.current = btCtx.status;
    if ((btCtx.status === 'completed' || btCtx.status === 'error') && prev === 'running') {
      history.refresh();
      setAutoExpandHistory(true);
    } else if (btCtx.status === 'idle' || btCtx.status === 'running') {
      setAutoExpandHistory(false);
    }
  }, [btCtx.status, history]);
  useWorkspaceEffects({ code: codeCtx.code, setCode: codeCtx.setCode, loadedTemplate: codeCtx.loadedTemplate, resetBacktestStatus: btCtx.resetStatus, activeAccounts: account.activeAccounts, accountId: account.accountId, setAccountId: account.setAccountId, setSymbol: account.setSymbol, fetchAccounts: account.fetchAccounts, loadTemplates: codeCtx.loadTemplates, datePreset: btCtx.datePreset, applyDatePreset: btCtx.applyDatePreset, financialsReady: qt.financialsReady, fetchTradeHistory: qt.fetchTradeHistory });
  const accountSlice = useMemo(() => ({ ...account, accountInfo: qt.accountInfo }), [account, qt.accountInfo]);
  const templatesSlice = useMemo(() => ({ list: codeCtx.templates, loading: codeCtx.templatesLoading, ...templates }), [codeCtx.templates, codeCtx.templatesLoading, templates]);
  const backtestSlice = useMemo(() => ({ runner: btCtx, submitting: btCtx.submitting, status: btCtx.status, metrics: btCtx.metrics, executionAssumptions: btCtx.executionAssumptions, error: btCtx.errorMsg, initialCapital: btCtx.initialCapital, setInitialCapital: btCtx.setInitialCapital, leverage: btCtx.leverage, setLeverage: btCtx.setLeverage, commission: btCtx.commission, setCommission: btCtx.setCommission, slippage: btCtx.slippage, setSlippage: btCtx.setSlippage, startDate: btCtx.startDate, setStartDate: btCtx.setStartDate, endDate: btCtx.endDate, setEndDate: btCtx.setEndDate, datePreset: btCtx.datePreset, tradeDirection: btCtx.tradeDirection, setTradeDirection: btCtx.setTradeDirection, strictMode: btCtx.strictMode, setStrictMode: btCtx.setStrictMode, strategyDirectives: btCtx.strategyDirectives, applyDatePreset: btCtx.applyDatePreset, applyPreset: btCtx.applyPreset, applyDefaults: btCtx.applyDefaults, getTimeframeWarning: btCtx.getTimeframeWarning, runId: btCtx.runId, runMeta: btCtx.runMeta, chartTrades: btCtx.chartTrades, run: handleRunBacktest, loadRunById: btCtx.loadRunById, btCollapsed, setBtCollapsed, activeTab: btCtx.activeTab, setActiveTab: btCtx.setActiveTab, panelHeight: btCtx.panelHeight, setPanelHeight: btCtx.setPanelHeight }), [btCtx, handleRunBacktest, btCollapsed]);
  const tuningSlice = useMemo(() => ({ subTab: btCtx.tuning.subTab, setSubTab: btCtx.tuning.setSubTab, method: btCtx.tuning.tuneMethod, setMethod: btCtx.tuning.setTuneMethod, sweepDimensions: btCtx.tuning.sweepDimensions, toggleDimension: btCtx.tuning.toggleDimension, enabledDims: btCtx.tuning.enabledSweepDims, cartesianSize: btCtx.tuning.cartesianSize, running: btCtx.tuning.tuningRunning, run: handleRunTuning }), [btCtx.tuning, handleRunTuning]);
  const gateSlice = useMemo(() => ({ loading: btCtx.gate.gateLoading, gates: btCtx.gate.gateGates, summary: btCtx.gate.gateSummary, error: btCtx.gate.gateError, run: btCtx.gate.runGate }), [btCtx.gate]);
  const quickTradeSlice = useMemo(() => ({ positionCount: qt.positionCount, allPositions: qt.allPositions, qtPositions: qt.qtPositions, qtRecentTrades: qt.qtRecentTrades, handleClosePosition: qt.handleClosePosition }), [qt.positionCount, qt.allPositions, qt.qtPositions, qt.qtRecentTrades, qt.handleClosePosition]);
  const layoutSlice = layout;
  // eslint-disable-next-line react-hooks/exhaustive-deps -- codeCtx.setCode is stable, full codeCtx not needed  | REF: rd.md#part-0.2-hooks-deps
  const aiSlice = useMemo(() => ({ optimize: ai.handleAIOptimize, optimizePrompt: ai.aiOptimizePrompt, askForValidation: ai.handleAskAIForValidation, chatAutoApply: ai.chatAutoApply, autoFixing: ai.autoFixing, autoFix: ai.handleAutoFix, autoFixDebug: ai.autoFixDebug, dismissDebug: ai.dismissDebug, applyTunedParams: (code: string) => { codeCtx.setCode(code); } }), [ai, codeCtx.setCode]);

  return {
    account: accountSlice,
    code: codeCtx,
    templates: templatesSlice,
    backtest: backtestSlice,
    tuning: tuningSlice,
    gate: gateSlice,
    quickTrade: quickTradeSlice,
    layout: layoutSlice,
    history,
    autoExpandHistory,
    ai: aiSlice,
  };
}
