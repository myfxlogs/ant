import { useState } from 'react';
import { Table, Tag, Typography, Button, Space, Empty, Drawer } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { EyeOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'

;
import { formatDateTime } from '@/utils/date';
import { useMarketplaceCtx } from '../MarketplaceContext';
import type { PurchasedItem } from '../hooks/useMarketplace';
import ProtectedBacktestPanel from './ProtectedBacktestPanel';

const { Text } = Typography;

export default function PurchaseTab() {
  const { t } = useTranslation();
  const m = useMarketplaceCtx();
  const [backtestDrawerOpen, setBacktestDrawerOpen] = useState(false);
  const [backtestStrategyId, setBacktestStrategyId] = useState('');

  if (!m.purchasesLoading && m.purchases.length === 0) {
    return <Empty description={t('marketplace.purchases.empty')} />;
  }

  const columns: ColumnsType<PurchasedItem> = [
    {
      title: t('marketplace.purchases.strategy'),
      dataIndex: 'strategyId', key: 'strategy',
      render: (id: string, row: PurchasedItem) => {
        const s = m.strategies.find(st => st.strategyId === row.strategyId);
        return <Text>{s?.title || s?.strategyName || String(id).slice(0, 12)}</Text>;
      },
    },
    {
      title: t('marketplace.purchases.date'),
      key: 'date',
      render: (_: unknown, row: PurchasedItem) => <Text>{formatDateTime(String(row.createdAt || row.purchasedAt || ''))}</Text>,
    },
    {
      title: t('marketplace.purchases.status'),
      key: 'status',
      render: (_: unknown, row: PurchasedItem) => (
        row.active
          ? <Tag color="green">{t('common.active')}</Tag>
          : <Tag>{t('common.inactive')}</Tag>
      ),
    },
    {
      title: t('marketplace.purchases.actions'),
      key: 'actions',
      render: (_: unknown, row: PurchasedItem) => (
        <Space>
          <Button size="small" icon={<EyeOutlined />} onClick={() => {
            const s = m.strategies.find(s => s.strategyId === row.strategyId);
            if (s) m.openDetail(s);
          }}>
            {t('strategy.backtestHistory.actions.view')}
          </Button>
          <Button size="small" type="primary" icon={<ThunderboltOutlined />} onClick={() => {
            setBacktestStrategyId(row.strategyId);
            setBacktestDrawerOpen(true);
          }}>
            {t('marketplace.purchases.runBacktest', 'Run Backtest')}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <>
      <Table
        rowKey="subscriptionId"
        columns={columns}
        dataSource={m.purchases}
        loading={m.purchasesLoading}
        pagination={{ pageSize: 10 }}
        size="small"
      />

      <Drawer
        title={t('marketplace.backtest.title', 'Strategy Backtest')}
        open={backtestDrawerOpen}
        onClose={() => setBacktestDrawerOpen(false)}
        width={640}
        destroyOnClose
      >
        {backtestStrategyId && <ProtectedBacktestPanel strategyId={backtestStrategyId} />}
      </Drawer>
    </>
  );
}
