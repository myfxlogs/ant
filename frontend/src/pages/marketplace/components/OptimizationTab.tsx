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
  if (!original && !optimized) return <Empty description={t('marketplace.optimization.noBacktest', '无回测数据')} />;
  const metrics: [string, string | undefined, string | undefined][] = [
    [t('marketplace.optimization.totalReturn', '总收益'), original?.totalReturn, optimized?.totalReturn],
    [t('marketplace.optimization.annualReturn', '年化收益'), original?.annualReturn, optimized?.annualReturn],
    [t('marketplace.optimization.maxDrawdown', '最大回撤'), original?.maxDrawdown, optimized?.maxDrawdown],
    [t('marketplace.optimization.sharpe', '夏普比率'), original?.sharpeRatio, optimized?.sharpeRatio],
    [t('marketplace.optimization.winRate', '胜率'), original?.winRate, optimized?.winRate],
    [t('marketplace.optimization.totalTrades', '总交易数'), original?.totalTrades?.toString(), optimized?.totalTrades?.toString()],
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
                {t('marketplace.optimization.original', '原版')}: {orig || '-'}
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
      message.error(e?.message || t('marketplace.optimization.previewFailed', '预览失败'));
      setPreviewData(null);
    } finally {
      setPreviewLoading(false);
    }
  }, [t]);

  const handleReject = useCallback(async (taskId: string) => {
    setActionLoading(taskId);
    try {
      await marketplaceClient.rejectOptimizationTask(create(RejectOptimizationTaskRequestSchema, { taskId }));
      message.success(t('marketplace.optimization.rejected', '已拒绝'));
      fetchTasks();
    } catch (e: any) {
      message.error(e?.message || t('marketplace.optimization.rejectFailed', '操作失败'));
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
      message.error(e?.message || t('marketplace.optimization.publishFailed', '发布失败'));
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
        message.warning(t('marketplace.optimization.decayDetected', '检测到策略衰减！'));
      } else {
        message.success(t('marketplace.optimization.noDecay', '策略表现正常'));
      }
    } catch (e: any) {
      message.error(e?.message || t('marketplace.optimization.detectFailed', '检测失败'));
    } finally {
      setDecayLoading(false);
    }
  }, [decayStrategyId, t]);

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Text strong style={{ fontSize: 15 }}>
          <ThunderboltOutlined style={{ marginRight: 8 }} />
          {t('marketplace.optimization.title', 'AI 策略优化')}
        </Text>
      </div>

      {/* ── Decay Detection ── */}
      <Card size="small" style={{ marginBottom: 16 }} title={t('marketplace.optimization.decayDetection', '衰减检测')}>
        <Space>
          <Input
            placeholder={t('marketplace.optimization.strategyIdPlaceholder', '输入策略 ID')}
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
            {t('marketplace.optimization.detect', '检测衰减')}
          </Button>
        </Space>
        {decayResult && (
          <Alert
            style={{ marginTop: 12 }}
            type={decayResult.isDecaying ? 'warning' : 'success'}
            showIcon
            icon={decayResult.isDecaying ? <WarningOutlined /> : <CheckOutlined />}
            message={decayResult.isDecaying
              ? t('marketplace.optimization.decayDetected', '检测到策略衰减！')
              : t('marketplace.optimization.noDecay', '策略表现正常')}
            description={decayResult.isDecaying
              ? `${t('marketplace.optimization.decayScore', '衰减评分')}: ${decayResult.score}/3 — ${t('marketplace.optimization.decayHint', '建议创建优化任务以改进策略')}`
              : `${t('marketplace.optimization.decayScore', '衰减评分')}: ${decayResult.score}/3`}
          />
        )}
      </Card>

      {/* ── Optimization Tasks ── */}
      <Card size="small" title={t('marketplace.optimization.tasks', '优化任务列表')}>
        <Space style={{ marginBottom: 12 }}>
          {['', 'pending', 'completed', 'rejected', 'published'].map(s => (
            <Button
              key={s}
              size="small"
              type={statusFilter === s ? 'primary' : 'default'}
              onClick={() => setStatusFilter(s)}
            >
              {s === '' ? t('marketplace.optimization.all', '全部') : s}
            </Button>
          ))}
        </Space>
        {tasks.length === 0 && !loading ? (
          <Empty description={t('marketplace.optimization.noTasks', '暂无优化任务')} />
        ) : (
          <Table<OptimizationTaskInfo>
            rowKey="id"
            dataSource={tasks}
            loading={loading}
            pagination={{ pageSize: 10, total }}
            size="small"
            columns={[
              { title: t('marketplace.optimization.strategyId', '策略'), dataIndex: 'strategyId', key: 'sid', width: 120, render: (v: string) => <Text copyable style={{ fontSize: 12 }}>{v?.slice(0, 8)}...</Text> },
              { title: t('marketplace.optimization.status', '状态'), dataIndex: 'status', key: 'status', width: 100, render: (s: string) => <Tag color={STATUS_COLORS[s] || 'default'}>{s}</Tag> },
              { title: t('marketplace.optimization.trigger', '触发原因'), dataIndex: 'triggerReason', key: 'trigger', width: 120, ellipsis: true },
              { title: t('marketplace.optimization.changeSummary', '变更摘要'), dataIndex: 'changeSummary', key: 'summary', ellipsis: true },
              {
                title: t('marketplace.optimization.actions', '操作'), key: 'actions', width: 240,
                render: (_: unknown, row: OptimizationTaskInfo) => (
                  <Space>
                    {row.status === 'completed' && (
                      <>
                        <Button size="small" icon={<EyeOutlined />} onClick={() => handlePreview(row.id)}>
                          {t('marketplace.optimization.preview', '预览')}
                        </Button>
                        <Button size="small" type="primary" icon={<CheckOutlined />}
                          loading={actionLoading === row.id}
                          onClick={() => handlePublish(row.id)}>
                          {t('marketplace.optimization.publish', '发布')}
                        </Button>
                        <Button size="small" danger icon={<CloseOutlined />}
                          loading={actionLoading === row.id}
                          onClick={() => handleReject(row.id)}>
                          {t('marketplace.optimization.reject', '拒绝')}
                        </Button>
                      </>
                    )}
                    {row.status === 'published' && (
                      <Tag color="green">{t('marketplace.optimization.versionId', '版本')}: {row.publishedVersionId?.slice(0, 8)}</Tag>
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
        title={t('marketplace.optimization.previewTitle', '优化预览对比')}
        open={previewOpen}
        onCancel={() => { setPreviewOpen(false); setPreviewData(null); }}
        footer={null}
        width={900}
      >
        {previewLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}>{t('common.loading', '加载中...')}</div>
        ) : previewData ? (
          <div>
            <Descriptions size="small" column={2} style={{ marginBottom: 16 }}>
              <Descriptions.Item label={t('marketplace.optimization.trigger', '触发原因')}>{previewData.task?.triggerReason}</Descriptions.Item>
              <Descriptions.Item label={t('marketplace.optimization.status', '状态')}><Tag color={STATUS_COLORS[previewData.task?.status || '']}>{previewData.task?.status}</Tag></Descriptions.Item>
              <Descriptions.Item label={t('marketplace.optimization.changeSummary', '变更摘要')} span={2}>{previewData.changeSummary}</Descriptions.Item>
            </Descriptions>

            {previewData.decayMetrics && (
              <Alert
                type="warning"
                style={{ marginBottom: 16 }}
                message={t('marketplace.optimization.decayMetrics', '衰减指标')}
                description={
                  <Descriptions size="small" column={2} style={{ marginTop: 8 }}>
                    <Descriptions.Item label="Decay Score">{previewData.decayMetrics.decayScore}</Descriptions.Item>
                    <Descriptions.Item label="Trigger">{previewData.decayMetrics.triggerReason}</Descriptions.Item>
                    <Descriptions.Item label="Sharpe Decline">{previewData.decayMetrics.sharpeDeclinePct}</Descriptions.Item>
                    <Descriptions.Item label="Win Rate Decline">{previewData.decayMetrics.winrateDeclinePct}</Descriptions.Item>
                    <Descriptions.Item label="Return Delta">{previewData.decayMetrics.returnDelta}</Descriptions.Item>
                  </Descriptions>
                }
              />
            )}

            <Text strong style={{ display: 'block', marginBottom: 8 }}>
              {t('marketplace.optimization.backtestCompare', '回测对比')}
            </Text>
            <BacktestCompare original={previewData.originalBacktest} optimized={previewData.optimizedBacktest} />

            {previewData.suggestedCodePreview && (
              <Card size="small" style={{ marginTop: 16 }} title={t('marketplace.optimization.codePreview', '优化后代码预览')}>
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
