import { useState } from 'react';
import { Button, message, Popconfirm, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType, TableRowSelection } from 'antd/es/table';
import { DeleteOutlined, SyncOutlined } from '@ant-design/icons';
import { formatDateTime } from '@/utils/date';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

function statusLabel(s: unknown, t: (key: string) => string): string {
  switch (Number(s)) {
    case 1: return t('strategy.templates.backtestRuns.status.queued');
    case 2: return t('strategy.templates.backtestRuns.status.running');
    case 3: return t('strategy.templates.backtestRuns.status.completed');
    case 4: return t('strategy.templates.backtestRuns.status.failed');
    case 5: return t('strategy.templates.backtestRuns.status.canceling');
    case 6: return t('strategy.templates.backtestRuns.status.canceled');
    default: return String(s ?? '-');
  }
}

interface Props {
  runs: any[];
  loading: boolean;
  page: number;
  pageSize: number;
  total: number;
  deleting: boolean;
  onPageChange: (page: number, pageSize: number) => void;
  onViewRun: (runId: string) => void;
  onDeleteRun: (runId: string) => void;
  onBatchDelete: (runIds: string[]) => void;
  onRefresh: () => void;
}

export default function LibraryBacktestHistoryTab({
  runs, loading, page, pageSize, total,
  deleting, onPageChange, onViewRun, onDeleteRun, onBatchDelete, onRefresh,
}: Props) {
  const { t } = useTranslation();
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  const rowSelection: TableRowSelection<any> = {
    selectedRowKeys,
    onChange: (keys) => setSelectedRowKeys(keys),
    selections: [Table.SELECTION_ALL, Table.SELECTION_INVERT, Table.SELECTION_NONE],
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;
    onBatchDelete(selectedRowKeys.map(String));
    setSelectedRowKeys([]);
  };

  const columns: ColumnsType<any> = [
    {
      title: t('strategy.templates.backtestRuns.table.status'),
      dataIndex: 'status', key: 'status', width: 100,
      render: (s: unknown) => <Tag>{statusLabel(s, t)}</Tag>,
    },
    {
      title: t('strategy.templates.backtestRuns.table.symbol'),
      dataIndex: 'symbol', key: 'symbol', width: 120,
      render: (v: string) => <Text>{v || '-'}</Text>,
    },
    {
      title: t('strategy.templates.backtestRuns.table.timeframe'),
      dataIndex: 'timeframe', key: 'timeframe', width: 90,
      render: (v: string) => <Text>{v || '-'}</Text>,
    },
    {
      title: t('strategy.templates.backtestRuns.table.createdAt'),
      dataIndex: 'createdAt', key: 'createdAt', width: 170,
      render: (v: string | undefined) => <Text>{v ? formatDateTime(v) : '-'}</Text>,
    },
    {
      title: t('strategy.templates.backtestRuns.table.actions'),
      key: 'actions', width: 160,
      render: (_: unknown, record: any) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => onViewRun(record.id)}>
            {t('strategy.templates.backtestRuns.actions.view', '查看')}
          </Button>
          <Popconfirm
            title={t('strategy.templates.backtestRuns.deleteConfirm', '删除这条回测记录?')}
            onConfirm={() => onDeleteRun(record.id)}
            okText={String(t('common.yes'))} cancelText={String(t('common.no'))}
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '16px 0' }}>
      <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between' }}>
        <Space>
          <Button icon={<SyncOutlined />} onClick={onRefresh} loading={loading}>
            {t('common.refresh')}
          </Button>
          {selectedRowKeys.length > 0 && (
            <Popconfirm
              title={t('strategy.templates.backtestRuns.batchDeleteConfirm', { count: selectedRowKeys.length })}
              onConfirm={handleBatchDelete}
              okText={String(t('common.yes'))} cancelText={String(t('common.no'))}
            >
              <Button danger loading={deleting}>
                {t('strategy.backtestHistory.batchDelete', '删除 {{count}} 条', { count: selectedRowKeys.length })}
              </Button>
            </Popconfirm>
          )}
        </Space>
      </div>
      <Table
        rowKey="id"
        columns={columns}
        dataSource={runs}
        loading={loading}
        rowSelection={rowSelection}
        pagination={{
          current: page, pageSize, total,
          showSizeChanger: true, showQuickJumper: true,
          showTotal: (t: number) => `${t} runs`,
          pageSizeOptions: ['10', '20', '50'],
          onChange: onPageChange,
        }}
        size="small"
        scroll={{ x: 640 }}
        locale={{ emptyText: t('strategy.backtestHistory.empty', '暂无回测记录') }}
      />
    </div>
  );
}
