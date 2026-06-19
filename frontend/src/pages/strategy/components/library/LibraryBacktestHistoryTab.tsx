import { useState } from 'react';
import { Button, Popconfirm, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType, TableRowSelection } from 'antd/es/table';
import { DeleteOutlined, SyncOutlined } from '@ant-design/icons';
import { formatDateTime } from '@/utils/date';
import { useTranslation } from 'react-i18next'
import { BACKTEST_RUNS_ACTIONS_VIEW_KEY, BACKTEST_RUNS_BATCH_DELETE_CONFIRM_KEY, BACKTEST_RUNS_DELETE_CONFIRM_KEY, BACKTEST_RUNS_EMPTY_KEY, BACKTEST_RUNS_STATUS_CANCELED_KEY, BACKTEST_RUNS_STATUS_CANCELING_KEY, BACKTEST_RUNS_STATUS_COMPLETED_KEY, BACKTEST_RUNS_STATUS_FAILED_KEY, BACKTEST_RUNS_STATUS_QUEUED_KEY, BACKTEST_RUNS_STATUS_RUNNING_KEY, BACKTEST_RUNS_TABLE_ACTIONS_KEY, BACKTEST_RUNS_TABLE_CREATED_AT_KEY, BACKTEST_RUNS_TABLE_STATUS_KEY, BACKTEST_RUNS_TABLE_SYMBOL_KEY, BACKTEST_RUNS_TABLE_TIMEFRAME_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

;
import { useLibraryCtx } from '../../LibraryContext';
import { StatusResult } from '@/components/common/StatusResult';
import type { BacktestRunRow } from '../../hooks/libraryTypes';

const { Text } = Typography;

function statusLabel(s: unknown, t: (key: string) => string): string {
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

export default function LibraryBacktestHistoryTab() {
  const { t } = useTranslation();
  const lib = useLibraryCtx();
  const b = lib.backtestProps;
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  const rowSelection: TableRowSelection<BacktestRunRow> = {
    selectedRowKeys,
    onChange: (keys) => setSelectedRowKeys(keys),
    selections: [Table.SELECTION_ALL, Table.SELECTION_INVERT, Table.SELECTION_NONE],
  };

  const columns: ColumnsType<BacktestRunRow> = [
    { title: t(BACKTEST_RUNS_TABLE_STATUS_KEY), dataIndex: 'status', key: 'status', width: 100,
      render: (s: unknown) => <Tag>{statusLabel(s, t)}</Tag> },
    { title: t(BACKTEST_RUNS_TABLE_TRADING_SYMBOL_KEY), dataIndex: 'symbol', key: 'symbol', width: 120,
      render: (v: string) => <Text>{v || '-'}</Text> },
    { title: t(BACKTEST_RUNS_TABLE_TIMEFRAME_KEY), dataIndex: 'timeframe', key: 'timeframe', width: 90,
      render: (v: string) => <Text>{v || '-'}</Text> },
    { title: t(BACKTEST_RUNS_TABLE_CREATED_AT_KEY), dataIndex: 'createdAt', key: 'createdAt', width: 170,
      render: (v: string | undefined) => <Text>{v ? formatDateTime(v) : '-'}</Text> },
    { title: t(BACKTEST_RUNS_TABLE_ACTIONS_KEY), key: 'actions', width: 160,
      render: (_: unknown, record: BacktestRunRow) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => b.onViewRun(record.id)}>{t(BACKTEST_RUNS_ACTIONS_VIEW_KEY)}</Button>
          <Popconfirm title={t(BACKTEST_RUNS_DELETE_CONFIRM_KEY)} onConfirm={() => b.onDeleteRun(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '16px 0' }}>
      <StatusResult loading={b.loading} error={b.error || undefined} onRetry={b.onRefresh}>
        <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between' }}>
          <Space>
            <Button icon={<SyncOutlined />} onClick={b.onRefresh} loading={b.loading}>{t('common.refresh')}</Button>
            {selectedRowKeys.length > 0 && (
              <Popconfirm title={t(BACKTEST_RUNS_BATCH_DELETE_CONFIRM_KEY, { count: selectedRowKeys.length })}
                onConfirm={b.onBatchDelete}>
                <Button danger loading={b.deleting}>
                  {t('strategy.templates.backtestRuns.batchDelete', '删除 {{count}} 条', { count: selectedRowKeys.length })}
                </Button>
              </Popconfirm>
            )}
          </Space>
        </div>
        <Table<BacktestRunRow>
          rowKey="id" columns={columns} dataSource={b.runs}
          loading={b.loading} rowSelection={rowSelection}
          pagination={{ current: b.page, pageSize: b.pageSize, total: b.total, showSizeChanger: true, showQuickJumper: true,
            showTotal: (t: number) => `${t} runs`, pageSizeOptions: ['10', '20', '50'], onChange: b.onPageChange }}
          size="small" scroll={{ x: 640 }}
          locale={{ emptyText: t(BACKTEST_RUNS_EMPTY_KEY) }}
        />
      </StatusResult>
    </div>
  );
}
