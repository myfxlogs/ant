import { Table, Tag, Typography, Button, Space, Empty } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { EyeOutlined, ExportOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { formatDateTime } from '@/utils/date';
import { useMarketplaceCtx } from '../MarketplaceContext';

const { Text } = Typography;

export default function PurchaseTab() {
  const { t } = useTranslation();
  const m = useMarketplaceCtx();

  if (!m.purchasesLoading && m.purchases.length === 0) {
    return <Empty description={t('marketplace.purchases.empty')} />;
  }

  const columns: ColumnsType<any> = [
    {
      title: t('marketplace.purchases.strategy'),
      dataIndex: 'strategyId', key: 'strategy',
      render: (id: string) => <Text>{String(id).slice(0, 12)}</Text>,
    },
    {
      title: t('marketplace.purchases.date'),
      key: 'date',
      render: (_: unknown, row: any) => <Text>{formatDateTime(row.createdAt || row.purchasedAt)}</Text>,
    },
    {
      title: t('marketplace.purchases.status'),
      key: 'status',
      render: (_: unknown, row: any) => (
        row.active
          ? <Tag color="green">{t('common.active')}</Tag>
          : <Tag>{t('common.inactive')}</Tag>
      ),
    },
    {
      title: t('marketplace.purchases.actions'),
      key: 'actions',
      render: (_: unknown, row: any) => (
        <Space>
          <Button size="small" icon={<EyeOutlined />} onClick={() => {
            const s = m.strategies.find(s => s.strategyId === row.strategyId);
            if (s) m.openDetail(s);
          }}>
            {t('strategy.backtestHistory.actions.view')}
          </Button>
          <Button size="small" icon={<ExportOutlined />} onClick={() => {
            window.open(`/strategy/workspace?templateId=${row.strategyId}`, '_blank');
          }}>
            {t('strategy.library.openInWorkspace')}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <Table
      rowKey="subscriptionId"
      columns={columns}
      dataSource={m.purchases}
      loading={m.purchasesLoading}
      pagination={{ pageSize: 10 }}
      size="small"
    />
  );
}
