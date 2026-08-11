import { useEffect, useState, useRef } from 'react';
import { Card, Col, Row, Statistic, Tag, Progress, Space, Alert, Descriptions } from 'antd';
import { MonitorOutlined, DatabaseOutlined, CloudServerOutlined, ApiOutlined, ThunderboltOutlined, WarningOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { TFunction } from 'i18next';
import { adminMonitorStreamClient } from '@/client/connect';
import { create } from '@bufbuild/protobuf';
import { SubscribeMetricsRequestSchema } from '@/gen/ant/v1/admin_monitor_pb';
import type { MonitorSnapshot } from '@/gen/ant/v1/admin_monitor_pb';

function formatBytes(bytes: bigint | number): string {
  const b = Number(bytes);
  if (b >= 1 << 30) return (b / (1 << 30)).toFixed(2) + ' GB';
  if (b >= 1 << 20) return (b / (1 << 20)).toFixed(2) + ' MB';
  if (b >= 1 << 10) return (b / (1 << 10)).toFixed(2) + ' KB';
  return b + ' B';
}

function formatUptime(seconds: number, t: TFunction): string {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  if (d > 0) return t('monitoring.uptimeDays', { d, h, defaultValue: `${d}d ${h}h` });
  if (h > 0) return t('monitoring.uptimeHours', { h, m, defaultValue: `${h}h ${m}m` });
  if (m > 0) return t('monitoring.uptimeMinutes', { m, s, defaultValue: `${m}m ${s}s` });
  return t('monitoring.uptimeSeconds', { s, defaultValue: `${s}s` });
}

function formatNs(ns: number): string {
  if (ns >= 1e6) return (ns / 1e6).toFixed(2) + ' ms';
  if (ns >= 1e3) return (ns / 1e3).toFixed(2) + ' µs';
  return ns.toFixed(0) + ' ns';
}

function StatusTag({ status, t }: { status: string; t: TFunction }) {
  if (!status) return <Tag>{t('monitoring.unknown', { defaultValue: 'Unknown' })}</Tag>;
  const isOk = status === 'ok';
  return (
    <Tag color={isOk ? 'green' : 'red'} icon={!isOk ? <WarningOutlined /> : undefined}>
      {isOk ? t('monitoring.healthy', { defaultValue: 'OK' }) : status}
    </Tag>
  );
}

export default function MonitoringPage() {
  const { t } = useTranslation();
  const [snapshot, setSnapshot] = useState<MonitorSnapshot | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [connected, setConnected] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    const ac = new AbortController();
    abortRef.current = ac;
    setError(null);

    (async () => {
      try {
        const stream = adminMonitorStreamClient.subscribeMetrics(
          create(SubscribeMetricsRequestSchema, {}),
          { signal: ac.signal },
        );
        setConnected(true);
        for await (const snap of stream) {
          setSnapshot(snap);
        }
      } catch (err: unknown) {
        if (ac.signal.aborted) return;
        setError(err instanceof Error ? err.message : String(err));
        setConnected(false);
      }
    })();

    return () => {
      ac.abort();
      abortRef.current = null;
    };
  }, []);

  const snap = snapshot;
  const heapAlloc = snap ? Number(snap.heapAllocBytes) : 0;
  const heapSys = snap ? Number(snap.heapSysBytes) : 0;
  const heapUsagePct = heapSys > 0 ? Math.round((heapAlloc / heapSys) * 100) : 0;

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <MonitorOutlined style={{ fontSize: 24, color: '#D4AF37' }} />
        <h1 className="text-2xl font-bold" style={{ color: 'var(--color-text)' }}>
          {t('monitoring.title', { defaultValue: 'System Monitoring' })}
        </h1>
        <Tag color={connected ? 'green' : 'red'} style={{ marginLeft: 8 }}>
          {connected ? t('monitoring.sseConnected', { defaultValue: 'SSE Connected' }) : t('monitoring.disconnected', { defaultValue: 'Disconnected' })}
        </Tag>
      </div>

      {error && (
        <Alert type="error" message={t('monitoring.streamError', { defaultValue: 'Stream Error' })} description={error} showIcon closable onClose={() => setError(null)} />
      )}

      {!snap && !error && (
        <Card><div className="text-center py-8" style={{ color: 'var(--color-text-secondary)' }}>{t('monitoring.waitingData', { defaultValue: 'Waiting for data...' })}</div></Card>
      )}

      {snap && (
        <>
          <Card title={<span><CloudServerOutlined /> {t('monitoring.serviceHealth', { defaultValue: 'Service Health' })}</span>} size="small">
            <Descriptions column={4} size="small" bordered>
              <Descriptions.Item label={t('monitoring.uptime', { defaultValue: 'Uptime' })}>{formatUptime(Number(snap.uptimeSeconds), t)}</Descriptions.Item>
              <Descriptions.Item label={t('monitoring.database', { defaultValue: 'Database' })}><StatusTag status={snap.dbStatus} t={t} /></Descriptions.Item>
              <Descriptions.Item label="Redis"><StatusTag status={snap.redisStatus} t={t} /></Descriptions.Item>
              <Descriptions.Item label="NATS"><StatusTag status={snap.natsStatus} t={t} /></Descriptions.Item>
            </Descriptions>
            <div className="mt-3">
              <div className="mb-1 flex items-center justify-between" style={{ color: 'var(--color-text-secondary)', fontSize: 12 }}>
                <span>{t('monitoring.diskUsage', { defaultValue: 'Disk Usage' })}</span>
                <span>{formatBytes(snap.diskUsedBytes)} / {formatBytes(snap.diskTotalBytes)} ({snap.diskUsagePct.toFixed(1)}%)</span>
              </div>
              <Progress
                percent={Math.round(snap.diskUsagePct)}
                status={snap.diskUsagePct > 90 ? 'exception' : snap.diskUsagePct > 75 ? 'active' : 'normal'}
                size="small"
              />
            </div>
          </Card>

          <Card title={<span><ThunderboltOutlined /> {t('monitoring.goRuntime', { defaultValue: 'Go Runtime' })}</span>} size="small">
            <Row gutter={[16, 12]}>
              <Col xs={12} sm={8} md={6}>
                <Statistic title={t('monitoring.goroutines', { defaultValue: 'Goroutines' })} value={Number(snap.goroutines)} />
              </Col>
              <Col xs={12} sm={8} md={6}>
                <Statistic title={t('monitoring.gcCount', { defaultValue: 'GC Count' })} value={Number(snap.numGc)} />
              </Col>
              <Col xs={12} sm={8} md={6}>
                <Statistic title={t('monitoring.gcPauseAvg', { defaultValue: 'GC Pause Avg' })} value={formatNs(snap.gcPauseAvgNs)} />
              </Col>
              <Col xs={12} sm={8} md={6}>
                <Statistic title={t('monitoring.stackUsage', { defaultValue: 'Stack Usage' })} value={formatBytes(snap.stackInuseBytes)} />
              </Col>
              <Col xs={24} md={12}>
                <div className="mb-1" style={{ color: 'var(--color-text-secondary)', fontSize: 12 }}>{t('monitoring.heapMemory', { defaultValue: 'Heap Memory' })}</div>
                <Space>
                  <span>{formatBytes(heapAlloc)} / {formatBytes(heapSys)}</span>
                  <Progress percent={heapUsagePct} size="small" style={{ width: 120 }} />
                </Space>
              </Col>
            </Row>
          </Card>

          <Card title={<span><DatabaseOutlined /> {t('monitoring.dbPool', { defaultValue: 'DB Connection Pool' })}</span>} size="small">
            <Row gutter={[16, 12]}>
              <Col xs={8}>
                <Statistic title={t('monitoring.totalConns', { defaultValue: 'Total' })} value={Number(snap.dbPoolTotal)} />
              </Col>
              <Col xs={8}>
                <Statistic title={t('monitoring.idle', { defaultValue: 'Idle' })} value={Number(snap.dbPoolIdle)} valueStyle={{ color: '#52c41a' }} />
              </Col>
              <Col xs={8}>
                <Statistic title={t('monitoring.acquired', { defaultValue: 'Acquired' })} value={Number(snap.dbPoolAcquired)} valueStyle={{ color: Number(snap.dbPoolAcquired) > 20 ? '#ff4d4f' : '#1677ff' }} />
              </Col>
            </Row>
          </Card>

          <Card title={<span><ApiOutlined /> {t('monitoring.mdGateway', { defaultValue: 'MD Gateway' })}</span>} size="small">
            <Row gutter={[16, 12]}>
              <Col xs={12} sm={6}>
                <Statistic title={t('monitoring.spillFiles', { defaultValue: 'Spill Files' })} value={Number(snap.spillPendingFiles)} valueStyle={{ color: Number(snap.spillPendingFiles) > 0 ? '#ff4d4f' : undefined }} />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic title={t('monitoring.droppedBars', { defaultValue: 'Dropped Bars' })} value={Number(snap.barDroppedTotal)} valueStyle={{ color: Number(snap.barDroppedTotal) > 0 ? '#faad14' : undefined }} />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic title={t('monitoring.droppedSignals', { defaultValue: 'Dropped Signals' })} value={Number(snap.signalDroppedTotal)} valueStyle={{ color: Number(snap.signalDroppedTotal) > 0 ? '#faad14' : undefined }} />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic title={t('monitoring.consumerLag', { defaultValue: 'Consumer Lag' })} value={Number(snap.consumerLag)} />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic title={t('monitoring.staleAccounts', { defaultValue: 'Stale Accounts' })} value={Number(snap.staleAccounts)} />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic title={t('monitoring.deadAccounts', { defaultValue: 'Dead Accounts' })} value={Number(snap.deadAccounts)} valueStyle={{ color: Number(snap.deadAccounts) > 0 ? '#ff4d4f' : undefined }} />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic title={t('monitoring.avgGapSec', { defaultValue: 'Avg Gap (s)' })} value={snap.mdGapAvgSeconds.toFixed(2)} />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic title={t('monitoring.maxGapSec', { defaultValue: 'Max Gap (s)' })} value={snap.mdGapMaxSeconds.toFixed(2)} />
              </Col>
            </Row>
          </Card>

          <Card title={<span><WarningOutlined /> {t('monitoring.dlq', { defaultValue: 'Dead Letter Queue (DLQ)' })}</span>} size="small">
            <Row gutter={[16, 12]}>
              <Col xs={8}>
                <Statistic title={t('monitoring.parseErrors', { defaultValue: 'Parse Errors' })} value={Number(snap.dlqParseErrors)} />
              </Col>
              <Col xs={8}>
                <Statistic title={t('monitoring.bidGtAsk', { defaultValue: 'Bid>Ask' })} value={Number(snap.dlqBidGtAsk)} />
              </Col>
              <Col xs={8}>
                <Statistic title={t('monitoring.nonPositive', { defaultValue: 'Non-Positive' })} value={Number(snap.dlqNonPositive)} />
              </Col>
            </Row>
          </Card>

          <div className="text-center" style={{ color: 'var(--color-text-tertiary)', fontSize: 12 }}>
            {t('monitoring.pushInterval', { defaultValue: 'Push interval: 5s' })} · {t('monitoring.lastUpdate', { defaultValue: 'Last update' })}: {snap.timestamp ? new Date(Number(snap.timestamp.seconds) * 1000 + Math.floor(Number(snap.timestamp.nanos) / 1e6)).toLocaleTimeString() : '-'}
          </div>
        </>
      )}
    </div>
  );
}
