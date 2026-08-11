import { useState, useCallback, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next'
import { message } from 'antd';
import { MESSAGES_CONNECTING_MT_SERVER_KEY, MESSAGES_CONNECT_FAILED_KEY, MESSAGES_DISABLED_SUCCESS_KEY, MESSAGES_ENABLED_SUCCESS_KEY, DETAIL_ACTIONS_DELETE_PASSWORD_WRONG_KEY, MESSAGES_DELETE_FAILED_KEY } from '@/gen/ant/v1/i18n/accounts_keys';
import { showSuccessModal, showErrorModal, showLoadingModal, showSuccess, showError } from '@/utils/message';
import { translateMaybeI18nKey } from '@/utils/error';
import { getErrorMessage } from '@/utils/error';
import type { ConnectAccountResult } from '@/client/account';
import { useAccountDetailQuery } from '@/queries/useAccountDetailQuery';
import { useAccountFinancials } from '@/queries/useAccountFinancials';
import { usePositionsQuery } from '@/queries/usePositionsQuery';
import { useConnectAccountMutation } from '@/mutations/useConnectAccountMutation';
import { useEnableDisableAccountMutation } from '@/mutations/useEnableDisableAccountMutation';
import { useDeleteAccountMutation } from '@/mutations/useDeleteAccountMutation';
import { useConnect } from '@/providers/useConnect';
import { isPendingOrder } from '../components/AccountDetail.utils';
import { useHistoryTrades } from './useHistoryTrades';
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

  // ── History trades (for AccountTradeTabs) ──
  const {
    historyTrades, historyTotal, historyPage, historyLoading,
    setHistoryTrades, setHistoryTotal, setHistoryPage,
    handleRefresh: handleRefreshHistory, handleRetry: handleRetryHistory,
  } = useHistoryTrades(id);

  // ── Account ── (must come before analytics — isDataReceived used below)
  const currentAccount = accountDetailQ.data ?? null;
  const hasReceivedData = financialsQ.isSuccess || financialsQ.isError;
  const isDataReceived = !!id && hasReceivedData;

  // ── Analytics (account-level performance charts) ──
  const analytics = useAccountAnalytics(id, isDataReceived, chartPeriod);

  // ── Action state ──
  const [connecting, setConnecting] = useState(false);
  const [disabling, setDisabling] = useState(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deletePassword, setDeletePassword] = useState('');
  const [deleting, setDeleting] = useState(false);
  const isStreamLoading = !isDataReceived && financialsQ.isLoading;
  const accountLoadError = accountDetailQ.isError && !currentAccount;
  const positions = useMemo(() => positionsQ.data ?? [], [positionsQ.data]);

  // ── Financial values (prefer SSE over snapshot) ──
  const financials = useMemo(() => computeFinancials(id, hasReceivedData, financialsQ.data as Record<string, unknown> | null | undefined, currentAccount as Record<string, unknown> | null), [id, hasReceivedData, financialsQ.data, currentAccount]);

  // ── Account actions ──
  const handleConnect = useCallback(async () => {
    if (!currentAccount || connecting) return;
    setConnecting(true);
    try {
      const result: ConnectAccountResult = await connectMut.mutateAsync(currentAccount.id);
      if (result?.success) {
        const msg = translateMaybeI18nKey(result?.message, '');
        if (msg) showSuccess(msg);
        reconnect();
      } else {
        showError(translateMaybeI18nKey(result?.message, t(MESSAGES_CONNECT_FAILED_KEY)));
      }
    } catch { /* mutation onError handles toast */ }
    finally { setConnecting(false); }
  }, [currentAccount, connecting, connectMut, reconnect, t]);

  const handleToggleStatus = useCallback(async () => {
    if (!currentAccount) return;
    const s = (currentAccount.status || currentAccount.accountStatus || '').toLowerCase();
    if (s === 'disconnected' || s === 'frozen' || s === 'error' || s === 'connecting' || currentAccount.isDisabled === true) {
      const modal = showLoadingModal(t(MESSAGES_CONNECTING_MT_SERVER_KEY), t('common.pleaseWait'));
      try {
        await toggleMut.mutateAsync({ id: currentAccount.id, isDisabled: false });
        modal.destroy();
        showSuccessModal(t(MESSAGES_ENABLED_SUCCESS_KEY));
      } catch { modal.destroy(); showErrorModal(t('common.operationFailed')); }
    } else {
      setDisabling(true);
      try {
        await toggleMut.mutateAsync({ id: currentAccount.id, isDisabled: true });
        showSuccessModal(t(MESSAGES_DISABLED_SUCCESS_KEY));
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
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      if (msg.includes('password verification failed') || msg.includes('Invalid account')) {
        message.error(t(DETAIL_ACTIONS_DELETE_PASSWORD_WRONG_KEY));
      } else {
        message.error(getErrorMessage(err, t(MESSAGES_DELETE_FAILED_KEY)));
      }
    } finally { setDeleting(false); }
  }, [currentAccount, deletePassword, deleteMut, navigate, t]);

  // ── Position filtering ──
  const { realPositions, pendingOrders } = useMemo(() => {
    const list = Array.isArray(positions) ? positions : [];
    return {
      realPositions: list.filter((p) => !isPendingOrder(p.type)),
      pendingOrders: list.filter((p) => isPendingOrder(p.type)),
    };
  }, [positions]);

  return {
    currentAccount, isDataReceived, isStreamLoading, accountLoadError, financials,
    positions: realPositions, pendingOrders,
    connecting, disabling,
    handleConnect, handleToggleStatus,
    deleteModalOpen, setDeleteModalOpen,
    deletePassword, setDeletePassword,
    deleting, handleDelete,
    togglePending: toggleMut.isPending,
    // history trades
    historyTrades, historyTotal, historyPage, historyLoading,
    setHistoryTrades, setHistoryTotal, setHistoryPage,
    handleRefresh: () => { accountDetailQ.refetch(); handleRefreshHistory(); analytics.handleRefresh(); },
    handleRetry: () => { accountDetailQ.refetch(); handleRetryHistory(); analytics.handleRetry(); },
    // analytics (account-level performance)
    chartType, setChartType, chartPeriod, setChartPeriod,
    analyticsLoading: analytics.analyticsLoading, analyticsError: analytics.analyticsError,
    equityChartData: analytics.equityChartData, profitByMonthData: analytics.profitByMonthData,
    symbolDistributionData: analytics.symbolDistributionData,
    dailyPnLData: analytics.dailyPnLData, hourlyData: analytics.hourlyData,
    tradeStats: analytics.tradeStats, riskMetrics: analytics.riskMetrics,
    monthlyAnalysisYears: analytics.monthlyAnalysisYears,
    monthlyAnalysisData: analytics.monthlyAnalysisData,
  };
}

const FINANCIAL_KEYS = ['balance', 'equity', 'margin', 'freeMargin', 'marginLevel', 'profit', 'profitPercent', 'credit'] as const;

type Financials = { balance: number; equity: number; margin: number; freeMargin: number; marginLevel: number; profit: number; profitPercent: number; credit: number };

function computeFinancials(id: string | undefined, hasReceivedData: boolean, sse: Record<string, unknown> | null, acc: Record<string, unknown> | null) {
  const src = (id && hasReceivedData && sse) ? sse : acc;
  const result: Record<string, number> = {};
  for (const key of FINANCIAL_KEYS) {
    result[key] = Number(src?.[key] ?? 0);
  }
  return result as Financials;
}
