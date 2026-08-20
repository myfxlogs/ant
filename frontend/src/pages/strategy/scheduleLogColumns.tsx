import { Typography, Tag } from 'antd';
import { CopyOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { TFunction } from 'i18next';
import type { ScheduleRunLog } from '@/gen/ant/v1/log_schedule_pb';
import type { OrderHistoryRecord } from '@/gen/ant/v1/log_order_pb';
import { ACTION_CLEANUP_KEY, ACTION_REGISTER_KEY, ACTION_RESTART_KEY, ACTION_START_KEY, ACTION_STOP_KEY, EXEC_STATUS_FAILED_KEY, EXEC_STATUS_PENDING_KEY, EXEC_STATUS_RUNNING_KEY, EXEC_STATUS_STOPPED_KEY, EXEC_STATUS_SUCCESS_KEY, EXEC_TABLE_ACTION_KEY, EXEC_TABLE_DURATION_MS_KEY, EXEC_TABLE_ERROR_KEY, EXEC_TABLE_EXECUTE_KEY, EXEC_TABLE_STATUS_KEY, EXEC_TABLE_TIME_KEY, ORDERS_TABLE_CLOSE_PRICE_KEY, ORDERS_TABLE_LOTS_KEY, ORDERS_TABLE_OPEN_PRICE_KEY, ORDERS_TABLE_PROFIT_KEY, ORDERS_TABLE_SIDE_KEY, ORDERS_TABLE_SYMBOL_KEY, ORDERS_TABLE_TICKET_KEY, ORDERS_TABLE_TIME_KEY, ORDER_SIDE_BUY_KEY, ORDER_SIDE_CLOSE_KEY, ORDER_SIDE_SELL_KEY, STATUS_FAILED_KEY, STATUS_SUCCESS_KEY } from '@/gen/ant/v1/i18n/strategy_schedule_logs_keys';
;

const { Text } = Typography;

interface ColOpts { t: TFunction; formatTime: (v: unknown) => string; }

export function formatLogTime(v: unknown) {
  if (!v) return '-';
  if (typeof v === 'string') {
    const d = new Date(v);
    if (!isNaN(d.getTime())) return d.toLocaleString();
    return v;
  }
  if (v instanceof Date && !isNaN(v.getTime())) return v.toLocaleString();
  if (typeof v === 'object' && v !== null) {
    const vo = v as Record<string, unknown>;
    const secNum = typeof vo['seconds'] === 'bigint' ? Number(vo['seconds']) : typeof vo['seconds'] === 'number' ? vo['seconds'] : undefined;
    if (typeof secNum === 'number' && isFinite(secNum)) {
      return new Date(secNum * 1000).toLocaleString();
    }
  }
  return String(v);
}

export function renderExecStatus(v: unknown, t: TFunction) {
  const s = String(v || '').toLowerCase();
  if (s === 'success' || s === 'completed' || s === 'succeeded') return <Tag color="green">{t(EXEC_STATUS_SUCCESS_KEY)}</Tag>;
  if (s === 'failed' || s === 'error') return <Tag color="red">{t(EXEC_STATUS_FAILED_KEY)}</Tag>;
  if (s === 'running') return <Tag color="blue">{t(EXEC_STATUS_RUNNING_KEY)}</Tag>;
  if (s === 'pending' || s === 'queued') return <Tag color="orange">{t(EXEC_STATUS_PENDING_KEY)}</Tag>;
  if (s === 'stopped') return <Tag>{t(EXEC_STATUS_STOPPED_KEY)}</Tag>;
  return <Text>{s || '-'}</Text>;
}

export function renderMs(v: unknown) {
  const n = typeof v === 'number' ? v : Number(v);
  if (!isFinite(n) || n <= 0) return <Text>-</Text>;
  if (n >= 1000) return <Text>{(n / 1000).toFixed(1)}s</Text>;
  return <Text>{n}ms</Text>;
}

export function renderOperationAction(v: unknown, t: TFunction) {
  const s = String(v || '').toLowerCase();
  const map: Record<string, string> = { start: t(ACTION_START_KEY), stop: t(ACTION_STOP_KEY), restart: t(ACTION_RESTART_KEY), register: t(ACTION_REGISTER_KEY), cleanup: t(ACTION_CLEANUP_KEY) };
  return <Text>{map[s] || s}</Text>;
}

export function renderOperationStatus(v: unknown, t: TFunction) {
  const s = String(v || '').toLowerCase();
  if (s === 'success') return <Tag color="green">{t(STATUS_SUCCESS_KEY)}</Tag>;
  if (s === 'failed') return <Tag color="red">{t(STATUS_FAILED_KEY)}</Tag>;
  return <Text>{s || '-'}</Text>;
}

export function renderOrderTypeTag(value: string, t: TFunction) {
  if (!value) return <Text>-</Text>;
  const s = value.toLowerCase();
  if (s === 'buy' || s === 'market_buy') return <Tag color="green">{t(ORDER_SIDE_BUY_KEY)}</Tag>;
  if (s === 'sell' || s === 'market_sell') return <Tag color="red">{t(ORDER_SIDE_SELL_KEY)}</Tag>;
  return <Tag>{value.toUpperCase()}</Tag>;
}

export function buildExecColumns({ t, formatTime }: ColOpts): ColumnsType<ScheduleRunLog> {
  return [
    { title: t(EXEC_TABLE_TIME_KEY), dataIndex: 'createdAt', key: 'createdAt', width: 180, render: (_v: unknown, row: ScheduleRunLog) => <Text>{formatTime(row?.createdAt)}</Text> },
    { title: t(EXEC_TABLE_ACTION_KEY), key: 'action', width: 160, render: (_: unknown, row: ScheduleRunLog) => {
      if (String(row?.kind || '').toLowerCase() === 'operation') return renderOperationAction(row?.action, t);
      const st = String(row?.signalType || row?.action || '').toLowerCase();
      if (st === 'close') return <Text>{t(ORDER_SIDE_CLOSE_KEY)}</Text>;
      if (st === 'register') return <Text>{t(ACTION_REGISTER_KEY)}</Text>;
      if (st === 'cleanup') return <Text>{t(ACTION_CLEANUP_KEY)}</Text>;
      return <Text>{String(row?.signalType || row?.action || t(EXEC_TABLE_EXECUTE_KEY))}</Text>;
    }},
    { title: t(EXEC_TABLE_STATUS_KEY), dataIndex: 'status', key: 'status', width: 120, render: (_: unknown, row: ScheduleRunLog) => {
      if (String(row?.kind || '').toLowerCase() === 'operation') return renderOperationStatus(row?.status, t);
      return renderExecStatus(row?.status, t);
    }},
    { title: t(EXEC_TABLE_DURATION_MS_KEY), key: 'duration', width: 110, render: (_: unknown, row: ScheduleRunLog) => renderMs(row?.durationMs) },
    { title: t(EXEC_TABLE_ERROR_KEY), dataIndex: 'errorMessage', key: 'errorMessage', render: (v: unknown) => {
      const s = String(v || '').trim();
      if (!s) return <Text type="secondary">{t('common.none')}</Text>;
      return <Text type="danger" copyable={{
        text: s,
        icon: <CopyOutlined />,
        tooltips: [t('common.copy', { defaultValue: '复制' }), t('common.copied', { defaultValue: '已复制' })],
      }} ellipsis={{ tooltip: s }} style={{ maxWidth: 360, display: 'inline-block' }}>{s}</Text>;
    }},
  ];
}

export function buildOrderColumns({ t, formatTime }: ColOpts): ColumnsType<OrderHistoryRecord> {
  return [
    { title: t(ORDERS_TABLE_TIME_KEY), key: 'time', width: 180, render: (_: unknown, row: OrderHistoryRecord) => <Text>{formatTime(row?.closeTime || row?.openTime)}</Text> },
    { title: t(ORDERS_TABLE_SIDE_KEY), dataIndex: 'orderType', key: 'orderType', width: 100, render: (v: unknown) => renderOrderTypeTag(String(v || ''), t) },
    { title: t(ORDERS_TABLE_SYMBOL_KEY), dataIndex: 'symbol', key: 'symbol', width: 120, render: (v: unknown) => <Text>{String(v || '-')}</Text> },
    { title: t(ORDERS_TABLE_LOTS_KEY), dataIndex: 'lots', key: 'lots', width: 90, render: (v: unknown) => <Text>{typeof v === 'number' ? v : '-'}</Text> },
    { title: t(ORDERS_TABLE_OPEN_PRICE_KEY), dataIndex: 'openPrice', key: 'openPrice', width: 120, render: (v: unknown) => <Text>{typeof v === 'number' ? v : '-'}</Text> },
    { title: t(ORDERS_TABLE_CLOSE_PRICE_KEY), dataIndex: 'closePrice', key: 'closePrice', width: 120, render: (v: unknown) => <Text>{typeof v === 'number' ? v : '-'}</Text> },
    { title: t(ORDERS_TABLE_PROFIT_KEY), dataIndex: 'profit', key: 'profit', width: 120, render: (v: unknown) => {
      const n = typeof v === 'number' ? v : Number(v);
      if (!isFinite(n)) return <Text>-</Text>;
      if (n > 0) return <Text style={{ color: '#00A651' }}>{n.toFixed(2)}</Text>;
      if (n < 0) return <Text type="danger">{n.toFixed(2)}</Text>;
      return <Text>{n.toFixed(2)}</Text>;
    }},
    { title: t(ORDERS_TABLE_TICKET_KEY), dataIndex: 'ticket', key: 'ticket', width: 110, render: (v: unknown) => <Text>{typeof v === 'number' ? v : '-'}</Text> },
  ];
}
