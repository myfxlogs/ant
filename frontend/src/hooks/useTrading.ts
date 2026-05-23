import { useCallback } from 'react';
import { useTradingStore } from '@/stores/tradingStore';
import { tradingApi } from '@/client/trading';
import { accountApi } from '@/client/account';
import { getErrorMessage, translateMaybeI18nKey } from '@/utils/error';
import { showError, showSuccess } from '@/utils/message';
import i18n from '@/i18n';
import { Code, ConnectError } from '@connectrpc/connect';
import { getTradingRiskToastMessage } from '@/utils/tradingRiskError';

const reconnectAttemptAtMs = new Map<string, number>();
const RECONNECT_COOLDOWN_MS = 15_000;

export function useTrading() {
  const positions = useTradingStore((state) => state.positions);
  const tradeLogs = useTradingStore((state) => state.tradeLogs);
  const loading = useTradingStore((state) => state.loading);
  const setLoading = useTradingStore((state) => state.setLoading);
  const setPositions = useTradingStore((state) => state.setPositions);

  const fetchPositions = useCallback(async (accountId: string, showLoading = true) => {
    if (!accountId) {
      return [];
    }
    if (showLoading) {
      setLoading(true);
    }
    try {
      const positions = await tradingApi.getPositions(accountId);
      const positionsArray = Array.isArray(positions) ? positions : [];
      setPositions(accountId, positionsArray as any);
      return positionsArray;
    } catch (error) {
      const isFailedPrecondition = error instanceof ConnectError && error.code === Code.FailedPrecondition;
      const rawMessage = typeof (error as Error)?.message === 'string' ? (error as Error).message : '';
      const normalizedMessage = rawMessage.toLowerCase();
      const isAccountNotConnected = isFailedPrecondition || normalizedMessage.includes('account not connected');

      if (isAccountNotConnected) {
        const now = Date.now();
        const lastAttempt = reconnectAttemptAtMs.get(accountId) || 0;
        if (now - lastAttempt > RECONNECT_COOLDOWN_MS) {
          reconnectAttemptAtMs.set(accountId, now);
          try {
            const result = await accountApi.connect(accountId);
            if (result?.success !== false) {
              const positionsAfterReconnect = await tradingApi.getPositions(accountId);
              const positionsArray = Array.isArray(positionsAfterReconnect) ? positionsAfterReconnect : [];
              setPositions(accountId, positionsArray as any);
              return positionsArray;
            }
          } catch (_e) {
            // ignore
          }
        }
      }

      if (!isAccountNotConnected) {
        showError(getErrorMessage(error, i18n.t('trading.messages.fetchPositionsFailed')));
        setPositions(accountId, []);
        return [];
      }
      const state = useTradingStore.getState();
      return state.positionsMap.get(accountId) || [];
    } finally {
      if (showLoading) {
        setLoading(false);
      }
    }
  }, [setLoading, setPositions]);

  const sendOrder = useCallback(async (params: {
    accountId: string;
    symbol: string;
    type: string;
    volume: number;
    price?: number;
    stopLoss?: number;
    takeProfit?: number;
    comment?: string;
  }) => {
    setLoading(true);
    try {
      const result = await tradingApi.orderSend(params);
      if (result.error) {
        showError(
          getTradingRiskToastMessage({
            riskCode: result.riskError?.code,
            error: result.error,
            message: result.message,
            fallback: translateMaybeI18nKey(result.error, String(result.error)),
          }),
        );
        return null;
      }
      showSuccess(i18n.t('trading.messages.orderSendSuccess'));
      return result.order;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('trading.messages.orderSendFailed')));
      throw error;
    } finally {
      setLoading(false);
    }
  }, [setLoading]);

  const modifyOrder = useCallback(async (params: {
    accountId: string;
    ticket: bigint;
    stopLoss?: number;
    takeProfit?: number;
    price?: number;
  }) => {
    setLoading(true);
    try {
      const result = await tradingApi.orderModify(params);
      if (result.error) {
        showError(
          getTradingRiskToastMessage({
            riskCode: result.riskError?.code,
            error: result.error,
            message: result.message,
            fallback: translateMaybeI18nKey(result.error, String(result.error)),
          }),
        );
        return null;
      }
      showSuccess(i18n.t('trading.messages.orderModifySuccess'));
      return result.order;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('trading.messages.orderModifyFailed')));
      throw error;
    } finally {
      setLoading(false);
    }
  }, [setLoading]);

  const closeOrder = useCallback(async (params: {
    accountId: string;
    ticket: bigint;
    volume?: number;
    price?: number;
  }) => {
    setLoading(true);
    try {
      const result = await tradingApi.orderClose(params);
      if (result.error) {
        showError(
          getTradingRiskToastMessage({
            riskCode: result.riskError?.code,
            error: result.error,
            message: result.message,
            fallback: translateMaybeI18nKey(result.error, String(result.error)),
          }),
        );
        return null;
      }
      showSuccess(i18n.t('trading.messages.orderCloseSuccess'));
      return result.order;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('trading.messages.orderCloseFailed')));
      throw error;
    } finally {
      setLoading(false);
    }
  }, [setLoading]);

  const getPendingOrders = useCallback(async (accountId: string) => {
    try {
      const orders = await tradingApi.getPendingOrders(accountId);
      return orders;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('trading.messages.fetchPendingOrdersFailed')));
      return [];
    }
  }, []);

  const getOrderHistory = useCallback(async (params: {
    accountId: string;
    from?: string;
    to?: string;
    page?: number;
    pageSize?: number;
  }) => {
    try {
      const result = await tradingApi.getOrderHistory(params);
      return result;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('trading.messages.fetchOrderHistoryFailed')));
      return { orders: [], total: 0, page: 1, pageSize: 50 };
    }
  }, []);

  const connectAccount = useCallback(async (accountId: string) => {
    try {
      const result = await accountApi.connect(accountId);
      return result;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('accounts.messages.connectFailed')));
      throw error;
    }
  }, []);

  return {
    positions,
    tradeLogs,
    loading,
    fetchPositions,
    sendOrder,
    modifyOrder,
    closeOrder,
    getPendingOrders,
    getOrderHistory,
    connectAccount,
  };
}
