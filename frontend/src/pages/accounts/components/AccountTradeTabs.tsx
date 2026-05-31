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
import { HistoryTradeRow, PendingOrderRow, PositionRow } from './AccountDetail.shared';
import { useTranslation } from 'react-i18next';

type Props = {
  id: string | undefined;
  realPositions: any[];
  pendingOrders: any[];
  historyTrades: any[];
  historyTotal: number;
  historyPage: number;
  historyPageSize: number;
  onHistoryTradesChange: (trades: any[]) => void;
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
  const autoSyncDone = useRef(false);

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
    } catch (err) {
      console.error('Sync history failed:', err);
    } finally {
      setSyncing(false);
    }
  };

  const handleSync = () => { doSync(); };

  // Auto-sync: if history is empty and not loading, trigger a one-time sync
  // from the MT broker. Covers new accounts that haven't synced yet.
  useEffect(() => {
    if (!id) return;
    if (autoSyncDone.current) return;
    if (historyLoading) return;
    if (historyTrades.length > 0) return;
    autoSyncDone.current = true;
    doSync();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, historyLoading, historyTrades.length]);

  const tradeTabs: TabsProps['items'] = [
    {
      key: 'positions',
      label: (
        <span className="flex items-center gap-2">
          <UnorderedListOutlined size={16} stroke={1.5} />
          {t('accounts.tradeTabs.positionsWithCount', { count: realPositions.length })}
          {pendingOrders.length > 0 && ` | ${t('accounts.tradeTabs.pendingWithCount', { count: pendingOrders.length })}`}
        </span>
      ),
      children:
        realPositions.length === 0 && pendingOrders.length === 0 ? (
          <div className="text-center py-12" style={{ color: '#8A9AA5' }}>
            <LineChartOutlined size={48} stroke={1} color="#D4AF37" style={{ opacity: 0.3 }} />
            <p className="mt-4">{t('accounts.tradeTabs.emptyPositions')}</p>
          </div>
        ) : (
          <div>
            {realPositions.length > 0 && (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr style={{ background: '#F5F7F9' }}>
                      <th className="text-left p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.orderId')}</th>
                      <th className="text-left p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.symbol')}</th>
                      <th className="text-left p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.side')}</th>
                      <th className="text-right p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.volume')}</th>
                      <th className="text-right p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.openPrice')}</th>
                      <th className="text-right p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.currentPrice')}</th>
                      <th className="text-right p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.profit')}</th>
                      <th className="text-left p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.openTime')}</th>
                    </tr>
                  </thead>
                  <tbody>{realPositions.map((p) => <PositionRow key={p.ticket} position={p} />)}</tbody>
                </table>
              </div>
            )}
            {pendingOrders.length > 0 && (
              <div className="mt-4">
                <div className="px-3 py-2 text-sm font-medium" style={{ color: '#8A9AA5', background: '#F5F7F9' }}>
                  {t('accounts.tradeTabs.pendingWithCount', { count: pendingOrders.length })}
                </div>
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead>
                      <tr style={{ background: '#FAFBFC' }}>
                        <th className="text-left p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.orderId')}</th>
                        <th className="text-left p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.symbol')}</th>
                        <th className="text-left p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.type')}</th>
                        <th className="text-right p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.volume')}</th>
                        <th className="text-right p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.pendingPrice')}</th>
                        <th className="text-right p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.currentPrice')}</th>
                        <th className="text-left p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.pendingTime')}</th>
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
          <HistoryOutlined size={16} stroke={1.5} />
          {t('accounts.tradeTabs.historyWithCount', { count: historyTotal })}
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
                {t('accounts.tradeTabs.syncHistory')}
              </Button>
            </div>
            <div className="text-center py-12" style={{ color: '#8A9AA5' }}>
              <HistoryOutlined size={48} stroke={1} color="#D4AF37" style={{ opacity: 0.3 }} />
              <p className="mt-4">{t('accounts.tradeTabs.emptyHistory')}</p>
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
                {t('accounts.tradeTabs.syncHistory')}
              </Button>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr style={{ background: '#F5F7F9' }}>
                    <th className="text-left p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.orderId')}</th>
                    <th className="text-left p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.symbol')}</th>
                    <th className="text-left p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.side')}</th>
                    <th className="text-right p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.volume')}</th>
                    <th className="text-right p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.openPrice')}</th>
                    <th className="text-right p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.closePrice')}</th>
                    <th className="text-right p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.profit')}</th>
                    <th className="text-left p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>{t('accounts.tradeTabs.table.closeTime')}</th>
                  </tr>
                </thead>
                <tbody>
                  {historyTrades.map((trade: any) => (
                    <HistoryTradeRow key={trade.id || trade.ticket} trade={trade} />
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
                  }).catch((err) => {
                    if (import.meta.env.DEV) console.debug('[AccountTradeTabs] pagination error', err);
                    onHistoryPageChange(historyPage); // reset to current page on error
                  }).finally(() => {
                    setLocalHistoryLoading(false);
                  });
                }}
                showSizeChanger={false}
                showTotal={(total) => t('accounts.tradeTabs.pagination.total', { total })}
              />
            </div>
            </Spin>
        ),
    },
  ];

  return <Tabs defaultActiveKey="positions" items={tradeTabs} className="px-4 pt-4" />;
}
