import { useState, useEffect, useCallback, useMemo } from 'react';
import { useAccount } from '@/hooks/useAccount';
import { marketApi } from '@/client/market';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import { useStrategyCode } from './useStrategyCode';
import { useBacktestParams, DATE_PRESETS } from './useBacktestParams';
import type { SweepDimension, BacktestMetrics, StrategyDirective, PresetKey, BacktestSubTab } from './useBacktestParams';
import { useQuickTradeData } from './useQuickTradeData';
import type { QuickTradePosition, RecentTrade } from './useQuickTradeData';
import { useAIWorkflow } from './useAIWorkflow';

export type { SweepDimension, BacktestMetrics, StrategyDirective, PresetKey, BacktestSubTab, QuickTradePosition, RecentTrade };
export { DATE_PRESETS };
export type BacktestStatus = 'idle' | 'running' | 'completed' | 'error';

export function useStrategyWorkspaceState() {
  // Account
  const { accounts: allAccounts, fetchAccounts } = useAccount();
  const activeAccounts = useMemo(() => (allAccounts || []).filter(a => !a.isDisabled), [allAccounts]);
  const wsStore = useWorkspaceStore();
  const accountId = wsStore.accountId; const setAccountId = wsStore.setAccountId;
  const symbol = wsStore.symbol; const setSymbol = wsStore.setSymbol;
  const timeframe = wsStore.timeframe; const setTimeframe = wsStore.setTimeframe;

  const selectedAccountMeta = useMemo(() => {
    const a = activeAccounts.find(a => a.id === accountId);
    if (!a) return null;
    return { brokerCompany: a.brokerCompany, brokerServer: a.brokerServer, mtType: a.mtType, leverage: a.leverage ?? 0 };
  }, [activeAccounts, accountId]);

  const handleAccountChange = useCallback((id: string) => {
    setAccountId(id); setSymbol(''); marketApi.clearSymbolCache();
  }, [setAccountId, setSymbol]);

  // Backtest + Smart Tuning (must precede useStrategyCode for onValidateResult wiring)
  const btCtx = useBacktestParams();

  // Code + Templates + Save
  const codeCtx = useStrategyCode({
    onValidateResult: (result) => {
      if (result.sweepDimensions.length > 0) btCtx.updateSweepFromCode(result.sweepDimensions);
      if (result.strategyDirectives.length > 0) btCtx.updateStrategyDirectivesFromCode(result.strategyDirectives);
    },
  });
  const handleRunBacktest = useCallback(() => {
    btCtx.runBacktest({ code: codeCtx.code, symbol, accountId, timeframe });
  }, [codeCtx.code, symbol, accountId, timeframe, btCtx.runBacktest]);

  const handleRunTuning = useCallback(async (): Promise<string> => {
    return btCtx.runTuning({
      code: codeCtx.code, symbol, timeframe,
      startDate: btCtx.startDate, endDate: btCtx.endDate,
    });
  }, [codeCtx.code, symbol, timeframe, btCtx.startDate, btCtx.endDate, btCtx.runTuning]);

  // Quick Trade data
  const qt = useQuickTradeData(accountId, symbol);

  // Layout
  const codePanelVisible = wsStore.codePanelVisible; const setCodePanelVisible = wsStore.setCodePanelVisible;
  const quickTradeVisible = wsStore.quickTradeVisible; const setQuickTradeVisible = wsStore.setQuickTradeVisible;
  const positionsPanelVisible = wsStore.positionsPanelVisible; const setPositionsPanelVisible = wsStore.setPositionsPanelVisible;

  // History drawer
  const [historyDrawerOpen, setHistoryDrawerOpen] = useState(false);
  const [historyRunId, setHistoryRunId] = useState('');
  const handleOpenHistory = useCallback(async () => {
    try {
      const { pythonStrategyApi } = await import('@/client/pythonStrategy');
      const resp = await pythonStrategyApi.listBacktestRuns({ accountId, limit: 1 });
      if (resp.runs?.length) setHistoryRunId(resp.runs[0].id);
    } catch (e) { console.warn('fetch history run failed', e); }
    setHistoryDrawerOpen(true);
  }, [accountId]);
  const handleCloseHistory = useCallback(() => { setHistoryDrawerOpen(false); setHistoryRunId(''); }, []);
  const handleViewHistoryRun = useCallback((runId: string) => { setHistoryRunId(runId); setHistoryDrawerOpen(true); }, []);

  // AI workflow
  const ai = useAIWorkflow(codeCtx, btCtx.metrics, setCodePanelVisible);

  // Init
  useEffect(() => { fetchAccounts(); codeCtx.loadTemplates(); }, []);
  useEffect(() => { if (accountId && qt.financialsReady) qt.fetchTradeHistory(); }, [accountId, qt.financialsReady, qt.fetchTradeHistory]);
  useEffect(() => {
    const preset = DATE_PRESETS.find(p => p.key === btCtx.datePreset);
    if (preset) btCtx.applyDatePreset(preset);
  }, []);

  return {
    account: { activeAccounts, accountId, setAccountId, symbol, setSymbol, timeframe, setTimeframe, handleAccountChange, accountInfo: qt.accountInfo, selectedAccountMeta },
    code: codeCtx,
    backtest: {
      submitting: btCtx.submitting, status: btCtx.status, metrics: btCtx.metrics,
      executionAssumptions: btCtx.executionAssumptions, error: btCtx.errorMsg,
      initialCapital: btCtx.initialCapital, setInitialCapital: btCtx.setInitialCapital,
      leverage: btCtx.leverage, setLeverage: btCtx.setLeverage,
      commission: btCtx.commission, setCommission: btCtx.setCommission,
      slippage: btCtx.slippage, setSlippage: btCtx.setSlippage,
      startDate: btCtx.startDate, setStartDate: btCtx.setStartDate,
      endDate: btCtx.endDate, setEndDate: btCtx.setEndDate,
      datePreset: btCtx.datePreset, tradeDirection: btCtx.tradeDirection, setTradeDirection: btCtx.setTradeDirection,
      strictMode: btCtx.strictMode, setStrictMode: btCtx.setStrictMode,
      paramsExpanded: btCtx.paramsExpanded, setParamsExpanded: btCtx.setParamsExpanded,
      resultsExpanded: btCtx.resultsExpanded, setResultsExpanded: btCtx.setResultsExpanded,
      strategyDirectives: btCtx.strategyDirectives,
      applyDatePreset: btCtx.applyDatePreset, applyPreset: btCtx.applyPreset,
      applyDefaults: btCtx.applyDefaults, getTimeframeWarning: btCtx.getTimeframeWarning,
      runId: btCtx.backtestRunId, chartTrades: btCtx.chartTrades, run: handleRunBacktest,
    },
    tuning: {
      subTab: btCtx.subTab, setSubTab: btCtx.setSubTab,
      method: btCtx.tuneMethod, setMethod: btCtx.setTuneMethod,
      sweepDimensions: btCtx.sweepDimensions, toggleDimension: btCtx.toggleDimension,
      enabledDims: btCtx.enabledSweepDims, cartesianSize: btCtx.cartesianSize,
      running: btCtx.tuningRunning, run: handleRunTuning,
    },
    gate: { loading: btCtx.gateLoading, gates: btCtx.gateGates, summary: btCtx.gateSummary, error: btCtx.gateError, run: btCtx.runGate },
    quickTrade: { positionCount: qt.positionCount, allPositions: qt.allPositions, qtPositions: qt.qtPositions, qtRecentTrades: qt.qtRecentTrades, handleClosePosition: qt.handleClosePosition },
    layout: { codePanelVisible, setCodePanelVisible, positionsPanelVisible, setPositionsPanelVisible, quickTradeVisible, setQuickTradeVisible },
    history: { drawerOpen: historyDrawerOpen, runId: historyRunId, open: handleOpenHistory, close: handleCloseHistory },
    ai: { optimize: ai.handleAIOptimize, optimizePrompt: ai.aiOptimizePrompt, askForValidation: ai.handleAskAIForValidation, chatAutoApply: ai.chatAutoApply, autoFixing: ai.autoFixing, autoFix: ai.handleAutoFix, autoFixDebug: ai.autoFixDebug, dismissDebug: ai.dismissDebug, applyTunedParams: (code: string) => { codeCtx.setCode(code); } },
  };
}
