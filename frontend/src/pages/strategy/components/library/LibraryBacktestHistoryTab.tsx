import { useState } from 'react';
import { Button, Popconfirm, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType, TableRowSelection } from 'antd/es/table';
import { DeleteOutlined, SyncOutlined } from '@ant-design/icons';
import { formatDateTime } from '@/utils/date';
import { useTranslation } from 'react-i18next';
import { useLibraryCtx } from '../../LibraryContext';
import { StatusResult } from '@/components/common/StatusResult';
import type { BacktestRunRow } from '../../hooks/libraryTypes';

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
    { title: t('strategy.templates.backtestRuns.table.status'), dataIndex: 'status', key: 'status', width: 100,
      render: (s: unknown) => <Tag>{statusLabel(s, t)}</Tag> },
    { title: t('strategy.templates.backtestRuns.table.symbol'), dataIndex: 'symbol', key: 'symbol', width: 120,
      render: (v: string) => <Text>{v || '-'}</Text> },
    { title: t('strategy.templates.backtestRuns.table.timeframe'), dataIndex: 'timeframe', key: 'timeframe', width: 90,
      render: (v: string) => <Text>{v || '-'}</Text> },
    { title: t('strategy.templates.backtestRuns.table.createdAt'), dataIndex: 'createdAt', key: 'createdAt', width: 170,
      render: (v: string | undefined) => <Text>{v ? formatDateTime(v) : '-'}</Text> },
    { title: t('strategy.templates.backtestRuns.table.actions'), key: 'actions', width: 160,
      render: (_: unknown, record: BacktestRunRow) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => b.onViewRun(record.id)}>{t('strategy.templates.backtestRuns.actions.view')}</Button>
          <Popconfirm title={t('strategy.templates.backtestRuns.deleteConfirm')} onConfirm={() => b.onDeleteRun(record.id)}>
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
              <Popconfirm title={t('strategy.templates.backtestRuns.batchDeleteConfirm', { count: selectedRowKeys.length })}
                onConfirm={b.onBatchDelete}>
                <Button danger loading={b.deleting}>
                  {t('strategy.backtestHistory.batchDelete', '删除 {{count}} 条', { count: selectedRowKeys.length })}
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
          locale={{ emptyText: t('strategy.backtestHistory.empty') }}
        />
      </StatusResult>
    </div>
  );
}
