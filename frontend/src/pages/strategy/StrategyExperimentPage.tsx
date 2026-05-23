import { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Card, Form, Input, InputNumber, List, Progress, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { strategyApi, type StrategyTemplate } from '@/client/strategy';
import { strategyExperimentApi, type StrategyExperiment, type StrategyExperimentCandidate } from '@/client/strategyExperiment';
import { jobApi, type JobEvent } from '@/client/job';
import { showError, showSuccess } from '@/utils/message';

const { Text, Title } = Typography;
const maxJobEvents = 20;

type SubmitExperimentValues = {
  baseTemplateId: string;
  parameterSpace: string;
  searchMethod: string;
  maxCandidates: number;
  objective: string;
};

function defaultParameterSpace() {
  return JSON.stringify({ fast_period: [8, 12, 16], slow_period: [24, 30], risk_pct: [0.5, 1] }, null, 2);
}

function JobEventStreamCard({ jobId, events }: { jobId?: string; events: JobEvent[] }) {
  const latestEvent = events[events.length - 1];
  const progressPercent = Math.round((latestEvent?.progress || 0) * 100);

  return (
    <Card title="Job 事件流">
      {jobId ? (
        <div className="space-y-3">
          <Progress percent={progressPercent} size="small" />
          <List
            size="small"
            dataSource={events}
            locale={{ emptyText: '暂无事件' }}
            renderItem={event => (
              <List.Item>
                <Space>
                  <Tag>{event.type}</Tag>
                  <Text>{event.stage || '-'}</Text>
                  <Text type="secondary">{event.message || event.status}</Text>
                </Space>
              </List.Item>
            )}
          />
        </div>
      ) : (
        <Text type="secondary">选择带 Job 的实验后显示事件。</Text>
      )}
    </Card>
  );
}

export default function StrategyExperimentPage() {
  const [form] = Form.useForm();
  const [templates, setTemplates] = useState<StrategyTemplate[]>([]);
  const [experiments, setExperiments] = useState<StrategyExperiment[]>([]);
  const [candidates, setCandidates] = useState<StrategyExperimentCandidate[]>([]);
  const [selectedExperimentId, setSelectedExperimentId] = useState('');
  const [loading, setLoading] = useState(false);
  const [candidateLoading, setCandidateLoading] = useState(false);
  const [jobEvents, setJobEvents] = useState<JobEvent[]>([]);

  const selectedExperiment = useMemo(
    () => experiments.find(item => item.id === selectedExperimentId),
    [experiments, selectedExperimentId],
  );

  const loadTemplates = async () => {
    try {
      setTemplates(await strategyApi.listTemplates());
    } catch {
      showError('加载策略模板失败');
    }
  };

  const loadExperiments = async () => {
    try {
      const rows = await strategyExperimentApi.list();
      setExperiments(rows);
      if (!selectedExperimentId && rows.length > 0) {
        setSelectedExperimentId(rows[0].id);
      }
    } catch {
      showError('加载实验列表失败');
    }
  };

  const loadCandidates = async (experimentId: string) => {
    if (!experimentId) {
      setCandidates([]);
      return;
    }
    setCandidateLoading(true);
    try {
      setCandidates(await strategyExperimentApi.listCandidates(experimentId));
    } catch {
      showError('加载候选失败');
    } finally {
      setCandidateLoading(false);
    }
  };

  useEffect(() => {
    void loadTemplates();
    void loadExperiments();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    void loadCandidates(selectedExperimentId);
  }, [selectedExperimentId]);

  useEffect(() => {
    let cancelled = false;
    setJobEvents([]);
    if (!selectedExperiment?.jobId) {
      return;
    }
    void jobApi.subscribe(selectedExperiment.jobId, event => {
      if (!cancelled) {
        setJobEvents(prev => [...prev.slice(1 - maxJobEvents), event]);
      }
    }).catch(() => {
      if (!cancelled) {
        showError('订阅实验 Job 事件失败');
      }
    });
    return () => {
      cancelled = true;
    };
  }, [selectedExperiment?.jobId]);

  const handleSubmit = async (values: SubmitExperimentValues) => {
    setLoading(true);
    try {
      const parameterSpace = JSON.parse(values.parameterSpace || '{}') as Record<string, unknown>;
      const res = await strategyExperimentApi.submit({
        baseTemplateId: values.baseTemplateId,
        parameterSpace,
        searchMethod: values.searchMethod,
        maxCandidates: values.maxCandidates,
        objective: values.objective,
      });
      showSuccess('策略实验已生成候选');
      await loadExperiments();
      if (res.experiment?.id) {
        setSelectedExperimentId(res.experiment.id);
      }
    } catch {
      showError('提交策略实验失败，请确认参数空间是合法 JSON');
    } finally {
      setLoading(false);
    }
  };

  const promote = async (candidate: StrategyExperimentCandidate) => {
    try {
      const res = await strategyExperimentApi.promoteCandidateToDraft(candidate.id, `实验候选 ${candidate.rank}`);
      showSuccess(`已生成草稿模板：${res.templateId}`);
    } catch {
      showError('提升候选为草稿失败');
    }
  };

  const experimentColumns: ColumnsType<StrategyExperiment> = [
    { title: '状态', dataIndex: 'status', render: v => <Tag color={v === 'SUCCEEDED' ? 'green' : 'blue'}>{v}</Tag> },
    { title: '搜索方式', dataIndex: 'searchMethod' },
    { title: '候选上限', dataIndex: 'maxCandidates' },
    { title: '目标', dataIndex: 'objective' },
    { title: 'Job', dataIndex: 'jobId', ellipsis: true },
    {
      title: '操作',
      render: (_, row) => <Button size="small" onClick={() => setSelectedExperimentId(row.id)}>查看候选</Button>,
    },
  ];

  const candidateColumns: ColumnsType<StrategyExperimentCandidate> = [
    { title: '排名', dataIndex: 'rank', width: 80 },
    { title: '等级', dataIndex: 'grade', width: 80, render: v => <Tag color={v === 'A' ? 'gold' : v === 'B' ? 'blue' : 'default'}>{v}</Tag> },
    { title: '评分', dataIndex: 'score', width: 100, render: v => Number(v).toFixed(1) },
    { title: '参数', dataIndex: 'parameters', render: v => <Text code>{JSON.stringify(v)}</Text> },
    { title: '摘要', dataIndex: 'summary' },
    { title: '建议', dataIndex: 'recommendation' },
    {
      title: '操作',
      width: 120,
      render: (_, row) => <Button size="small" type="primary" onClick={() => void promote(row)}>生成草稿</Button>,
    },
  ];

  return (
    <div className="space-y-4">
      <div>
        <Title level={3}>策略实验</Title>
        <Text type="secondary">参数实验、候选评分与草稿生成均由后端完成，前端只负责提交和展示。</Text>
      </div>

      <Alert type="info" showIcon message="当前为确定性参数实验最小闭环，候选仅生成草稿，不会自动发布、调度或交易。" />

      <Card title="提交实验">
        <Form
          form={form}
          layout="vertical"
          initialValues={{ parameterSpace: defaultParameterSpace(), searchMethod: 'grid', maxCandidates: 12, objective: 'balanced' }}
          onFinish={handleSubmit}
        >
          <Form.Item name="baseTemplateId" label="基础策略模板" rules={[{ required: true, message: '请选择基础策略模板' }]}>
            <Select
              showSearch
              options={templates.map(t => ({ value: t.id, label: `${t.name || t.id} (${t.status || '-'})` }))}
              placeholder="选择模板"
            />
          </Form.Item>
          <Form.Item name="parameterSpace" label="参数空间 JSON" rules={[{ required: true, message: '请输入参数空间 JSON' }]}>
            <Input.TextArea rows={8} />
          </Form.Item>
          <Space size="large" wrap>
            <Form.Item name="searchMethod" label="搜索方式">
              <Select style={{ width: 160 }} options={[{ value: 'grid', label: 'Grid' }, { value: 'random', label: 'Random' }]} />
            </Form.Item>
            <Form.Item name="maxCandidates" label="候选上限">
              <InputNumber min={1} max={50} />
            </Form.Item>
            <Form.Item name="objective" label="目标">
              <Input style={{ width: 220 }} />
            </Form.Item>
          </Space>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading}>提交实验</Button>
          </Form.Item>
        </Form>
      </Card>

      <Card title="实验列表">
        <Table rowKey="id" size="small" dataSource={experiments} columns={experimentColumns} pagination={false} />
      </Card>

      <JobEventStreamCard jobId={selectedExperiment?.jobId} events={jobEvents} />

      <Card title={selectedExperiment ? `候选列表：${selectedExperiment.id}` : '候选列表'}>
        <Table rowKey="id" size="small" loading={candidateLoading} dataSource={candidates} columns={candidateColumns} pagination={false} scroll={{ x: 1100 }} />
      </Card>
    </div>
  );
}
