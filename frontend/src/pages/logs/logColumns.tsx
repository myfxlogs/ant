import { Tag } from 'antd';
import type { JsonObject } from '@bufbuild/protobuf';
import type { ConnectionLog } from '@/gen/ant/v1/log_connection_pb';
import type { OperationLog } from '@/gen/ant/v1/log_operation_pb';
import type { TFunction } from 'i18next'
import { ACTION_KEY, COST_KEY, DETAILS_KEY, DURATION_KEY, ERROR_KEY, EVENT_TYPE_KEY, EXECUTION_PRICE_KEY, FAILED_KEY, IP_KEY, LOGIN_ID_KEY, MESSAGE_KEY, MODULE_KEY, ORDER_TABLE_CLOSE_KEY, ORDER_TABLE_LOTS_KEY, ORDER_TABLE_OPEN_KEY, ORDER_TABLE_TICKET_KEY, ORDER_TABLE_TYPE_KEY, PERIOD_KEY, PRODUCT_KEY, PROFIT_KEY, REQUEST_ID_KEY, RESULT_KEY, RISK_CODE_KEY, SERVER_KEY, SIGNAL_KEY, SIGNAL_PRICE_KEY, STATUS_KEY, SUCCESS_KEY, TIME_KEY, TRIGGER_SOURCE_KEY } from '@/gen/ant/v1/i18n/logs_keys';
;

interface AccountLike { id: string; brokerServer?: string; brokerHost?: string; brokerCompany?: string; [key: string]: unknown; }
interface OperationDetails { result?: string; risk_code?: string; request_id?: string; trigger_source?: string; [key: string]: unknown; }

export function parseOperationDetails(details?: JsonObject): OperationDetails {
  if (!details || typeof details !== 'object') return {};
  return details as OperationDetails;
}

export function getStatusTag(status: string | undefined, t: TFunction) {
  const n = String(status || '').toLowerCase();
  const colors: Record<string, string> = { success: 'green', failed: 'red', completed: 'green', running: 'blue', pending: 'orange', skipped: 'default' };
  return <Tag color={colors[n] || 'default'}>{n === 'success' ? t(SUCCESS_KEY) : n === 'failed' ? t(FAILED_KEY) : n ? n.toUpperCase() : '-'}</Tag>;
}

export function getEventTypeTag(type: string | undefined) {
  const n = String(type || '').toLowerCase();
  const colors: Record<string, string> = { connect: 'blue', disconnect: 'orange', reconnect: 'cyan', error: 'red', heartbeat: 'green' };
  return <Tag color={colors[n] || 'default'}>{n ? n.toUpperCase() : '-'}</Tag>;
}

export function getSignalTypeTag(type: string | undefined, t?: (k: string, o?: Record<string, unknown>) => string) {
  if (!type) return '-';
  const colors: Record<string, string> = { buy: 'green', sell: 'red', close: 'orange', hold: 'default', modify: 'blue' };
  const tl = t ?? ((k: string) => k);
  const key = `logs.signalType.${type.toLowerCase()}`;
  const label = tl(key, { defaultValue: type.toUpperCase() });
  return <Tag color={colors[type] || 'default'}>{label}</Tag>;
}

interface ColumnOpts {
  t: TFunction;
  formatTime: (v: unknown) => string;
  accountById: Map<string, AccountLike>;
}

export function buildConnectionColumns({ t, formatTime, accountById }: ColumnOpts) {
  return [
    { title: t(TIME_KEY), dataIndex: 'createdAt', key: 'createdAt', width: 180, render: (v: unknown) => formatTime(v) },
    { title: t(EVENT_TYPE_KEY), dataIndex: 'eventType', key: 'eventType', width: 120, render: getEventTypeTag },
    { title: t(STATUS_KEY), dataIndex: 'status', key: 'status', width: 100, render: (v: string) => getStatusTag(v, t) },
    { title: t(SERVER_KEY), key: 'server', width: 200, render: (_: unknown, r: ConnectionLog) => {
      const a = accountById.get(String(r.accountId || ''));
      const name = String(a?.brokerServer || a?.brokerHost || a?.brokerCompany || '').trim();
      if (name) return name;
      const host = String(r.serverHost || '').trim(); const port = String(r.serverPort ?? '').trim();
      return host && port ? `${host}:${port}` : host || '-';
    }},
    { title: t(LOGIN_ID_KEY), dataIndex: 'loginId', key: 'loginId', width: 100, render: (v: bigint | number | undefined) => (v !== undefined && v !== null ? String(v) : '-') },
    { title: t(MESSAGE_KEY), dataIndex: 'message', key: 'message', ellipsis: true },
    { title: t(DURATION_KEY), dataIndex: 'connectionDurationSeconds', key: 'duration', width: 100, render: (v: bigint | number | undefined) => (v ? `${String(v)}s` : '-') },
  ];
}

export function buildExecutionColumns({ t, formatTime }: ColumnOpts) {
  return [
    { title: t(TIME_KEY), dataIndex: 'createdAt', key: 'createdAt', width: 180, render: (v: unknown) => formatTime(v) },
    { title: t(PRODUCT_KEY), dataIndex: 'symbol', key: 'symbol', width: 100 },
    { title: t(PERIOD_KEY), dataIndex: 'timeframe', key: 'timeframe', width: 80 },
    { title: t(STATUS_KEY), dataIndex: 'status', key: 'status', width: 100, render: (v: string) => getStatusTag(v, t) },
    { title: t(SIGNAL_KEY), dataIndex: 'signalType', key: 'signalType', width: 80, render: (v: string) => getSignalTypeTag(v, t) },
    { title: t(SIGNAL_PRICE_KEY), dataIndex: 'signalPrice', key: 'signalPrice', width: 100, render: (v: number) => v?.toFixed(5) || '-' },
    { title: t(EXECUTION_PRICE_KEY), dataIndex: 'executedPrice', key: 'executedPrice', width: 100, render: (v: number) => v?.toFixed(5) || '-' },
    { title: t(PROFIT_KEY), dataIndex: 'profit', key: 'profit', width: 100, render: (v: number) => v ? <span style={{ color: v >= 0 ? 'green' : 'red' }}>{v.toFixed(2)}</span> : '-' },
    { title: t(COST_KEY), dataIndex: 'executionTimeMs', key: 'executionTimeMs', width: 80, render: (v: number) => v ? `${v}ms` : '-' },
    { title: t(ERROR_KEY), dataIndex: 'errorMessage', key: 'errorMessage', ellipsis: true },
  ];
}

export function buildOrderColumns({ t, formatTime }: ColumnOpts) {
  return [
    { title: t(TIME_KEY), dataIndex: 'openTime', key: 'openTime', width: 180, render: (v: unknown) => formatTime(v) },
    { title: t(ORDER_TABLE_TICKET_KEY), dataIndex: 'ticket', key: 'ticket', width: 100 },
    { title: t(PRODUCT_KEY), dataIndex: 'symbol', key: 'symbol', width: 100 },
    { title: t(ORDER_TABLE_TYPE_KEY), dataIndex: 'orderType', key: 'orderType', width: 100 },
    { title: t(ORDER_TABLE_LOTS_KEY), dataIndex: 'lots', key: 'lots', width: 80 },
    { title: t(ORDER_TABLE_OPEN_KEY), dataIndex: 'openPrice', key: 'openPrice', width: 100, render: (v: number) => v?.toFixed(5) || '-' },
    { title: t(ORDER_TABLE_CLOSE_KEY), dataIndex: 'closePrice', key: 'closePrice', width: 100, render: (v: number) => v?.toFixed(5) || '-' },
    { title: t(PROFIT_KEY), dataIndex: 'profit', key: 'profit', width: 100, render: (v: number) => v ? <span style={{ color: v >= 0 ? 'green' : 'red' }}>{v.toFixed(2)}</span> : '-' },
  ];
}

export function buildOperationColumns({ t, formatTime }: ColumnOpts) {
  return [
    { title: t(TIME_KEY), dataIndex: 'createdAt', key: 'createdAt', width: 180, render: (v: unknown) => formatTime(v) },
    { title: t(MODULE_KEY), dataIndex: 'module', key: 'module', width: 120 },
    { title: t(ACTION_KEY), dataIndex: 'action', key: 'action', width: 150 },
    { title: t(RESULT_KEY), key: 'riskResult', width: 100, render: (_: unknown, r: OperationLog) => {
      const d = parseOperationDetails(r?.details); const val = String(d?.result || '').toLowerCase();
      if (!val) return '-';
      return <Tag color={val === 'pass' ? 'green' : val === 'reject' ? 'red' : 'default'}>{val.toUpperCase()}</Tag>;
    }},
    { title: t(RISK_CODE_KEY), key: 'riskCode', width: 220, render: (_: unknown, r: OperationLog) => { const d = parseOperationDetails(r?.details); return d?.risk_code || '-'; } },
    { title: t(REQUEST_ID_KEY), key: 'requestId', width: 220, render: (_: unknown, r: OperationLog) => { const d = parseOperationDetails(r?.details); return d?.request_id || '-'; } },
    { title: t(TRIGGER_SOURCE_KEY), key: 'triggerSource', width: 120, render: (_: unknown, r: OperationLog) => { const d = parseOperationDetails(r?.details); return d?.trigger_source || '-'; } },
    { title: t(DETAILS_KEY), key: 'details', ellipsis: true, render: (_: unknown, r: OperationLog) => r.details ? JSON.stringify(r.details) : '-' },
    { title: t(IP_KEY), dataIndex: 'ip', key: 'ip', width: 120 },
  ];
}
