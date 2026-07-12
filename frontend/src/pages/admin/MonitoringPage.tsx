import { useEffect, useState, useRef } from 'react';
import { Card, Col, Row, Statistic, Tag, Progress, Space, Alert, Descriptions } from 'antd';
import { MonitorOutlined, DatabaseOutlined, CloudServerOutlined, ApiOutlined, ThunderboltOutlined, WarningOutlined } from '@ant-design/icons';
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

function formatUptime(seconds: number): string {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  if (d > 0) return `${d}天 ${h}小时`;
  if (h > 0) return `${h}小时 ${m}分`;
  if (m > 0) return `${m}分 ${s}秒`;
  return `${s}秒`;
}

function formatNs(ns: number): string {
  if (ns >= 1e6) return (ns / 1e6).toFixed(2) + ' ms';
  if (ns >= 1e3) return (ns / 1e3).toFixed(2) + ' µs';
  return ns.toFixed(0) + ' ns';
}

function StatusTag({ status }: { status: string }) {
  if (!status) return <Tag>未知</Tag>;
  const isOk = status === 'ok';
  return (
    <Tag color={isOk ? 'green' : 'red'} icon={!isOk ? <WarningOutlined /> : undefined}>
      {isOk ? '正常' : status}
    </Tag>
  );
}

export default function MonitoringPage() {
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
      } catch (err: any) {
        if (ac.signal.aborted) return;
        setError(String(err?.message ?? err));
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
          系统监控
        </h1>
        <Tag color={connected ? 'green' : 'red'} style={{ marginLeft: 8 }}>
          {connected ? 'SSE 已连接' : '未连接'}
        </Tag>
      </div>

      {error && (
        <Alert type="error" message="监控流错误" description={error} showIcon closable onClose={() => setError(null)} />
      )}

      {!snap && !error && (
        <Card><div className="text-center py-8" style={{ color: 'var(--color-text-secondary)' }}>等待数据推送...</div></Card>
      )}

      {snap && (
        <>
          <Card title={<span><CloudServerOutlined /> 服务健康</span>} size="small">
            <Descriptions column={4} size="small" bordered>
              <Descriptions.Item label="运行时间">{formatUptime(Number(snap.uptimeSeconds))}</Descriptions.Item>
              <Descriptions.Item label="数据库"><StatusTag status={snap.dbStatus} /></Descriptions.Item>
              <Descriptions.Item label="Redis"><StatusTag status={snap.redisStatus} /></Descriptions.Item>
              <Descriptions.Item label="NATS"><StatusTag status={snap.natsStatus} /></Descriptions.Item>
            </Descriptions>
          </Card>

          <Card title={<span><ThunderboltOutlined /> Go 运行时</span>} size="small">
            <Row gutter={[16, 12]}>
              <Col xs={12} sm={8} md={6}>
                <Statistic title="Goroutines" value={Number(snap.goroutines)} />
              </Col>
              <Col xs={12} sm={8} md={6}>
                <Statistic title="GC 次数" value={Number(snap.numGc)} />
              </Col>
              <Col xs={12} sm={8} md={6}>
                <Statistic title="GC 平均暂停" value={formatNs(snap.gcPauseAvgNs)} />
              </Col>
              <Col xs={12} sm={8} md={6}>
                <Statistic title="栈使用" value={formatBytes(snap.stackInuseBytes)} />
              </Col>
              <Col xs={24} md={12}>
                <div className="mb-1" style={{ color: 'var(--color-text-secondary)', fontSize: 12 }}>堆内存</div>
                <Space>
                  <span>{formatBytes(heapAlloc)} / {formatBytes(heapSys)}</span>
                  <Progress percent={heapUsagePct} size="small" style={{ width: 120 }} />
                </Space>
              </Col>
            </Row>
          </Card>

          <Card title={<span><DatabaseOutlined /> 数据库连接池</span>} size="small">
            <Row gutter={[16, 12]}>
              <Col xs={8}>
                <Statistic title="总连接" value={Number(snap.dbPoolTotal)} />
              </Col>
              <Col xs={8}>
                <Statistic title="空闲" value={Number(snap.dbPoolIdle)} valueStyle={{ color: '#52c41a' }} />
              </Col>
              <Col xs={8}>
                <Statistic title="已获取" value={Number(snap.dbPoolAcquired)} valueStyle={{ color: Number(snap.dbPoolAcquired) > 20 ? '#ff4d4f' : '#1677ff' }} />
              </Col>
            </Row>
          </Card>

          <Card title={<span><ApiOutlined /> 行情网关</span>} size="small">
            <Row gutter={[16, 12]}>
              <Col xs={12} sm={6}>
                <Statistic title="堆积文件" value={Number(snap.spillPendingFiles)} valueStyle={{ color: Number(snap.spillPendingFiles) > 0 ? '#ff4d4f' : undefined }} />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic title="丢弃 Bar" value={Number(snap.barDroppedTotal)} valueStyle={{ color: Number(snap.barDroppedTotal) > 0 ? '#faad14' : undefined }} />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic title="丢弃信号" value={Number(snap.signalDroppedTotal)} valueStyle={{ color: Number(snap.signalDroppedTotal) > 0 ? '#faad14' : undefined }} />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic title="消费者延迟" value={Number(snap.consumerLag)} />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic title="过期账户" value={Number(snap.staleAccounts)} />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic title="死账户" value={Number(snap.deadAccounts)} valueStyle={{ color: Number(snap.deadAccounts) > 0 ? '#ff4d4f' : undefined }} />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic title="平均间隔(s)" value={snap.mdGapAvgSeconds.toFixed(2)} />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic title="最大间隔(s)" value={snap.mdGapMaxSeconds.toFixed(2)} />
              </Col>
            </Row>
          </Card>

          <Card title={<span><WarningOutlined /> 死信队列 (DLQ)</span>} size="small">
            <Row gutter={[16, 12]}>
              <Col xs={8}>
                <Statistic title="解析错误" value={Number(snap.dlqParseErrors)} />
              </Col>
              <Col xs={8}>
                <Statistic title="Bid>Ask" value={Number(snap.dlqBidGtAsk)} />
              </Col>
              <Col xs={8}>
                <Statistic title="非正数" value={Number(snap.dlqNonPositive)} />
              </Col>
            </Row>
          </Card>

          <div className="text-center" style={{ color: 'var(--color-text-tertiary)', fontSize: 12 }}>
            数据推送间隔: 5 秒 · 最后更新: {snap.timestamp ? new Date(snap.timestamp.toDate()).toLocaleTimeString('zh-CN') : '-'}
          </div>
        </>
      )}
    </div>
  );
}
