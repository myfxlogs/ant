import { Tag } from 'antd';
import type { ConnectionLog } from '@/gen/ant/v1/log_connection_pb';
import type { ExecutionLog } from '@/gen/ant/v1/log_execution_pb';
import type { OrderHistoryRecord } from '@/gen/ant/v1/log_order_pb';
import type { OperationLog } from '@/gen/ant/v1/log_operation_pb';
import type { TFunction } from 'react-i18next';

interface AccountLike { id: string; brokerServer?: string; brokerHost?: string; brokerCompany?: string; [key: string]: unknown; }
interface OperationDetails { result?: string; risk_code?: string; request_id?: string; trigger_source?: string; [key: string]: unknown; }

export function parseOperationDetails(raw?: string): OperationDetails {
  if (!raw) return {};
  try { const p = JSON.parse(raw); return p && typeof p === 'object' ? p : {}; } catch { return {}; }
}

export function getStatusTag(status: string | undefined, t: TFunction) {
  const n = String(status || '').toLowerCase();
  const colors: Record<string, string> = { success: 'green', failed: 'red', completed: 'green', running: 'blue', pending: 'orange', skipped: 'default' };
  return <Tag color={colors[n] || 'default'}>{n === 'success' ? t('logs.success') : n === 'failed' ? t('logs.failed') : n ? n.toUpperCase() : '-'}</Tag>;
}

export function getEventTypeTag(type: string | undefined) {
  const n = String(type || '').toLowerCase();
  const colors: Record<string, string> = { connect: 'blue', disconnect: 'orange', reconnect: 'cyan', error: 'red', heartbeat: 'green' };
  return <Tag color={colors[n] || 'default'}>{n ? n.toUpperCase() : '-'}</Tag>;
}

export function getSignalTypeTag(type: string | undefined) {
  if (!type) return '-';
  const colors: Record<string, string> = { buy: 'green', sell: 'red', close: 'orange', hold: 'default', modify: 'blue' };
  return <Tag color={colors[type] || 'default'}>{type.toUpperCase()}</Tag>;
}

interface ColumnOpts {
  t: TFunction;
  formatTime: (v: unknown) => string;
  accountById: Map<string, AccountLike>;
}

export function buildConnectionColumns({ t, formatTime, accountById }: ColumnOpts) {
  return [
    { title: t('logs.time'), dataIndex: 'createdAt', key: 'createdAt', width: 180, render: (v: unknown) => formatTime(v) },
    { title: t('logs.eventType'), dataIndex: 'eventType', key: 'eventType', width: 120, render: getEventTypeTag },
    { title: t('logs.status'), dataIndex: 'status', key: 'status', width: 100, render: (v: string) => getStatusTag(v, t) },
    { title: t('logs.server'), key: 'server', width: 200, render: (_: unknown, r: ConnectionLog) => {
      const a = accountById.get(String(r.accountId || ''));
      const name = String(a?.brokerServer || a?.brokerHost || a?.brokerCompany || '').trim();
      if (name) return name;
      const host = String(r.serverHost || '').trim(); const port = String(r.serverPort ?? '').trim();
      return host && port ? `${host}:${port}` : host || '-';
    }},
    { title: t('logs.loginId'), dataIndex: 'loginId', key: 'loginId', width: 100, render: (v: bigint | number | undefined) => (v !== undefined && v !== null ? String(v) : '-') },
    { title: t('logs.message'), dataIndex: 'message', key: 'message', ellipsis: true },
    { title: t('logs.duration'), dataIndex: 'connectionDurationSeconds', key: 'duration', width: 100, render: (v: bigint | number | undefined) => (v ? `${String(v)}s` : '-') },
  ];
}

export function buildExecutionColumns({ t, formatTime }: ColumnOpts) {
  return [
    { title: t('logs.time'), dataIndex: 'createdAt', key: 'createdAt', width: 180, render: (v: unknown) => formatTime(v) },
    { title: t('logs.product'), dataIndex: 'symbol', key: 'symbol', width: 100 },
    { title: t('logs.period'), dataIndex: 'timeframe', key: 'timeframe', width: 80 },
    { title: t('logs.status'), dataIndex: 'status', key: 'status', width: 100, render: (v: string) => getStatusTag(v, t) },
    { title: t('logs.signal'), dataIndex: 'signalType', key: 'signalType', width: 80, render: getSignalTypeTag },
    { title: t('logs.signalPrice'), dataIndex: 'signalPrice', key: 'signalPrice', width: 100, render: (v: number) => v?.toFixed(5) || '-' },
    { title: t('logs.executionPrice'), dataIndex: 'executedPrice', key: 'executedPrice', width: 100, render: (v: number) => v?.toFixed(5) || '-' },
    { title: t('logs.profit'), dataIndex: 'profit', key: 'profit', width: 100, render: (v: number) => v ? <span style={{ color: v >= 0 ? 'green' : 'red' }}>{v.toFixed(2)}</span> : '-' },
    { title: t('logs.cost'), dataIndex: 'executionTimeMs', key: 'executionTimeMs', width: 80, render: (v: number) => v ? `${v}ms` : '-' },
    { title: t('logs.error'), dataIndex: 'errorMessage', key: 'errorMessage', ellipsis: true },
  ];
}

export function buildOrderColumns({ t, formatTime }: ColumnOpts) {
  return [
    { title: t('logs.time'), dataIndex: 'openTime', key: 'openTime', width: 180, render: (v: unknown) => formatTime(v) },
    { title: t('logs.orderTable.ticket'), dataIndex: 'ticket', key: 'ticket', width: 100 },
    { title: t('logs.product'), dataIndex: 'symbol', key: 'symbol', width: 100 },
    { title: t('logs.orderTable.type'), dataIndex: 'orderType', key: 'orderType', width: 100 },
    { title: t('logs.orderTable.lots'), dataIndex: 'lots', key: 'lots', width: 80 },
    { title: t('logs.orderTable.open'), dataIndex: 'openPrice', key: 'openPrice', width: 100, render: (v: number) => v?.toFixed(5) || '-' },
    { title: t('logs.orderTable.close'), dataIndex: 'closePrice', key: 'closePrice', width: 100, render: (v: number) => v?.toFixed(5) || '-' },
    { title: t('logs.profit'), dataIndex: 'profit', key: 'profit', width: 100, render: (v: number) => v ? <span style={{ color: v >= 0 ? 'green' : 'red' }}>{v.toFixed(2)}</span> : '-' },
  ];
}

export function buildOperationColumns({ t, formatTime }: ColumnOpts) {
  return [
    { title: t('logs.time'), dataIndex: 'createdAt', key: 'createdAt', width: 180, render: (v: unknown) => formatTime(v) },
    { title: t('logs.module'), dataIndex: 'module', key: 'module', width: 120 },
    { title: t('logs.action'), dataIndex: 'action', key: 'action', width: 150 },
    { title: t('logs.result'), key: 'riskResult', width: 100, render: (_: unknown, r: OperationLog) => {
      const d = parseOperationDetails(r?.details); const val = String(d?.result || '').toLowerCase();
      if (!val) return '-';
      return <Tag color={val === 'pass' ? 'green' : val === 'reject' ? 'red' : 'default'}>{val.toUpperCase()}</Tag>;
    }},
    { title: t('logs.riskCode'), key: 'riskCode', width: 220, render: (_: unknown, r: OperationLog) => { const d = parseOperationDetails(r?.details); return d?.risk_code || '-'; } },
    { title: t('logs.requestId'), key: 'requestId', width: 220, render: (_: unknown, r: OperationLog) => { const d = parseOperationDetails(r?.details); return d?.request_id || '-'; } },
    { title: t('logs.triggerSource'), key: 'triggerSource', width: 120, render: (_: unknown, r: OperationLog) => { const d = parseOperationDetails(r?.details); return d?.trigger_source || '-'; } },
    { title: t('logs.details'), dataIndex: 'details', key: 'details', ellipsis: true },
    { title: t('logs.ip'), dataIndex: 'ip', key: 'ip', width: 120 },
  ];
}
