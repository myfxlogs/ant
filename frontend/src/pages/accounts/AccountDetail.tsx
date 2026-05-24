import { useEffect, useState, useCallback, useMemo } from 'react';
import { Tag, Button, Spin, Dropdown, Modal } from 'antd';
import { showSuccessModal, showErrorModal, showLoadingModal, showSuccess, showError } from '@/utils/message';
import type { MenuProps } from 'antd';
import {
  IconArrowLeft,
  IconRefresh,
  IconPlayerPause,
  IconPlayerPlay,
  IconDotsVertical,
  IconWallet,
  IconChartLine,
  IconTrendingUp,
  IconTrendingDown,
  IconCoin,
  IconPercentage,
  IconAlertTriangle,
  IconCloudDownload,
} from '@tabler/icons-react';
import { useParams, useNavigate } from 'react-router-dom';
import { useAccount } from '@/hooks/useAccount';
import { useTrading } from '@/hooks/useTrading';
import { useRealtimeUpdates } from '@/hooks/useRealtimeUpdates';
import { useTradingStore } from '@/stores/tradingStore';
import { useAccountStore } from '@/stores/accountStore';
import { useShallow } from 'zustand/react/shallow';
import { analyticsApi } from '@/client/analytics';
import { tradingApi } from '@/client/trading';
import AccountTradeTabs from './components/AccountTradeTabs';
import AccountAnalyticsSection from './components/AccountAnalyticsSection';
import {
  InfoCard,
  SmallInfoCard,
} from './components/AccountDetail.shared';
import { formatTimestamp, isPendingOrder } from './components/AccountDetail.utils';
import { getErrorMessage, translateMaybeI18nKey } from '@/utils/error';
import { useTranslation } from 'react-i18next';

export default function AccountDetail() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { currentAccount, fetchAccount, fetchAccounts, disableAccount, enableAccount, setCurrentAccount } = useAccount();
  const { connectAccount, fetchPositions, positions } = useTrading();
  const setCurrentAccountId = useTradingStore((state) => state.setCurrentAccountId);
  const accountInfo = useTradingStore(useShallow((state) => id ? state.accountInfoMap.get(id) : null));
  const hasReceivedData = useTradingStore((state) => state.hasReceivedData);
  const { connectionState } = useRealtimeUpdates(id);
  const enablingAccount = useAccountStore((state) => state.enablingAccount);
  
  const isDataReceived = id ? hasReceivedData(id) : true;
  // Show loading only while actively connecting. If already connected but no first stream frame yet,
  // still render snapshot/account values to avoid perpetual "loading..." cards.
  const isStreamLoading = !isDataReceived && connectionState === 'connecting';
  
  const [chartPeriod, setChartPeriod] = useState<'day' | 'week' | 'month' | 'all'>('month');
  const [chartType, setChartType] = useState<'equity' | 'balance' | 'profit'>('equity');

  const [selectedYear, setSelectedYear] = useState<number>(new Date().getFullYear());
  const [analyticsLoading, setAnalyticsLoading] = useState(false);
  const [analytics, setAnalytics] = useState<any>(null);
  const [monthlyPnL, setMonthlyPnL] = useState<any[]>([]);
  const [monthlyAnalysisYears, setMonthlyAnalysisYears] = useState<number[]>([]);
  const [monthlyAnalysisData, setMonthlyAnalysisData] = useState<any[]>([]);
  const [syncingHistory, setSyncingHistory] = useState(false);
  const [connecting, setConnecting] = useState(false);
  const [historyTrades, setHistoryTrades] = useState<any[]>([]);
  const [historyTotal, setHistoryTotal] = useState(0);
  const [historyPage, setHistoryPage] = useState(1);
  const historyPageSize = 10;

  const loadAllData = useCallback(async (accountId: string) => {
    setAnalyticsLoading(true);
    try {
      const [analyticsData, tradesData, monthlyData, monthlyAnalysisResp] = await Promise.all([
        analyticsApi.getAccountAnalytics(accountId),
        analyticsApi.getRecentTrades(accountId, 1, historyPageSize),
        analyticsApi.getMonthlyPnL(accountId, selectedYear),
        analyticsApi.getMonthlyAnalysis(accountId),
      ]);
      setAnalytics(analyticsData as any);
      setHistoryTrades((tradesData as any).trades || []);
      setHistoryTotal((tradesData as any).total || 0);
      setHistoryPage(1);
      setMonthlyPnL((monthlyData as any).monthlyPnl || []);
      setMonthlyAnalysisYears((monthlyAnalysisResp as any).years || []);
      setMonthlyAnalysisData((monthlyAnalysisResp as any).data || []);
    } catch (error) {
      showError(getErrorMessage(error, '加载分析数据失败'));
    } finally {
      setAnalyticsLoading(false);
    }
  }, [selectedYear]);

  useEffect(() => {
    if (!id) return;
    
    setCurrentAccountId(id);
    
    const init = async () => {
      // 并行加载所有数据，但不显示全局 loading
      const account = useAccountStore.getState().accounts.find(a => a.id === id);

      // 如果列表里已有该账户，直接用缓存填充 currentAccount，避免详情页一直等待
      if (account) {
        setCurrentAccount(account as any);
      }
      
      // 如果 store 中没有当前账户，才获取详情（不显示 loading）
      if (!account) {
        const loaded = await fetchAccount(id, false);
        if (!loaded) {
          showErrorModal(t('accounts.detail.messages.fetchAccountFailed'));
          navigate('/accounts');
          return;
        }
      }
      
      // 加载分析数据（不阻塞 UI）
      loadAllData(id).catch((error) => showError(getErrorMessage(error, '加载分析数据失败')));

      // 无条件拉持仓：useTrading.fetchPositions 内部已处理
      // 账户未连接→自动重连→重取 的兜底，账户禁用时后端返回空数组即可。
      // 此处若再加 status==='connected' 前置判断，会因缓存的账户 status
      // 过期而把首屏持仓“卡”住。
      if (!useAccountStore.getState().currentAccount?.isDisabled) {
        fetchPositions(id, false);
      }
    };
    
    init();
    
    const handlePositionChange = (event: Event) => {
      const customEvent = event as CustomEvent;
      const { action, order } = customEvent.detail;
      
      if (action === 'PositionClose' && order) {
        // Same rationale as init(): don't gate on the possibly-stale cached
        // status; fetchPositions handles not-connected via auto-reconnect.
        if (!useAccountStore.getState().currentAccount?.isDisabled) {
          fetchPositions(id);
        }
        
        const newTrade = {
          ticket: order.ticket,
          symbol: order.symbol,
          type: order.type,
          volume: order.volume,
          openPrice: order.openPrice,
          closePrice: order.closePrice,
          profit: order.profit,
          openTime: order.openTime,
          closeTime: order.closeTime,
          swap: order.swap || 0,
          commission: order.commission || 0,
          comment: order.comment || '',
        };
        
        setHistoryTrades((prev: any[]) => {
          const exists = prev.some((t: any) => t.ticket === order.ticket);
          if (exists) return prev;
          return [newTrade, ...prev];
        });

        // Keep all business metrics authoritative from backend.
        // Re-fetch analytics + recent trades/total instead of client-side accumulation.
        loadAllData(id).catch((error) => showError(getErrorMessage(error, '加载分析数据失败')));
      } else if (action === 'PositionOpen' || action === 'PendingOpen') {
        if (id) {
          if (!useAccountStore.getState().currentAccount?.isDisabled) {
            fetchPositions(id);
          }
          loadAllData(id).catch((error) => showError(getErrorMessage(error, '加载分析数据失败')));
        }
      }
    };
    
    window.addEventListener('position-change', handlePositionChange);
    
    return () => {
      // Keep positionsMap[id] intact so navigating back to the detail page
      // can render cached rows immediately while the next fetch is in flight.
      // Only detach from the "current" pointer; per-account cache survives.
      setCurrentAccountId(null);
      window.removeEventListener('position-change', handlePositionChange);
    };
  }, [id, loadAllData, fetchAccounts, fetchAccount, fetchPositions, setCurrentAccount, navigate, setCurrentAccountId, t]);

  const handleConnect = useCallback(async () => {
    if (!currentAccount || connecting) return;
    setConnecting(true);
    try {
      const result: any = await connectAccount(currentAccount.id);
      const msg = translateMaybeI18nKey(result?.message, t('common.operationFailed'));
      if (result?.success === false) {
        showError(msg);
      } else if (result?.message) {
        showSuccess(msg);
      }
      fetchPositions(currentAccount.id, false); // 连接后获取持仓，但不显示 loading
      await fetchAccount(currentAccount.id, false); // 获取账户详情，但不显示 loading
    } finally {
      setConnecting(false);
    }
  }, [currentAccount, connecting, connectAccount, fetchPositions, fetchAccount, t]);

  const handleRefreshAnalytics = useCallback(async () => {
    if (!id) return;
    await loadAllData(id);
  }, [id, loadAllData]);

  const handleToggleStatus = useCallback(async () => {
    if (!currentAccount) return;
    
    if (currentAccount.isDisabled) {
      const modal = showLoadingModal(t('accounts.messages.connectingMtServer'), t('common.pleaseWait'));
      try {
        await enableAccount(currentAccount.id);
        await fetchAccount(currentAccount.id, false); // 刷新账户信息，但不显示 loading
        modal.destroy();
        showSuccessModal(t('accounts.messages.enabledSuccess'));
      } catch (_error) {
        modal.destroy();
        showErrorModal(t('common.operationFailed'));
      }
    } else {
      try {
        await disableAccount(currentAccount.id);
        await fetchAccount(currentAccount.id, false); // 刷新账户信息，但不显示 loading
        showSuccessModal(t('accounts.messages.disabledSuccess'));
      } catch (_error) {
        showErrorModal(t('common.operationFailed'));
      }
    }
  }, [currentAccount, enableAccount, disableAccount, fetchAccount, t]);

  const handleSyncHistory = useCallback(async () => {
    if (!id) return;
    
    Modal.confirm({
      title: t('accounts.detail.syncHistory.title'),
      content: t('accounts.detail.syncHistory.content'),
      okText: t('accounts.detail.syncHistory.ok'),
      cancelText: t('common.cancel'),
      onOk: async () => {
        setSyncingHistory(true);
        try {
          await tradingApi.syncOrderHistory(id);
          showSuccess(t('accounts.detail.messages.syncHistorySuccess'));
          await loadAllData(id);
        } catch (_error) {
          showError(t('accounts.detail.messages.syncHistoryFailed'));
        } finally {
          setSyncingHistory(false);
        }
      },
    });
  }, [id, loadAllData, t]);

  const formatCurrency = useCallback((value: number) => {
    const isNegative = value < 0;
    return `${isNegative ? '-' : ''}${Math.abs(value).toFixed(2)} ${currentAccount?.currency || 'USD'}`;
  }, [currentAccount?.currency]);

  const statusConfig = useMemo(() => {
    if (!currentAccount) return { color: '#8A9AA5', bg: 'rgba(138, 154, 165, 0.1)', text: t('common.unknown') };
    if (currentAccount.isDisabled) return { color: '#8A9AA5', bg: 'rgba(138, 154, 165, 0.1)', text: t('accounts.detail.status.disabled') };
    switch (currentAccount.status) {
      case 'connected': return { color: '#00A651', bg: 'rgba(0, 166, 81, 0.1)', text: t('accounts.detail.status.connected') };
      case 'connecting': return { color: '#FF9800', bg: 'rgba(255, 152, 0, 0.1)', text: t('accounts.detail.status.connecting') };
      case 'disconnected': return { color: '#E53935', bg: 'rgba(229, 57, 53, 0.1)', text: t('accounts.detail.status.disconnected') };
      case 'error': return { color: '#E53935', bg: 'rgba(229, 57, 53, 0.1)', text: t('accounts.detail.status.error') };
      default: return { color: '#8A9AA5', bg: 'rgba(138, 154, 165, 0.1)', text: t('common.unknown') };
    }
  }, [currentAccount, t]);

  const menuItems: MenuProps['items'] = useMemo(() => [
    { 
      key: 'toggle', 
      label: currentAccount?.isDisabled ? t('accounts.detail.actions.enableAccount') : t('accounts.detail.actions.disableAccount'), 
      icon: currentAccount?.isDisabled ? (
        enablingAccount === currentAccount.id ? (
          <Spin size="small" />
        ) : (
          <IconPlayerPlay size={16} stroke={1.5} />
        )
      ) : (
        <IconPlayerPause size={16} stroke={1.5} />
      ), 
      onClick: handleToggleStatus 
    },
  ], [currentAccount?.isDisabled, currentAccount?.id, enablingAccount, handleToggleStatus, t]);

  const { equityChartData, profitByMonthData, symbolDistributionData, dailyPnLData, hourlyData, tradeStats, riskMetrics } = useMemo(() => {
    const equityCurve = analytics?.equityCurve?.map((point: any) => ({ date: point.date, equity: point.equity, balance: point.balance, profit: point.profit })) || [];
    const profitByMonth = monthlyPnL
      .map((m: any) => {
        const monthValue = (m as any)?.month ?? (m as any)?.monthNum ?? (m as any)?.month_num;
        return {
          month: String(monthValue ?? ''),
          profit: m.profit,
          trades: m.trades,
        };
      })
      .filter((m: any) => m.month);
    const dailyPnlRaw = analytics?.dailyPnl || [];
    const dailyPnl = dailyPnlRaw.map((d: any) => ({
      day: d.day,
      date: d.date,
      profit: d.pnl ?? d.profit,
      trades: d.trades,
      lots: d.lots ?? 0,
      balance: d.balance ?? 0,
      profitFactor: d.profitFactor ?? 0,
      maxFloatingLossAmount: d.maxFloatingLossAmount ?? 0,
      maxFloatingLossRatio: d.maxFloatingLossRatio ?? 0,
      maxFloatingProfitAmount: d.maxFloatingProfitAmount ?? 0,
      maxFloatingProfitRatio: d.maxFloatingProfitRatio ?? 0,
    }));
    
    const symbolStats = analytics?.symbolStats || [];
    const symbolDistribution = symbolStats
      .slice(0, 6)
      .map((s: any) => ({
        name: s.symbol,
        value: Math.round(Number(s.tradeSharePercent || 0)),
        profit: s.profit,
      }));
    
    return {
      equityChartData: equityCurve,
      profitByMonthData: profitByMonth,
      symbolDistributionData: symbolDistribution,
      dailyPnLData: dailyPnl,
      hourlyData: (analytics?.hourlyStats || []).map((h: any) => ({
        ...h,
        hourLabel: `${String(Number(h.hour ?? 0)).padStart(2, '0')}:00`,
        lots: h.lots ?? 0,
        balance: h.balance ?? 0,
        profitFactor: h.profitFactor ?? 0,
        maxFloatingLossAmount: h.maxFloatingLossAmount ?? 0,
        maxFloatingLossRatio: h.maxFloatingLossRatio ?? 0,
        maxFloatingProfitAmount: h.maxFloatingProfitAmount ?? 0,
        maxFloatingProfitRatio: h.maxFloatingProfitRatio ?? 0,
      })),
      tradeStats: analytics?.tradeStats || { totalTrades: 0, winRate: 0, profitFactor: 0, averageProfit: 0, averageLoss: 0, largestWin: 0, largestLoss: 0, maxConsecutiveWins: 0, maxConsecutiveLosses: 0, averageHoldingTime: '-', netProfit: 0, totalDeposit: 0, totalWithdrawal: 0, netDeposit: 0 },
      riskMetrics: analytics?.riskMetrics || { maxDrawdownPercent: 0, sharpeRatio: 0, sortinoRatio: 0, calmarRatio: 0, volatility: 0, averageDailyReturn: 0 },
    };
  }, [analytics, monthlyPnL]);

  const { realPositions, pendingOrders } = useMemo(() => {
    const positionsList = Array.isArray(positions) ? positions : [];
    const real = positionsList.map(p => ({ ...p, open_price: p.openPrice || p.openPrice || 0, current_price: p.closePrice || p.currentPrice || 0, open_time: formatTimestamp(p.openTime || p.openTime) })).filter(p => !isPendingOrder(p.type));
    const pending = positionsList.map(p => ({ ...p, open_price: p.openPrice || p.openPrice || 0, current_price: p.closePrice || p.currentPrice || 0, open_time: formatTimestamp(p.openTime || p.openTime) })).filter(p => isPendingOrder(p.type));
    return { realPositions: real, pendingOrders: pending };
  }, [positions]);

  const { balance, equity, margin, freeMargin, marginLevel, profit, profitPercent, credit } = useMemo(() => {
    const hasRealtimeData = Boolean(id && hasReceivedData && accountInfo);
    const b = hasRealtimeData ? (accountInfo?.balance ?? 0) : (currentAccount?.balance || 0);
    const e = hasRealtimeData ? (accountInfo?.equity ?? 0) : (currentAccount?.equity || 0);
    const m = hasRealtimeData ? (accountInfo?.margin ?? 0) : (currentAccount?.margin || 0);
    const fm = hasRealtimeData ? (accountInfo?.freeMargin ?? 0) : (currentAccount?.freeMargin || 0);
    const ml = hasRealtimeData ? (accountInfo?.marginLevel ?? 0) : (currentAccount?.marginLevel || 0);
    const p = hasRealtimeData ? (accountInfo?.profit ?? 0) : (currentAccount?.profit || 0);
    const pp = hasRealtimeData ? (accountInfo?.profitPercent ?? 0) : (currentAccount?.profitPercent || 0);
    const c = hasRealtimeData ? (accountInfo?.credit ?? 0) : (currentAccount?.credit || 0);
    return { balance: b, equity: e, margin: m, freeMargin: fm, marginLevel: ml, profit: p, profitPercent: pp, credit: c };
  }, [accountInfo, currentAccount, id, hasReceivedData]);

  // 只在首次加载时显示 loading，后续使用缓存数据
  if (!currentAccount) {
    return <div className="p-4 flex justify-center items-center h-64"><Spin size="large" /></div>;
  }

// ...
  return (
    <div className="min-h-screen" style={{ background: '#F5F7F9' }}>
      <div className="max-w-7xl mx-auto p-4">
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-4">
            <Button type="text" icon={<IconArrowLeft size={20} stroke={1.5} />} onClick={() => navigate('/accounts')} style={{ color: '#8A9AA5' }} />
            <div>
              <div className="flex items-center gap-3">
                <h1 className="text-2xl font-bold" style={{ color: '#141D22' }}>{currentAccount.login}</h1>
                <Tag color={currentAccount.mtType === 'MT4' ? 'blue' : 'purple'} style={{ borderRadius: '6px' }}>{currentAccount.mtType}</Tag>
                {currentAccount.accountType && <Tag style={{ borderRadius: '6px', background: currentAccount.accountType === 'real' ? 'rgba(229, 57, 53, 0.1)' : 'rgba(33, 150, 243, 0.1)', color: currentAccount.accountType === 'real' ? '#E53935' : '#2196F3', border: 'none' }}>{currentAccount.accountType === 'real' ? t('accounts.detail.accountType.real') : t('accounts.detail.accountType.demo')}</Tag>}
                <Tag style={{ borderRadius: '6px', background: currentAccount.isInvestor ? 'rgba(255, 152, 0, 0.1)' : 'rgba(0, 166, 81, 0.1)', color: currentAccount.isInvestor ? '#FF9800' : '#00A651', border: 'none' }}>{currentAccount.isInvestor ? t('accounts.detail.mode.investor') : t('accounts.detail.mode.trader')}</Tag>
                <Tag style={{ background: statusConfig.bg, color: statusConfig.color, border: 'none', borderRadius: '6px', cursor: currentAccount.status === 'disconnected' || currentAccount.status === 'error' ? 'pointer' : 'default' }} onClick={() => { if (currentAccount.status === 'disconnected' || currentAccount.status === 'error') handleConnect(); }}>{connecting ? t('accounts.detail.status.connecting') : statusConfig.text}</Tag>
              </div>
              <div className="flex items-center gap-4 mt-1" style={{ color: '#8A9AA5', fontSize: '14px' }}><span>{currentAccount.brokerCompany}</span><span>•</span><span>{currentAccount.brokerServer}</span><span>•</span><span>{t('accounts.detail.leverage', { leverage: currentAccount.leverage })}</span></div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button icon={<IconRefresh size={16} stroke={1.5} />} onClick={handleRefreshAnalytics} loading={analyticsLoading} style={{ borderRadius: '8px' }}>{t('common.refresh')}</Button>
            <Button icon={<IconCloudDownload size={16} stroke={1.5} />} onClick={handleSyncHistory} loading={syncingHistory} disabled={currentAccount.status !== 'connected'} style={{ borderRadius: '8px' }}>{t('accounts.detail.actions.syncHistory')}</Button>
            <Dropdown menu={{ items: menuItems }} trigger={['click']}><Button icon={<IconDotsVertical size={16} stroke={1.5} />} style={{ borderRadius: '8px' }} /></Dropdown>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
          <InfoCard icon={<IconWallet size={18} stroke={1.5} color="#8A9AA5" />} label={t('accounts.detail.cards.balance')} value={formatCurrency(balance)} loading={isStreamLoading} />
          <InfoCard icon={<IconChartLine size={18} stroke={1.5} color="#8A9AA5" />} label={t('accounts.detail.cards.equity')} value={formatCurrency(equity)} loading={isStreamLoading} />
          <div className="rounded-2xl p-5" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}>
            <div className="flex items-center gap-2 mb-3">{profit >= 0 ? <IconTrendingUp size={18} stroke={1.5} color="#00A651" /> : <IconTrendingDown size={18} stroke={1.5} color="#E53935" />}<span style={{ color: '#8A9AA5', fontSize: '14px' }}>{t('accounts.detail.cards.floatingProfit')}</span></div>
            {isStreamLoading ? <div className="text-lg" style={{ color: '#8A9AA5' }}>{t('common.loading')}</div> : <div className="flex items-baseline gap-2"><span className="text-2xl font-bold" style={{ color: profit >= 0 ? '#00A651' : '#E53935' }}>{profit >= 0 ? '+' : ''}{formatCurrency(profit)}</span><span style={{ color: profit >= 0 ? '#00A651' : '#E53935', fontSize: '14px' }}>({profitPercent >= 0 ? '+' : ''}{profitPercent.toFixed(2)}%)</span></div>}
          </div>
        </div>

        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
          <SmallInfoCard icon={<IconCoin size={16} stroke={1.5} color="#8A9AA5" />} label={t('accounts.detail.cards.marginUsed')} value={formatCurrency(margin)} loading={isStreamLoading} />
          <SmallInfoCard icon={<IconCoin size={16} stroke={1.5} color="#8A9AA5" />} label={t('accounts.detail.cards.marginFree')} value={formatCurrency(freeMargin)} loading={isStreamLoading} />
          <SmallInfoCard icon={<IconPercentage size={16} stroke={1.5} color="#8A9AA5" />} label={t('accounts.detail.cards.marginLevel')} value={margin > 0 ? `${(marginLevel || 0).toFixed(2)}%` : '--'} loading={isStreamLoading} valueColor={margin > 0 && (marginLevel || 0) < 100 ? '#E53935' : '#141D22'} />
          <SmallInfoCard icon={<IconAlertTriangle size={16} stroke={1.5} color="#8A9AA5" />} label={t('accounts.detail.cards.credit')} value={formatCurrency(credit)} loading={isStreamLoading} />
        </div>

        <div className="rounded-2xl overflow-hidden mb-6" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}>
          <AccountTradeTabs
            id={id}
            realPositions={realPositions}
            pendingOrders={pendingOrders}
            historyTrades={historyTrades}
            historyTotal={historyTotal}
            historyPage={historyPage}
            historyPageSize={historyPageSize}
            onHistoryTradesChange={setHistoryTrades}
            onHistoryTotalChange={setHistoryTotal}
            onHistoryPageChange={setHistoryPage}
          />
        </div>

        <AccountAnalyticsSection
          analyticsLoading={analyticsLoading}
          chartType={chartType}
          setChartType={setChartType}
          chartPeriod={chartPeriod}
          setChartPeriod={setChartPeriod}
          selectedYear={selectedYear}
          setSelectedYear={setSelectedYear}
          equityChartData={equityChartData}
          profitByMonthData={profitByMonthData}
          symbolDistributionData={symbolDistributionData}
          dailyPnLData={dailyPnLData}
          hourlyData={hourlyData}
          tradeStats={tradeStats}
          riskMetrics={riskMetrics}
          monthlyAnalysisYears={monthlyAnalysisYears}
          monthlyAnalysisData={monthlyAnalysisData}
          currency={currentAccount?.currency || 'USD'}
          accountId={id}
        />
    </div>
  </div>
);
}