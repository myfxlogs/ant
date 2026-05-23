import { Pagination, Tabs } from 'antd';
import type { TabsProps } from 'antd';
import {
  IconChartLine,
  IconHistory,
  IconList,
} from '@tabler/icons-react';
import { analyticsApi } from '@/client/analytics';
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
}: Props) {
  const { t } = useTranslation();

  const tradeTabs: TabsProps['items'] = [
    {
      key: 'positions',
      label: (
        <span className="flex items-center gap-2">
          <IconList size={16} stroke={1.5} />
          {t('accounts.tradeTabs.positionsWithCount', { count: realPositions.length })}
          {pendingOrders.length > 0 && ` | ${t('accounts.tradeTabs.pendingWithCount', { count: pendingOrders.length })}`}
        </span>
      ),
      children:
        realPositions.length === 0 && pendingOrders.length === 0 ? (
          <div className="text-center py-12" style={{ color: '#8A9AA5' }}>
            <IconChartLine size={48} stroke={1} color="#D4AF37" style={{ opacity: 0.3 }} />
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
          <IconHistory size={16} stroke={1.5} />
          {t('accounts.tradeTabs.historyWithCount', { count: historyTotal })}
        </span>
      ),
      children:
        historyTrades.length === 0 ? (
          <div className="text-center py-12" style={{ color: '#8A9AA5' }}>
            <IconHistory size={48} stroke={1} color="#D4AF37" style={{ opacity: 0.3 }} />
            <p className="mt-4">{t('accounts.tradeTabs.emptyHistory')}</p>
          </div>
        ) : (
          <div>
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
                  analyticsApi.getRecentTrades(id, page, historyPageSize).then((data) => {
                    onHistoryTradesChange((data as any).trades || []);
                    onHistoryTotalChange((data as any).total || 0);
                    onHistoryPageChange(page);
                  });
                }}
                showSizeChanger={false}
                showTotal={(total) => t('accounts.tradeTabs.pagination.total', { total })}
              />
            </div>
          </div>
        ),
    },
  ];

  return <Tabs defaultActiveKey="positions" items={tradeTabs} className="px-4 pt-4" />;
}
