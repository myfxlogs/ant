import { useEffect, useMemo, useState, useCallback } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Button, Card, Descriptions, Space, Table, Tabs, Tag, Typography, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { Timestamp } from '@bufbuild/protobuf/wkt';
import { timestampDate } from '@bufbuild/protobuf/wkt';

import { logApi } from '@/client/log';
import { strategyScheduleV2Api } from '@/client/strategy';
import type { OrderHistoryRecord } from '@/types/log';
import { getDeviceLocale, getDeviceTimeZone } from '@/utils/date';
import { useTranslation } from 'react-i18next';
import { StatusResult } from '@/components/common/StatusResult';

const { Title, Text } = Typography;

function formatTime(v: unknown) {
  if (!v) return '-';
  const locale = getDeviceLocale();
  const timeZone = getDeviceTimeZone();
  try {
    if (typeof v === 'string') {
      const s = v.trim();
      if (/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(\.\d+)?$/.test(s)) {
        const d = new Date(s.replace(' ', 'T') + 'Z');
        if (!Number.isNaN(d.getTime())) {
          return d.toLocaleString(locale, { timeZone, hour12: false });
        }
      }
    }

    if (typeof v === 'object') {
      const ts = v as Partial<Timestamp>;
      const secNum =
        typeof ts.seconds === 'number'
          ? ts.seconds
          : typeof ts.seconds === 'bigint'
            ? Number(ts.seconds)
            : undefined;
      if (typeof secNum === 'number' && Number.isFinite(secNum)) {
        try {
          const d = timestampDate(v as Timestamp);
          if (d instanceof Date && !Number.isNaN(d.getTime())) {
            return d.toLocaleString(locale, { timeZone, hour12: false });
          }
        } catch {
          // ignore
        }
      }
    }

    if (v instanceof Date) {
      return v.toLocaleString(locale, { timeZone, hour12: false });
    }

    const d = typeof v === 'string' || typeof v === 'number' ? new Date(v) : new Date(String(v));
    if (Number.isNaN(d.getTime())) return String(v);
    return d.toLocaleString(locale, { timeZone, hour12: false });
  } catch {
    return String(v);
  }
}

function renderExecStatus(v: unknown, t: (key: string, opts?: unknown) => string) {
  const s = String(v || '').trim().toLowerCase();
  const text = (() => {
    if (s === 'pending') return t('strategy.scheduleLogs.execStatus.pending');
    if (s === 'running') return t('strategy.scheduleLogs.execStatus.running');
    if (s === 'completed' || s === 'success') return t('strategy.scheduleLogs.execStatus.completed');
    if (s === 'failed') return t('strategy.scheduleLogs.execStatus.failed');
    if (s === 'skipped') return t('strategy.scheduleLogs.execStatus.skipped');
    return String(v || '-');
  })();
  if (s === 'completed' || s === 'success') return <Tag color="green">{text}</Tag>;
  if (s === 'failed') return <Tag color="red">{text}</Tag>;
  if (s === 'running') return <Tag color="blue">{text}</Tag>;
  return <Tag>{text}</Tag>;
}

function renderMs(v: unknown) {
  if (typeof v === 'number') return <Text>{v}</Text>;
  if (typeof v === 'bigint') return <Text>{Number(v)}</Text>;
  if (typeof v === 'string') {
    const n = Number(v);
    return <Text>{Number.isFinite(n) ? n : '-'}</Text>;
  }
  return <Text>-</Text>;
}

function renderOperationAction(v: unknown, t: (key: string, opts?: unknown) => string) {
  const s = String(v || '').trim().toLowerCase();
  if (s === 'enable') return <Text>{t('common.enable')}</Text>;
  if (s === 'disable') return <Text>{t('common.disable')}</Text>;
  return <Text>{String(v || '-')}</Text>;
}

function renderOperationStatus(v: unknown, t: (key: string, opts?: unknown) => string) {
  const s = String(v || '').trim().toLowerCase();
  const text = (() => {
    if (s === 'success' || s === 'completed') return t('strategy.scheduleLogs.operationStatus.success');
    if (s === 'failed') return t('strategy.scheduleLogs.operationStatus.failed');
    if (s === 'running') return t('strategy.scheduleLogs.operationStatus.running');
    return String(v || '-');
  })();
  if (s === 'success' || s === 'completed') return <Tag color="green">{text}</Tag>;
  if (s === 'failed') return <Tag color="red">{text}</Tag>;
  if (s === 'running') return <Tag color="blue">{text}</Tag>;
  return <Tag>{text}</Tag>;
}

function renderOrderTypeTag(value: string, t: (key: string, opts?: unknown) => string) {
  const s = String(value || '').toLowerCase();
  if (s === 'close') return <Tag color="gold">{t('strategy.scheduleLogs.orderSide.close')}</Tag>;
  if (s.includes('buy_stop_limit')) return <Tag color="cyan">{t('strategy.scheduleLogs.orderSide.buyStopLimit')}</Tag>;
  if (s.includes('sell_stop_limit')) return <Tag color="cyan">{t('strategy.scheduleLogs.orderSide.sellStopLimit')}</Tag>;
  if (s.includes('buy_limit')) return <Tag color="blue">{t('strategy.scheduleLogs.orderSide.buyLimit')}</Tag>;
  if (s.includes('sell_limit')) return <Tag color="blue">{t('strategy.scheduleLogs.orderSide.sellLimit')}</Tag>;
  if (s.includes('buy_stop')) return <Tag color="purple">{t('strategy.scheduleLogs.orderSide.buyStop')}</Tag>;
  if (s.includes('sell_stop')) return <Tag color="purple">{t('strategy.scheduleLogs.orderSide.sellStop')}</Tag>;
  if (s.includes('buy')) return <Tag color="green">{t('strategy.scheduleLogs.orderSide.buy')}</Tag>;
  if (s.includes('sell')) return <Tag color="red">{t('strategy.scheduleLogs.orderSide.sell')}</Tag>;
  return <Tag>{value || '-'}</Tag>;
}

export default function StrategyScheduleLogsPage() {
  const { t } = useTranslation();
  const { id } = useParams();
  const scheduleId = String(id || '');
  const navigate = useNavigate();

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [schedule, setSchedule] = useState<any>(null);

  const [activeTab, setActiveTab] = useState<'exec' | 'orders'>('exec');

  const [execPage, setExecPage] = useState(1);
  const [execPageSize, setExecPageSize] = useState(20);
  const [execTotal, setExecTotal] = useState(0);
  const [execLogs, setExecLogs] = useState<any[]>([]);

  const [orderPage, setOrderPage] = useState(1);
  const [orderPageSize, setOrderPageSize] = useState(20);
  const [orderTotal, setOrderTotal] = useState(0);
  const [orders, setOrders] = useState<OrderHistoryRecord[]>([]);

  const title = useMemo(() => {
    const name = String(schedule?.name || '').trim();
    return name ? t('strategy.scheduleLogs.titleWithName', { name }) : t('strategy.scheduleLogs.title');
  }, [schedule?.name, t]);

  const refreshSchedule = useCallback(async () => {
    if (!scheduleId) return;
    const s: any = await strategyScheduleV2Api.get(scheduleId);
    setSchedule(s);
  }, [scheduleId]);

  const refreshExec = useCallback(async () => {
    if (!scheduleId) return;
    const res = await logApi.getScheduleRunLogs({
      page: execPage,
      pageSize: execPageSize,
      scheduleId,
    });
    setExecLogs(res.logs || []);
    setExecTotal(res.total || 0);
  }, [scheduleId, execPage, execPageSize]);

  const refreshOrders = useCallback(async () => {
    if (!scheduleId) return;
    const res = await logApi.getOrderHistory({
      page: orderPage,
      pageSize: orderPageSize,
      scheduleId,
    });
    setOrders(res.orders || []);
    setOrderTotal(res.total || 0);
  }, [scheduleId, orderPage, orderPageSize]);

  const refresh = useCallback(async () => {
    if (!scheduleId) {
      message.error(t('strategy.scheduleLogs.messages.missingScheduleId'));
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await Promise.all([
        refreshSchedule(),
        refreshExec(),
        refreshOrders(),
      ]);
    } catch (e: unknown) {
      const msg = e?.message || t('common.loadingFailed');
      setError(msg);
      message.error(msg);
    } finally {
      setLoading(false);
    }
  }, [scheduleId, t, refreshSchedule, refreshExec, refreshOrders]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    if (activeTab === 'exec') void refreshExec();
  }, [activeTab, refreshExec]);

  useEffect(() => {
    if (activeTab === 'orders') void refreshOrders();
  }, [activeTab, refreshOrders]);

  const execColumns: ColumnsType<any> = [
    {
      title: t('strategy.scheduleLogs.execTable.time'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (_v: unknown, row: ScheduleRunLog) => <Text>{formatTime(row?.createdAt)}</Text>,
    },
    {
      title: t('strategy.scheduleLogs.execTable.action'),
      key: 'action',
      width: 160,
      render: (_: unknown, row: ScheduleRunLog) => {
        if (String(row?.kind || '').toLowerCase() === 'operation') return renderOperationAction(row?.action, t);
        const st = String(row?.signalType || row?.action || '').toLowerCase();
        if (st === 'close') return <Text>{t('strategy.scheduleLogs.orderSide.close')}</Text>;
        return <Text>{String(row?.signalType || row?.action || t('strategy.scheduleLogs.execTable.execute'))}</Text>;
      },
    },
    {
      title: t('strategy.scheduleLogs.execTable.status'),
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (_: unknown, row: ScheduleRunLog) => {
        if (String(row?.kind || '').toLowerCase() === 'operation') return renderOperationStatus(row?.status, t);
        return renderExecStatus(row?.status, t);
      },
    },
    {
      title: t('strategy.scheduleLogs.execTable.durationMs'),
      key: 'duration',
      width: 110,
      render: (_: unknown, row: ScheduleRunLog) => {
        return renderMs(row?.durationMs);
      },
    },
    {
      title: t('strategy.scheduleLogs.execTable.error'),
      dataIndex: 'errorMessage',
      key: 'errorMessage',
      render: (v: unknown) => {
        const s = String(v || '').trim();
        if (!s) return <Text type="secondary">{t('common.none')}</Text>;
        return (
          <Text type="danger" ellipsis={{ tooltip: s }} style={{ maxWidth: 360, display: 'inline-block' }}>
            {s}
          </Text>
        );
      },
    },
  ];

  const orderColumns: ColumnsType<OrderHistoryRecord> = [
    {
      title: t('strategy.scheduleLogs.ordersTable.time'),
      key: 'time',
      width: 180,
      render: (_: unknown, row: ScheduleRunLog) => <Text>{formatTime(row?.closeTime || row?.openTime)}</Text>,
    },
    {
      title: t('strategy.scheduleLogs.ordersTable.side'),
      dataIndex: 'orderType',
      key: 'orderType',
      width: 100,
      render: (v: unknown) => renderOrderTypeTag(String(v || ''), t),
    },
    {
      title: t('strategy.scheduleLogs.ordersTable.symbol'),
      dataIndex: 'symbol',
      key: 'symbol',
      width: 120,
      render: (v: unknown) => <Text>{String(v || '-')}</Text>,
    },
    {
      title: t('strategy.scheduleLogs.ordersTable.lots'),
      dataIndex: 'lots',
      key: 'lots',
      width: 90,
      render: (v: unknown) => <Text>{typeof v === 'number' ? v : '-'}</Text>,
    },
    {
      title: t('strategy.scheduleLogs.ordersTable.openPrice'),
      dataIndex: 'openPrice',
      key: 'openPrice',
      width: 120,
      render: (v: unknown) => <Text>{typeof v === 'number' ? v : '-'}</Text>,
    },
    {
      title: t('strategy.scheduleLogs.ordersTable.closePrice'),
      dataIndex: 'closePrice',
      key: 'closePrice',
      width: 120,
      render: (v: unknown) => <Text>{typeof v === 'number' ? v : '-'}</Text>,
    },
    {
      title: t('strategy.scheduleLogs.ordersTable.profit'),
      dataIndex: 'profit',
      key: 'profit',
      width: 120,
      render: (v: unknown) => {
        const n = typeof v === 'number' ? v : Number(v);
        if (!Number.isFinite(n)) return <Text>-</Text>;
        if (n > 0) return <Text style={{ color: '#00A651' }}>{n.toFixed(2)}</Text>;
        if (n < 0) return <Text type="danger">{n.toFixed(2)}</Text>;
        return <Text>{n.toFixed(2)}</Text>;
      },
    },
    {
      title: t('strategy.scheduleLogs.ordersTable.ticket'),
      dataIndex: 'ticket',
      key: 'ticket',
      width: 110,
      render: (v: unknown) => <Text>{typeof v === 'number' ? v : '-'}</Text>,
    },
  ];

  return (
    <Space orientation="vertical" style={{ width: '100%' }} size={16}>
      <Card>
        <Space align="start" style={{ width: '100%', justifyContent: 'space-between' }}>
          <Space orientation="vertical" size={4}>
            <Title level={4} style={{ margin: 0 }}>{title}</Title>
            <Text type="secondary">{t('strategy.scheduleLogs.scheduleIdLabel')} {scheduleId || '-'}</Text>
          </Space>
          <Space>
            <Button onClick={() => navigate('/strategy/schedules')}>{t('common.back')}</Button>
            <Button onClick={() => void refresh()} loading={loading}>{t('common.refresh')}</Button>
          </Space>
        </Space>

        <Descriptions size="small" column={2} style={{ marginTop: 16 }}>
          <Descriptions.Item label={t('strategy.scheduleLogs.summary.name')}>{schedule?.name || '-'}</Descriptions.Item>
          <Descriptions.Item label={t('strategy.scheduleLogs.summary.status')}>{schedule?.isActive ? <Tag color="green">{t('strategy.schedules.status.running')}</Tag> : <Tag>{t('strategy.schedules.status.disabled')}</Tag>}</Descriptions.Item>
          <Descriptions.Item label={t('strategy.scheduleLogs.summary.trade')}>{`${schedule?.symbol || '-'} / ${schedule?.timeframe || '-'}`}</Descriptions.Item>
          <Descriptions.Item label={t('strategy.scheduleLogs.summary.enableCount')}>{typeof schedule?.enableCount === 'number' ? schedule.enableCount : '-'}</Descriptions.Item>
          <Descriptions.Item label={t('strategy.scheduleLogs.summary.lastRun')}>{formatTime(schedule?.lastRunAt)}</Descriptions.Item>
          <Descriptions.Item label={t('strategy.scheduleLogs.summary.lastError')}>{String(schedule?.lastError || '').trim() || '-'}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Card>
        <StatusResult error={error} onRetry={() => refresh()}>
        <Tabs
          activeKey={activeTab}
          onChange={(k) => setActiveTab(k as 'exec' | 'orders')}
          items={[
            {
              key: 'exec',
              label: t('strategy.scheduleLogs.tabs.exec'),
              children: (
                <Table
                  scroll={{ x: "max-content" }}
                  rowKey={(r: Record<string, unknown>) => String(r?.id || '')}
                  loading={loading}
                  columns={execColumns}
                  dataSource={execLogs}
                  pagination={{
                    current: execPage,
                    pageSize: execPageSize,
                    total: execTotal,
                    showSizeChanger: true,
                    onChange: (p, ps) => {
                      setExecPage(p);
                      setExecPageSize(ps);
                    },
                  }}
                />
              ),
            },
            {
              key: 'orders',
              label: t('strategy.scheduleLogs.tabs.orders'),
              children: (
                <Table
                  scroll={{ x: "max-content" }}
                  rowKey={(r) => String(r.id || r.ticket)}
                  loading={loading}
                  columns={orderColumns}
                  dataSource={orders}
                  pagination={{
                    current: orderPage,
                    pageSize: orderPageSize,
                    total: orderTotal,
                    showSizeChanger: true,
                    onChange: (p, ps) => {
                      setOrderPage(p);
                      setOrderPageSize(ps);
                    },
                  }}
                />
              ),
            },
          ]}
        />
        </StatusResult>
      </Card>
    </Space>
  );
}
