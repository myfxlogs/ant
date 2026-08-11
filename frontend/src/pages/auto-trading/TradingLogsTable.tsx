import { Table, Tag } from 'antd';
import { useTranslation } from 'react-i18next';
import type { TradingLog } from '@/gen/ant/v1/auto_trading_logs_pb';

interface Props {
  logs: TradingLog[];
  loading: boolean;
}

export default function TradingLogsTable({ logs, loading }: Props) {
  const { t } = useTranslation();

  const columns = [
    {
      title: t('autoTrading.logs.columns.time'),
      dataIndex: 'createdAt',
      key: 'time',
      width: 160,
      render: (v: Date | undefined) =>
        v ? new Date(v).toLocaleString() : '-',
    },
    {
      title: t('autoTrading.logs.columns.symbol'),
      dataIndex: 'symbol',
      key: 'symbol',
      width: 80,
      render: (v: string) => <Tag>{v || '-'}</Tag>,
    },
    {
      title: t('autoTrading.logs.columns.action'),
      dataIndex: 'action',
      key: 'action',
      width: 70,
      render: (v: string) => (
        <Tag color={v === 'BUY' ? 'green' : v === 'SELL' ? 'red' : 'default'}>
          {v || '-'}
        </Tag>
      ),
    },
    {
      title: t('autoTrading.logs.columns.volume'),
      dataIndex: 'volume',
      key: 'volume',
      width: 70,
      render: (v: number) => v?.toFixed(2) || '-',
    },
    {
      title: t('autoTrading.logs.columns.price'),
      dataIndex: 'price',
      key: 'price',
      width: 80,
      render: (v: number) => v?.toFixed(5) || '-',
    },
    {
      title: t('autoTrading.logs.columns.profit'),
      dataIndex: 'profit',
      key: 'profit',
      width: 90,
      render: (v: number) => {
        if (v === undefined || v === null) return '-';
        const color = v >= 0 ? '#26a69a' : '#ef5350';
        return <span style={{ color, fontWeight: 500 }}>{v >= 0 ? '+' : ''}{v.toFixed(2)}</span>;
      },
    },
    {
      title: t('autoTrading.logs.columns.ticket'),
      dataIndex: 'ticket',
      key: 'ticket',
      width: 90,
      render: (v: bigint | number) => v ? String(v) : '-',
    },
  ];

  return (
    <Table
      columns={columns}
      dataSource={logs}
      rowKey="id"
      loading={loading}
      size="small"
      pagination={{ pageSize: 10, size: 'small', showTotal: (n) => `${n}` }}
      locale={{ emptyText: t('autoTrading.logs.empty') }}
    />
  );
}
