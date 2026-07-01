import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useAccount } from '@/hooks/useAccount';
import { marketApi } from '@/client/market';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import { useStrategyCode } from './useStrategyCode';
import { useBacktestRunner, DATE_PRESETS } from '@/components/backtest/useBacktestRunner';
import type { SweepDimension, BacktestMetrics, StrategyDirective, PresetKey, ExtractedParam, BacktestSubTab } from '@/components/backtest/useBacktestRunner';
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
  const btCtx = useBacktestRunner();

  // Code + Templates + Save
  const codeCtx = useStrategyCode({
    onValidateResult: (result) => {
      btCtx.handleValidationResult(result);
    },
  });
  const [searchParams] = useSearchParams();
  const [selectedTemplateId, setSelectedTemplateId] = useState<string>('');

  const handleSelectTemplate = useCallback(async (templateId: string | null) => {
    if (!templateId) {
      setSelectedTemplateId('');
      btCtx.updateExtractedParams(null);
      return;
    }
    setSelectedTemplateId(templateId);
    const tpl = await codeCtx.handleLoadTemplate(templateId);
    if (!tpl) { setSelectedTemplateId(''); return; }
    // Populate strategy params from stored template metadata.
    if (tpl.parameters?.length) {
      const params = tpl.parameters.map((p: any) => ({
        name: p.name || '', type: p.type || 'string',
        default: p.default || '', label: p.label || p.name || '',
      }));
      btCtx.updateExtractedParams(params);
    } else if (tpl.code?.trim()) {
      // No stored parameters — auto-validate to extract ctx.Param() calls.
      codeCtx.validateCode(tpl.code);
    }
  }, [codeCtx.handleLoadTemplate, codeCtx.validateCode, btCtx.updateExtractedParams]);

  // Load template from URL on mount (e.g. from Library "Open in Workspace").
  useEffect(() => {
    const tid = searchParams.get('templateId');
    if (tid) handleSelectTemplate(tid);
  }, [searchParams, handleSelectTemplate]);

  const handleRunBacktest = useCallback(() => {
    btCtx.run({
      strategyCode: codeCtx.code, symbol, accountId, timeframe,
      templateId: selectedTemplateId || undefined,
      strategyId: codeCtx.strategyId,
    });
  }, [codeCtx.code, codeCtx.strategyId, symbol, accountId, timeframe, selectedTemplateId, btCtx.run]);

  const handleRunTuning = useCallback(async (): Promise<string> => {
    return btCtx.tuning.run({
      code: codeCtx.code, symbol, timeframe,
      startDate: btCtx.startDate, endDate: btCtx.endDate,
      templateId: selectedTemplateId || undefined,
    });
  }, [codeCtx.code, symbol, timeframe, btCtx.startDate, btCtx.endDate, selectedTemplateId, btCtx.tuning]);

  // Quick Trade data
  const qt = useQuickTradeData(accountId, symbol);

  // Layout
  const codePanelVisible = wsStore.codePanelVisible; const setCodePanelVisible = wsStore.setCodePanelVisible;
  const quickTradeVisible = wsStore.quickTradeVisible; const setQuickTradeVisible = wsStore.setQuickTradeVisible;
  const positionsPanelVisible = wsStore.positionsPanelVisible; const setPositionsPanelVisible = wsStore.setPositionsPanelVisible;

  // Backtest panel collapse state
  const [btCollapsed, setBtCollapsed] = useState(false);

  // History
  const [historyDrawerOpen, setHistoryDrawerOpen] = useState(false);
  const [historyRunId, setHistoryRunId] = useState('');
  const [historyModalOpen, setHistoryModalOpen] = useState(false);
  const [historyRuns, setHistoryRuns] = useState<any[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [historyPage, setHistoryPage] = useState(1);
  const [historyPageSize, setHistoryPageSize] = useState(20);
  const [historyTotal, setHistoryTotal] = useState(0);
  const [historySelectedKeys, setHistorySelectedKeys] = useState<React.Key[]>([]);
  const [historyDeleting, setHistoryDeleting] = useState(false);

  const fetchHistoryRuns = useCallback(async (page: number, pageSize: number) => {
    setHistoryLoading(true);
    try {
      const { strategyRuntimeApi } = await import('@/client/strategyRuntime');
      const resp = await strategyRuntimeApi.listBacktestRuns({ accountId: accountId || undefined, limit: pageSize, offset: (page - 1) * pageSize });
      const runs = resp.runs ?? [];
      setHistoryRuns(runs);
      // API has no total field — infer from returned count
      setHistoryTotal(runs.length < pageSize ? (page - 1) * pageSize + runs.length : page * pageSize + 1);
    } catch (e) {
      setHistoryRuns([]);
      console.error('fetchHistoryRuns failed', e);
    }
    finally { setHistoryLoading(false); }
  }, [accountId]);

  const handleOpenHistory = useCallback(() => {
    setHistoryModalOpen(true);
    setHistoryPage(1); setHistoryPageSize(20); setHistorySelectedKeys([]);
    fetchHistoryRuns(1, 20);
  }, [fetchHistoryRuns]);

  const handleCloseHistory = useCallback(() => { setHistoryDrawerOpen(false); setHistoryRunId(''); }, []);
  const handleCloseHistoryModal = useCallback(() => { setHistoryModalOpen(false); setHistorySelectedKeys([]); }, []);
  const handleViewHistoryRun = useCallback((runId: string) => { setHistoryRunId(runId); setHistoryDrawerOpen(true); }, []);

  const handleHistoryPageChange = useCallback((p: number, ps: number) => {
    setHistoryPage(p); setHistoryPageSize(ps); setHistorySelectedKeys([]);
    fetchHistoryRuns(p, ps);
  }, [fetchHistoryRuns]);

  const handleDeleteHistoryRun = useCallback(async (runId: string) => {
    setHistoryDeleting(true);
    try {
      const { strategyRuntimeApi } = await import('@/client/strategyRuntime');
      await strategyRuntimeApi.deleteBacktestRun(runId);
      // If page now empty, go back one page
      const newPage = historyRuns.length <= 1 && historyPage > 1 ? historyPage - 1 : historyPage;
      setHistoryPage(newPage);
      fetchHistoryRuns(newPage, historyPageSize);
    } catch (e) {
      console.error('deleteBacktestRun failed', e);
    }
    finally { setHistoryDeleting(false); }
  }, [historyRuns.length, historyPage, historyPageSize, fetchHistoryRuns]);

  const handleBatchDeleteHistory = useCallback(async () => {
    if (!historySelectedKeys.length) return;
    setHistoryDeleting(true);
    try {
      const { strategyRuntimeApi } = await import('@/client/strategyRuntime');
      await strategyRuntimeApi.deleteBacktestRuns(historySelectedKeys.map(String));
      setHistorySelectedKeys([]);
      const newPage = historyRuns.length <= historySelectedKeys.length && historyPage > 1 ? historyPage - 1 : historyPage;
      setHistoryPage(newPage);
      fetchHistoryRuns(newPage, historyPageSize);
    } catch (e) {
      console.error('deleteBacktestRuns failed', e);
    }
    finally { setHistoryDeleting(false); }
  }, [historySelectedKeys, historyRuns.length, historyPage, historyPageSize, fetchHistoryRuns]);

  // Refetch when account changes while modal is open
  useEffect(() => {
    if (historyModalOpen) { setHistoryPage(1); fetchHistoryRuns(1, historyPageSize); }
  }, [accountId]); // eslint-disable-line react-hooks/exhaustive-deps

  // AI workflow
  const ai = useAIWorkflow(codeCtx, btCtx.metrics, setCodePanelVisible);

  // Reset backtest state when code changes (AI apply, template load, manual edit).
  useEffect(() => {
    btCtx.resetStatus();
  }, [codeCtx.code]); // eslint-disable-line react-hooks/exhaustive-deps

  // Clear stale accountId from localStorage when the persisted account no longer
  // exists (deleted) or the user has no accounts — prevents 404/403 API calls.
  useEffect(() => {
    if (!accountId) return;
    if (activeAccounts.length === 0 || !activeAccounts.some(a => a.id === accountId)) {
      setAccountId(''); setSymbol('');
    }
  }, [activeAccounts, accountId, setAccountId, setSymbol]);

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
    templates: { list: codeCtx.templates, loading: codeCtx.templatesLoading, selectedId: selectedTemplateId, onSelect: handleSelectTemplate },
    backtest: {
      runner: btCtx,
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
      runId: btCtx.runId, chartTrades: btCtx.chartTrades, run: handleRunBacktest,
      btCollapsed, setBtCollapsed,
      activeTab: btCtx.activeTab, setActiveTab: btCtx.setActiveTab,
      panelHeight: btCtx.panelHeight, setPanelHeight: btCtx.setPanelHeight,
    },
    tuning: {
      subTab: btCtx.tuning.subTab, setSubTab: btCtx.tuning.setSubTab,
      method: btCtx.tuning.method, setMethod: btCtx.tuning.setMethod,
      sweepDimensions: btCtx.tuning.sweepDimensions, toggleDimension: btCtx.tuning.toggleDimension,
      enabledDims: btCtx.tuning.enabledDims, cartesianSize: btCtx.tuning.cartesianSize,
      running: btCtx.tuning.running, run: handleRunTuning,
    },
    gate: { loading: btCtx.gate.loading, gates: btCtx.gate.gates, summary: btCtx.gate.summary, error: btCtx.gate.error, run: btCtx.gate.run },
    quickTrade: { positionCount: qt.positionCount, allPositions: qt.allPositions, qtPositions: qt.qtPositions, qtRecentTrades: qt.qtRecentTrades, handleClosePosition: qt.handleClosePosition },
    layout: { codePanelVisible, setCodePanelVisible, positionsPanelVisible, setPositionsPanelVisible, quickTradeVisible, setQuickTradeVisible },
    history: {
      drawerOpen: historyDrawerOpen, runId: historyRunId,
      open: handleOpenHistory, close: handleCloseHistory,
      modalOpen: historyModalOpen, closeModal: handleCloseHistoryModal,
      runs: historyRuns, loading: historyLoading,
      page: historyPage, pageSize: historyPageSize, total: historyTotal,
      selectedRowKeys: historySelectedKeys, setSelectedRowKeys: setHistorySelectedKeys,
      deleting: historyDeleting,
      onPageChange: handleHistoryPageChange,
      onViewRun: handleViewHistoryRun,
      onDeleteRun: handleDeleteHistoryRun,
      onBatchDelete: handleBatchDeleteHistory,
      onRefresh: () => fetchHistoryRuns(historyPage, historyPageSize),
    },
    ai: { optimize: ai.handleAIOptimize, optimizePrompt: ai.aiOptimizePrompt, askForValidation: ai.handleAskAIForValidation, chatAutoApply: ai.chatAutoApply, autoFixing: ai.autoFixing, autoFix: ai.handleAutoFix, autoFixDebug: ai.autoFixDebug, dismissDebug: ai.dismissDebug, applyTunedParams: (code: string) => { codeCtx.setCode(code); } },
  };
}
