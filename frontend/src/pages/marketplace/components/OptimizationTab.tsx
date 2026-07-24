import { useState, useCallback, useEffect } from 'react';
import { Card, Table, Button, Tag, Typography, Space, Modal, Descriptions, Empty, message, Alert, Row, Col, Statistic, Input } from 'antd';
import { ExperimentOutlined, CheckOutlined, CloseOutlined, EyeOutlined, ThunderboltOutlined, WarningOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { create } from '@bufbuild/protobuf';
import {
  ListOptimizationTasksRequestSchema,
  RejectOptimizationTaskRequestSchema,
  PublishOptimizationRequestSchema,
  PreviewOptimizationRequestSchema,
  DetectStrategyDecayRequestSchema,
} from '@/gen/ant/v1/marketplace_service_pb';
import type { OptimizationTaskInfo, PreviewOptimizationResponse, BacktestSnapshot } from '@/gen/ant/v1/marketplace_service_pb';

const { Text, Paragraph } = Typography;

const STATUS_COLORS: Record<string, string> = {
  pending: 'default',
  generating: 'processing',
  completed: 'gold',
  rejected: 'red',
  published: 'green',
};

function BacktestCompare({ original, optimized }: { original?: BacktestSnapshot; optimized?: BacktestSnapshot }) {
  const { t } = useTranslation();
  if (!original && !optimized) return <Empty description={t('marketplace.optimization.noBacktest')} />;
  const metrics: [string, string | undefined, string | undefined][] = [
    [t('marketplace.optimization.totalReturn'), original?.totalReturn, optimized?.totalReturn],
    [t('marketplace.optimization.annualReturn'), original?.annualReturn, optimized?.annualReturn],
    [t('marketplace.optimization.maxDrawdown'), original?.maxDrawdown, optimized?.maxDrawdown],
    [t('marketplace.optimization.sharpe'), original?.sharpeRatio, optimized?.sharpeRatio],
    [t('marketplace.optimization.winRate'), original?.winRate, optimized?.winRate],
    [t('marketplace.optimization.totalTrades'), original?.totalTrades?.toString(), optimized?.totalTrades?.toString()],
  ];
  return (
    <Row gutter={[8, 8]}>
      {metrics.map(([label, orig, opt]) => {
        const improved = orig && opt && Number(opt) > Number(orig);
        return (
          <Col xs={24} sm={12} md={8} key={label}>
            <Card size="small" style={{ borderRadius: 8 }}>
              <Statistic
                title={label}
                value={opt || '-'}
                prefix={improved ? <CheckOutlined style={{ color: '#52c41a' }} /> : undefined}
              />
              <Text type="secondary" style={{ fontSize: 12 }}>
                {t('marketplace.optimization.original')}: {orig || '-'}
              </Text>
            </Card>
          </Col>
        );
      })}
    </Row>
  );
}

export default function OptimizationTab() {
  const { t } = useTranslation();
  const [tasks, setTasks] = useState<OptimizationTaskInfo[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [statusFilter, setStatusFilter] = useState('');
  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewData, setPreviewData] = useState<PreviewOptimizationResponse | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [decayResult, setDecayResult] = useState<{ strategyId: string; isDecaying: boolean; score: number } | null>(null);
  const [decayStrategyId, setDecayStrategyId] = useState('');
  const [decayLoading, setDecayLoading] = useState(false);

  const fetchTasks = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await marketplaceClient.listOptimizationTasks(create(ListOptimizationTasksRequestSchema, {
        status: statusFilter, limit: 50, offset: 0,
      }));
      setTasks(resp.tasks || []);
      setTotal(resp.total || 0);
    } catch {
      setTasks([]);
      setTotal(0);
    }
    setLoading(false);
  }, [statusFilter]);

  useEffect(() => { fetchTasks(); }, [fetchTasks]);

  const handlePreview = useCallback(async (taskId: string) => {
    setPreviewLoading(true);
    setPreviewOpen(true);
    try {
      const resp = await marketplaceClient.previewOptimization(create(PreviewOptimizationRequestSchema, { taskId }));
      setPreviewData(resp);
    } catch (e: any) {
      message.error(e?.message || t('marketplace.optimization.previewFailed'));
      setPreviewData(null);
    } finally {
      setPreviewLoading(false);
    }
  }, [t]);

  const handleReject = useCallback(async (taskId: string) => {
    setActionLoading(taskId);
    try {
      await marketplaceClient.rejectOptimizationTask(create(RejectOptimizationTaskRequestSchema, { taskId }));
      message.success(t('marketplace.optimization.rejected'));
      fetchTasks();
    } catch (e: any) {
      message.error(e?.message || t('marketplace.optimization.rejectFailed'));
    } finally {
      setActionLoading(null);
    }
  }, [t, fetchTasks]);

  const handlePublish = useCallback(async (taskId: string) => {
    setActionLoading(taskId);
    try {
      const resp = await marketplaceClient.publishOptimization(create(PublishOptimizationRequestSchema, { taskId }));
      message.success(t('marketplace.optimization.published', `已发布新版本 v${resp.versionId}`));
      fetchTasks();
    } catch (e: any) {
      message.error(e?.message || t('marketplace.optimization.publishFailed'));
    } finally {
      setActionLoading(null);
    }
  }, [t, fetchTasks]);

  const handleDetectDecay = useCallback(async () => {
    if (!decayStrategyId.trim()) return;
    setDecayLoading(true);
    try {
      const resp = await marketplaceClient.detectStrategyDecay(create(DetectStrategyDecayRequestSchema, {
        strategyId: decayStrategyId.trim(),
      }));
      setDecayResult({
        strategyId: decayStrategyId.trim(),
        isDecaying: resp.isDecaying,
        score: resp.decayScore,
      });
      if (resp.isDecaying) {
        message.warning(t('marketplace.optimization.decayDetected'));
      } else {
        message.success(t('marketplace.optimization.noDecay'));
      }
    } catch (e: any) {
      message.error(e?.message || t('marketplace.optimization.detectFailed'));
    } finally {
      setDecayLoading(false);
    }
  }, [decayStrategyId, t]);

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Text strong style={{ fontSize: 15 }}>
          <ThunderboltOutlined style={{ marginRight: 8 }} />
          {t('marketplace.optimization.title')}
        </Text>
      </div>

      {/* ── Decay Detection ── */}
      <Card size="small" style={{ marginBottom: 16 }} title={t('marketplace.optimization.decayDetection')}>
        <Space>
          <Input
            placeholder={t('marketplace.optimization.strategyIdPlaceholder')}
            value={decayStrategyId}
            onChange={e => setDecayStrategyId(e.target.value)}
            style={{ width: 360 }}
          />
          <Button
            type="primary"
            loading={decayLoading}
            onClick={handleDetectDecay}
            icon={<ExperimentOutlined />}
          >
            {t('marketplace.optimization.detect')}
          </Button>
        </Space>
        {decayResult && (
          <Alert
            style={{ marginTop: 12 }}
            type={decayResult.isDecaying ? 'warning' : 'success'}
            showIcon
            icon={decayResult.isDecaying ? <WarningOutlined /> : <CheckOutlined />}
            message={decayResult.isDecaying
              ? t('marketplace.optimization.decayDetected')
              : t('marketplace.optimization.noDecay')}
            description={decayResult.isDecaying
              ? `${t('marketplace.optimization.decayScore')}: ${decayResult.score}/3 — ${t('marketplace.optimization.decayHint')}`
              : `${t('marketplace.optimization.decayScore')}: ${decayResult.score}/3`}
          />
        )}
      </Card>

      {/* ── Optimization Tasks ── */}
      <Card size="small" title={t('marketplace.optimization.tasks')}>
        <Space style={{ marginBottom: 12 }}>
          {['', 'pending', 'completed', 'rejected', 'published'].map(s => (
            <Button
              key={s}
              size="small"
              type={statusFilter === s ? 'primary' : 'default'}
              onClick={() => setStatusFilter(s)}
            >
              {s === '' ? t('marketplace.optimization.all') : s}
            </Button>
          ))}
        </Space>
        {tasks.length === 0 && !loading ? (
          <Empty description={t('marketplace.optimization.noTasks')} />
        ) : (
          <Table<OptimizationTaskInfo>
            rowKey="id"
            dataSource={tasks}
            loading={loading}
            pagination={{ pageSize: 10, total }}
            size="small"
            columns={[
              { title: t('marketplace.optimization.strategyId'), dataIndex: 'strategyId', key: 'sid', width: 120, render: (v: string) => <Text copyable style={{ fontSize: 12 }}>{v?.slice(0, 8)}...</Text> },
              { title: t('marketplace.optimization.status'), dataIndex: 'status', key: 'status', width: 100, render: (s: string) => <Tag color={STATUS_COLORS[s] || 'default'}>{s}</Tag> },
              { title: t('marketplace.optimization.trigger'), dataIndex: 'triggerReason', key: 'trigger', width: 120, ellipsis: true },
              { title: t('marketplace.optimization.changeSummary'), dataIndex: 'changeSummary', key: 'summary', ellipsis: true },
              {
                title: t('marketplace.optimization.actions'), key: 'actions', width: 240,
                render: (_: unknown, row: OptimizationTaskInfo) => (
                  <Space>
                    {row.status === 'completed' && (
                      <>
                        <Button size="small" icon={<EyeOutlined />} onClick={() => handlePreview(row.id)}>
                          {t('marketplace.optimization.preview')}
                        </Button>
                        <Button size="small" type="primary" icon={<CheckOutlined />}
                          loading={actionLoading === row.id}
                          onClick={() => handlePublish(row.id)}>
                          {t('marketplace.optimization.publish')}
                        </Button>
                        <Button size="small" danger icon={<CloseOutlined />}
                          loading={actionLoading === row.id}
                          onClick={() => handleReject(row.id)}>
                          {t('marketplace.optimization.reject')}
                        </Button>
                      </>
                    )}
                    {row.status === 'published' && (
                      <Tag color="green">{t('marketplace.optimization.versionId')}: {row.publishedVersionId?.slice(0, 8)}</Tag>
                    )}
                  </Space>
                ),
              },
            ]}
          />
        )}
      </Card>

      {/* ── Preview Modal ── */}
      <Modal
        title={t('marketplace.optimization.previewTitle')}
        open={previewOpen}
        onCancel={() => { setPreviewOpen(false); setPreviewData(null); }}
        footer={null}
        width={900}
      >
        {previewLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}>{t('common.loading')}</div>
        ) : previewData ? (
          <div>
            <Descriptions size="small" column={2} style={{ marginBottom: 16 }}>
              <Descriptions.Item label={t('marketplace.optimization.trigger')}>{previewData.task?.triggerReason}</Descriptions.Item>
              <Descriptions.Item label={t('marketplace.optimization.status')}><Tag color={STATUS_COLORS[previewData.task?.status || '']}>{previewData.task?.status}</Tag></Descriptions.Item>
              <Descriptions.Item label={t('marketplace.optimization.changeSummary')} span={2}>{previewData.changeSummary}</Descriptions.Item>
            </Descriptions>

            {previewData.decayMetrics && (
              <Alert
                type="warning"
                style={{ marginBottom: 16 }}
                message={t('marketplace.optimization.decayMetrics')}
                description={
                  <Descriptions size="small" column={2} style={{ marginTop: 8 }}>
                    <Descriptions.Item label={t('marketplace.optimization.decayScore', { defaultValue: 'Decay Score' })}>{previewData.decayMetrics.decayScore}</Descriptions.Item>
                    <Descriptions.Item label={t('marketplace.optimization.trigger', { defaultValue: 'Trigger' })}>{previewData.decayMetrics.triggerReason}</Descriptions.Item>
                    <Descriptions.Item label={t('marketplace.optimization.sharpeDecline', { defaultValue: 'Sharpe Decline' })}>{previewData.decayMetrics.sharpeDeclinePct}</Descriptions.Item>
                    <Descriptions.Item label={t('marketplace.optimization.winRateDecline', { defaultValue: 'Win Rate Decline' })}>{previewData.decayMetrics.winrateDeclinePct}</Descriptions.Item>
                    <Descriptions.Item label={t('marketplace.optimization.returnDelta', { defaultValue: 'Return Delta' })}>{previewData.decayMetrics.returnDelta}</Descriptions.Item>
                  </Descriptions>
                }
              />
            )}

            <Text strong style={{ display: 'block', marginBottom: 8 }}>
              {t('marketplace.optimization.backtestCompare')}
            </Text>
            <BacktestCompare original={previewData.originalBacktest} optimized={previewData.optimizedBacktest} />

            {previewData.suggestedCodePreview && (
              <Card size="small" style={{ marginTop: 16 }} title={t('marketplace.optimization.codePreview')}>
                <pre style={{ maxHeight: 300, overflow: 'auto', fontSize: 12, background: '#f5f5f5', padding: 12, borderRadius: 8 }}>
                  {previewData.suggestedCodePreview}
                </pre>
              </Card>
            )}
          </div>
        ) : (
          <Empty />
        )}
      </Modal>
    </div>
  );
}
