import { Table, Tag, Typography, Button, Space, Empty } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { EyeOutlined, ExportOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { formatDateTime } from '@/utils/date';

const { Text } = Typography;

interface Props {
  purchases: any[];
  loading: boolean;
  onViewDetail: (strategyId: string) => void;
  onOpenInWorkspace: (strategyId: string) => void;
}

export default function PurchaseTab({ purchases, loading, onViewDetail, onOpenInWorkspace }: Props) {
  const { t } = useTranslation();

  if (purchases.length === 0 && !loading) {
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
          <Button size="small" icon={<EyeOutlined />} onClick={() => onViewDetail(row.strategyId)}>
            {t('strategy.backtestHistory.actions.view')}
          </Button>
          <Button size="small" icon={<ExportOutlined />} onClick={() => onOpenInWorkspace(row.strategyId)}>
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
      dataSource={purchases}
      loading={loading}
      pagination={{ pageSize: 10 }}
      size="small"
    />
  );
}
