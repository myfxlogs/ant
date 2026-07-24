import { useState, useEffect, useCallback } from 'react';
import { Card, Table, Button, Tag, Space, Select, Modal, Input, Typography, Popconfirm, message } from 'antd';
import { CheckOutlined, CloseOutlined, ThunderboltOutlined, ReloadOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { create } from '@bufbuild/protobuf';
import {
  ListAutoGenTasksRequestSchema,
  ApproveAutoGenTaskRequestSchema,
  RejectAutoGenTaskRequestSchema,
  TriggerBatchGenerationRequestSchema,
  type AutoGenTaskInfo,
} from '@/gen/ant/v1/marketplace_service_pb';

const { Title, Text } = Typography;

const STATUS_COLORS: Record<string, string> = {
  pending: 'default',
  generating: 'processing',
  compiling: 'processing',
  backtesting: 'processing',
  evaluating: 'processing',
  awaiting_review: 'warning',
  published: 'success',
  rejected: 'error',
  failed: 'error',
};

export default function AutoGenTaskReview() {
  const { t } = useTranslation();
  const [tasks, setTasks] = useState<AutoGenTaskInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState('awaiting_review');
  const [triggerOpen, setTriggerOpen] = useState(false);
  const [triggerSymbols, setTriggerSymbols] = useState('EURUSD,GBPUSD,USDJPY');
  const [triggerTimeframes, setTriggerTimeframes] = useState('M15,H1,H4,D1');
  const [triggerTypes, setTriggerTypes] = useState('trend_following,mean_reversion,breakout');
  const [triggering, setTriggering] = useState(false);

  const fetchTasks = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await marketplaceClient.listAutoGenTasks(
        create(ListAutoGenTasksRequestSchema, { status: statusFilter, limit: 50 })
      );
      setTasks(resp.tasks);
    } catch (e: any) {
      message.error(e?.message || t('admin.autogen.loadFailed', { defaultValue: 'Failed to load tasks' }));
    } finally {
      setLoading(false);
    }
  }, [statusFilter]);

  useEffect(() => { fetchTasks(); }, [fetchTasks]);

  const handleApprove = useCallback(async (taskId: string) => {
    try {
      await marketplaceClient.approveAutoGenTask(
        create(ApproveAutoGenTaskRequestSchema, { taskId })
      );
      message.success(t('admin.autogen.approved', { defaultValue: 'Task approved and published' }));
      fetchTasks();
    } catch (e: any) {
      message.error(e?.message || t('admin.autogen.approveFailed', { defaultValue: 'Approve failed' }));
    }
  }, [fetchTasks, t]);

  const handleReject = useCallback(async (taskId: string) => {
    try {
      await marketplaceClient.rejectAutoGenTask(
        create(RejectAutoGenTaskRequestSchema, { taskId })
      );
      message.success(t('admin.autogen.rejected', { defaultValue: 'Task rejected' }));
      fetchTasks();
    } catch (e: any) {
      message.error(e?.message || t('admin.autogen.rejectFailed', { defaultValue: 'Reject failed' }));
    }
  }, [fetchTasks, t]);

  const handleTrigger = useCallback(async () => {
    setTriggering(true);
    try {
      const resp = await marketplaceClient.triggerBatchGeneration(
        create(TriggerBatchGenerationRequestSchema, {
          symbols: triggerSymbols.split(',').map(s => s.trim()).filter(Boolean),
          timeframes: triggerTimeframes.split(',').map(s => s.trim()).filter(Boolean),
          strategyTypes: triggerTypes.split(',').map(s => s.trim()).filter(Boolean),
        })
      );
      message.success(t('admin.autogen.enqueued', { defaultValue: '{{count}} tasks enqueued', count: resp.enqueued }));
      setTriggerOpen(false);
      fetchTasks();
    } catch (e: any) {
      message.error(e?.message || t('admin.autogen.triggerFailed', { defaultValue: 'Trigger failed' }));
    } finally {
      setTriggering(false);
    }
  }, [triggerSymbols, triggerTimeframes, triggerTypes, fetchTasks, t]);

  const columns = [
    {
      title: t('admin.autogen.symbol', { defaultValue: 'Symbol' }),
      dataIndex: 'symbol',
      key: 'symbol',
      width: 100,
    },
    {
      title: t('admin.autogen.timeframe', { defaultValue: 'TF' }),
      dataIndex: 'timeframe',
      key: 'timeframe',
      width: 70,
    },
    {
      title: t('admin.autogen.strategyType', { defaultValue: 'Type' }),
      dataIndex: 'strategyType',
      key: 'strategyType',
      width: 120,
    },
    {
      title: t('admin.autogen.status', { defaultValue: 'Status' }),
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: string) => <Tag color={STATUS_COLORS[status] || 'default'}>{status}</Tag>,
    },
    {
      title: t('admin.autogen.quality', { defaultValue: 'Quality' }),
      dataIndex: 'qualityPassed',
      key: 'qualityPassed',
      width: 80,
      render: (v: boolean) => v ? <Tag color="green">PASS</Tag> : <Tag color="red">FAIL</Tag>,
    },
    {
      title: t('admin.autogen.error', { defaultValue: 'Error' }),
      dataIndex: 'errorMessage',
      key: 'errorMessage',
      ellipsis: true,
      render: (v: string) => v ? <Text type="danger" style={{ fontSize: 12 }}>{v}</Text> : '-',
    },
    {
      title: t('admin.autogen.actions', { defaultValue: 'Actions' }),
      key: 'actions',
      width: 160,
      render: (_: any, record: AutoGenTaskInfo) => (
        <Space>
          {record.status === 'awaiting_review' && (
            <>
              <Popconfirm title={t('admin.autogen.confirmApprove', { defaultValue: 'Approve and publish?' })} onConfirm={() => handleApprove(record.id)}>
                <Button size="small" type="primary" icon={<CheckOutlined />}>{t('admin.autogen.approve', { defaultValue: 'Approve' })}</Button>
              </Popconfirm>
              <Popconfirm title={t('admin.autogen.confirmReject', { defaultValue: 'Reject this task?' })} onConfirm={() => handleReject(record.id)}>
                <Button size="small" danger icon={<CloseOutlined />}>{t('admin.autogen.reject', { defaultValue: 'Reject' })}</Button>
              </Popconfirm>
            </>
          )}
        </Space>
      ),
    },
  ];

  return (
    <Card>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={4} style={{ margin: 0 }}>{t('admin.autogen.title', { defaultValue: 'AI Strategy Generation Tasks' })}</Title>
        <Space>
          <Select
            value={statusFilter}
            onChange={setStatusFilter}
            style={{ width: 160 }}
            options={[
              { value: '', label: t('admin.autogen.allStatus', { defaultValue: 'All Status' }) },
              { value: 'pending', label: 'Pending' },
              { value: 'awaiting_review', label: 'Awaiting Review' },
              { value: 'published', label: 'Published' },
              { value: 'rejected', label: 'Rejected' },
              { value: 'failed', label: 'Failed' },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={fetchTasks}>{t('admin.autogen.refresh', { defaultValue: 'Refresh' })}</Button>
          <Button type="primary" icon={<ThunderboltOutlined />} onClick={() => setTriggerOpen(true)}>
            {t('admin.autogen.triggerBatch', { defaultValue: 'Trigger Batch' })}
          </Button>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={tasks}
        rowKey="id"
        loading={loading}
        size="small"
        pagination={{ pageSize: 20 }}
      />

      <Modal
        title={t('admin.autogen.triggerBatch', { defaultValue: 'Trigger Batch Generation' })}
        open={triggerOpen}
        onCancel={() => setTriggerOpen(false)}
        onOk={handleTrigger}
        confirmLoading={triggering}
        okText={t('admin.autogen.enqueue', { defaultValue: 'Enqueue' })}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <div>
            <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{t('admin.autogen.symbols', { defaultValue: 'Symbols (comma-separated)' })}</Text>
            <Input value={triggerSymbols} onChange={e => setTriggerSymbols(e.target.value)} />
          </div>
          <div>
            <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{t('admin.autogen.timeframes', { defaultValue: 'Timeframes (comma-separated)' })}</Text>
            <Input value={triggerTimeframes} onChange={e => setTriggerTimeframes(e.target.value)} />
          </div>
          <div>
            <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>{t('admin.autogen.strategyTypes', { defaultValue: 'Strategy Types (comma-separated)' })}</Text>
            <Input value={triggerTypes} onChange={e => setTriggerTypes(e.target.value)} />
          </div>
        </Space>
      </Modal>
    </Card>
  );
}
