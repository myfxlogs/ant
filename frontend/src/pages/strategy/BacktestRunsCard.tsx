import React from 'react';
import { Button, Card, Popconfirm, Space, Table, Tag, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, SyncOutlined } from '@ant-design/icons';
import { formatDateTime } from '@/utils/date';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

type Props = {
  runs: any[];
  loading: boolean;
  onRefresh: () => void;
  onView: (runId: string) => void;
  onViewScore?: (runId: string) => void;
  onAddToSchedule?: (run: any) => void;
  onDelete: (runId: string) => void;
};

function statusText(s: any, t: (key: string) => string) {
  switch (Number(s)) {
    case 1:
      return t('strategy.templates.backtestRuns.status.queued');
    case 2:
      return t('strategy.templates.backtestRuns.status.running');
    case 3:
      return t('strategy.templates.backtestRuns.status.completed');
    case 4:
      return t('strategy.templates.backtestRuns.status.failed');
    case 5:
      return t('strategy.templates.backtestRuns.status.canceling');
    case 6:
      return t('strategy.templates.backtestRuns.status.canceled');
    default:
      return String(s ?? '-');
  }
}

const BacktestRunsCard: React.FC<Props> = ({ runs, loading, onRefresh, onView, onViewScore, onDelete }) => {
  const { t } = useTranslation();
  const columns: ColumnsType<any> = [
    {
      title: t('strategy.templates.backtestRuns.table.title'),
      dataIndex: 'title',
      key: 'title',
      width: 260,
      ellipsis: true,
      render: (t: any, r: any) => {
        const base = String(t || '').trim();
        const fallback = [
          formatDateTime((r as any)?.createdAt),
          String((r as any)?.symbol || '').trim(),
          String((r as any)?.timeframe || '').trim(),
        ]
          .filter(Boolean)
          .join(' ');
        const titleText = base || fallback || '-';
        return (
          <Tooltip title={String(r?.id || '')}>
            <Text>{titleText}</Text>
          </Tooltip>
        );
      },
    },
    {
      title: t('strategy.templates.backtestRuns.table.status'),
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (s: any) => <Tag>{statusText(s, t)}</Tag>,
    },
    {
      title: t('strategy.templates.backtestRuns.table.symbol'),
      dataIndex: 'symbol',
      key: 'symbol',
      width: 120,
    },
    {
      title: t('strategy.templates.backtestRuns.table.timeframe'),
      dataIndex: 'timeframe',
      key: 'timeframe',
      width: 100,
    },
    {
      title: t('strategy.templates.backtestRuns.table.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (v: any) => formatDateTime(v),
    },
    {
      title: t('strategy.templates.backtestRuns.table.actions'),
      key: 'action',
      width: 220,
      fixed: 'right',
      render: (_: any, r: any) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => onView(String(r.id || ''))}>
            {t('strategy.templates.backtestRuns.actions.view')}
          </Button>
          {typeof onViewScore === 'function' ? (
            <Button type="link" size="small" onClick={() => onViewScore(String(r.id || ''))}>
              {t('strategy.templates.backtestRuns.actions.launchSchedule')}
            </Button>
          ) : null}
          <Popconfirm title={t('strategy.templates.backtestRuns.deleteConfirm')} onConfirm={() => onDelete(String(r.id || ''))}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              {t('common.delete')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Card
      title={t('strategy.templates.backtestRuns.title')}
      extra={
        <Button icon={<SyncOutlined />} onClick={onRefresh} loading={loading}>
          {t('common.refresh')}
        </Button>
      }
    >
      <Table
        columns={columns}
        dataSource={runs}
        rowKey={(r) => String((r as any)?.id || '')}
        loading={loading}
        scroll={{ x: 'max-content' }}
        pagination={{
          defaultPageSize: 10,
          pageSizeOptions: ['10', '20', '50'],
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => t('common.totalItems', { total }),
        }}
        locale={{ emptyText: t('strategy.templates.backtestRuns.empty') }}
      />
    </Card>
  );
};

export default BacktestRunsCard;
