import React, { useState } from 'react';
import { Button, Card, message, Popconfirm, Space, Table, Tag, Tooltip, Typography } from 'antd';
import type { ColumnsType, TableRowSelection } from 'antd/es/table';
import { DeleteOutlined, SyncOutlined } from '@ant-design/icons';
import { formatDateTime } from '@/utils/date';
import { useTranslation } from 'react-i18next'
import { BACKTEST_RUNS_ACTIONS_LAUNCH_SCHEDULE_KEY, BACKTEST_RUNS_ACTIONS_VIEW_KEY, BACKTEST_RUNS_BATCH_DELETE_CONFIRM_KEY, BACKTEST_RUNS_BATCH_DELETE_SUCCESS_KEY, BACKTEST_RUNS_DELETE_CONFIRM_KEY, BACKTEST_RUNS_EMPTY_KEY, BACKTEST_RUNS_STATUS_CANCELED_KEY, BACKTEST_RUNS_STATUS_CANCELING_KEY, BACKTEST_RUNS_STATUS_COMPLETED_KEY, BACKTEST_RUNS_STATUS_FAILED_KEY, BACKTEST_RUNS_STATUS_QUEUED_KEY, BACKTEST_RUNS_STATUS_RUNNING_KEY, BACKTEST_RUNS_TABLE_ACTIONS_KEY, BACKTEST_RUNS_TABLE_CREATED_AT_KEY, BACKTEST_RUNS_TABLE_STATUS_KEY, BACKTEST_RUNS_TABLE_SYMBOL_KEY, BACKTEST_RUNS_TABLE_TIMEFRAME_KEY, BACKTEST_RUNS_TABLE_TITLE_KEY, BACKTEST_RUNS_TITLE_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

;

const { Text } = Typography;

type Props = {
  runs: any[];
  loading: boolean;
  onRefresh: () => void;
  onView: (runId: string) => void;
  onViewScore?: (runId: string) => void;
  onAddToSchedule?: (run: BacktestRun) => void;
  onDelete: (runId: string) => void;
  onBatchDelete?: (runIds: string[]) => Promise<void>;
};

function statusText(s: unknown, t: (key: string) => string) {
  switch (Number(s)) {
    case 1:
      return t(BACKTEST_RUNS_STATUS_QUEUED_KEY);
    case 2:
      return t(BACKTEST_RUNS_STATUS_RUNNING_KEY);
    case 3:
      return t(BACKTEST_RUNS_STATUS_COMPLETED_KEY);
    case 4:
      return t(BACKTEST_RUNS_STATUS_FAILED_KEY);
    case 5:
      return t(BACKTEST_RUNS_STATUS_CANCELING_KEY);
    case 6:
      return t(BACKTEST_RUNS_STATUS_CANCELED_KEY);
    default:
      return String(s ?? '-');
  }
}

const BacktestRunsCard: React.FC<Props> = ({ runs, loading, onRefresh, onView, onViewScore, onDelete, onBatchDelete }) => {
  const { t } = useTranslation();
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [batchDeleting, setBatchDeleting] = useState(false);

  const rowSelection: TableRowSelection<unknown> = {
    selectedRowKeys,
    onChange: (keys) => setSelectedRowKeys(keys),
    selections: [
      Table.SELECTION_ALL,
      Table.SELECTION_INVERT,
      Table.SELECTION_NONE,
    ],
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;
    setBatchDeleting(true);
    try {
      if (onBatchDelete) {
        await onBatchDelete(selectedRowKeys.map(String));
      } else {
        // Fallback: delete one by one
        for (const id of selectedRowKeys) {
          onDelete(String(id));
        }
      }
      message.success(
        t(BACKTEST_RUNS_BATCH_DELETE_SUCCESS_KEY, {
          count: selectedRowKeys.length,
        }),
      );
      setSelectedRowKeys([]);
    } catch {
      message.error(t('common.deleteFailed'));
    } finally {
      setBatchDeleting(false);
    }
  };
  const columns: ColumnsType<unknown> = [
    {
      title: t(BACKTEST_RUNS_TABLE_TITLE_KEY),
      dataIndex: 'title',
      key: 'title',
      width: 260,
      ellipsis: true,
      render: (_text: unknown, r: BacktestRun) => {
        const base = String(_text || '').trim();
        const fallback = [
          formatDateTime(r?.createdAt),
          String(r?.symbol || '').trim(),
          String(r?.timeframe || '').trim(),
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
      title: t(BACKTEST_RUNS_TABLE_STATUS_KEY),
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (s: unknown) => <Tag>{statusText(s, t)}</Tag>,
    },
    {
      title: t(BACKTEST_RUNS_TABLE_SYMBOL_KEY),
      dataIndex: 'symbol',
      key: 'symbol',
      width: 120,
    },
    {
      title: t(BACKTEST_RUNS_TABLE_TIMEFRAME_KEY),
      dataIndex: 'timeframe',
      key: 'timeframe',
      width: 100,
    },
    {
      title: t(BACKTEST_RUNS_TABLE_CREATED_AT_KEY),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (v: unknown) => formatDateTime(v as string),
    },
    {
      title: t(BACKTEST_RUNS_TABLE_ACTIONS_KEY),
      key: 'action',
      width: 220,
      fixed: 'right',
      render: (_: unknown, r: BacktestRun) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => onView(String(r.id || ''))}>
            {t(BACKTEST_RUNS_ACTIONS_VIEW_KEY)}
          </Button>
          {typeof onViewScore === 'function' ? (
            <Button type="link" size="small" onClick={() => onViewScore(String(r.id || ''))}>
              {t(BACKTEST_RUNS_ACTIONS_LAUNCH_SCHEDULE_KEY)}
            </Button>
          ) : null}
          <Popconfirm title={t(BACKTEST_RUNS_DELETE_CONFIRM_KEY)} onConfirm={() => onDelete(String(r.id || ''))}>
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
      title={t(BACKTEST_RUNS_TITLE_KEY)}
      extra={
        <Space>
          {selectedRowKeys.length > 0 && (
            <Popconfirm
              title={t(BACKTEST_RUNS_BATCH_DELETE_CONFIRM_KEY, {
                count: selectedRowKeys.length,
              })}
              onConfirm={handleBatchDelete}
            >
              <Button
                danger
                icon={<DeleteOutlined />}
                loading={batchDeleting}
              >
                {t('common.deleteSelected', { count: selectedRowKeys.length })}
              </Button>
            </Popconfirm>
          )}
          <Button icon={<SyncOutlined />} onClick={onRefresh} loading={loading}>
            {t('common.refresh')}
          </Button>
        </Space>
      }
    >
      <Table
        rowSelection={rowSelection}
        columns={columns}
        dataSource={runs}
        rowKey={(r) => String(r?.id || '')}
        loading={loading}
        scroll={{ x: 'max-content' }}
        pagination={{
          defaultPageSize: 10,
          pageSizeOptions: ['10', '20', '50'],
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => t('common.totalItems', { total }),
        }}
        locale={{ emptyText: t(BACKTEST_RUNS_EMPTY_KEY) }}
      />
    </Card>
  );
};

export default BacktestRunsCard;
