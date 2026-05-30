import { useState, useCallback, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { showSuccessModal, showErrorModal, showLoadingModal, showSuccess, showError } from '@/utils/message';
import { translateMaybeI18nKey } from '@/utils/error';
import type { ConnectAccountResult } from '@/client/account';
import { useAccountDetailQuery } from '@/queries/useAccountDetailQuery';
import { useAccountFinancials } from '@/queries/useAccountFinancials';
import { usePositionsQuery } from '@/queries/usePositionsQuery';
import { useConnectAccountMutation } from '@/mutations/useConnectAccountMutation';
import { useEnableDisableAccountMutation } from '@/mutations/useEnableDisableAccountMutation';
import { useDeleteAccountMutation } from '@/mutations/useDeleteAccountMutation';
import { useConnect } from '@/providers/useConnect';
import { formatTimestamp, isPendingOrder } from '../components/AccountDetail.utils';
import { useAccountAnalytics } from './useAccountAnalytics';

export function useAccountDetailData(id: string | undefined) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { reconnect } = useConnect();

  // ── TanStack Query ──
  const accountDetailQ = useAccountDetailQuery(id ?? '');
  const financialsQ = useAccountFinancials(id ?? '');
  const positionsQ = usePositionsQuery(id ?? '');
  const connectMut = useConnectAccountMutation();
  const toggleMut = useEnableDisableAccountMutation();
  const deleteMut = useDeleteAccountMutation();

  // ── Chart UI ──
  const [chartType, setChartType] = useState<'equity' | 'balance' | 'profit'>('equity');
  const [chartPeriod, setChartPeriod] = useState<'day' | 'week' | 'month' | 'all'>('month');

  // ── Action state ──
  const [connecting, setConnecting] = useState(false);
  const [disabling, setDisabling] = useState(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deletePassword, setDeletePassword] = useState('');
  const [deleting, setDeleting] = useState(false);

  // ── Account ──
  const currentAccount = accountDetailQ.data ?? null;
  const hasReceivedData = financialsQ.isSuccess;
  const isDataReceived = !!id && hasReceivedData;
  const isStreamLoading = !isDataReceived;
  const positions = positionsQ.data ?? [];

  // ── Analytics (delegated) ──
  const analytics = useAccountAnalytics(id, isDataReceived, chartPeriod);

  // ── Financial values (prefer SSE over snapshot) ──
  const financials = useMemo(() => {
    const sse = financialsQ.data;
    const acc = currentAccount;
    const useSse = Boolean(id && hasReceivedData && sse);
    return {
      balance: useSse ? (sse?.balance ?? 0) : (acc?.balance ?? 0),
      equity: useSse ? (sse?.equity ?? 0) : (acc?.equity ?? 0),
      margin: useSse ? (sse?.margin ?? 0) : (acc?.margin ?? 0),
      freeMargin: useSse ? (sse?.freeMargin ?? 0) : (acc?.freeMargin ?? 0),
      marginLevel: useSse ? (sse?.marginLevel ?? 0) : (acc?.marginLevel ?? 0),
      profit: useSse ? (sse?.profit ?? 0) : (acc?.profit ?? 0),
      profitPercent: useSse ? (sse?.profitPercent ?? 0) : (acc?.profitPercent ?? 0),
      credit: useSse ? (sse?.credit ?? 0) : (acc?.credit ?? 0),
    };
  }, [id, hasReceivedData, financialsQ.data, currentAccount]);

  // ── Account actions ──
  const handleConnect = useCallback(async () => {
    if (!currentAccount || connecting) return;
    setConnecting(true);
    try {
      const result: ConnectAccountResult = await connectMut.mutateAsync(currentAccount.id);
      const msg = translateMaybeI18nKey(result?.message, t('common.operationFailed'));
      if (result?.success === false) showError(msg);
      else if (result?.message) showSuccess(msg);
      reconnect();
    } finally { setConnecting(false); }
  }, [currentAccount, connecting, connectMut, reconnect, t]);

  const handleToggleStatus = useCallback(async () => {
    if (!currentAccount) return;
    if (currentAccount.isDisabled) {
      const modal = showLoadingModal(t('accounts.messages.connectingMtServer'), t('common.pleaseWait'));
      try {
        await toggleMut.mutateAsync({ id: currentAccount.id, isDisabled: false });
        modal.destroy();
        showSuccessModal(t('accounts.messages.enabledSuccess'));
      } catch { modal.destroy(); showErrorModal(t('common.operationFailed')); }
    } else {
      setDisabling(true);
      try {
        await toggleMut.mutateAsync({ id: currentAccount.id, isDisabled: true });
        showSuccessModal(t('accounts.messages.disabledSuccess'));
      } catch { showErrorModal(t('common.operationFailed')); }
      finally { setDisabling(false); }
    }
  }, [currentAccount, toggleMut, t]);

  const handleDelete = useCallback(async () => {
    if (!currentAccount || !deletePassword.trim()) return;
    setDeleting(true);
    try {
      await deleteMut.mutateAsync({ id: currentAccount.id, password: deletePassword.trim() });
      setDeleteModalOpen(false);
      navigate('/');
    } catch { /* mutation handles error */ }
    finally { setDeleting(false); }
  }, [currentAccount, deletePassword, deleteMut, navigate]);

  // ── Position filtering ──
  const { realPositions, pendingOrders } = useMemo(() => {
    const list = Array.isArray(positions) ? positions : [];
    const withDisplay = list.map((p) => ({
      ...p, open_price: p.openPrice || 0,
      current_price: p.closePrice || p.currentPrice || 0,
      open_time: formatTimestamp(p.openTime),
    }));
    return {
      realPositions: withDisplay.filter((p) => !isPendingOrder(p.type)),
      pendingOrders: withDisplay.filter((p) => isPendingOrder(p.type)),
    };
  }, [positions]);

  return {
    currentAccount, isDataReceived, isStreamLoading, financials,
    positions: realPositions, pendingOrders,
    chartType, setChartType, chartPeriod, setChartPeriod,
    connecting, disabling,
    handleConnect, handleToggleStatus,
    deleteModalOpen, setDeleteModalOpen,
    deletePassword, setDeletePassword,
    deleting, handleDelete,
    togglePending: toggleMut.isPending,
    // analytics (spread from useAccountAnalytics)
    analyticsLoading: analytics.analyticsLoading,
    analyticsError: analytics.analyticsError,
    equityChartData: analytics.equityChartData,
    profitByMonthData: analytics.profitByMonthData,
    symbolDistributionData: analytics.symbolDistributionData,
    dailyPnLData: analytics.dailyPnLData,
    hourlyData: analytics.hourlyData,
    tradeStats: analytics.tradeStats,
    riskMetrics: analytics.riskMetrics,
    monthlyAnalysisYears: analytics.monthlyAnalysisYears,
    monthlyAnalysisData: analytics.monthlyAnalysisData,
    historyTrades: analytics.historyTrades,
    historyTotal: analytics.historyTotal,
    historyPage: analytics.historyPage,
    historyLoading: analytics.historyLoading,
    setHistoryTrades: analytics.setHistoryTrades,
    setHistoryTotal: analytics.setHistoryTotal,
    setHistoryPage: analytics.setHistoryPage,
    handleRefresh: analytics.handleRefresh,
    handleRetry: analytics.handleRetry,
  };
}
