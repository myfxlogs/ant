import { useEffect, useState, useCallback, useMemo, useRef } from 'react';
import { Tag, Button, Spin, Dropdown, Modal, Input } from 'antd';
import { showSuccessModal, showErrorModal, showLoadingModal, showSuccess, showError } from '@/utils/message';
import type { MenuProps } from 'antd';
import {
  ArrowLeftOutlined,
  ReloadOutlined,
  PauseCircleOutlined,
  CaretRightOutlined,
  MoreOutlined,
  WalletOutlined,
  LineChartOutlined,
  RiseOutlined,
  FallOutlined,
  DollarOutlined,
  PercentageOutlined,
  WarningOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import { useParams, useNavigate } from 'react-router-dom';
import { useAccount } from '@/hooks/useAccount';
import { useTrading } from '@/hooks/useTrading';
import { useRealtimeUpdates } from '@/hooks/useRealtimeUpdates';
import { useTradingStore } from '@/stores/tradingStore';
import { useAccountStore } from '@/stores/accountStore';
import { useShallow } from 'zustand/react/shallow';
import { analyticsApi } from '@/client/analytics';
import type { ConnectAccountResult } from '@/client/account';
import AccountTradeTabs from './components/AccountTradeTabs';
import AccountAnalyticsSection from './components/AccountAnalyticsSection';
import {
  InfoCard,
  SmallInfoCard,
} from './components/AccountDetail.shared';
import { formatTimestamp, isPendingOrder } from './components/AccountDetail.utils';
import { getErrorMessage, translateMaybeI18nKey } from '@/utils/error';
import { useTranslation } from 'react-i18next';

// Analytics proto types are not yet generated (backend stubs).
// Define local interfaces matching the expected API response shapes.
interface AccountAnalyticsData {
  tradeStats?: AccountTradeStats;
  riskMetrics?: AccountRiskMetrics;
  symbolStats?: AccountSymbolStat[];
  equityCurve?: AccountEquityPoint[];
  dailyPnl?: AccountDailyPnlItem[];
  hourlyStats?: AccountHourlyStat[];
}

interface AccountTradeStats {
  totalTrades?: number;
  winRate?: number;
  profitFactor?: number;
  averageProfit?: number;
  averageLoss?: number;
  largestWin?: number;
  largestLoss?: number;
  maxConsecutiveWins?: number;
  maxConsecutiveLosses?: number;
  averageHoldingTime?: string;
  netProfit?: number;
  totalDeposit?: number;
  totalWithdrawal?: number;
  netDeposit?: number;
}

interface AccountRiskMetrics {
  maxDrawdownPercent?: number;
  sharpeRatio?: number;
  sortinoRatio?: number;
  calmarRatio?: number;
  volatility?: number;
  averageDailyReturn?: number;
}

interface AccountSymbolStat {
  symbol: string;
  profit: number;
  tradeSharePercent?: number;
}

interface AccountEquityPoint {
  date: string;
  equity: number;
  balance: number;
  profit: number;
}

interface AccountDailyPnlItem {
  day: string;
  date: string;
  pnl?: number;
  profit?: number;
  trades: number;
  lots?: number;
  balance?: number;
  profitFactor?: number;
  maxFloatingLossAmount?: number;
  maxFloatingLossRatio?: number;
  maxFloatingProfitAmount?: number;
  maxFloatingProfitRatio?: number;
}

interface AccountHourlyStat {
  hour?: string;
  lots?: number;
  balance?: number;
  profitFactor?: number;
  maxFloatingLossAmount?: number;
  maxFloatingLossRatio?: number;
  maxFloatingProfitAmount?: number;
  maxFloatingProfitRatio?: number;
}

interface AccountRecentTradesResponse {
  trades?: Array<{
    ticket: bigint | number;
    symbol: string;
    type: string;
    volume: number;
    openPrice: number;
    closePrice: number;
    profit: number;
    openTime?: string | Date;
    closeTime?: string | Date;
    swap?: number;
    commission?: number;
    comment?: string;
  }>;
  total?: number;
}

interface AccountMonthlyPnLItem {
  month?: string;
  monthNum?: number;
  month_num?: number;
  profit: number;
  trades: number;
}

interface AccountMonthlyPnLResponse {
  monthlyPnl?: AccountMonthlyPnLItem[];
}

interface AccountMonthlyAnalysisResponse {
  years?: number[];
  data?: unknown[];
}

export default function AccountDetail() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { currentAccount, fetchAccount, fetchAccounts, disableAccount, enableAccount, deleteAccount, setCurrentAccount } = useAccount();
  const { connectAccount, positions, fetchPositions } = useTrading();
  const setCurrentAccountId = useTradingStore((state) => state.setCurrentAccountId);
  const accountInfo = useTradingStore(useShallow((state) => id ? state.accountInfoMap.get(id) : null));
  const hasReceivedData = useTradingStore((state) => state.hasReceivedData);
  const { connectionState } = useRealtimeUpdates(id);
  const enablingAccount = useAccountStore((state) => state.enablingAccount);
  
  const isDataReceived = id ? hasReceivedData(id) : true;
  // Show loading only while actively connecting. If already connected but no first stream frame yet,
  // still render snapshot/account values to avoid perpetual "loading..." cards.
  const isStreamLoading = !isDataReceived;
  
  const [chartType, setChartType] = useState<'equity' | 'balance' | 'profit'>('equity');
  const [chartPeriod, setChartPeriod] = useState<'day' | 'week' | 'month' | 'all'>('month');

  // selectedYear 已下移到 AccountAnalyticsSection 内部管理，不再在 AccountDetail 层控制
  const [analyticsLoading, setAnalyticsLoading] = useState(false);
  const [analyticsError, setAnalyticsError] = useState<string | null>(null);
  const [analytics, setAnalytics] = useState<AccountAnalyticsData | null>(null);
  const [monthlyPnL, setMonthlyPnL] = useState<AccountMonthlyPnLItem[]>([]);
  const [monthlyAnalysisYears, setMonthlyAnalysisYears] = useState<number[]>([]);
  const [monthlyAnalysisData, setMonthlyAnalysisData] = useState<unknown[]>([]);
  const [connecting, setConnecting] = useState(false);
  const [disabling, setDisabling] = useState(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deletePassword, setDeletePassword] = useState('');
  const [deleting, setDeleting] = useState(false);
  const [historyTrades, setHistoryTrades] = useState<NonNullable<AccountRecentTradesResponse['trades']>>([]);
  const [historyTotal, setHistoryTotal] = useState(0);
  const [historyPage, setHistoryPage] = useState(1);
  const [historyLoading, setHistoryLoading] = useState(false);
  const historyPageSize = 10;
  const lastAnalyticsReloadRef = useRef(0);
  const ANALYTICS_RELOAD_MIN_INTERVAL_MS = 5000;

  const loadHistory = useCallback(async (accountId: string, page: number) => {
    setHistoryLoading(true);
    try {
      const tradesData = await analyticsApi.getRecentTrades(accountId, page, historyPageSize);
      setHistoryTrades((tradesData as AccountRecentTradesResponse).trades || []);
      setHistoryTotal(Number((tradesData as AccountRecentTradesResponse).total || 0));
      setHistoryPage(page);
    } catch (_error) {
      if (import.meta.env.DEV) console.debug('[AccountDetail] loadHistory error', _error);
    } finally {
      setHistoryLoading(false);
    }
  }, [historyPageSize]);

  const loadAllData = useCallback(async (accountId: string) => {
    setAnalyticsLoading(true);
    setAnalyticsError(null);
    try {
      const [analyticsData, tradesData, monthlyData, monthlyAnalysisResp] = await Promise.all([
        analyticsApi.getAccountAnalytics(accountId, chartPeriod),
        analyticsApi.getRecentTrades(accountId, 1, historyPageSize),
        analyticsApi.getMonthlyPnL(accountId, new Date().getFullYear()),
        analyticsApi.getMonthlyAnalysis(accountId),
      ]);
      setAnalytics(analyticsData as AccountAnalyticsData);
      setHistoryTrades((tradesData as AccountRecentTradesResponse).trades || []);
      setHistoryTotal(Number((tradesData as AccountRecentTradesResponse).total || 0));
      setHistoryPage(1);
      setMonthlyAnalysisYears((monthlyAnalysisResp as AccountMonthlyAnalysisResponse).years || []);
      setMonthlyAnalysisData((monthlyAnalysisResp as AccountMonthlyAnalysisResponse).data || []);
      // monthlyPnL 已下移到 AccountAnalyticsSection 内部管理
      setMonthlyPnL((monthlyData as AccountMonthlyPnLResponse).monthlyPnl || []);
    } catch (error) {
      setAnalyticsError(getErrorMessage(error, '加载分析数据失败'));
    } finally {
      setAnalyticsLoading(false);
    }
  }, [historyPageSize]);

  // chartPeriod changes only refresh the equity curve chart, not the full analytics section.
  const loadAnalyticsForPeriod = useCallback(async (accountId: string) => {
    try {
      const analyticsData = await analyticsApi.getAccountAnalytics(accountId, chartPeriod);
      setAnalytics(analyticsData as AccountAnalyticsData);
    } catch (_error) {
      // Keep existing analytics on period-switch error.
    }
  }, [chartPeriod]);

  // When stream data first arrives (gateway connected), load analytics/history.
  // Retry after 5s to cover syncHistory timing gap.
  // Positions arrive via SSE position_snapshot — no RPC fetchPositions needed.
  useEffect(() => {
    if (!id || !isDataReceived) return;
    loadAllData(id).catch((error) => showError(getErrorMessage(error, '加载分析数据失败')));
    const timer = setTimeout(() => {
      loadAllData(id).catch(() => {});
    }, 5000);
    return () => clearTimeout(timer);
  }, [id, isDataReceived]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!id) return;
    loadAnalyticsForPeriod(id);
  }, [id, chartPeriod, loadAnalyticsForPeriod]);

  useEffect(() => {
    if (!id) return;
    
    setCurrentAccountId(id);
    
    const init = async () => {
      // 并行加载所有数据，但不显示全局 loading
      const account = useAccountStore.getState().accounts.find(a => a.id === id);

      // 如果列表里已有该账户，直接用缓存填充 currentAccount，避免详情页一直等待
      if (account) {
        setCurrentAccount(account);
      }
      
      // 如果 store 中没有当前账户，才获取详情（不显示 loading）
      if (!account) {
        const loaded = await fetchAccount(id, false);
        if (!loaded) {
          showErrorModal(t('accounts.detail.messages.fetchAccountFailed'));
          navigate('/');
          return;
        }
      }
      
      // Load analytics data (non-blocking). Positions + history are
      // triggered by isDataReceived event when gateway connects.
      loadAllData(id).catch((error) => showError(getErrorMessage(error, '加载分析数据失败')));
    };
    
    init();
    
    const throttledReloadAnalytics = () => {
      const now = Date.now();
      if (now - lastAnalyticsReloadRef.current < ANALYTICS_RELOAD_MIN_INTERVAL_MS) return;
      lastAnalyticsReloadRef.current = now;
      loadAllData(id).catch((error) => showError(getErrorMessage(error, '加载分析数据失败')));
    };

    const handlePositionChange = (event: Event) => {
      const customEvent = event as CustomEvent;
      const { action, order } = customEvent.detail;

      if (action === 'PositionClose' && order) {
        // Positions come from SSE stream (position_snapshot). Only refresh
        // analytics / trade history on position changes.
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

        setHistoryTrades((prev) => {
          const exists = prev.some((t) => t.ticket === order.ticket);
          if (exists) return prev;
          return [newTrade, ...prev];
        });

        // Keep all business metrics authoritative from backend.
        // Throttle analytics reload to max once per 5 seconds.
        throttledReloadAnalytics();
      } else if (action === 'PositionOpen' || action === 'PendingOpen') {
        if (id) {
          throttledReloadAnalytics();
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
  }, [id, loadAllData, fetchAccounts, fetchAccount, setCurrentAccount, navigate, setCurrentAccountId, t]);

  const handleConnect = useCallback(async () => {
    if (!currentAccount || connecting) return;
    setConnecting(true);
    try {
      const result: ConnectAccountResult = await connectAccount(currentAccount.id);
      const msg = translateMaybeI18nKey(result?.message, t('common.operationFailed'));
      if (result?.success === false) {
        showError(msg);
      } else if (result?.message) {
        showSuccess(msg);
      }
      // Positions arrive via SSE stream after reconnect — no explicit fetchPositions needed.
      await fetchAccount(currentAccount.id, false); // 获取账户详情，但不显示 loading
    } finally {
      setConnecting(false);
    }
  }, [currentAccount, connecting, connectAccount, fetchAccount, t]);

  const handleRefreshAnalytics = useCallback(async () => {
    if (!id) return;
    await loadAllData(id);
  }, [id, loadAllData]);

  const handleRetryAnalytics = useCallback(() => {
    if (id) loadAllData(id);
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
      setDisabling(true);
      try {
        await disableAccount(currentAccount.id);
        await fetchAccount(currentAccount.id, false);
        showSuccessModal(t('accounts.messages.disabledSuccess'));
      } catch (_error) {
        showErrorModal(t('common.operationFailed'));
      } finally {
        setDisabling(false);
      }
    }
  }, [currentAccount, enableAccount, disableAccount, fetchAccount, t]);

  const handleDeleteClick = useCallback(() => {
    setDeletePassword('');
    setDeleteModalOpen(true);
  }, []);

  const handleConfirmDelete = useCallback(async () => {
    if (!currentAccount || !deletePassword.trim()) return;
    setDeleting(true);
    try {
      await deleteAccount(currentAccount.id, deletePassword.trim());
      setDeleteModalOpen(false);
      navigate('/');
    } catch {
      // Error already shown by hook
    } finally {
      setDeleting(false);
    }
  }, [currentAccount, deletePassword, deleteAccount, navigate]);

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
          <CaretRightOutlined size={16} stroke={1.5} />
        )
      ) : (
        disabling ? <Spin size="small" /> : <PauseCircleOutlined size={16} stroke={1.5} />
      ),
      onClick: handleToggleStatus,
      disabled: disabling,
    },
    {
      key: 'delete',
      label: t('accounts.detail.actions.deleteAccount'),
      icon: <DeleteOutlined size={16} stroke={1.5} style={{ color: '#E53935' }} />,
      onClick: handleDeleteClick,
      danger: true,
    },
  ], [currentAccount?.isDisabled, currentAccount?.id, enablingAccount, disabling, handleToggleStatus, handleDeleteClick, t]);

  const { equityChartData, profitByMonthData, symbolDistributionData, dailyPnLData, hourlyData, tradeStats, riskMetrics } = useMemo(() => {
    // Backend returns pre-filtered ISO `YYYY-MM-DD` equity curve per chartPeriod.
    // Frontend only formats date labels for chart x-axis display (MM/DD).
    // Period-aware date labels. Data is untouched; only x-axis display format changes.
    const DAY_NAMES = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
    const equityCurve = analytics?.equityCurve?.map((point) => {
      const raw = point.date || '';
      let label: string;
      if (raw.indexOf(' ') > 0) {
        // Hourly: "2026-05-28 14:00" → "14:00"
        label = raw.slice(raw.indexOf(' ') + 1, raw.indexOf(' ') + 6);
      } else if (chartPeriod === 'week') {
        // Daily → day-name: "Mon", "Tue", ...
        try { label = DAY_NAMES[new Date(raw + 'T00:00:00').getDay()]; } catch { label = raw; }
      } else {
        // Day/Month/All: "2026-05-28" → "5/28"
        const parts = raw.split('-');
        label = parts.length >= 3
          ? `${parseInt(parts[1], 10)}/${parseInt(parts[2], 10)}`
          : raw;
      }
      return {
        date: label,
        equity: point.equity,
        balance: point.balance,
        profit: point.profit,
      };
    }) || [];
    const profitByMonth = monthlyPnL
      .map((m) => {
        const monthValue = m?.month ?? m?.monthNum ?? m?.month_num;
        return {
          month: String(monthValue ?? ''),
          profit: m.profit,
          trades: Number(m.trades),
        };
      })
      .filter((m) => m.month);
    const dailyPnlRaw = analytics?.dailyPnl || [];
    const dailyPnl = dailyPnlRaw.map((d) => ({
      day: d.day,
      date: d.date,
      profit: d.pnl ?? d.profit,
      trades: Number(d.trades),
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
      .map((s) => ({
        name: s.symbol,
        value: Math.round(Number(s.tradeSharePercent || 0)),
        profit: s.profit,
      }));

    return {
      equityChartData: equityCurve,
      profitByMonthData: profitByMonth,
      symbolDistributionData: symbolDistribution,
      dailyPnLData: dailyPnl,
      hourlyData: (analytics?.hourlyStats || []).map((h) => ({
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
  }, [analytics, monthlyPnL, chartPeriod]);

  const { realPositions, pendingOrders } = useMemo(() => {
    const positionsList = Array.isArray(positions) ? positions : [];
    const real = positionsList.map(p => ({ ...p, open_price: p.openPrice || 0, current_price: p.closePrice || p.currentPrice || 0, open_time: formatTimestamp(p.openTime) })).filter(p => !isPendingOrder(p.type));
    const pending = positionsList.map(p => ({ ...p, open_price: p.openPrice || 0, current_price: p.closePrice || p.currentPrice || 0, open_time: formatTimestamp(p.openTime) })).filter(p => isPendingOrder(p.type));
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
            <Button type="text" icon={<ArrowLeftOutlined size={20} stroke={1.5} />} onClick={() => navigate('/')} style={{ color: '#8A9AA5' }} />
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
            <Button icon={<ReloadOutlined size={16} stroke={1.5} />} onClick={handleRefreshAnalytics} loading={analyticsLoading} style={{ borderRadius: '8px' }}>{t('common.refresh')}</Button>
            <Dropdown menu={{ items: menuItems }} trigger={['click']}><Button icon={<MoreOutlined size={16} stroke={1.5} />} style={{ borderRadius: '8px' }} /></Dropdown>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
          <InfoCard icon={<WalletOutlined size={18} stroke={1.5} color="#8A9AA5" />} label={t('accounts.detail.cards.balance')} value={formatCurrency(balance)} loading={isStreamLoading} />
          <InfoCard icon={<LineChartOutlined size={18} stroke={1.5} color="#8A9AA5" />} label={t('accounts.detail.cards.equity')} value={formatCurrency(equity)} loading={isStreamLoading} />
          <div className="rounded-2xl p-5" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}>
            <div className="flex items-center gap-2 mb-3">{profit >= 0 ? <RiseOutlined size={18} stroke={1.5} color="#00A651" /> : <FallOutlined size={18} stroke={1.5} color="#E53935" />}<span style={{ color: '#8A9AA5', fontSize: '14px' }}>{t('accounts.detail.cards.floatingProfit')}</span></div>
            {isStreamLoading ? <div className="text-lg" style={{ color: '#8A9AA5' }}>{t('common.loading')}</div> : <div className="flex items-baseline gap-2"><span className="text-2xl font-bold" style={{ color: profit >= 0 ? '#00A651' : '#E53935' }}>{profit >= 0 ? '+' : ''}{formatCurrency(profit)}</span><span style={{ color: profit >= 0 ? '#00A651' : '#E53935', fontSize: '14px' }}>({profitPercent >= 0 ? '+' : ''}{profitPercent.toFixed(2)}%)</span></div>}
          </div>
        </div>

        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
          <SmallInfoCard icon={<DollarOutlined size={16} stroke={1.5} color="#8A9AA5" />} label={t('accounts.detail.cards.marginUsed')} value={formatCurrency(margin)} loading={isStreamLoading} />
          <SmallInfoCard icon={<DollarOutlined size={16} stroke={1.5} color="#8A9AA5" />} label={t('accounts.detail.cards.marginFree')} value={formatCurrency(freeMargin)} loading={isStreamLoading} />
          <SmallInfoCard icon={<PercentageOutlined size={16} stroke={1.5} color="#8A9AA5" />} label={t('accounts.detail.cards.marginLevel')} value={margin > 0 ? `${(marginLevel || 0).toFixed(2)}%` : '--'} loading={isStreamLoading} valueColor={margin > 0 && (marginLevel || 0) < 100 ? '#E53935' : '#141D22'} />
          <SmallInfoCard icon={<WarningOutlined size={16} stroke={1.5} color="#8A9AA5" />} label={t('accounts.detail.cards.credit')} value={formatCurrency(credit)} loading={isStreamLoading} />
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
            historyLoading={historyLoading}
          />
        </div>

        <AccountAnalyticsSection
          analyticsLoading={analyticsLoading}
          analyticsError={analyticsError}
          onRetryAnalytics={handleRetryAnalytics}
          chartType={chartType}
          setChartType={setChartType}
          chartPeriod={chartPeriod}
          setChartPeriod={setChartPeriod}
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

        <Modal
          title={t('accounts.detail.actions.deleteAccount')}
          open={deleteModalOpen}
          onOk={handleConfirmDelete}
          onCancel={() => setDeleteModalOpen(false)}
          confirmLoading={deleting}
          okText={t('accounts.detail.actions.deleteConfirm')}
          cancelText={t('common.cancel')}
          okButtonProps={{ danger: true }}
          destroyOnClose
        >
          <div style={{ marginBottom: 16, color: '#E53935' }}>
            {t('accounts.detail.actions.deleteWarning')}
          </div>
          <div style={{ marginBottom: 8, color: '#8A9AA5' }}>
            {t('accounts.detail.actions.deletePasswordHint')}
          </div>
          <Input
            placeholder={t('accounts.detail.actions.deletePasswordPlaceholder')}
            value={deletePassword}
            onChange={(e) => setDeletePassword(e.target.value)}
            onPressEnter={handleConfirmDelete}
            disabled={deleting}
          />
        </Modal>
    </div>
  </div>
);
}