import { useState, useEffect, useCallback, useMemo } from 'react';
import { Table, Card, Form, DatePicker, Select, Button, Tag, Tabs, Space, Input } from 'antd';
import { accountApi } from '@/client/account';
import { logApi } from '@/client/log';
import type { ConnectionLog } from '@/gen/log_connection_pb';
import { getDeviceLocale, getDeviceTimeZone } from '@/utils/date';
import { getErrorMessage } from '@/utils/error';
import { showError } from '@/utils/message';
import { useTranslation } from 'react-i18next';

const { RangePicker } = DatePicker;

export default function LogManagement() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('connection');
  const [loading, setLoading] = useState(false);
  const [logs, setLogs] = useState<any[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [accounts, setAccounts] = useState<any[]>([]);
  const [opRiskCode, setOpRiskCode] = useState('');
  const [opRequestId, setOpRequestId] = useState('');
  const [opTriggerSource, setOpTriggerSource] = useState('');
  const [opResult, setOpResult] = useState('');
  const [form] = Form.useForm();

  const accountById = useMemo(() => {
    const m = new Map<string, any>();
    (accounts || []).forEach((a: any) => {
      if (a?.id) m.set(String(a.id), a);
    });
    return m;
  }, [accounts]);

  const toDateSafe = (v: any): Date | null => {
    if (!v) return null;
    if (v instanceof Date && !Number.isNaN(v.getTime())) return v;
    if (typeof v === 'string') {
      const d = new Date(v);
      return Number.isNaN(d.getTime()) ? null : d;
    }
    if (typeof v === 'number' && Number.isFinite(v)) {
      const d = new Date(v);
      return Number.isNaN(d.getTime()) ? null : d;
    }
    if (typeof v === 'object') {
      const toDate = (v as any)?.toDate;
      if (typeof toDate === 'function') {
        try {
          const d = toDate.call(v);
          if (d instanceof Date && !Number.isNaN(d.getTime())) return d;
        } catch (_e) {
          // ignore
        }
      }
      const seconds = (v as any)?.seconds;
      const nanos = (v as any)?.nanos;
      const secNum = typeof seconds === 'bigint' ? Number(seconds) : typeof seconds === 'number' ? seconds : undefined;
      const nanoNum = typeof nanos === 'bigint' ? Number(nanos) : typeof nanos === 'number' ? nanos : 0;
      if (typeof secNum === 'number' && Number.isFinite(secNum)) {
        const ms = secNum * 1000 + (Number.isFinite(nanoNum) ? Math.floor(nanoNum / 1_000_000) : 0);
        const d = new Date(ms);
        return Number.isNaN(d.getTime()) ? null : d;
      }
    }
    return null;
  };

  const formatTime = (v: any) => {
    const d = toDateSafe(v);
    if (!d) return '-';
    const locale = getDeviceLocale();
    const timeZone = getDeviceTimeZone();
    return d.toLocaleString(locale, { timeZone, hour12: false });
  };

  const fetchLogs = useCallback(async (filters?: any) => {
    setLoading(true);
    try {
      let result: any;
      
      switch (activeTab) {
        case 'connection':
          result = await logApi.getConnectionLogs({
            page,
            pageSize,
            ...filters,
          });
          setLogs(result.logs);
          setTotal(result.total);
          break;
        case 'execution':
          result = await logApi.getExecutionLogs({
            page,
            pageSize,
            ...filters,
          });
          setLogs(result.logs);
          setTotal(result.total);
          break;
        case 'orders':
          result = await logApi.getOrderHistory({
            page,
            pageSize,
            ...filters,
          });
          setLogs(result.orders);
          setTotal(result.total);
          break;
        case 'operations':
          result = await logApi.getOperationLogs({
            page,
            pageSize,
            ...filters,
          });
          setLogs(result.logs);
          setTotal(result.total);
          break;
      }
    } catch (error) {
      showError(getErrorMessage(error, '加载日志失败'));
    } finally {
      setLoading(false);
    }
  }, [activeTab, page, pageSize]);

  useEffect(() => {
    accountApi
      .list()
      .then((accs: any) => setAccounts(Array.isArray(accs) ? accs : []))
      .catch(() => setAccounts([]));
  }, []);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const handleSearch = () => {
    const values = form.getFieldsValue();
    const filters: any = {};
    
    if (values.dateRange) {
      filters.startDate = values.dateRange[0].format('YYYY-MM-DD');
      filters.endDate = values.dateRange[1].format('YYYY-MM-DD');
    }
    if (values.status) filters.status = values.status;
    if (values.symbol) filters.symbol = values.symbol;
    if (values.accountId) filters.accountId = values.accountId;
    if (values.module) filters.module = values.module;
    if (values.action) filters.action = values.action;
    
    setPage(1);
    fetchLogs(filters);
  };

  const handleReset = () => {
    form.resetFields();
    setOpRiskCode('');
    setOpRequestId('');
    setOpTriggerSource('');
    setOpResult('');
    setPage(1);
    fetchLogs();
  };

  const getStatusTag = (status?: string) => {
    const normalized = String(status || '').toLowerCase();
    const colors: Record<string, string> = {
      success: 'green',
      failed: 'red',
      completed: 'green',
      running: 'blue',
      pending: 'orange',
      skipped: 'default',
    };
    const label =
      normalized === 'success'
        ? t('logs.success')
        : normalized === 'failed'
          ? t('logs.failed')
          : normalized
            ? normalized.toUpperCase()
            : '-';
    return <Tag color={colors[normalized] || 'default'}>{label}</Tag>;
  };

  const getEventTypeTag = (type?: string) => {
    const normalized = String(type || '').toLowerCase();
    const colors: Record<string, string> = {
      connect: 'blue',
      disconnect: 'orange',
      reconnect: 'cyan',
      error: 'red',
      heartbeat: 'green',
    };
    return <Tag color={colors[normalized] || 'default'}>{normalized ? normalized.toUpperCase() : '-'}</Tag>;
  };

  const getSignalTypeTag = (type?: string) => {
    if (!type) return '-';
    const colors: Record<string, string> = {
      buy: 'green',
      sell: 'red',
      close: 'orange',
      hold: 'default',
      modify: 'blue',
    };
    return <Tag color={colors[type] || 'default'}>{type.toUpperCase()}</Tag>;
  };

  const parseOperationDetails = (raw?: string): Record<string, any> => {
    if (!raw) return {};
    try {
      const parsed = JSON.parse(raw);
      if (parsed && typeof parsed === 'object') return parsed;
      return {};
    } catch (_e) {
      return {};
    }
  };

  const connectionColumns = [
    { title: t('logs.time'), dataIndex: 'createdAt', key: 'createdAt', width: 180, render: (v: any) => formatTime(v) },
    { title: t('logs.eventType'), dataIndex: 'eventType', key: 'eventType', width: 120, render: getEventTypeTag },
    { title: t('logs.status'), dataIndex: 'status', key: 'status', width: 100, render: getStatusTag },
    {
      title: t('logs.server'),
      key: 'server',
      width: 200,
      render: (_: unknown, r: ConnectionLog) => {
        const a = accountById.get(String(r.accountId || ''));
        const name = String(a?.brokerServer || a?.brokerHost || a?.brokerCompany || '').trim();
        if (name) return name;
        const host = String(r.serverHost || '').trim();
        const port = String(r.serverPort ?? '').trim();
        return host && port ? `${host}:${port}` : host || '-';
      },
    },
    {
      title: t('logs.loginId'),
      dataIndex: 'loginId',
      key: 'loginId',
      width: 100,
      render: (v: bigint | number | undefined) => (v !== undefined && v !== null ? String(v) : '-'),
    },
    { title: t('logs.message'), dataIndex: 'message', key: 'message', ellipsis: true },
    {
      title: t('logs.duration'),
      dataIndex: 'connectionDurationSeconds',
      key: 'duration',
      width: 100,
      render: (v: bigint | number | undefined) => (v ? `${String(v)}s` : '-'),
    },
  ];

  const executionColumns = [
    { title: t('logs.time'), dataIndex: 'createdAt', key: 'createdAt', width: 180, render: (v: any) => formatTime(v) },
    { title: t('logs.product'), dataIndex: 'symbol', key: 'symbol', width: 100 },
    { title: t('logs.period'), dataIndex: 'timeframe', key: 'timeframe', width: 80 },
    { title: t('logs.status'), dataIndex: 'status', key: 'status', width: 100, render: getStatusTag },
    { title: t('logs.signal'), dataIndex: 'signalType', key: 'signalType', width: 80, render: getSignalTypeTag },
    { title: t('logs.signalPrice'), dataIndex: 'signalPrice', key: 'signalPrice', width: 100, render: (v: number) => v?.toFixed(5) || '-' },
    { title: t('logs.executionPrice'), dataIndex: 'executedPrice', key: 'executedPrice', width: 100, render: (v: number) => v?.toFixed(5) || '-' },
    { title: t('logs.profit'), dataIndex: 'profit', key: 'profit', width: 100, render: (v: number) => v ? <span style={{ color: v >= 0 ? 'green' : 'red' }}>{v.toFixed(2)}</span> : '-' },
    { title: t('logs.cost'), dataIndex: 'executionTimeMs', key: 'executionTimeMs', width: 80, render: (v: number) => v ? `${v}ms` : '-' },
    { title: t('logs.error'), dataIndex: 'errorMessage', key: 'errorMessage', ellipsis: true },
  ];

  const orderColumns = [
    { title: t('logs.time'), dataIndex: 'openTime', key: 'openTime', width: 180, render: (v: any) => formatTime(v) },
    { title: t('logs.orderTable.ticket'), dataIndex: 'ticket', key: 'ticket', width: 100 },
    { title: t('logs.product'), dataIndex: 'symbol', key: 'symbol', width: 100 },
    { title: t('logs.orderTable.type'), dataIndex: 'orderType', key: 'orderType', width: 100 },
    { title: t('logs.orderTable.lots'), dataIndex: 'lots', key: 'lots', width: 80 },
    { title: t('logs.orderTable.open'), dataIndex: 'openPrice', key: 'openPrice', width: 100, render: (v: number) => v?.toFixed(5) || '-' },
    { title: t('logs.orderTable.close'), dataIndex: 'closePrice', key: 'closePrice', width: 100, render: (v: number) => v?.toFixed(5) || '-' },
    { title: t('logs.profit'), dataIndex: 'profit', key: 'profit', width: 100, render: (v: number) => v ? <span style={{ color: v >= 0 ? 'green' : 'red' }}>{v.toFixed(2)}</span> : '-' },
  ];

  const operationColumns = [
    { title: t('logs.time'), dataIndex: 'createdAt', key: 'createdAt', width: 180, render: (v: any) => formatTime(v) },
    { title: t('logs.module'), dataIndex: 'module', key: 'module', width: 120 },
    { title: t('logs.action'), dataIndex: 'action', key: 'action', width: 150 },
    {
      title: '结果',
      key: 'riskResult',
      width: 100,
      render: (_: any, r: any) => {
        const d = parseOperationDetails(r?.details);
        const val = String(d?.result || '').toLowerCase();
        if (!val) return '-';
        return <Tag color={val === 'pass' ? 'green' : val === 'reject' ? 'red' : 'default'}>{val.toUpperCase()}</Tag>;
      },
    },
    {
      title: '风险码',
      key: 'riskCode',
      width: 220,
      render: (_: any, r: any) => {
        const d = parseOperationDetails(r?.details);
        return d?.risk_code || '-';
      },
    },
    {
      title: '请求ID',
      key: 'requestId',
      width: 220,
      render: (_: any, r: any) => {
        const d = parseOperationDetails(r?.details);
        return d?.request_id || '-';
      },
    },
    {
      title: '触发源',
      key: 'triggerSource',
      width: 120,
      render: (_: any, r: any) => {
        const d = parseOperationDetails(r?.details);
        return d?.trigger_source || '-';
      },
    },
    { title: t('logs.details'), dataIndex: 'details', key: 'details', ellipsis: true },
    { title: t('logs.ip'), dataIndex: 'ip', key: 'ip', width: 120 },
  ];

  const filteredLogs = useMemo(() => {
    if (activeTab !== 'operations') return logs;
    return logs.filter((r: any) => {
      const d = parseOperationDetails(r?.details);
      if (opRiskCode && !String(d?.risk_code || '').toLowerCase().includes(opRiskCode.toLowerCase())) return false;
      if (opRequestId && !String(d?.request_id || '').toLowerCase().includes(opRequestId.toLowerCase())) return false;
      if (opTriggerSource && String(d?.trigger_source || '').toLowerCase() !== opTriggerSource.toLowerCase()) return false;
      if (opResult && String(d?.result || '').toLowerCase() !== opResult.toLowerCase()) return false;
      return true;
    });
  }, [activeTab, logs, opRiskCode, opRequestId, opTriggerSource, opResult]);

  const tabItems = [
    { key: 'connection', label: t('logs.connectionLogs') },
    { key: 'execution', label: t('logs.executionLogs') },
    { key: 'orders', label: t('logs.orderHistory') },
    { key: 'operations', label: t('logs.operationLogs') },
  ];

  const renderFilterForm = () => (
    <Form form={form} layout="inline" className="mb-4">
      <Space wrap>
        <Form.Item name="dateRange" label={t('logs.dateRange')}>
          <RangePicker style={{ width: 240 }} />
        </Form.Item>
        {activeTab === 'connection' && (
          <Form.Item name="status" label={t('logs.status')}>
            <Select style={{ width: 120 }} allowClear>
              <Select.Option value="success">{t('logs.success')}</Select.Option>
              <Select.Option value="failed">{t('logs.failed')}</Select.Option>
            </Select>
          </Form.Item>
        )}
        {(activeTab === 'execution' || activeTab === 'orders') && (
          <Form.Item name="symbol" label={t('logs.symbol')}>
            <Input style={{ width: 120 }} placeholder={t('logs.exampleSymbolPlaceholder')} />
          </Form.Item>
        )}
        {activeTab === 'operations' && (
          <>
            <Form.Item name="module" hidden><Input /></Form.Item>
            <Form.Item name="action" hidden><Input /></Form.Item>
            <Form.Item>
              <Button
                onClick={() => {
                  form.setFieldsValue({ module: 'trading_risk', action: 'pre_trade_validate' });
                  const values = form.getFieldsValue();
                  setPage(1);
                  fetchLogs({
                    ...values,
                    module: 'trading_risk',
                    action: 'pre_trade_validate',
                  });
                }}
              >
                风控日志快速筛选
              </Button>
            </Form.Item>
            <Form.Item label="风险码">
              <Input
                style={{ width: 220 }}
                placeholder="RISK_MARGIN_INSUFFICIENT"
                value={opRiskCode}
                onChange={(e) => setOpRiskCode(e.target.value)}
              />
            </Form.Item>
            <Form.Item label="请求ID">
              <Input
                style={{ width: 220 }}
                placeholder="request_id"
                value={opRequestId}
                onChange={(e) => setOpRequestId(e.target.value)}
              />
            </Form.Item>
            <Form.Item label="触发源">
              <Select
                allowClear
                style={{ width: 130 }}
                value={opTriggerSource || undefined}
                onChange={(v) => setOpTriggerSource(v || '')}
                options={[
                  { label: 'manual', value: 'manual' },
                  { label: 'strategy', value: 'strategy' },
                  { label: 'recovery', value: 'recovery' },
                ]}
              />
            </Form.Item>
            <Form.Item label="结果">
              <Select
                allowClear
                style={{ width: 120 }}
                value={opResult || undefined}
                onChange={(v) => setOpResult(v || '')}
                options={[
                  { label: 'PASS', value: 'pass' },
                  { label: 'REJECT', value: 'reject' },
                ]}
              />
            </Form.Item>
          </>
        )}
        <Form.Item name="accountId" label={t('logs.accountId')}>
          <Input style={{ width: 200 }} placeholder={t('logs.accountId')} />
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" onClick={handleSearch}>{t('logs.search')}</Button>
            <Button onClick={handleReset}>{t('logs.reset')}</Button>
          </Space>
        </Form.Item>
      </Space>
    </Form>
  );

  return (
    <div className="p-6">
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={tabItems}
          className="mb-4"
        />
        {renderFilterForm()}
        <Table
          scroll={{ x: "max-content" }}
          columns={
            activeTab === 'connection' ? connectionColumns :
            activeTab === 'execution' ? executionColumns :
            activeTab === 'orders' ? orderColumns :
            operationColumns
          }
          dataSource={filteredLogs}
          rowKey="id"
          loading={loading}
          pagination={{
            current: page,
            pageSize,
            total,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </Card>
    </div>
  );
}
