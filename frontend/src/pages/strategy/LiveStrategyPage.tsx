import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { Table, Tag, Typography, Button, Card, Space, message, Popconfirm, Tabs, Empty, Tooltip, Alert } from 'antd';
import { ReloadOutlined, StopOutlined, EyeOutlined, MonitorOutlined, ClockCircleOutlined, FileTextOutlined, HeartOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { strategyActiveApi, strategyRunsApi } from '@/client/strategy';
import { accountApi } from '@/client/account';
import type { Account } from '@/types/account';
import type { ActiveStrategy, StrategyRun, StrategySignalEvent } from '@/gen/ant/v1/strategy_runtime_pb';
import { SignalDrawer, formatTime, shortId, STATUS_COLORS, MODE_COLORS } from './LiveStrategyPageSignalDrawer';
import LiveSchedulesTab from './components/workspace/LiveSchedulesTab';
import ScheduleLogsModal from './components/ScheduleLogsModal';

const { Text } = Typography;

export function isLogButtonDisabled(scheduleId: string): boolean { return !scheduleId; }

export function isHealthButtonDisabled(scheduleId: string): boolean { return !scheduleId; }

function tsToMs(ts: { seconds?: bigint; nanos?: number } | null | undefined): number | null {
  if (!ts || ts.seconds === undefined) return null;
  return Number(ts.seconds) * 1000 + Math.floor((ts.nanos || 0) / 1_000_000);
}

function formatAgo(ts: { seconds?: bigint; nanos?: number } | null | undefined): string {
  const ms = tsToMs(ts);
  if (ms === null) return '-';
  const diff = Math.max(0, Date.now() - ms);
  if (diff < 60_000) return 'now';
  const m = Math.floor(diff / 60_000);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  return `${h}h ago`;
}

function secondsSince(ts: { seconds?: bigint; nanos?: number } | null | undefined): number {
  const ms = tsToMs(ts);
  if (ms === null) return Infinity;
  return (Date.now() - ms) / 1000;
}

export default function LiveStrategyPage() {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const initialTab = searchParams.get('tab') || 'active';
  const highlightScheduleId = searchParams.get('scheduleId') || null;
  const healthId = searchParams.get('healthId') || null;
  const [activeTab, setActiveTab] = useState(initialTab);
  const [activeStrategies, setActiveStrategies] = useState<ActiveStrategy[]>([]);
  const [runs, setRuns] = useState<StrategyRun[]>([]);
  const [loading, setLoading] = useState(false);
  const [stopping, setStopping] = useState<string | null>(null);
  const [signalDrawerOpen, setSignalDrawerOpen] = useState(false);
  const [watchingRunId, setWatchingRunId] = useState<string | null>(null);
  const [signals, setSignals] = useState<StrategySignalEvent[]>([]);
  const abortRef = useRef<AbortController | null>(null);
  const [streamError, setStreamError] = useState(false);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [logsModalScheduleId, setLogsModalScheduleId] = useState<string | null>(null);

  useEffect(() => { void accountApi.list().then(setAccounts).catch(() => {}); }, []);
  const accountById = useMemo(() => { const m = new Map<string, Account>(); accounts.forEach(a => { if (a?.id) m.set(a.id, a); }); return m; }, [accounts]);
  const fmtAccount = useCallback((id: string) => { const a = accountById.get(id); return a?.login ? `${a.login} (${a.mtType})` : id; }, [accountById]);

  const fetchRuns = useCallback(async () => {
    setLoading(true);
    try {
      const r = await strategyRunsApi.listRuns({ limit: 100 });
      setRuns(r as StrategyRun[]);
    } catch {
      setRuns([]);
    }
    setLoading(false);
  }, []);

  useEffect(() => {
    if (activeTab !== 'active') return;
    let active = true;
    const connect = async () => {
      while (active) {
        const ctrl = new AbortController();
        try {
          setLoading(true);
          for await (const event of strategyActiveApi.watchActive('', ctrl.signal)) {
            if (!event.strategies?.length) continue; // skip heartbeat keepalive
            setActiveStrategies((event.strategies || []) as ActiveStrategy[]);
            setLoading(false);
            setStreamError(false);
          }
        } catch {
          // Stream ended — keep existing data, show reconnect banner
        }
        ctrl.abort();
        if (!active) break;
        setStreamError(true);
        await new Promise(r => setTimeout(r, 2000));
      }
    };
    connect();
    return () => { active = false; };
  }, [activeTab]);

  useEffect(() => {
    if (activeTab === 'history') fetchRuns();
  }, [activeTab, fetchRuns]);

  const handleStop = async (runId: string) => {
    setStopping(runId);
    try {
      const r = await strategyActiveApi.stop(runId);
      if (r.success) {
        message.success(t('strategy.live.stopSuccess', { defaultValue: 'Strategy stopped' }));
      } else {
        message.error(r.error || t('strategy.live.stopFailed', { defaultValue: 'Failed to stop' }));
      }
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t('strategy.live.stopFailed', { defaultValue: 'Failed to stop' }));
    }
    setStopping(null);
  };

  const handleWatchSignals = useCallback((runId: string) => {
    if (abortRef.current) abortRef.current.abort();
    abortRef.current = new AbortController();

    setWatchingRunId(runId);
    setSignals([]);
    setSignalDrawerOpen(true);

    (async () => {
      try {
        for await (const event of strategyActiveApi.watchSignals(runId, abortRef.current!.signal)) {
          if (!event.signalType) continue; // skip heartbeat keepalive
          setSignals(prev => [...prev.slice(-199), event as StrategySignalEvent]);
        }
      } catch (e: unknown) {
        if (e instanceof Error && e.name !== 'AbortError') { /* stream ended */ }
      }
    })();
  }, []);

  useEffect(() => {
    return () => { if (abortRef.current) abortRef.current.abort(); };
  }, []);

  const activeColumns = [
    { title: t('strategy.live.runId', { defaultValue: 'Run ID' }), dataIndex: 'runId', width: 100, render: (v: string) => <Text code copyable>{shortId(v)}</Text> },
    { title: t('strategy.live.strategyName', { defaultValue: 'Strategy' }), dataIndex: 'strategyName', width: 120, render: (v: string, record: ActiveStrategy) => v || <Text type="secondary">{shortId(record.runId)}</Text> },
    { title: t('strategy.live.account', { defaultValue: 'Account' }), dataIndex: 'accountId', width: 120, render: (v: string) => <Text style={{ fontSize: 12 }}>{fmtAccount(v)}</Text> },
    { title: t('strategy.live.symbol', { defaultValue: 'Symbol' }), dataIndex: 'symbol', width: 80 },
    { title: t('strategy.live.price', { defaultValue: 'Price' }), dataIndex: 'bid', width: 100, render: (_: string, record: ActiveStrategy) => {
      const stale = secondsSince(record.lastTickAt) > 60;
      if (!record.bid && !record.ask) return <Text type="secondary">-</Text>;
      const spread = record.bid && record.ask ? ` / ${record.ask}` : '';
      return (
        <Space direction="vertical" size={0}>
          <Text style={{ fontSize: 12, fontVariantNumeric: 'tabular-nums' }} type={stale ? 'secondary' : undefined}>
            {record.bid}{spread}
          </Text>
          {stale && <Tag color="default" style={{ fontSize: 10 }}>stale</Tag>}
        </Space>
      );
    } },
    { title: t('strategy.live.lastSignal', { defaultValue: 'Last Signal' }), dataIndex: 'lastSignalAt', width: 110, render: (_: { seconds?: bigint; nanos?: number } | null, record: ActiveStrategy) => {
      const s = secondsSince(record.lastSignalAt);
      return <Text style={{ fontSize: 12 }} type={s > 300 ? 'warning' : undefined}>{formatAgo(record.lastSignalAt)}</Text>;
    } },
    { title: t('strategy.live.pnl', { defaultValue: 'PnL' }), dataIndex: 'pnl', width: 90, render: (_: string, record: ActiveStrategy) => {
      if (!record.pnl) return <Text type="secondary">-</Text>;
      const n = Number(record.pnl);
      const color = n >= 0 ? 'success' : 'danger';
      return <Text style={{ fontSize: 12 }} type={color}>{n >= 0 ? `+${record.pnl}` : record.pnl}</Text>;
    } },
    { title: t('strategy.live.timeframe', { defaultValue: 'TF' }), dataIndex: 'timeframe', width: 60 },
    { title: t('strategy.live.mode', { defaultValue: 'Mode' }), dataIndex: 'mode', width: 70, render: (v: string) => <Tag color={MODE_COLORS[v] || 'default'}>{v}</Tag> },
    { title: t('strategy.live.signals', { defaultValue: 'Signals' }), dataIndex: 'signalCount', width: 70, render: (v: number) => <Text strong>{v}</Text> },
    { title: t('strategy.live.errors', { defaultValue: 'Errors' }), dataIndex: 'errorCount', width: 60, render: (v: number, record: ActiveStrategy) => v > 0 ? <Tooltip title={record.lastError || t('strategy.live.unknownError', { defaultValue: 'Unknown error' })}><Tag color="red">{v}</Tag></Tooltip> : <Text type="secondary">0</Text> },
    { title: t('strategy.live.startedAt', { defaultValue: 'Started' }), dataIndex: 'startedAt', width: 140, render: (v: { seconds?: bigint; nanos?: number } | null) => <Text style={{ fontSize: 12 }}>{formatTime(v)}</Text> },
    {
      title: '', width: 180,
      render: (_: unknown, record: ActiveStrategy) => (
        <Space size="small">
          <Tooltip title={t('strategy.live.watchSignals', { defaultValue: 'Watch Signals' })}>
            <Button size="small" icon={<MonitorOutlined />} onClick={() => handleWatchSignals(record.runId)} />
          </Tooltip>
          <Tooltip title={t('strategy.live.logs', { defaultValue: 'Logs' })}>
            <Button size="small" icon={<FileTextOutlined />} disabled={isLogButtonDisabled(record.scheduleId)} onClick={() => setLogsModalScheduleId(record.scheduleId)} />
          </Tooltip>
          <Tooltip title={t('strategy.live.health', { defaultValue: 'Health' })}>
            <Button size="small" icon={<HeartOutlined />} disabled={isHealthButtonDisabled(record.scheduleId)} onClick={() => navigate(`/strategy/live?tab=schedules&healthId=${record.scheduleId}`)} />
          </Tooltip>
          <Popconfirm title={t('strategy.live.confirmStop', { defaultValue: 'Stop this strategy?' })} onConfirm={() => handleStop(record.runId)}>
            <Button size="small" danger icon={<StopOutlined />} loading={stopping === record.runId} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const runColumns = [
    { title: t('strategy.live.runId', { defaultValue: 'Run ID' }), dataIndex: 'id', width: 100, render: (v: string) => <Text code copyable>{shortId(v)}</Text> },
    { title: t('strategy.live.account', { defaultValue: 'Account' }), dataIndex: 'accountId', width: 120, render: (v: string) => <Text style={{ fontSize: 12 }}>{fmtAccount(v)}</Text> },
    { title: t('strategy.live.symbol', { defaultValue: 'Symbol' }), dataIndex: 'symbol', width: 80 },
    { title: t('strategy.live.timeframe', { defaultValue: 'TF' }), dataIndex: 'timeframe', width: 60 },
    { title: t('strategy.live.mode', { defaultValue: 'Mode' }), dataIndex: 'mode', width: 70, render: (v: string) => <Tag color={MODE_COLORS[v] || 'default'}>{v}</Tag> },
    { title: t('strategy.live.status', { defaultValue: 'Status' }), dataIndex: 'status', width: 90, render: (v: string) => <Tag color={STATUS_COLORS[v] || 'default'}>{v}</Tag> },
    { title: t('strategy.live.totalSignals', { defaultValue: 'Total Signals' }), dataIndex: 'totalSignals', width: 90, render: (v: number) => <Text strong>{v}</Text> },
    { title: t('strategy.live.startedAt', { defaultValue: 'Started' }), dataIndex: 'startedAt', width: 140, render: (v: { seconds?: bigint; nanos?: number } | null) => <Text style={{ fontSize: 12 }}>{formatTime(v)}</Text> },
    { title: t('strategy.live.stoppedAt', { defaultValue: 'Stopped' }), dataIndex: 'stoppedAt', width: 140, render: (v: { seconds?: bigint; nanos?: number } | null) => <Text style={{ fontSize: 12 }}>{formatTime(v)}</Text> },
    { title: t('strategy.live.error', { defaultValue: 'Error' }), dataIndex: 'error', ellipsis: true, render: (v: string) => v ? <Tooltip title={v}><Text type="danger" style={{ fontSize: 12 }}>{v}</Text></Tooltip> : <Text type="secondary">-</Text> },
  ];

  return (
    <div style={{ padding: '0 0 12px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
        <h2 style={{ margin: 0, fontSize: 18, fontWeight: 600 }}>{t('strategy.live.title', { defaultValue: 'Live Strategy Monitor' })}</h2>
        {activeTab === 'history' && (
          <Button icon={<ReloadOutlined />} onClick={fetchRuns} loading={loading}>
            {t('common.refresh', { defaultValue: 'Refresh' })}
          </Button>
        )}
      </div>

      {streamError && activeTab === 'active' && (
        <Alert
          type="warning"
          message={t('strategy.live.streamDisconnected', { defaultValue: 'Connection interrupted, reconnecting…' })}
          showIcon
          style={{ marginBottom: 12 }}
        />
      )}

      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: 'active',
            label: <span><ClockCircleOutlined /> {t('strategy.live.activeTab', { defaultValue: 'Active Runs' })}</span>,
            children: (
              <Card size="small">
                <Table
                  size="small"
                  dataSource={[...activeStrategies].sort((a, b) => a.runId.localeCompare(b.runId))}
                  rowKey="runId"
                  loading={loading}
                  columns={activeColumns}
                  pagination={false}
                  locale={{ emptyText: (
                    <Empty description={t('strategy.live.noActive', { defaultValue: 'No active strategies' })}>
                      <Button type="primary" onClick={() => setActiveTab('schedules')}>{t('strategy.live.goSchedules', { defaultValue: 'Go to Schedules' })}</Button>
                    </Empty>
                  ) }}
                />
              </Card>
            ),
          },
          {
            key: 'schedules',
            label: <span><ClockCircleOutlined /> {t('strategy.live.schedulesTab', { defaultValue: 'Schedules' })}</span>,
            children: <LiveSchedulesTab highlightScheduleId={highlightScheduleId} healthId={healthId} />,
          },
          {
            key: 'history',
            label: <span><EyeOutlined /> {t('strategy.live.historyTab', { defaultValue: 'Run History' })}</span>,
            children: (
              <Card size="small">
                <Table
                  size="small"
                  dataSource={runs}
                  rowKey="id"
                  loading={loading}
                  columns={runColumns}
                  pagination={{ pageSize: 20, showSizeChanger: false }}
                  locale={{ emptyText: <Empty description={t('strategy.live.noRuns', { defaultValue: 'No strategy runs' })} /> }}
                />
              </Card>
            ),
          },
        ]}
      />

      <SignalDrawer
        open={signalDrawerOpen}
        onClose={() => {
          if (abortRef.current) abortRef.current.abort();
          setSignalDrawerOpen(false);
        }}
        watchingRunId={watchingRunId}
        signals={signals}
        activeStrategies={activeStrategies}
      />
      <ScheduleLogsModal
        open={logsModalScheduleId !== null}
        scheduleId={logsModalScheduleId}
        onClose={() => setLogsModalScheduleId(null)}
      />
    </div>
  );
}
