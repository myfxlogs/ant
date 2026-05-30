import { useCallback } from 'react';
import { useTradingStore } from '@/stores/tradingStore';
import { tradingApi } from '@/client/trading';
import { accountApi } from '@/client/account';
import { getErrorMessage, translateMaybeI18nKey } from '@/utils/error';
import { showError, showSuccess } from '@/utils/message';
import i18n from '@/i18n';

export function useTrading() {
  const positions = useTradingStore((state) => state.positions);
  const tradeLogs = useTradingStore((state) => state.tradeLogs);
  const loading = useTradingStore((state) => state.loading);
  const setLoading = useTradingStore((state) => state.setLoading);

  // Position data now flows exclusively through SSE → TanStack Query.
  // No RPC fetchPositions — the SSEQueryBridge handles initial snapshots
  // and incremental updates. This eliminates the race condition between
  // SSE and RPC position data.

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
    sendOrder,
    closeOrder,
    getOrderHistory,
    connectAccount,
  };
}
