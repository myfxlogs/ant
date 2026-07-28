import type { ColumnsType } from 'antd/es/table';
import type { TFunction } from 'i18next';
import dayjs from 'dayjs';
import { Tag, Typography, Space, Button, Popconfirm } from 'antd';
import { DeleteOutlined } from '@ant-design/icons';
import { formatDateTime } from '@/utils/date';
import type { BacktestTrade } from '@/client/backtestRuns';
import {
  BACKTEST_RUNS_ACTIONS_VIEW_KEY, BACKTEST_RUNS_DELETE_CONFIRM_KEY,
  BACKTEST_RUNS_STATUS_QUEUED_KEY, BACKTEST_RUNS_STATUS_RUNNING_KEY,
  BACKTEST_RUNS_STATUS_COMPLETED_KEY, BACKTEST_RUNS_STATUS_FAILED_KEY,
  BACKTEST_RUNS_STATUS_CANCELING_KEY, BACKTEST_RUNS_STATUS_CANCELED_KEY,
  BACKTEST_RUNS_TABLE_ACTIONS_KEY, BACKTEST_RUNS_TABLE_CREATED_AT_KEY,
  BACKTEST_RUNS_TABLE_STATUS_KEY, BACKTEST_RUNS_TABLE_SYMBOL_KEY,
  BACKTEST_RUNS_TABLE_TIMEFRAME_KEY,
} from '@/gen/ant/v1/i18n/strategy_templates_keys';
import {
  TRADES_CLOSE_PRICE_KEY, TRADES_CLOSE_TIME_KEY, TRADES_COMMISSION_KEY,
  TRADES_OPEN_PRICE_KEY, TRADES_OPEN_TIME_KEY, TRADES_PNL_KEY, TRADES_REASON_KEY,
  TRADES_SIDE_BUY_KEY, TRADES_SIDE_KEY, TRADES_SIDE_SELL_KEY, TRADES_TICKET_KEY,
  TRADES_VOLUME_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_run_keys';
import { COMMON_YES_KEY, COMMON_NO_KEY } from '@/gen/ant/v1/i18n/base_keys';

const { Text } = Typography;

export function statusText(s: unknown, t: TFunction): string {
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

export function statusColor(s: unknown): string {
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

export const fmt = (n: number | null | undefined, digits = 4): string =>
  n === null || n === undefined || Number.isNaN(n) ? '-' : Number(n).toFixed(digits);

export const fmtTs = (ms: number | undefined): string =>
  !ms || ms <= 0 ? '-' : dayjs(ms).format('YYYY-MM-DD HH:mm:ss');

export function buildTradeColumns(t: TFunction): ColumnsType<BacktestTrade> {
  return [
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
  ];
}

export function buildHistoryColumns(
  t: TFunction,
  onViewRun: (runId: string) => void,
  onDeleteRun: (runId: string) => void,
): ColumnsType<unknown> {
  return [
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
  ];
}
