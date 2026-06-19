import { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Card, Form, Input, InputNumber, List, Progress, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { strategyApi, type StrategyTemplate } from '@/client/strategy';
import { strategyExperimentApi, type StrategyExperiment, type StrategyExperimentCandidate } from '@/client/strategyExperiment';
import { jobApi, type JobEvent } from '@/client/job';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import { showError, showSuccess } from '@/utils/message';
import { useTranslation } from 'react-i18next'
import { CANDIDATES_COLUMN_ACTIONS_KEY, CANDIDATES_COLUMN_GENERATE_DRAFT_KEY, CANDIDATES_COLUMN_GRADE_KEY, CANDIDATES_COLUMN_PARAMETERS_KEY, CANDIDATES_COLUMN_RANK_KEY, CANDIDATES_COLUMN_RECOMMENDATION_KEY, CANDIDATES_COLUMN_SCORE_KEY, CANDIDATES_COLUMN_SUMMARY_KEY, CANDIDATES_TITLE_KEY, CANDIDATES_TITLE_WITH_ID_KEY, JOB_EVENT_STREAM_KEY, LIST_COLUMN_ACTIONS_KEY, LIST_COLUMN_MAX_CANDIDATES_KEY, LIST_COLUMN_OBJECTIVE_KEY, LIST_COLUMN_SEARCH_METHOD_KEY, LIST_COLUMN_STATUS_KEY, LIST_COLUMN_VIEW_CANDIDATES_KEY, LIST_TITLE_KEY, MESSAGES_CANDIDATES_GENERATED_KEY, MESSAGES_DRAFT_GENERATED_KEY, MESSAGES_PROMOTE_FAILED_KEY, MESSAGES_SUBMIT_FAILED_KEY, MESSAGES_SUBSCRIBE_JOB_FAILED_KEY, NO_EVENTS_KEY, RULE_VERSION_ALERT_KEY, SELECT_JOB_TO_VIEW_KEY, SUBMIT_FORM_BASE_TEMPLATE_KEY, SUBMIT_FORM_BASE_TEMPLATE_PLACEHOLDER_KEY, SUBMIT_FORM_BASE_TEMPLATE_REQUIRED_KEY, SUBMIT_FORM_MAX_CANDIDATES_KEY, SUBMIT_FORM_OBJECTIVE_KEY, SUBMIT_FORM_PARAMETER_SPACE_KEY, SUBMIT_FORM_PARAMETER_SPACE_REQUIRED_KEY, SUBMIT_FORM_SEARCH_METHOD_KEY, SUBMIT_FORM_SUBMIT_KEY, SUBMIT_FORM_TITLE_KEY, SUBTITLE_KEY, TITLE_KEY } from '@/gen/ant/v1/i18n/strategy_experiment_keys';

;

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
    <Card title={t(JOB_EVENT_STREAM_KEY)}>
      {jobId ? (
        <div className="space-y-3">
          <Progress percent={progressPercent} size="small" />
          <List
            size="small"
            dataSource={events}
            locale={{ emptyText: t(NO_EVENTS_KEY) }}
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
        <Text type="secondary">{t(SELECT_JOB_TO_VIEW_KEY)}</Text>
      )}
    </Card>
  );
}

export default function StrategyExperimentPage() {
  const { t } = useTranslation();
  const [selectedExperimentId, setSelectedExperimentId] = useState('');
  const [loading, setLoading] = useState(false);
  const [jobEvents, setJobEvents] = useState<JobEvent[]>([]);

  const { data: templates = [] } = useRpcQuery(
    ['strategy', 'templates'],
    () => strategyApi.listTemplates(),
  );

  const { data: experiments = [], refetch: refetchExperiments } = useRpcQuery(
    ['strategy', 'experiments'],
    () => strategyExperimentApi.list(),
  );

  const { data: candidates = [], isLoading: candidateLoading } = useRpcQuery(
    ['strategy', 'experiments', selectedExperimentId, 'candidates'],
    () => strategyExperimentApi.listCandidates(selectedExperimentId),
    { enabled: !!selectedExperimentId },
  );

  useEffect(() => {
    if (!selectedExperimentId && experiments.length > 0) {
      setSelectedExperimentId(experiments[0].id);
    }
  }, [experiments, selectedExperimentId]);

  const selectedExperiment = useMemo(
    () => experiments.find(item => item.id === selectedExperimentId),
    [experiments, selectedExperimentId],
  );

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
        showError(t(MESSAGES_SUBSCRIBE_JOB_FAILED_KEY));
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
      showSuccess(t(MESSAGES_CANDIDATES_GENERATED_KEY));
      await refetchExperiments();
      if (res.experiment?.id) {
        setSelectedExperimentId(res.experiment.id);
      }
    } catch {
      showError(t(MESSAGES_SUBMIT_FAILED_KEY));
    } finally {
      setLoading(false);
    }
  };

  const promote = async (candidate: StrategyExperimentCandidate) => {
    try {
      const res = await strategyExperimentApi.promoteCandidateToDraft(candidate.id, `实验候选 ${candidate.rank}`);
      showSuccess(t(MESSAGES_DRAFT_GENERATED_KEY, { templateId: res.templateId }));
    } catch {
      showError(t(MESSAGES_PROMOTE_FAILED_KEY));
    }
  };

  const experimentColumns: ColumnsType<StrategyExperiment> = [
    { title: t(LIST_COLUMN_STATUS_KEY), dataIndex: 'status', render: v => <Tag color={v === 'SUCCEEDED' ? 'green' : 'blue'}>{v}</Tag> },
    { title: t(LIST_COLUMN_SEARCH_METHOD_KEY), dataIndex: 'searchMethod' },
    { title: t(LIST_COLUMN_MAX_CANDIDATES_KEY), dataIndex: 'maxCandidates' },
    { title: t(LIST_COLUMN_OBJECTIVE_KEY), dataIndex: 'objective' },
    { title: 'Job', dataIndex: 'jobId', ellipsis: true },
    {
      title: t(LIST_COLUMN_ACTIONS_KEY),
      render: (_, row) => <Button size="small" onClick={() => setSelectedExperimentId(row.id)}>{t(LIST_COLUMN_VIEW_CANDIDATES_KEY)}</Button>,
    },
  ];

  const candidateColumns: ColumnsType<StrategyExperimentCandidate> = [
    { title: t(CANDIDATES_COLUMN_RANK_KEY), dataIndex: 'rank', width: 80 },
    { title: t(CANDIDATES_COLUMN_GRADE_KEY), dataIndex: 'grade', width: 80, render: v => <Tag color={v === 'A' ? 'gold' : v === 'B' ? 'blue' : 'default'}>{v}</Tag> },
    { title: t(CANDIDATES_COLUMN_SCORE_KEY), dataIndex: 'score', width: 100, render: v => Number(v).toFixed(1) },
    { title: t(CANDIDATES_COLUMN_PARAMETERS_KEY), dataIndex: 'parameters', render: v => <Text code>{JSON.stringify(v)}</Text> },
    { title: t(CANDIDATES_COLUMN_SUMMARY_KEY), dataIndex: 'summary' },
    { title: t(CANDIDATES_COLUMN_RECOMMENDATION_KEY), dataIndex: 'recommendation' },
    {
      title: t(CANDIDATES_COLUMN_ACTIONS_KEY),
      width: 120,
      render: (_, row) => <Button size="small" type="primary" onClick={() => void promote(row)}>{t(CANDIDATES_COLUMN_GENERATE_DRAFT_KEY)}</Button>,
    },
  ];

  return (
    <div className="space-y-4">
      <div>
        <Title level={3}>{t(TITLE_KEY)}</Title>
        <Text type="secondary">{t(SUBTITLE_KEY)}</Text>
      </div>

      <Alert type="info" showIcon message={t(RULE_VERSION_ALERT_KEY)} />

      <Card title={t(SUBMIT_FORM_TITLE_KEY)}>
        <Form
          layout="vertical"
          initialValues={{ parameterSpace: defaultParameterSpace(), searchMethod: 'grid', maxCandidates: 12, objective: 'balanced' }}
          onFinish={handleSubmit}
        >
          <Form.Item name="baseTemplateId" label={t(SUBMIT_FORM_BASE_TEMPLATE_KEY)} rules={[{ required: true, message: t(SUBMIT_FORM_BASE_TEMPLATE_REQUIRED_KEY) }]}>
            <Select
              showSearch
              options={templates.map(t => ({ value: t.id, label: `${t.name || t.id} (${t.status || '-'})` }))}
              placeholder={t(SUBMIT_FORM_BASE_TEMPLATE_PLACEHOLDER_KEY)}
            />
          </Form.Item>
          <Form.Item name="parameterSpace" label={t(SUBMIT_FORM_PARAMETER_SPACE_KEY)} rules={[{ required: true, message: t(SUBMIT_FORM_PARAMETER_SPACE_REQUIRED_KEY) }]}>
            <Input.TextArea rows={8} />
          </Form.Item>
          <Space size="large" wrap>
            <Form.Item name="searchMethod" label={t(SUBMIT_FORM_SEARCH_METHOD_KEY)}>
              <Select style={{ width: 160 }} options={[{ value: 'grid', label: 'Grid' }, { value: 'random', label: 'Random' }]} />
            </Form.Item>
            <Form.Item name="maxCandidates" label={t(SUBMIT_FORM_MAX_CANDIDATES_KEY)}>
              <InputNumber min={1} max={50} />
            </Form.Item>
            <Form.Item name="objective" label={t(SUBMIT_FORM_OBJECTIVE_KEY)}>
              <Input style={{ width: 220 }} />
            </Form.Item>
          </Space>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading}>{t(SUBMIT_FORM_SUBMIT_KEY)}</Button>
          </Form.Item>
        </Form>
      </Card>

      <Card title={t(LIST_TITLE_KEY)}>
        <Table rowKey="id" size="small" dataSource={experiments} columns={experimentColumns} pagination={false} />
      </Card>

      <JobEventStreamCard jobId={selectedExperiment?.jobId} events={jobEvents} t={t} />

      <Card title={selectedExperiment ? t(CANDIDATES_TITLE_WITH_ID_KEY, { id: selectedExperiment.id }) : t(CANDIDATES_TITLE_KEY)}>
        <Table rowKey="id" size="small" loading={candidateLoading} dataSource={candidates} columns={candidateColumns} pagination={false} scroll={{ x: 1100 }} />
      </Card>
    </div>
  );
}
