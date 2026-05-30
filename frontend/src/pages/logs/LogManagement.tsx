import { useState, useMemo } from 'react';
import { Table, Card, Tabs } from 'antd';
import { accountApi } from '@/client/account';
import { logApi } from '@/client/log';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import type { ConnectionLog } from '@/gen/ant/v1/log_connection_pb';
import type { ExecutionLog } from '@/gen/ant/v1/log_execution_pb';
import type { OrderHistoryRecord } from '@/gen/ant/v1/log_order_pb';
import type { OperationLog } from '@/gen/ant/v1/log_operation_pb';
import { getDeviceLocale, getDeviceTimeZone } from '@/utils/date';
import { getErrorMessage } from '@/utils/error';
import { StatusResult } from '@/components/common/StatusResult';
import { useTranslation } from 'react-i18next';
import LogFilterForm from './LogFilterForm';
import {
  buildConnectionColumns, buildExecutionColumns, buildOrderColumns, buildOperationColumns,
} from './logColumns';

type LogEntry = ConnectionLog | ExecutionLog | OrderHistoryRecord | OperationLog;

interface AccountLike {
  id: string;
  brokerServer?: string;
  brokerHost?: string;
  brokerCompany?: string;
  [key: string]: unknown;
}

export default function LogManagement() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('connection');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [opRiskCode, setOpRiskCode] = useState('');
  const [opRequestId, setOpRequestId] = useState('');
  const [opTriggerSource, setOpTriggerSource] = useState('');
  const [opResult, setOpResult] = useState('');

  const { data: accounts = [] } = useRpcQuery(['logs', 'accounts'],
    async () => { const accs = await accountApi.list(); return (Array.isArray(accs) ? accs : []) as AccountLike[]; });

  const { data: queryResult, isLoading: loading, error: queryError, refetch } = useRpcQuery(
    ['logs', activeTab, page, pageSize, filters],
    async () => {
      switch (activeTab) {
        case 'connection': return logApi.getConnectionLogs({ page, pageSize, ...filters });
        case 'execution': return logApi.getExecutionLogs({ page, pageSize, ...filters });
        case 'orders': return logApi.getOrderHistory({ page, pageSize, ...filters });
        default: return logApi.getOperationLogs({ page, pageSize, ...filters });
      }
    });

  const logs = (queryResult as { logs?: LogEntry[]; orders?: OrderHistoryRecord[]; total: number } | undefined)?.logs
    || (queryResult as { orders?: OrderHistoryRecord[] } | undefined)?.orders || [];
  const total = (queryResult as { total?: number } | undefined)?.total || 0;
  const error = queryError ? getErrorMessage(queryError, t('logs.loadFailed')) : null;

  const accountById = useMemo(() => {
    const m = new Map<string, AccountLike>();
    (accounts || []).forEach((a) => { if (a?.id) m.set(String(a.id), a); });
    return m;
  }, [accounts]);

  const toDateSafe = (v: unknown): Date | null => {
    if (!v) return null;
    if (v instanceof Date && !Number.isNaN(v.getTime())) return v;
    if (typeof v === 'string') { const d = new Date(v); return Number.isNaN(d.getTime()) ? null : d; }
    if (typeof v === 'number' && Number.isFinite(v)) { const d = new Date(v); return Number.isNaN(d.getTime()) ? null : d; }
    if (typeof v === 'object' && v !== null) {
      const vo = v as Record<string, unknown>;
      if (typeof vo['toDate'] === 'function') {
        try { const d = (vo['toDate'] as () => Date).call(v); if (d instanceof Date && !Number.isNaN(d.getTime())) return d; } catch { /* ignore */ }
      }
      const secNum = typeof vo['seconds'] === 'bigint' ? Number(vo['seconds']) : typeof vo['seconds'] === 'number' ? vo['seconds'] : undefined;
      const nanoNum = typeof vo['nanos'] === 'bigint' ? Number(vo['nanos']) : typeof vo['nanos'] === 'number' ? vo['nanos'] : 0;
      if (typeof secNum === 'number' && Number.isFinite(secNum)) {
        return new Date(secNum * 1000 + (Number.isFinite(nanoNum) ? Math.floor(nanoNum / 1_000_000) : 0));
      }
    }
    return null;
  };

  const formatTime = (v: unknown) => {
    const d = toDateSafe(v);
    if (!d) return '-';
    return d.toLocaleString(getDeviceLocale(), { timeZone: getDeviceTimeZone(), hour12: false });
  };

  const handleSearch = () => {
    // Search is handled by LogFilterForm which calls this via the onSearch prop
    // The actual filter values are managed through the operation-specific state
    setPage(1);
    setFilters({});
  };

  const handleReset = () => {
    setOpRiskCode(''); setOpRequestId(''); setOpTriggerSource(''); setOpResult('');
    setPage(1); setFilters({});
  };

  const handleQuickRiskFilter = () => { setPage(1); setFilters({ module: 'trading_risk', action: 'pre_trade_validate' }); };

  const filteredLogs = useMemo(() => {
    if (activeTab !== 'operations') return logs;
    return logs.filter((r) => {
      const op = r as OperationLog;
      let d: Record<string, unknown> = {};
      try { if (op?.details) { const p = JSON.parse(op.details); if (p && typeof p === 'object') d = p as Record<string, unknown>; } } catch { /* ignore */ }
      if (opRiskCode && !String(d?.risk_code || '').toLowerCase().includes(opRiskCode.toLowerCase())) return false;
      if (opRequestId && !String(d?.request_id || '').toLowerCase().includes(opRequestId.toLowerCase())) return false;
      if (opTriggerSource && String(d?.trigger_source || '').toLowerCase() !== opTriggerSource.toLowerCase()) return false;
      if (opResult && String(d?.result || '').toLowerCase() !== opResult.toLowerCase()) return false;
      return true;
    });
  }, [activeTab, logs, opRiskCode, opRequestId, opTriggerSource, opResult]);

  const colOpts = { t, formatTime, accountById };
  const columns =
    activeTab === 'connection' ? buildConnectionColumns(colOpts) :
    activeTab === 'execution' ? buildExecutionColumns(colOpts) :
    activeTab === 'orders' ? buildOrderColumns(colOpts) :
    buildOperationColumns(colOpts);

  return (
    <div className="p-6">
      <Card>
        <Tabs activeKey={activeTab} onChange={setActiveTab}
          items={[
            { key: 'connection', label: t('logs.connectionLogs') },
            { key: 'execution', label: t('logs.executionLogs') },
            { key: 'orders', label: t('logs.orderHistory') },
            { key: 'operations', label: t('logs.operationLogs') },
          ]} className="mb-4" />
        <LogFilterForm
          activeTab={activeTab}
          opRiskCode={opRiskCode} opRequestId={opRequestId} opTriggerSource={opTriggerSource} opResult={opResult}
          onRiskCodeChange={setOpRiskCode} onRequestIdChange={setOpRequestId}
          onTriggerSourceChange={setOpTriggerSource} onResultChange={setOpResult}
          onSearch={handleSearch} onReset={handleReset} onQuickRiskFilter={handleQuickRiskFilter}
        />
        <StatusResult loading={loading} error={error} empty={!loading && !error && filteredLogs.length === 0}
          emptyText={t('common.noData', { defaultValue: 'No logs found' })} onRetry={() => refetch()}>
          <Table scroll={{ x: 'max-content' }} columns={columns} dataSource={filteredLogs} rowKey="id"
            pagination={{ current: page, pageSize, total, onChange: (p, ps) => { setPage(p); setPageSize(ps); } }} />
        </StatusResult>
      </Card>
    </div>
  );
}
