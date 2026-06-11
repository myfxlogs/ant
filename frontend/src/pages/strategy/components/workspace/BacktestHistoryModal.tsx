import React, { useMemo } from 'react';
import { Button, Modal, Popconfirm, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType, TableRowSelection } from 'antd/es/table';
import { DeleteOutlined, ReloadOutlined } from '@ant-design/icons';
import { formatDateTime } from '@/utils/date';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

interface Props {
  open: boolean;
  runs: any[];
  loading: boolean;
  page: number;
  pageSize: number;
  total: number;
  selectedRowKeys: React.Key[];
  deleting: boolean;
  onPageChange: (page: number, pageSize: number) => void;
  onSelectionChange: (keys: React.Key[]) => void;
  onViewRun: (runId: string) => void;
  onDeleteRun: (runId: string) => void;
  onBatchDelete: () => void;
  onRefresh: () => void;
  onClose: () => void;
}

function statusText(s: unknown, t: (key: string) => string): string {
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

function statusColor(s: unknown): string {
  switch (Number(s)) {
    case 1: return 'default';
    case 2: return 'processing';
    case 3: return 'success';
    case 4: return 'error';
    case 5: return 'warning';
    case 6: return 'warning';
    default: return 'default';
  }
}

const BacktestHistoryModal: React.FC<Props> = ({
  open, runs, loading, page, pageSize, total,
  selectedRowKeys, deleting,
  onPageChange, onSelectionChange,
  onViewRun, onDeleteRun, onBatchDelete, onRefresh, onClose,
}) => {
  const { t } = useTranslation();

  const rowSelection: TableRowSelection<any> = {
    selectedRowKeys,
    onChange: (keys) => onSelectionChange(keys),
    selections: [Table.SELECTION_ALL, Table.SELECTION_INVERT, Table.SELECTION_NONE],
  };

  const columns: ColumnsType<any> = useMemo(() => [
    {
      title: t('strategy.templates.backtestRuns.status', 'Status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (s: unknown) => <Tag color={statusColor(s)}>{statusText(s, t)}</Tag>,
    },
    {
      title: t('strategy.templates.backtestRuns.symbol', 'Symbol'),
      dataIndex: 'symbol',
      key: 'symbol',
      width: 110,
      render: (v: string) => <Text>{v || '-'}</Text>,
    },
    {
      title: t('strategy.templates.backtestRuns.timeframe', 'Timeframe'),
      dataIndex: 'timeframe',
      key: 'timeframe',
      width: 90,
      render: (v: string) => <Text>{v || '-'}</Text>,
    },
    {
      title: t('strategy.templates.backtestRuns.created', 'Created'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      render: (v: string | undefined) => <Text>{v ? formatDateTime(v) : '-'}</Text>,
    },
    {
      title: t('strategy.templates.backtestRuns.actions', 'Actions'),
      key: 'actions',
      width: 140,
      render: (_: unknown, record: any) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => onViewRun(record.id)}>
            {t('strategy.backtestHistory.actions.view', 'View')}
          </Button>
          <Popconfirm
            title={t('strategy.backtestHistory.deleteConfirm', 'Delete this backtest run?')}
            onConfirm={() => onDeleteRun(record.id)}
            okText={t('common.yes', 'Yes')}
            cancelText={t('common.no', 'No')}
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ], [t, onViewRun, onDeleteRun]);

  return (
    <Modal
      title={t('strategy.backtestHistory.title', 'Backtest History')}
      open={open}
      onCancel={onClose}
      width={1100}
      footer={null}
      destroyOnClose
    >
      <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between' }}>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={onRefresh} loading={loading}>
            {t('common.refresh', 'Refresh')}
          </Button>
          {selectedRowKeys.length > 0 && (
            <Popconfirm
              title={t('strategy.backtestHistory.batchDeleteConfirm', 'Delete {{count}} selected backtest runs?', { count: selectedRowKeys.length })}
              onConfirm={onBatchDelete}
              okText={t('common.yes', 'Yes')}
              cancelText={t('common.no', 'No')}
            >
              <Button danger loading={deleting}>
                {t('strategy.backtestHistory.batchDelete', 'Delete {{count}} selected', { count: selectedRowKeys.length })}
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
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (t: number) => `${t} runs`,
          pageSizeOptions: ['10', '20', '50'],
          onChange: onPageChange,
        }}
        size="small"
        locale={{ emptyText: t('strategy.backtestHistory.empty', 'No backtest runs found') }}
        scroll={{ x: 610 }}
      />
    </Modal>
  );
};

export default BacktestHistoryModal;
