import { useState, useEffect, useRef } from 'react';
import { Button, Pagination, Spin, Tabs } from 'antd';
import type { TabsProps } from 'antd';
import {
  LineChartOutlined,
  HistoryOutlined,
  UnorderedListOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { analyticsApi } from '@/client/analytics';
import { tradingApi } from '@/client/trading';
import { queryKeys } from '@/queries/queryKeys';
import { showError } from '@/utils/message';
import { HistoryTradeRow, PendingOrderRow, PositionRow } from './AccountDetail.shared';
import { useTranslation } from 'react-i18next'
import { TRADE_TABS_EMPTY_HISTORY_KEY, TRADE_TABS_EMPTY_POSITIONS_KEY, TRADE_TABS_HISTORY_WITH_COUNT_KEY, TRADE_TABS_PAGINATION_TOTAL_KEY, TRADE_TABS_PENDING_WITH_COUNT_KEY, TRADE_TABS_POSITIONS_WITH_COUNT_KEY, TRADE_TABS_SYNC_HISTORY_KEY, TRADE_TABS_TABLE_CLOSE_PRICE_KEY, TRADE_TABS_TABLE_CLOSE_TIME_KEY, TRADE_TABS_TABLE_CURRENT_PRICE_KEY, TRADE_TABS_TABLE_MAGIC_KEY, TRADE_TABS_TABLE_OPEN_PRICE_KEY, TRADE_TABS_TABLE_OPEN_TIME_KEY, TRADE_TABS_TABLE_ORDER_ID_KEY, TRADE_TABS_TABLE_PENDING_PRICE_KEY, TRADE_TABS_TABLE_PENDING_TIME_KEY, TRADE_TABS_TABLE_PROFIT_KEY, TRADE_TABS_TABLE_SIDE_KEY, TRADE_TABS_TABLE_SYMBOL_KEY, TRADE_TABS_TABLE_TYPE_KEY, TRADE_TABS_TABLE_VOLUME_KEY } from '@/gen/ant/v1/i18n/accounts_keys';

;
import type { Position } from '@/types/trading';
import type { TradeRecordItem } from '@/client/analyticsTypes';

type Props = {
  id: string | undefined;
  realPositions: Position[];
  pendingOrders: Position[];
  historyTrades: TradeRecordItem[];
  historyTotal: number;
  historyPage: number;
  historyPageSize: number;
  onHistoryTradesChange: (trades: TradeRecordItem[]) => void;
  onHistoryTotalChange: (total: number) => void;
  onHistoryPageChange: (page: number) => void;
  historyLoading?: boolean;
};

export default function AccountTradeTabs({
  id,
  realPositions,
  pendingOrders,
  historyTrades,
  historyTotal,
  historyPage,
  historyPageSize,
  onHistoryTradesChange,
  onHistoryTotalChange,
  onHistoryPageChange,
  historyLoading = false,
}: Props) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [localHistoryLoading, setLocalHistoryLoading] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const isHistoryLoading = historyLoading || localHistoryLoading;
  const autoSyncDone = useRef<string | null>(null);

  // Reset auto-sync flag when the account changes so the new account gets synced.
  useEffect(() => {
    if (autoSyncDone.current !== id) {
      autoSyncDone.current = null;
    }
  }, [id]);

  const doSync = async () => {
    if (!id) return;
    setSyncing(true);
    try {
      await tradingApi.syncOrderHistory(id);
      // Reload history from DB after sync.
      const data = await analyticsApi.getRecentTrades(id, historyPage, historyPageSize);
      onHistoryTradesChange(data?.trades || []);
      onHistoryTotalChange(Number(data?.total || 0));
      // Invalidate analytics queries — equity curve, stats, etc. depend on
      // trade_records which were just populated by the sync.
      queryClient.invalidateQueries({ queryKey: queryKeys.analytics.detail(id, 'day') });
      queryClient.invalidateQueries({ queryKey: queryKeys.analytics.detail(id, 'week') });
      queryClient.invalidateQueries({ queryKey: queryKeys.analytics.detail(id, 'month') });
      queryClient.invalidateQueries({ queryKey: queryKeys.analytics.detail(id, 'all') });
    } catch (err: unknown) {
      showError(err instanceof Error ? err.message : String(err) || t('accounts.tradeTabs.syncHistoryFailed'));
    } finally {
      setSyncing(false);
    }
  };

  const handleSync = () => { doSync(); };

  // Auto-sync: if history is empty and not loading, trigger a one-time sync
  // from the MT broker. Covers new accounts that haven't synced yet.
  // Only mark done on success — allows retry on next mount if it fails.
  // Tracks by account ID so switching accounts re-triggers the sync.
  useEffect(() => {
    if (!id) return;
    if (autoSyncDone.current === id) return;
    if (historyLoading) return;
    if (historyTrades.length > 0) { autoSyncDone.current = id; return; }
    doSync().then(() => { autoSyncDone.current = id; }).catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, historyLoading, historyTrades.length]);

  const tradeTabs: TabsProps['items'] = [
    {
      key: 'positions',
      label: (
        <span className="flex items-center gap-2">
          <UnorderedListOutlined style={{ fontSize: 16 }} />
          {t(TRADE_TABS_POSITIONS_WITH_COUNT_KEY, { count: realPositions.length })}
          {pendingOrders.length > 0 && ` | ${t(TRADE_TABS_PENDING_WITH_COUNT_KEY, { count: pendingOrders.length })}`}
        </span>
      ),
      children:
        realPositions.length === 0 && pendingOrders.length === 0 ? (
          <div className="text-center py-12" style={{ color: 'var(--color-text-muted)' }}>
            <LineChartOutlined style={{ fontSize: 48, opacity: 0.3 }} color="#D4AF37" />
            <p className="mt-4">{t(TRADE_TABS_EMPTY_POSITIONS_KEY)}</p>
          </div>
        ) : (
          <div>
            {realPositions.length > 0 && (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr style={{ background: 'var(--color-bg-secondary)' }}>
                      <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_ORDER_ID_KEY)}</th>
                      <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_SYMBOL_KEY)}</th>
                      <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_SIDE_KEY)}</th>
                      <th className="text-right p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_VOLUME_KEY)}</th>
                      <th className="text-right p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_OPEN_PRICE_KEY)}</th>
                      <th className="text-right p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_CURRENT_PRICE_KEY)}</th>
                      <th className="text-right p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_PROFIT_KEY)}</th>
                      <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_OPEN_TIME_KEY)}</th>
                      <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_MAGIC_KEY)}</th>
                    </tr>
                  </thead>
                  <tbody>{realPositions.map((p) => <PositionRow key={p.ticket} position={p} />)}</tbody>
                </table>
              </div>
            )}
            {pendingOrders.length > 0 && (
              <div className="mt-4">
                <div className="px-3 py-2 text-sm font-medium" style={{ color: 'var(--color-text-muted)', background: 'var(--color-bg-secondary)' }}>
                  {t(TRADE_TABS_PENDING_WITH_COUNT_KEY, { count: pendingOrders.length })}
                </div>
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead>
                      <tr style={{ background: '#FAFBFC' }}>
                        <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_ORDER_ID_KEY)}</th>
                        <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_SYMBOL_KEY)}</th>
                        <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_TYPE_KEY)}</th>
                        <th className="text-right p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_VOLUME_KEY)}</th>
                        <th className="text-right p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_PENDING_PRICE_KEY)}</th>
                        <th className="text-right p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_CURRENT_PRICE_KEY)}</th>
                        <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_PENDING_TIME_KEY)}</th>
                        <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_MAGIC_KEY)}</th>
                      </tr>
                    </thead>
                    <tbody>{pendingOrders.map((p) => <PendingOrderRow key={p.ticket} order={p} />)}</tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        ),
    },
    {
      key: 'history',
      label: (
        <span className="flex items-center gap-2">
          <HistoryOutlined style={{ fontSize: 16 }} />
          {t(TRADE_TABS_HISTORY_WITH_COUNT_KEY, { count: historyTotal })}
        </span>
      ),
      children:
        historyTrades.length === 0 ? (
          <div>
            <div className="flex items-center justify-end mb-3">
              <Button
                icon={<ReloadOutlined spin={syncing} />}
                onClick={handleSync}
                loading={syncing}
                size="small"
              >
                {t(TRADE_TABS_SYNC_HISTORY_KEY)}
              </Button>
            </div>
            <div className="text-center py-12" style={{ color: 'var(--color-text-muted)' }}>
              <HistoryOutlined style={{ fontSize: 48, opacity: 0.3 }} color="#D4AF37" />
              <p className="mt-4">{t(TRADE_TABS_EMPTY_HISTORY_KEY)}</p>
            </div>
          </div>
        ) : (
          <Spin spinning={isHistoryLoading}>
            <div className="flex items-center justify-between mb-3">
              <div />
              <Button
                icon={<ReloadOutlined spin={syncing} />}
                onClick={handleSync}
                loading={syncing}
                size="small"
              >
                {t(TRADE_TABS_SYNC_HISTORY_KEY)}
              </Button>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr style={{ background: 'var(--color-bg-secondary)' }}>
                    <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_ORDER_ID_KEY)}</th>
                    <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_SYMBOL_KEY)}</th>
                    <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_SIDE_KEY)}</th>
                    <th className="text-right p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_VOLUME_KEY)}</th>
                    <th className="text-right p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_OPEN_PRICE_KEY)}</th>
                    <th className="text-right p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_CLOSE_PRICE_KEY)}</th>
                    <th className="text-right p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_PROFIT_KEY)}</th>
                    <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_CLOSE_TIME_KEY)}</th>
                    <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>{t(TRADE_TABS_TABLE_MAGIC_KEY)}</th>
                  </tr>
                </thead>
                <tbody>
                  {historyTrades.map((trade) => (
                    <HistoryTradeRow key={trade.ticket} trade={trade} />
                  ))}
                </tbody>
              </table>
            </div>
            <div className="flex justify-end mt-4 p-3">
              <Pagination
                current={historyPage}
                pageSize={historyPageSize}
                total={historyTotal}
                onChange={(page) => {
                  if (!id) return;
                  setLocalHistoryLoading(true);
                  analyticsApi.getRecentTrades(id, page, historyPageSize).then((data) => {
                    onHistoryTradesChange(data?.trades || []);
                    onHistoryTotalChange(Number(data?.total || 0));
                    onHistoryPageChange(page);
                  }).catch(() => {
                    onHistoryPageChange(historyPage); // reset to current page on error
                  }).finally(() => {
                    setLocalHistoryLoading(false);
                  });
                }}
                showSizeChanger={false}
                showTotal={(total) => t(TRADE_TABS_PAGINATION_TOTAL_KEY, { total })}
              />
            </div>
            </Spin>
        ),
    },
  ];

  return <Tabs defaultActiveKey="positions" items={tradeTabs} className="px-4 pt-4" />;
}
