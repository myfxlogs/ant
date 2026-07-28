import { useState, useCallback, useEffect } from 'react';
import { Card, Table, Button, Tag, Typography, Space, Modal, Descriptions, Empty, message, Alert, Input } from 'antd';
import { ExperimentOutlined, CheckOutlined, ThunderboltOutlined, WarningOutlined } from '@ant-design/icons';
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
import type { OptimizationTaskInfo, PreviewOptimizationResponse } from '@/gen/ant/v1/marketplace_service_pb';
import { BacktestCompare, buildTaskColumns, STATUS_COLORS } from './OptimizationTabHelpers';

const { Text } = Typography;

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
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : String(e) || t('marketplace.optimization.previewFailed'));
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
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : String(e) || t('marketplace.optimization.rejectFailed'));
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
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : String(e) || t('marketplace.optimization.publishFailed'));
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
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : String(e) || t('marketplace.optimization.detectFailed'));
    } finally {
      setDecayLoading(false);
    }
  }, [decayStrategyId, t]);

  const taskColumns = buildTaskColumns(t, handlePreview, handlePublish, handleReject, actionLoading);

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
            columns={taskColumns}
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
