import { useState, useEffect, useCallback, useRef } from 'react';
import { Table, Tag, Typography, Button, Card, Space, message, Popconfirm, Tabs, Empty, Tooltip } from 'antd';
import { ReloadOutlined, StopOutlined, EyeOutlined, MonitorOutlined, ClockCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { strategyActiveApi, strategyRunsApi } from '@/client/strategy';
import type { ActiveStrategy, StrategyRun, StrategySignalEvent } from '@/gen/ant/v1/strategy_runtime_pb';
import { SignalDrawer, formatTime, shortId, STATUS_COLORS, MODE_COLORS } from './LiveStrategyPageSignalDrawer';

const { Text } = Typography;

export default function LiveStrategyPage() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('active');
  const [activeStrategies, setActiveStrategies] = useState<ActiveStrategy[]>([]);
  const [runs, setRuns] = useState<StrategyRun[]>([]);
  const [loading, setLoading] = useState(false);
  const [stopping, setStopping] = useState<string | null>(null);
  const [signalDrawerOpen, setSignalDrawerOpen] = useState(false);
  const [watchingRunId, setWatchingRunId] = useState<string | null>(null);
  const [signals, setSignals] = useState<StrategySignalEvent[]>([]);
  const abortRef = useRef<AbortController | null>(null);

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
    const abort = new AbortController();
    setLoading(true);
    (async () => {
      try {
        for await (const event of strategyActiveApi.watchActive('', abort.signal)) {
          setActiveStrategies((event.strategies || []) as ActiveStrategy[]);
          setLoading(false);
        }
      } catch {
        setActiveStrategies([]);
        setLoading(false);
      }
    })();
    return () => abort.abort();
  }, [activeTab]);

  useEffect(() => {
    if (activeTab === 'history') fetchRuns();
  }, [activeTab, fetchRuns]);

  const handleStop = async (runId: string) => {
    setStopping(runId);
    try {
      const r = await strategyActiveApi.stop(runId);
      if (r.success) {
        message.success(t('strategy.live.stopSuccess', 'Strategy stopped'));
      } else {
        message.error(r.error || t('strategy.live.stopFailed', 'Failed to stop'));
      }
    } catch (e: any) {
      message.error(e?.message || t('strategy.live.stopFailed', 'Failed to stop'));
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
        for await (const event of strategyActiveApi.watchSignals(runId, abortRef.current.signal)) {
          setSignals(prev => [...prev.slice(-199), event as StrategySignalEvent]);
        }
      } catch (e: any) {
        if (e?.name !== 'AbortError') {
          // Stream ended (strategy stopped)
        }
      }
    })();
  }, []);

  useEffect(() => {
    return () => {
      if (abortRef.current) abortRef.current.abort();
    };
  }, []);

  const activeColumns = [
    {
      title: t('strategy.live.runId', 'Run ID'), dataIndex: 'runId', width: 100,
      render: (v: string) => <Text code copyable>{shortId(v)}</Text>,
    },
    {
      title: t('strategy.live.account', 'Account'), dataIndex: 'accountId', width: 120,
      render: (v: string) => <Text style={{ fontSize: 12 }}>{v}</Text>,
    },
    {
      title: t('strategy.live.symbol', 'Symbol'), dataIndex: 'symbol', width: 80,
    },
    {
      title: t('strategy.live.timeframe', 'TF'), dataIndex: 'timeframe', width: 60,
    },
    {
      title: t('strategy.live.mode', 'Mode'), dataIndex: 'mode', width: 70,
      render: (v: string) => <Tag color={MODE_COLORS[v] || 'default'}>{v}</Tag>,
    },
    {
      title: t('strategy.live.signals', 'Signals'), dataIndex: 'signalCount', width: 70,
      render: (v: number) => <Text strong>{v}</Text>,
    },
    {
      title: t('strategy.live.errors', 'Errors'), dataIndex: 'errorCount', width: 60,
      render: (v: number) => v > 0 ? <Tag color="red">{v}</Tag> : <Text type="secondary">0</Text>,
    },
    {
      title: t('strategy.live.startedAt', 'Started'), dataIndex: 'startedAt', width: 140,
      render: (v: any) => <Text style={{ fontSize: 12 }}>{formatTime(v)}</Text>,
    },
    {
      title: '', width: 120,
      render: (_: any, record: ActiveStrategy) => (
        <Space size="small">
          <Tooltip title={t('strategy.live.watchSignals', 'Watch Signals')}>
            <Button size="small" icon={<MonitorOutlined />} onClick={() => handleWatchSignals(record.runId)} />
          </Tooltip>
          <Popconfirm title={t('strategy.live.confirmStop', 'Stop this strategy?')} onConfirm={() => handleStop(record.runId)}>
            <Button size="small" danger icon={<StopOutlined />} loading={stopping === record.runId} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const runColumns = [
    {
      title: t('strategy.live.runId', 'Run ID'), dataIndex: 'id', width: 100,
      render: (v: string) => <Text code copyable>{shortId(v)}</Text>,
    },
    {
      title: t('strategy.live.account', 'Account'), dataIndex: 'accountId', width: 120,
      render: (v: string) => <Text style={{ fontSize: 12 }}>{v}</Text>,
    },
    {
      title: t('strategy.live.symbol', 'Symbol'), dataIndex: 'symbol', width: 80,
    },
    {
      title: t('strategy.live.timeframe', 'TF'), dataIndex: 'timeframe', width: 60,
    },
    {
      title: t('strategy.live.mode', 'Mode'), dataIndex: 'mode', width: 70,
      render: (v: string) => <Tag color={MODE_COLORS[v] || 'default'}>{v}</Tag>,
    },
    {
      title: t('strategy.live.status', 'Status'), dataIndex: 'status', width: 90,
      render: (v: string) => <Tag color={STATUS_COLORS[v] || 'default'}>{v}</Tag>,
    },
    {
      title: t('strategy.live.totalSignals', 'Total Signals'), dataIndex: 'totalSignals', width: 90,
      render: (v: number) => <Text strong>{v}</Text>,
    },
    {
      title: t('strategy.live.startedAt', 'Started'), dataIndex: 'startedAt', width: 140,
      render: (v: any) => <Text style={{ fontSize: 12 }}>{formatTime(v)}</Text>,
    },
    {
      title: t('strategy.live.stoppedAt', 'Stopped'), dataIndex: 'stoppedAt', width: 140,
      render: (v: any) => <Text style={{ fontSize: 12 }}>{formatTime(v)}</Text>,
    },
    {
      title: t('strategy.live.error', 'Error'), dataIndex: 'error', ellipsis: true,
      render: (v: string) => v ? <Tooltip title={v}><Text type="danger" style={{ fontSize: 12 }}>{v}</Text></Tooltip> : <Text type="secondary">-</Text>,
    },
  ];

  return (
    <div style={{ padding: '0 0 12px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
        <h2 style={{ margin: 0, fontSize: 18, fontWeight: 600 }}>{t('strategy.live.title', 'Live Strategy Monitor')}</h2>
        {activeTab === 'history' && (
          <Button icon={<ReloadOutlined />} onClick={fetchRuns} loading={loading}>
            {t('common.refresh', 'Refresh')}
          </Button>
        )}
      </div>

      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: 'active',
            label: <span><ClockCircleOutlined /> {t('strategy.live.activeTab', 'Active Runs')}</span>,
            children: (
              <Card size="small">
                <Table
                  size="small"
                  dataSource={activeStrategies}
                  rowKey="runId"
                  loading={loading}
                  columns={activeColumns}
                  pagination={false}
                  locale={{ emptyText: <Empty description={t('strategy.live.noActive', 'No active strategies')} /> }}
                />
              </Card>
            ),
          },
          {
            key: 'history',
            label: <span><EyeOutlined /> {t('strategy.live.historyTab', 'Run History')}</span>,
            children: (
              <Card size="small">
                <Table
                  size="small"
                  dataSource={runs}
                  rowKey="id"
                  loading={loading}
                  columns={runColumns}
                  pagination={{ pageSize: 20, showSizeChanger: false }}
                  locale={{ emptyText: <Empty description={t('strategy.live.noRuns', 'No strategy runs')} /> }}
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
