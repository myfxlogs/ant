import { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Card, Form, Input, InputNumber, List, Progress, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { strategyApi, type StrategyTemplate } from '@/client/strategy';
import { strategyExperimentApi, type StrategyExperiment, type StrategyExperimentCandidate } from '@/client/strategyExperiment';
import { jobApi, type JobEvent } from '@/client/job';
import { showError, showSuccess } from '@/utils/message';
import { useTranslation } from 'react-i18next';

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

function JobEventStreamCard({ jobId, events, t }: { jobId?: string; events: JobEvent[]; t: (key: string, options?: Record<string, unknown>) => string }) {
  const latestEvent = events[events.length - 1];
  const progressPercent = Math.round((latestEvent?.progress || 0) * 100);

  return (
    <Card title={t('strategy.experiment.jobEventStream')}>
      {jobId ? (
        <div className="space-y-3">
          <Progress percent={progressPercent} size="small" />
          <List
            size="small"
            dataSource={events}
            locale={{ emptyText: t('strategy.experiment.noEvents') }}
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
        <Text type="secondary">{t('strategy.experiment.selectJobToView')}</Text>
      )}
    </Card>
  );
}

export default function StrategyExperimentPage() {
  const { t } = useTranslation();
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
      showError(t('strategy.experiment.messages.loadTemplatesFailed'));
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
      showError(t('strategy.experiment.messages.loadExperimentsFailed'));
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
      showError(t('strategy.experiment.messages.loadCandidatesFailed'));
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
        showError(t('strategy.experiment.messages.subscribeJobFailed'));
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
      showSuccess(t('strategy.experiment.messages.candidatesGenerated'));
      await loadExperiments();
      if (res.experiment?.id) {
        setSelectedExperimentId(res.experiment.id);
      }
    } catch {
      showError(t('strategy.experiment.messages.submitFailed'));
    } finally {
      setLoading(false);
    }
  };

  const promote = async (candidate: StrategyExperimentCandidate) => {
    try {
      const res = await strategyExperimentApi.promoteCandidateToDraft(candidate.id, `实验候选 ${candidate.rank}`);
      showSuccess(t('strategy.experiment.messages.draftGenerated', { templateId: res.templateId }));
    } catch {
      showError(t('strategy.experiment.messages.promoteFailed'));
    }
  };

  const experimentColumns: ColumnsType<StrategyExperiment> = [
    { title: t('strategy.experiment.list.column.status'), dataIndex: 'status', render: v => <Tag color={v === 'SUCCEEDED' ? 'green' : 'blue'}>{v}</Tag> },
    { title: t('strategy.experiment.list.column.searchMethod'), dataIndex: 'searchMethod' },
    { title: t('strategy.experiment.list.column.maxCandidates'), dataIndex: 'maxCandidates' },
    { title: t('strategy.experiment.list.column.objective'), dataIndex: 'objective' },
    { title: 'Job', dataIndex: 'jobId', ellipsis: true },
    {
      title: t('strategy.experiment.list.column.actions'),
      render: (_, row) => <Button size="small" onClick={() => setSelectedExperimentId(row.id)}>{t('strategy.experiment.list.column.viewCandidates')}</Button>,
    },
  ];

  const candidateColumns: ColumnsType<StrategyExperimentCandidate> = [
    { title: t('strategy.experiment.candidates.column.rank'), dataIndex: 'rank', width: 80 },
    { title: t('strategy.experiment.candidates.column.grade'), dataIndex: 'grade', width: 80, render: v => <Tag color={v === 'A' ? 'gold' : v === 'B' ? 'blue' : 'default'}>{v}</Tag> },
    { title: t('strategy.experiment.candidates.column.score'), dataIndex: 'score', width: 100, render: v => Number(v).toFixed(1) },
    { title: t('strategy.experiment.candidates.column.parameters'), dataIndex: 'parameters', render: v => <Text code>{JSON.stringify(v)}</Text> },
    { title: t('strategy.experiment.candidates.column.summary'), dataIndex: 'summary' },
    { title: t('strategy.experiment.candidates.column.recommendation'), dataIndex: 'recommendation' },
    {
      title: t('strategy.experiment.candidates.column.actions'),
      width: 120,
      render: (_, row) => <Button size="small" type="primary" onClick={() => void promote(row)}>{t('strategy.experiment.candidates.column.generateDraft')}</Button>,
    },
  ];

  return (
    <div className="space-y-4">
      <div>
        <Title level={3}>{t('strategy.experiment.title')}</Title>
        <Text type="secondary">{t('strategy.experiment.subtitle')}</Text>
      </div>

      <Alert type="info" showIcon message={t('strategy.experiment.ruleVersionAlert')} />

      <Card title={t('strategy.experiment.submitForm.title')}>
        <Form
          form={form}
          layout="vertical"
          initialValues={{ parameterSpace: defaultParameterSpace(), searchMethod: 'grid', maxCandidates: 12, objective: 'balanced' }}
          onFinish={handleSubmit}
        >
          <Form.Item name="baseTemplateId" label={t('strategy.experiment.submitForm.baseTemplate')} rules={[{ required: true, message: t('strategy.experiment.submitForm.baseTemplateRequired') }]}>
            <Select
              showSearch
              options={templates.map(t => ({ value: t.id, label: `${t.name || t.id} (${t.status || '-'})` }))}
              placeholder={t('strategy.experiment.submitForm.baseTemplatePlaceholder')}
            />
          </Form.Item>
          <Form.Item name="parameterSpace" label={t('strategy.experiment.submitForm.parameterSpace')} rules={[{ required: true, message: t('strategy.experiment.submitForm.parameterSpaceRequired') }]}>
            <Input.TextArea rows={8} />
          </Form.Item>
          <Space size="large" wrap>
            <Form.Item name="searchMethod" label={t('strategy.experiment.submitForm.searchMethod')}>
              <Select style={{ width: 160 }} options={[{ value: 'grid', label: 'Grid' }, { value: 'random', label: 'Random' }]} />
            </Form.Item>
            <Form.Item name="maxCandidates" label={t('strategy.experiment.submitForm.maxCandidates')}>
              <InputNumber min={1} max={50} />
            </Form.Item>
            <Form.Item name="objective" label={t('strategy.experiment.submitForm.objective')}>
              <Input style={{ width: 220 }} />
            </Form.Item>
          </Space>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading}>{t('strategy.experiment.submitForm.submit')}</Button>
          </Form.Item>
        </Form>
      </Card>

      <Card title={t('strategy.experiment.list.title')}>
        <Table rowKey="id" size="small" dataSource={experiments} columns={experimentColumns} pagination={false} />
      </Card>

      <JobEventStreamCard jobId={selectedExperiment?.jobId} events={jobEvents} t={t} />

      <Card title={selectedExperiment ? t('strategy.experiment.candidates.titleWithId', { id: selectedExperiment.id }) : t('strategy.experiment.candidates.title')}>
        <Table rowKey="id" size="small" loading={candidateLoading} dataSource={candidates} columns={candidateColumns} pagination={false} scroll={{ x: 1100 }} />
      </Card>
    </div>
  );
}
