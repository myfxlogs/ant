import { useTranslation } from 'react-i18next';
import { Card, Table, Tag, Typography, Button, Space } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { depositApi } from '@/client/deposit';
import { queryKeys } from '@/queries/queryKeys';
import { formatAmount } from '@/utils/amount';
import { useMemo, useState } from 'react';

const { Title } = Typography;

export default function DepositManagement() {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);

  const { data, isLoading, refetch } = useQuery({
    queryKey: [...queryKeys.deposit.manualReview, page],
    queryFn: () => depositApi.listManualReviewDeposits({ page, pageSize: 20 }),
  });

  const columns = useMemo(() => [
    {
      title: t('admin.deposit.table.user', { defaultValue: 'User ID' }),
      dataIndex: 'userId',
      key: 'userId',
      width: 280,
      ellipsis: true,
      render: (v: string) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v}</span>,
    },
    {
      title: t('admin.deposit.table.amount', { defaultValue: 'USDT Amount' }),
      dataIndex: 'amount',
      key: 'amount',
      width: 120,
      render: (v: string) => <span style={{ color: '#00A651', fontWeight: 500 }}>+{formatAmount(v)}</span>,
    },
    {
      title: t('admin.deposit.table.txHash', { defaultValue: 'Tx Hash' }),
      dataIndex: 'txHash',
      key: 'txHash',
      width: 180,
      ellipsis: true,
      render: (v: string) => v ? <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v.slice(0, 20)}...</span> : '-',
    },
    {
      title: t('admin.deposit.table.block', { defaultValue: 'Block' }),
      dataIndex: 'blockNumber',
      key: 'blockNumber',
      width: 100,
    },
    {
      title: t('admin.deposit.table.confirmations', { defaultValue: 'Confirmations' }),
      dataIndex: 'confirmations',
      key: 'confirmations',
      width: 120,
    },
    {
      title: t('admin.deposit.table.status', { defaultValue: 'Status' }),
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (v: string) => {
        const colors: Record<string, string> = { CONFIRMED: 'green', MANUAL_REVIEW: 'orange' };
        return <Tag color={colors[v] || 'default'}>{v}</Tag>;
      },
    },
    {
      title: t('admin.deposit.table.time', { defaultValue: 'Time' }),
      dataIndex: 'confirmedAt',
      key: 'confirmedAt',
      width: 180,
      render: (v: unknown) => v ? new Date(Number((v as Record<string, unknown>).seconds) * 1000).toLocaleString() : '-',
    },
  ], [t]);

  return (
    <div>
      <Title level={4} style={{ margin: '0 0 16px 0', fontFamily: 'Poppins, sans-serif' }}>
        {t('admin.deposit.title', { defaultValue: 'Deposit Management' })}
      </Title>

      <Card size="small">
        <Space style={{ marginBottom: 16 }}>
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
            {t('common.refresh', { defaultValue: 'Refresh' })}
          </Button>
        </Space>

        <Table
          columns={columns}
          dataSource={data?.deposits || []}
          rowKey="id"
          loading={isLoading}
          pagination={{
            current: page,
            pageSize: 20,
            total: Number(data?.total || 0),
            onChange: setPage,
            size: 'small',
          }}
          size="small"
        />
      </Card>
    </div>
  );
}
