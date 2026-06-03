import { useState, useEffect, useCallback, useMemo } from 'react';
import { message } from 'antd';
import { useAccount } from '@/hooks/useAccount';
import { useTradingStore } from '@/stores/tradingStore';
import { useAccountFinancials } from '@/queries/useAccountFinancials';
import { marketApi } from '@/client/market';
import { tradingApi } from '@/client/trading';
import { codeAssistApi } from '@/client/codeAssist';
import { useStrategyCode } from './useStrategyCode';
import { useBacktestParams, DATE_PRESETS } from './useBacktestParams';
import type { SweepDimension, BacktestMetrics } from './useBacktestParams';

export type { SweepDimension, BacktestMetrics };
export { DATE_PRESETS };
export type BacktestStatus = 'idle' | 'running' | 'completed' | 'error';
export type BacktestSubTab = 'results' | 'tuning';

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
  const [accountId, setAccountId] = useState('');
  const [symbol, setSymbol] = useState('');
  const [timeframe, setTimeframe] = useState('1h');

  // Selected account metadata for QuickTrade (broker, leverage, mtType)
  const selectedAccountMeta = useMemo(() => {
    const a = activeAccounts.find(a => a.id === accountId);
    if (!a) return null;
    return {
      brokerCompany: a.brokerCompany,
      brokerServer: a.brokerServer,
      mtType: a.mtType,
      leverage: a.leverage ?? 0,
    };
  }, [activeAccounts, accountId]);

  const handleAccountChange = useCallback((id: string) => {
    setAccountId(id); setSymbol(''); marketApi.clearSymbolCache();
  }, []);

  // Code + Templates + Save
  const codeCtx = useStrategyCode();

  // Backtest + Smart Tuning
  const btCtx = useBacktestParams();

  const handleRunBacktest = useCallback(() => {
    btCtx.setSubTab('results');
    btCtx.runBacktest({ code: codeCtx.code, accountId, symbol, timeframe });
  }, [btCtx, codeCtx.code, accountId, symbol, timeframe]);

  // AI Generate
  const [aiPrompt, setAiPrompt] = useState('');
  const [aiGenerating, setAiGenerating] = useState(false);
  const handleGenerateCode = useCallback(async () => {
    if (!aiPrompt.trim()) return;
    setAiGenerating(true);
    try {
      const revised = await codeAssistApi.revise({ code: codeCtx.code, prompt: aiPrompt });
      if (revised?.code) codeCtx.setCode(revised.code);
    } catch (e: any) { message.error(e?.message || 'AI generation failed'); }
    finally { setAiGenerating(false); }
  }, [codeCtx.code, aiPrompt]);

  // Quick Trade data
  const tradingStore = useTradingStore();
  // Account financials flow through SSE → TanStack Query cache (bridgeStreamEvents.handleProfitUpdate).
  // tradingStore.setAccountInfoById is a dead path with zero callers — use the live query instead.
  const { data: accountInfo } = useAccountFinancials(accountId);

  const qtPositions: QuickTradePosition[] = useMemo(() => {
    if (!accountId || !symbol) return [];
    return (tradingStore.positionsMap.get(accountId) || [])
      .filter(p => p.symbol === symbol)
      .map(p => ({
        ticket: p.ticket, side: p.type.startsWith('buy') ? 'long' : 'short',
        volume: p.volume || 0, openPrice: p.openPrice || 0,
        markPrice: p.currentPrice, profit: p.profit || 0,
        leverage: undefined,
      }));
  }, [accountId, symbol, tradingStore.positionsMap]);

  const [qtRecentTrades, setQtRecentTrades] = useState<RecentTrade[]>([]);
  const fetchTradeHistory = useCallback(async () => {
    if (!accountId) return;
    // Only fetch if account has received data (connected)
    if (!tradingStore.hasReceivedData(accountId)) return;
    try {
      const result = await tradingApi.getOrderHistory({ accountId, pageSize: 5 });
      const trades: RecentTrade[] = (result.orders as any[] || []).slice(0, 5).map((o: any) => ({
        ticket: o.ticket, symbol: o.symbol || '', side: o.type || '',
        closePrice: o.closePrice, price: o.openPrice, profit: o.profit || 0,
        closeTime: o.closeTime ? new Date(o.closeTime * 1000).toISOString() : undefined,
        created_at: o.openTime ? new Date(o.openTime * 1000).toISOString() : undefined,
      }));
      setQtRecentTrades(trades);
    } catch { /* silent */ }
  }, [accountId, tradingStore]);

  const handleClosePosition = useCallback(async (ticket: number) => {
    if (!accountId) return;
    try { await tradingApi.orderClose({ accountId, ticket: BigInt(ticket) }); }
    catch (e: any) { message.error(e?.message || 'Close failed'); }
  }, [accountId]);

  // Layout
  const [codePanelVisible, setCodePanelVisible] = useState(true);
  const [quickTradeVisible, setQuickTradeVisible] = useState(true);

  // Init
  useEffect(() => { fetchAccounts(); codeCtx.loadTemplates(); }, []);
  useEffect(() => { if (accountId) fetchTradeHistory(); }, [accountId]);
  useEffect(() => {
    const preset = DATE_PRESETS.find(p => p.key === btCtx.datePreset);
    if (preset) btCtx.applyDatePreset(preset);
  }, []);

  return {
    activeAccounts, accountId, setAccountId, symbol, setSymbol, timeframe, setTimeframe, handleAccountChange,
    ...codeCtx,
    btSubmitting: btCtx.submitting, btStatus: btCtx.status, btMetrics: btCtx.metrics, btError: btCtx.errorMsg,
    btInitialCapital: btCtx.initialCapital, setBtInitialCapital: btCtx.setInitialCapital,
    btLeverage: btCtx.leverage, setBtLeverage: btCtx.setLeverage,
    btCommission: btCtx.commission, setBtCommission: btCtx.setCommission,
    btSlippage: btCtx.slippage, setBtSlippage: btCtx.setSlippage,
    btStartDate: btCtx.startDate, setBtStartDate: btCtx.setStartDate,
    btEndDate: btCtx.endDate, setBtEndDate: btCtx.setEndDate,
    btDatePreset: btCtx.datePreset, btTradeDirection: btCtx.tradeDirection, setBtTradeDirection: btCtx.setTradeDirection,
    btHighPrecision: btCtx.highPrecision, setBtHighPrecision: btCtx.setHighPrecision,
    btParamsExpanded: btCtx.paramsExpanded, setBtParamsExpanded: btCtx.setParamsExpanded,
    btResultsExpanded: btCtx.resultsExpanded, setBtResultsExpanded: btCtx.setResultsExpanded,
    applyDatePreset: btCtx.applyDatePreset,
    handleRunBacktest,
    backtestSubTab: btCtx.subTab, setBacktestSubTab: btCtx.setSubTab,
    tuneMethod: btCtx.tuneMethod, setTuneMethod: btCtx.setTuneMethod,
    sweepDimensions: btCtx.sweepDimensions, toggleDimension: btCtx.toggleDimension,
    enabledSweepDims: btCtx.enabledSweepDims, cartesianSize: btCtx.cartesianSize,
    tuningRunning: btCtx.tuningRunning, handleRunTuning: btCtx.runTuning,
    aiPrompt, setAiPrompt, aiGenerating, handleGenerateCode,
    accountInfo, selectedAccountMeta, qtPositions, qtRecentTrades, handleClosePosition,
    codePanelVisible, setCodePanelVisible, quickTradeVisible, setQuickTradeVisible,
  };
}
