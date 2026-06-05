import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { message } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/queries/queryKeys';
import { useAccount } from '@/hooks/useAccount';
import { useAccountFinancials } from '@/queries/useAccountFinancials';
import { usePositionsQuery } from '@/queries/usePositionsQuery';
import { marketApi } from '@/client/market';
import { tradingApi } from '@/client/trading';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import { useStrategyCode } from './useStrategyCode';
import { useBacktestParams, DATE_PRESETS } from './useBacktestParams';
import type { SweepDimension, BacktestMetrics, StrategyDirective, PresetKey, BacktestSubTab } from './useBacktestParams';

export type { SweepDimension, BacktestMetrics, StrategyDirective, PresetKey, BacktestSubTab };
export { DATE_PRESETS };
export type BacktestStatus = 'idle' | 'running' | 'completed' | 'error';

export interface QuickTradePosition {
  ticket: number; side: string; volume: number;
  openPrice: number; markPrice?: number; profit: number; leverage?: number;
}

export interface RecentTrade {
  ticket: number; symbol: string; side: string;
  closePrice?: number; price?: number; profit: number; closeTime?: string; created_at?: string;
}

export function useStrategyWorkspaceState() {
  // Account
  const { accounts: allAccounts, fetchAccounts } = useAccount();
  const activeAccounts = useMemo(() => (allAccounts || []).filter(a => !a.isDisabled), [allAccounts]);
  const wsStore = useWorkspaceStore();
  const accountId = wsStore.accountId;
  const setAccountId = wsStore.setAccountId;
  const symbol = wsStore.symbol;
  const setSymbol = wsStore.setSymbol;
  const timeframe = wsStore.timeframe;
  const setTimeframe = wsStore.setTimeframe;

  const selectedAccountMeta = useMemo(() => {
    const a = activeAccounts.find(a => a.id === accountId);
    if (!a) return null;
    return { brokerCompany: a.brokerCompany, brokerServer: a.brokerServer, mtType: a.mtType, leverage: a.leverage ?? 0 };
  }, [activeAccounts, accountId]);

  const handleAccountChange = useCallback((id: string) => {
    setAccountId(id); setSymbol(''); marketApi.clearSymbolCache();
  }, [setAccountId, setSymbol]);

  // Code + Templates + Save
  const codeCtx = useStrategyCode();

  // Backtest + Smart Tuning
  const btCtx = useBacktestParams();

  useEffect(() => {
    btCtx.updateSweepFromCode(codeCtx.code);
    btCtx.updateStrategyDirectivesFromCode(codeCtx.code);
  }, [codeCtx.code, btCtx.updateSweepFromCode, btCtx.updateStrategyDirectivesFromCode]);

  const handleRunBacktest = useCallback(() => {
    btCtx.setSubTab('results');
    btCtx.runBacktest({ code: codeCtx.code, accountId, symbol, timeframe });
  }, [btCtx, codeCtx.code, accountId, symbol, timeframe]);

  const handleRunTuning = useCallback(() => {
    btCtx.setSubTab('tuning');
    btCtx.runTuning({
      code: codeCtx.code,
      symbol,
      timeframe,
      startDate: btCtx.startDate,
      endDate: btCtx.endDate,
    });
  }, [btCtx, codeCtx.code, symbol, timeframe]);

  // Quick Trade data
  const { data: accountInfo, isSuccess: financialsReady } = useAccountFinancials(accountId);
  const { data: rawPositions } = usePositionsQuery(accountId);
  const positionCount = rawPositions?.length ?? 0;

  const allPositions: QuickTradePosition[] = useMemo(() => {
    if (!accountId || !rawPositions) return [];
    return rawPositions.map(p => ({
      ticket: p.ticket, side: p.type.startsWith('buy') ? 'long' : 'short',
      symbol: p.symbol, volume: p.volume || 0, openPrice: p.openPrice || 0,
      markPrice: p.currentPrice, profit: p.profit || 0,
    }));
  }, [rawPositions]);

  const qtPositions: QuickTradePosition[] = useMemo(() => {
    if (!symbol) return [];
    return (rawPositions || []).filter(p => p.symbol === symbol).map(p => ({
      ticket: p.ticket, side: p.type.startsWith('buy') ? 'long' : 'short',
      volume: p.volume || 0, openPrice: p.openPrice || 0,
      markPrice: p.currentPrice, profit: p.profit || 0, leverage: undefined,
    }));
  }, [symbol, rawPositions]);

  const tradeCacheRef = useRef<Set<string>>(new Set());
  const [qtRecentTrades, setQtRecentTrades] = useState<RecentTrade[]>([]);
  const fetchTradeHistory = useCallback(async () => {
    if (!accountId || !financialsReady) return;
    if (tradeCacheRef.current.has(accountId)) return;
    tradeCacheRef.current.add(accountId);
    try {
      const result = await tradingApi.getOrderHistory({ accountId, pageSize: 5 });
      setQtRecentTrades(result.orders.slice(0, 5).map(o => ({
        ticket: o.ticket, symbol: o.symbol || '', side: o.type || '',
        closePrice: o.closePrice, price: o.openPrice, profit: o.profit || 0,
        closeTime: o.closeTime ? new Date(o.closeTime * 1000).toISOString() : undefined,
        created_at: o.openTime ? new Date(o.openTime * 1000).toISOString() : undefined,
      })));
    } catch { /* silent */ }
  }, [accountId, financialsReady]);

  const queryClient = useQueryClient();
  const handleClosePosition = useCallback(async (ticket: number, volume?: number) => {
    if (!accountId) return;
    try {
      const result = await tradingApi.orderClose({ accountId, ticket: BigInt(ticket), volume });
      if (result.error) { message.error(result.message || result.error); } else {
        message.success(result.message || 'Position closed');
        queryClient.invalidateQueries({ queryKey: queryKeys.positions.byAccount(accountId) });
      }
    } catch (e: unknown) { message.error((e as Error)?.message || 'Close failed'); }
  }, [accountId, queryClient]);

  // Layout (persisted via workspaceStore).
  const codePanelVisible = wsStore.codePanelVisible;
  const setCodePanelVisible = wsStore.setCodePanelVisible;
  const quickTradeVisible = wsStore.quickTradeVisible;
  const setQuickTradeVisible = wsStore.setQuickTradeVisible;
  const [positionsPanelVisible, setPositionsPanelVisible] = useState(false);

  // History drawer.
  const [historyDrawerOpen, setHistoryDrawerOpen] = useState(false);
  const [historyRunId, setHistoryRunId] = useState('');
  const handleOpenHistory = useCallback(async () => {
    // Fetch most recent run ID and open drawer.
    try {
      const { pythonStrategyApi } = await import('@/client/pythonStrategy');
      const resp = await pythonStrategyApi.listBacktestRuns({ accountId, limit: 1 });
      const runs = resp.runs || [];
      if (runs.length > 0) setHistoryRunId(runs[0].id);
    } catch { /* no history available */ }
    setHistoryDrawerOpen(true);
  }, [accountId]);
  const handleCloseHistory = useCallback(() => { setHistoryDrawerOpen(false); setHistoryRunId(''); }, []);
  const handleViewHistoryRun = useCallback((runId: string) => {
    setHistoryRunId(runId); setHistoryDrawerOpen(true);
  }, []);

  // AI Optimize: store prompt for AIChatPanel to pick up.
  const [aiOptimizePrompt, setAiOptimizePrompt] = useState<string | null>(null);
  const [chatAutoApply, setChatAutoApply] = useState(true);
  const handleAIOptimize = useCallback(() => {
    const m = btCtx.metrics;
    if (!m) return;
    const prompt = [
      'Optimize the strategy based on these backtest results:',
      `Total Return: ${((m.totalReturn ?? 0) * 100).toFixed(2)}%`,
      `Max Drawdown: ${((m.maxDrawdown ?? 0) * 100).toFixed(2)}%`,
      `Sharpe Ratio: ${(m.sharpeRatio ?? 0).toFixed(3)}`,
      `Win Rate: ${((m.winRate ?? 0) * 100).toFixed(1)}%`,
      `Total Trades: ${m.totalTrades ?? 0}`,
      'Please suggest parameter improvements and generate the updated strategy code.',
    ].join('\n');
    setChatAutoApply(true);
    setAiOptimizePrompt(prompt);
    setCodePanelVisible(true);
  }, [btCtx.metrics, setCodePanelVisible]);

  // Ask AI for help with validation errors.
  const handleAskAIForValidation = useCallback(() => {
    const vr = codeCtx.validationResult;
    if (!vr || vr.valid) return;
    const errors = (vr.errors || []).map(e => `- ${e}`).join('\n');
    const warnings = (vr.warnings || []).map(w => `- ${w}`).join('\n');
    const prompt = [
      'I need help understanding and fixing validation issues in my Python trading strategy.',
      'Please analyze these issues and ask me clarifying questions about my trading logic,',
      'so I can explain what I intended. Help me fix them step by step.',
      '',
      '**Validation errors:**',
      errors || '(none)',
      '',
      warnings ? '**Warnings:**' : '',
      warnings || '',
      '',
      'Please ask me one question at a time. Do not rewrite the code yet — help me understand the problems first.',
    ].filter(Boolean).join('\n');
    setChatAutoApply(false);
    setAiOptimizePrompt(prompt);
    setCodePanelVisible(true);
  }, [codeCtx.validationResult, setCodePanelVisible]);

  // Auto-fix: AI iteratively fixes validation errors until code passes.
  const [autoFixing, setAutoFixing] = useState(false);
  const handleAutoFix = useCallback(async () => {
    const vr = codeCtx.validationResult;
    const currentCode = codeCtx.code;
    if (!vr || vr.valid || !currentCode) return;
    setAutoFixing(true);
    const maxIters = 3;
    let code = currentCode;
    let lastErrors: string[] = vr.errors || [];
    let lastWarnings: string[] = vr.warnings || [];
    let lastParams = vr.parameters || [];
    try {
      for (let iter = 1; iter <= maxIters; iter++) {
        // Build parameter hints for missing/required params.
        const paramHints = lastParams
          .filter(p => p.required || p.suggested !== undefined)
          .map(p => {
            const parts = [`@param ${p.key}`];
            if (p.type) parts.push(`type=${p.type}`);
            if (p.default !== undefined) parts.push(`default=${p.default}`);
            if (p.suggested !== undefined) parts.push(`suggested=${p.suggested}`);
            return parts.join(' ');
          });
        const errorsText = lastErrors.map(e => `- ${e}`).join('\n');
        const warningsText = lastWarnings.map(w => `- ${w}`).join('\n');
        const instruction = [
          'Fix ALL of the following validation errors in this Python trading strategy.',
          'Return the COMPLETE corrected code — do not omit any part.',
          '',
          '**Validation errors to fix:**',
          errorsText || '(none)',
          '',
          warningsText ? '**Warnings:**' : '',
          warningsText || '',
          '',
          paramHints.length ? '**Required parameters (add @param annotations at top of code):**' : '',
          ...paramHints.map(p => `  ${p}`),
          '',
          'Rules:',
          '1. Keep all existing logic unchanged unless it causes an error.',
          '2. Add missing @param annotations with reasonable defaults.',
          '3. Fix calculation errors (EMA, data length checks, etc).',
          '4. Return ONLY valid Python code — no explanations, no markdown.',
        ].filter(Boolean).join('\n');
        try {
          const { codeAssistApi } = await import('@/client/codeAssist');
          const result = await codeAssistApi.revise({ code, instruction });
          if (!result.python) throw new Error('AI returned no code');
          code = result.python;
          // Re-validate the fixed code.
          const recheck = await codeAssistApi.validateExtended(code);
          if (recheck.valid) {
            codeCtx.setCode(code);
            codeCtx.setLastValidatedCode(code);
            codeCtx.setValidationResult(recheck);
            message.success(`Auto-fix passed after ${iter} iteration${iter > 1 ? 's' : ''}`);
            setAutoFixing(false);
            return;
          }
          lastErrors = recheck.errors || [];
          lastWarnings = recheck.warnings || [];
          lastParams = recheck.parameters || [];
          codeCtx.setCode(code); // show progress
        } catch (e: unknown) {
          if (iter < maxIters) continue;
          throw e;
        }
      }
      // Max iterations reached — show remaining errors.
      codeCtx.setCode(code);
      codeCtx.setValidationResult({ valid: false, errors: lastErrors, warnings: lastWarnings, parameters: lastParams });
      message.warning(`Auto-fix: ${lastErrors.length} issue(s) remain after ${maxIters} iterations`);
    } catch (e: unknown) {
      message.error(e?.message || 'Auto-fix failed');
    } finally {
      setAutoFixing(false);
    }
  }, [codeCtx.code, codeCtx.validationResult, codeCtx.setCode]);

  // Init
  useEffect(() => { fetchAccounts(); codeCtx.loadTemplates(); }, []);
  useEffect(() => { if (accountId && financialsReady) fetchTradeHistory(); }, [accountId, financialsReady, fetchTradeHistory]);
  useEffect(() => {
    const preset = DATE_PRESETS.find(p => p.key === btCtx.datePreset);
    if (preset) btCtx.applyDatePreset(preset);
  }, []);

  return {
    activeAccounts, accountId, setAccountId, symbol, setSymbol, timeframe, setTimeframe, handleAccountChange,
    ...codeCtx,
    btSubmitting: btCtx.submitting, btStatus: btCtx.status, btMetrics: btCtx.metrics,
    btExecutionAssumptions: btCtx.executionAssumptions, btError: btCtx.errorMsg,
    btInitialCapital: btCtx.initialCapital, setBtInitialCapital: btCtx.setInitialCapital,
    btLeverage: btCtx.leverage, setBtLeverage: btCtx.setLeverage,
    btCommission: btCtx.commission, setBtCommission: btCtx.setCommission,
    btSlippage: btCtx.slippage, setBtSlippage: btCtx.setSlippage,
    btStartDate: btCtx.startDate, setBtStartDate: btCtx.setStartDate,
    btEndDate: btCtx.endDate, setBtEndDate: btCtx.setEndDate,
    btDatePreset: btCtx.datePreset, btTradeDirection: btCtx.tradeDirection, setBtTradeDirection: btCtx.setTradeDirection,
    btStrictMode: btCtx.strictMode, setBtStrictMode: btCtx.setStrictMode,
    btParamsExpanded: btCtx.paramsExpanded, setBtParamsExpanded: btCtx.setParamsExpanded,
    btResultsExpanded: btCtx.resultsExpanded, setBtResultsExpanded: btCtx.setResultsExpanded,
    btStrategyDirectives: btCtx.strategyDirectives,
    applyDatePreset: btCtx.applyDatePreset, applyPreset: btCtx.applyPreset,
    applyDefaults: btCtx.applyDefaults, getTimeframeWarning: btCtx.getTimeframeWarning,
    handleRunBacktest,
    backtestSubTab: btCtx.subTab, setBacktestSubTab: btCtx.setSubTab,
    tuneMethod: btCtx.tuneMethod, setTuneMethod: btCtx.setTuneMethod,
    sweepDimensions: btCtx.sweepDimensions, toggleDimension: btCtx.toggleDimension,
    enabledSweepDims: btCtx.enabledSweepDims, cartesianSize: btCtx.cartesianSize,
    tuningRunning: btCtx.tuningRunning, handleRunTuning,
    backtestRunId: btCtx.backtestRunId, gateLoading: btCtx.gateLoading,
    gateGates: btCtx.gateGates, gateSummary: btCtx.gateSummary,
    gateError: btCtx.gateError, handleRunGate: btCtx.runGate,
    accountInfo, selectedAccountMeta, positionCount, allPositions, qtPositions, qtRecentTrades, handleClosePosition,
    codePanelVisible, setCodePanelVisible, positionsPanelVisible, setPositionsPanelVisible, quickTradeVisible, setQuickTradeVisible,
    historyDrawerOpen, historyRunId, handleOpenHistory, handleCloseHistory, handleViewHistoryRun,
    handleAIOptimize, aiOptimizePrompt, handleAskAIForValidation, chatAutoApply,
    autoFixing, handleAutoFix,
    handleApplyTunedParams: (modifiedCode: string) => { codeCtx.setCode(modifiedCode); },
  };
}
