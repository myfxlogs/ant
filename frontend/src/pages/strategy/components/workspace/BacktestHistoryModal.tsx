import React, { useMemo } from 'react';
import { Button, Modal, Popconfirm, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType, TableRowSelection } from 'antd/es/table';
import { DeleteOutlined, ReloadOutlined } from '@ant-design/icons';
import { formatDateTime } from '@/utils/date';
import { useTranslation } from 'react-i18next'
import { BACKTEST_RUNS_ACTIONS_VIEW_KEY, BACKTEST_RUNS_DELETE_CONFIRM_KEY, BACKTEST_RUNS_EMPTY_KEY, BACKTEST_RUNS_STATUS_CANCELED_KEY, BACKTEST_RUNS_STATUS_CANCELING_KEY, BACKTEST_RUNS_STATUS_COMPLETED_KEY, BACKTEST_RUNS_STATUS_FAILED_KEY, BACKTEST_RUNS_STATUS_QUEUED_KEY, BACKTEST_RUNS_STATUS_RUNNING_KEY, BACKTEST_RUNS_TABLE_ACTIONS_KEY, BACKTEST_RUNS_TABLE_CREATED_AT_KEY, BACKTEST_RUNS_TABLE_STATUS_KEY, BACKTEST_RUNS_TABLE_SYMBOL_KEY, BACKTEST_RUNS_TABLE_TIMEFRAME_KEY, BACKTEST_RUNS_TITLE_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

;

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
    case 1: return t(BACKTEST_RUNS_STATUS_QUEUED_KEY);
    case 2: return t(BACKTEST_RUNS_STATUS_RUNNING_KEY);
    case 3: return t(BACKTEST_RUNS_STATUS_COMPLETED_KEY);
    case 4: return t(BACKTEST_RUNS_STATUS_FAILED_KEY);
    case 5: return t(BACKTEST_RUNS_STATUS_CANCELING_KEY);
    case 6: return t(BACKTEST_RUNS_STATUS_CANCELED_KEY);
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
      title: t(BACKTEST_RUNS_TABLE_STATUS_KEY, 'Status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (s: unknown) => <Tag color={statusColor(s)}>{statusText(s, t)}</Tag>,
    },
    {
      title: t(BACKTEST_RUNS_TABLE_TRADING_SYMBOL_KEY, 'Symbol'),
      dataIndex: 'symbol',
      key: 'symbol',
      width: 110,
      render: (v: string) => <Text>{v || '-'}</Text>,
    },
    {
      title: t(BACKTEST_RUNS_TABLE_TIMEFRAME_KEY, 'Timeframe'),
      dataIndex: 'timeframe',
      key: 'timeframe',
      width: 90,
      render: (v: string) => <Text>{v || '-'}</Text>,
    },
    {
      title: t(BACKTEST_RUNS_TABLE_CREATED_AT_KEY, 'Created'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      render: (v: string | undefined) => <Text>{v ? formatDateTime(v) : '-'}</Text>,
    },
    {
      title: t(BACKTEST_RUNS_TABLE_ACTIONS_KEY, 'Actions'),
      key: 'actions',
      width: 140,
      render: (_: unknown, record: any) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => onViewRun(record.id)}>
            {t(BACKTEST_RUNS_ACTIONS_VIEW_KEY, 'View')}
          </Button>
          <Popconfirm
            title={t(BACKTEST_RUNS_DELETE_CONFIRM_KEY, 'Delete this backtest run?')}
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
      title={t(BACKTEST_RUNS_TRADING_TITLE_KEY, 'Backtest History')}
      open={open}
      onCancel={onClose}
      width={1100}
      footer={null}
    >
      <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between' }}>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={onRefresh} loading={loading}>
            {t('common.refresh', 'Refresh')}
          </Button>
          {selectedRowKeys.length > 0 && (
            <Popconfirm
              title={t('strategy.templates.backtestRuns.batchDeleteConfirm', 'Delete {{count}} selected backtest runs?', { count: selectedRowKeys.length })}
              onConfirm={onBatchDelete}
              okText={t('common.yes', 'Yes')}
              cancelText={t('common.no', 'No')}
            >
              <Button danger loading={deleting}>
                {t('strategy.templates.backtestRuns.batchDelete', 'Delete {{count}}', { count: selectedRowKeys.length })}
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
        locale={{ emptyText: t(BACKTEST_RUNS_EMPTY_KEY, 'No backtest runs found') }}
        scroll={{ x: 610 }}
      />
    </Modal>
  );
};

export default BacktestHistoryModal;
