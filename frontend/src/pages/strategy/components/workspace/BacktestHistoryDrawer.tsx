import React, { useEffect, useMemo, useState } from 'react';
import { Button, Modal, Popconfirm, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType, TableRowSelection } from 'antd/es/table';
import dayjs from 'dayjs';
import { DeleteOutlined, ReloadOutlined, ArrowLeftOutlined } from '@ant-design/icons';
import { formatDateTime } from '@/utils/date';
import { useTranslation } from 'react-i18next';
import {
  BACKTEST_RUNS_ACTIONS_VIEW_KEY, BACKTEST_RUNS_DELETE_CONFIRM_KEY, BACKTEST_RUNS_EMPTY_KEY,
  BACKTEST_RUNS_STATUS_CANCELED_KEY, BACKTEST_RUNS_STATUS_CANCELING_KEY, BACKTEST_RUNS_STATUS_COMPLETED_KEY,
  BACKTEST_RUNS_STATUS_FAILED_KEY, BACKTEST_RUNS_STATUS_QUEUED_KEY, BACKTEST_RUNS_STATUS_RUNNING_KEY,
  BACKTEST_RUNS_TABLE_ACTIONS_KEY, BACKTEST_RUNS_TABLE_CREATED_AT_KEY, BACKTEST_RUNS_TABLE_STATUS_KEY,
  BACKTEST_RUNS_TABLE_SYMBOL_KEY, BACKTEST_RUNS_TABLE_TIMEFRAME_KEY, BACKTEST_RUNS_TITLE_KEY,
} from '@/gen/ant/v1/i18n/strategy_templates_keys';
import {
  ACTIONS_CANCEL_KEY, STATUS_CANCELED_KEY, STATUS_CANCELING_KEY, STATUS_COMPLETED_KEY,
  STATUS_ENDED_KEY, STATUS_FAILED_KEY, STATUS_QUEUED_KEY, STATUS_RUNNING_KEY, TITLE_KEY,
  TRADES_CLOSE_PRICE_KEY, TRADES_CLOSE_TIME_KEY, TRADES_COMMISSION_KEY, TRADES_OPEN_PRICE_KEY,
  TRADES_OPEN_TIME_KEY, TRADES_PNL_KEY, TRADES_REASON_KEY, TRADES_SIDE_BUY_KEY, TRADES_SIDE_KEY,
  TRADES_SIDE_SELL_KEY, TRADES_SUMMARY_KEY, TRADES_TICKET_KEY, TRADES_VOLUME_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_run_keys';
import { useWatchBacktestRun } from '@/hooks/useWatchBacktestRun';
import { backtestRunsApi, type BacktestTrade, type BacktestTradeSummary } from '@/client/backtestRuns';
import { isSucceededRun } from '@/pages/strategy/StrategyTemplatePage.utils';
import BacktestRunDrawerContent from '@/components/strategy/BacktestRunDrawerContent';
import {
  COMMON_CLOSE_KEY, COMMON_REFRESH_KEY, COMMON_YES_KEY, COMMON_NO_KEY, COMMON_TOTAL_ITEMS_KEY,
} from '@/gen/ant/v1/i18n/base_keys';
import {
  BACKTEST_RUNS_BATCH_DELETE_CONFIRM_KEY, BACKTEST_RUNS_BATCH_DELETE_KEY,
} from '@/gen/ant/v1/i18n/strategy_templates_keys';

const { Text } = Typography;

interface Props {
  open: boolean;
  runs: unknown[];
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
  // Run detail view
  runId: string;
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

const fmt = (n: number | null | undefined, digits = 4): string =>
  n === null || n === undefined || Number.isNaN(n) ? '-' : Number(n).toFixed(digits);

const fmtTs = (ms: number | undefined): string =>
  !ms || ms <= 0 ? '-' : dayjs(ms).format('YYYY-MM-DD HH:mm:ss');

const BacktestHistoryDrawer: React.FC<Props> = ({
  open, runs, loading, page, pageSize, total,
  selectedRowKeys, deleting,
  onPageChange, onSelectionChange,
  onViewRun, onDeleteRun, onBatchDelete, onRefresh, onClose,
  runId,
}) => {
  const { t } = useTranslation();
  const [trades, setTrades] = useState<BacktestTrade[]>([]);
  const [tradeSummary, setTradeSummary] = useState<BacktestTradeSummary | null>(null);
  const [tradesLoading, setTradesLoading] = useState(false);
  const [tradesError, setTradesError] = useState<string | null>(null);

  const watched = useWatchBacktestRun(runId || null);
  const isCompleted = isSucceededRun(watched.run);
  const showDetail = !!runId;

  useEffect(() => {
    if (!open || !runId || !isCompleted) return;
    let cancelled = false;
    setTradesLoading(true);
    setTradesError(null);
    backtestRunsApi
      .getTrades(runId)
      .then((result) => {
        if (cancelled) return;
        setTrades(result.trades);
        setTradeSummary(result.summary);
      })
      .catch((e: unknown) => { if (!cancelled) setTradesError(e instanceof Error ? e.message : String(e)); })
      .finally(() => { if (!cancelled) setTradesLoading(false); });
    return () => { cancelled = true; };
  }, [open, runId, isCompleted]);

  const tradesActive = open && !!runId && isCompleted;
  const visibleTrades = tradesActive ? trades : [];
  const visibleTradeSummary = tradesActive ? tradeSummary : null;
  const visibleTradesError = tradesActive ? tradesError : null;

  const runStatusText = (() => {
    switch (watched.run?.status) {
      case 1: return t(STATUS_QUEUED_KEY);
      case 2: return t(STATUS_RUNNING_KEY);
      case 3: return t(STATUS_COMPLETED_KEY);
      case 4: return t(STATUS_FAILED_KEY);
      case 5: return t(STATUS_CANCELING_KEY);
      case 6: return t(STATUS_CANCELED_KEY);
      default: return watched.run?.status != null ? String(watched.run.status) : '-';
    }
  })();

  const summary = useMemo(() => {
    if (!visibleTradeSummary || !visibleTradeSummary.count) return null;
    return t(TRADES_SUMMARY_KEY, {
      count: visibleTradeSummary.count,
      wins: visibleTradeSummary.wins,
      losses: visibleTradeSummary.losses,
      pnl: (visibleTradeSummary.netPnl ?? 0).toFixed(2),
    });
  }, [visibleTradeSummary, t]);

  const tradeColumns = useMemo<ColumnsType<BacktestTrade>>(() => [
    { title: t(TRADES_TICKET_KEY), dataIndex: 'ticket', key: 'ticket', width: 70 },
    { title: t(TRADES_SIDE_KEY), dataIndex: 'side', key: 'side', width: 70,
      render: (v: string) => { const isBuy = String(v).toLowerCase() === 'buy';
        return <Tag color={isBuy ? 'green' : 'red'}>{isBuy ? t(TRADES_SIDE_BUY_KEY) : t(TRADES_SIDE_SELL_KEY)}</Tag>; } },
    { title: t(TRADES_VOLUME_KEY), dataIndex: 'volume', key: 'volume', width: 80, render: (v: number) => fmt(v, 2) },
    { title: t(TRADES_OPEN_TIME_KEY), dataIndex: 'open_ts', key: 'open_ts', render: (v: number) => fmtTs(v) },
    { title: t(TRADES_OPEN_PRICE_KEY), dataIndex: 'open_price', key: 'open_price', width: 100, render: (v: number) => fmt(v, 5) },
    { title: t(TRADES_CLOSE_TIME_KEY), dataIndex: 'close_ts', key: 'close_ts', render: (v: number) => fmtTs(v) },
    { title: t(TRADES_CLOSE_PRICE_KEY), dataIndex: 'close_price', key: 'close_price', width: 100, render: (v: number) => fmt(v, 5) },
    { title: t(TRADES_PNL_KEY), dataIndex: 'pnl', key: 'pnl', width: 100, align: 'right',
      render: (v: number) => <Typography.Text type={v > 0 ? 'success' : v < 0 ? 'danger' : undefined}>{fmt(v, 2)}</Typography.Text>,
      sorter: (a, b) => a.pnl - b.pnl },
    { title: t(TRADES_COMMISSION_KEY), dataIndex: 'commission', key: 'commission', width: 100, render: (v: number) => fmt(v, 2) },
    { title: t(TRADES_REASON_KEY), dataIndex: 'reason', key: 'reason', width: 110,
      render: (v: string) => t(`strategy.backtestRun.trades.reasons.${v}`, { defaultValue: v || '-' }) },
  ], [t]);

  const rowSelection: TableRowSelection<unknown> = {
    selectedRowKeys,
    onChange: (keys) => onSelectionChange(keys),
    selections: [Table.SELECTION_ALL, Table.SELECTION_INVERT, Table.SELECTION_NONE],
  };

  const historyColumns: ColumnsType<unknown> = useMemo(() => [
    { title: t(BACKTEST_RUNS_TABLE_STATUS_KEY, 'Status'), dataIndex: 'status', key: 'status', width: 100,
      render: (s: unknown) => <Tag color={statusColor(s)}>{statusText(s, t)}</Tag> },
    { title: t(BACKTEST_RUNS_TABLE_SYMBOL_KEY, 'Symbol'), dataIndex: 'symbol', key: 'symbol', width: 110,
      render: (v: string) => <Text>{v || '-'}</Text> },
    { title: t(BACKTEST_RUNS_TABLE_TIMEFRAME_KEY, 'Timeframe'), dataIndex: 'timeframe', key: 'timeframe', width: 90,
      render: (v: string) => <Text>{v || '-'}</Text> },
    { title: t(BACKTEST_RUNS_TABLE_CREATED_AT_KEY, 'Created'), dataIndex: 'createdAt', key: 'createdAt', width: 170,
      render: (v: string | undefined) => <Text>{v ? formatDateTime(v) : '-'}</Text> },
    { title: t(BACKTEST_RUNS_TABLE_ACTIONS_KEY, 'Actions'), key: 'actions', width: 140,
      render: (_: unknown, record: { id: string }) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => onViewRun(record.id)}>
            {t(BACKTEST_RUNS_ACTIONS_VIEW_KEY, 'View')}
          </Button>
          <Popconfirm title={t(BACKTEST_RUNS_DELETE_CONFIRM_KEY, 'Delete this backtest run?')}
            onConfirm={() => onDeleteRun(record.id)} okText={t(COMMON_YES_KEY)} cancelText={t(COMMON_NO_KEY)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ], [t, onViewRun, onDeleteRun]);

  return (
    <Modal
      title={showDetail ? (
        <Space>
          <Button type="text" size="small" icon={<ArrowLeftOutlined />} onClick={onClose} />
          {t(TITLE_KEY)}
        </Space>
      ) : t(BACKTEST_RUNS_TITLE_KEY, 'Backtest History')}
      open={open}
      onCancel={onClose}
      destroyOnClose
      width={1100}
      styles={{ body: { maxHeight: 'calc(100vh - 200px)', overflowY: 'auto' } }}
      footer={
        showDetail ? (
          <Space>
            {watched.isTerminal ? (
              <Button disabled>{runStatusText || t(STATUS_ENDED_KEY)}</Button>
            ) : (
              <Button onClick={onClose} disabled={!runId || watched.isTerminal}>
                {t(ACTIONS_CANCEL_KEY)}
              </Button>
            )}
            <Button type="primary" onClick={onClose}>{t(COMMON_CLOSE_KEY)}</Button>
          </Space>
        ) : null
      }
    >
      {showDetail ? (
        <BacktestRunDrawerContent
          watched={watched}
          statusText={runStatusText}
          trades={visibleTrades}
          summary={summary}
          tradesLoading={tradesLoading}
          tradesError={visibleTradesError}
          columns={tradeColumns}
        />
      ) : (
        <>
          <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between' }}>
            <Space>
              <Button icon={<ReloadOutlined />} onClick={onRefresh} loading={loading}>
                {t(COMMON_REFRESH_KEY)}
              </Button>
              {selectedRowKeys.length > 0 && (
                <Popconfirm
                  title={t(BACKTEST_RUNS_BATCH_DELETE_CONFIRM_KEY, { count: selectedRowKeys.length })}
                  onConfirm={onBatchDelete} okText={t(COMMON_YES_KEY)} cancelText={t(COMMON_NO_KEY)}>
                  <Button danger loading={deleting}>
                    {t(BACKTEST_RUNS_BATCH_DELETE_KEY, { count: selectedRowKeys.length })}
                  </Button>
                </Popconfirm>
              )}
            </Space>
          </div>
          <Table
            rowKey="id"
            columns={historyColumns}
            dataSource={runs}
            loading={loading}
            rowSelection={rowSelection}
            pagination={{
              current: page, pageSize, total,
              showSizeChanger: true, showQuickJumper: true,
              showTotal: (total: number) => t(COMMON_TOTAL_ITEMS_KEY, { total }),
              pageSizeOptions: ['10', '20', '50'],
              onChange: onPageChange,
            }}
            size="small"
            locale={{ emptyText: t(BACKTEST_RUNS_EMPTY_KEY, 'No backtest runs found') }}
            scroll={{ x: 610 }}
          />
        </>
      )}
    </Modal>
  );
};

export default BacktestHistoryDrawer;
