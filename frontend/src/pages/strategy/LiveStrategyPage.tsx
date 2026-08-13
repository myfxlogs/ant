import { useState, useEffect, useCallback, useRef } from 'react';
import { Table, Tag, Typography, Button, Card, Space, message, Popconfirm, Tabs, Empty, Tooltip, Alert } from 'antd';
import { ReloadOutlined, StopOutlined, EyeOutlined, MonitorOutlined, ClockCircleOutlined, FileTextOutlined, HeartOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { strategyActiveApi, strategyRunsApi } from '@/client/strategy';
import type { ActiveStrategy, StrategyRun, StrategySignalEvent } from '@/gen/ant/v1/strategy_runtime_pb';
import { SignalDrawer, formatTime, shortId, STATUS_COLORS, MODE_COLORS } from './LiveStrategyPageSignalDrawer';
import LiveSchedulesTab from './components/workspace/LiveSchedulesTab';

const { Text } = Typography;

export function isLogButtonDisabled(scheduleId: string): boolean { return !scheduleId; }

export function isHealthButtonDisabled(scheduleId: string): boolean { return !scheduleId; }

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
    { title: t('strategy.live.account', { defaultValue: 'Account' }), dataIndex: 'accountId', width: 120, render: (v: string) => <Text style={{ fontSize: 12 }}>{v}</Text> },
    { title: t('strategy.live.symbol', { defaultValue: 'Symbol' }), dataIndex: 'symbol', width: 80 },
    { title: t('strategy.live.timeframe', { defaultValue: 'TF' }), dataIndex: 'timeframe', width: 60 },
    { title: t('strategy.live.mode', { defaultValue: 'Mode' }), dataIndex: 'mode', width: 70, render: (v: string) => <Tag color={MODE_COLORS[v] || 'default'}>{v}</Tag> },
    { title: t('strategy.live.signals', { defaultValue: 'Signals' }), dataIndex: 'signalCount', width: 70, render: (v: number) => <Text strong>{v}</Text> },
    { title: t('strategy.live.errors', { defaultValue: 'Errors' }), dataIndex: 'errorCount', width: 60, render: (v: number) => v > 0 ? <Tag color="red">{v}</Tag> : <Text type="secondary">0</Text> },
    { title: t('strategy.live.startedAt', { defaultValue: 'Started' }), dataIndex: 'startedAt', width: 140, render: (v: { seconds?: bigint; nanos?: number } | null) => <Text style={{ fontSize: 12 }}>{formatTime(v)}</Text> },
    {
      title: '', width: 180,
      render: (_: unknown, record: ActiveStrategy) => (
        <Space size="small">
          <Tooltip title={t('strategy.live.watchSignals', { defaultValue: 'Watch Signals' })}>
            <Button size="small" icon={<MonitorOutlined />} onClick={() => handleWatchSignals(record.runId)} />
          </Tooltip>
          <Tooltip title={t('strategy.live.logs', { defaultValue: 'Logs' })}>
            <Button size="small" icon={<FileTextOutlined />} disabled={isLogButtonDisabled(record.scheduleId)} onClick={() => navigate(`/strategy/schedules/${record.scheduleId}/logs`)} />
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
    { title: t('strategy.live.account', { defaultValue: 'Account' }), dataIndex: 'accountId', width: 120, render: (v: string) => <Text style={{ fontSize: 12 }}>{v}</Text> },
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
                  dataSource={activeStrategies}
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
    </div>
  );
}
