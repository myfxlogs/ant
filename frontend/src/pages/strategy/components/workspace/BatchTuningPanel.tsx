import { useEffect, useState } from 'react';
import { Button, Card, Form, Input, InputNumber, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { strategyApi, type StrategyTemplate } from '@/client/strategy';
import { strategyExperimentApi, type StrategyExperiment, type StrategyExperimentCandidate } from '@/client/strategyExperiment';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import { showError, showSuccess } from '@/utils/message';
import { useTranslation } from 'react-i18next';
import {
  MESSAGES_CANDIDATES_GENERATED_KEY, MESSAGES_SUBMIT_FAILED_KEY, MESSAGES_DRAFT_GENERATED_KEY,
  MESSAGES_PROMOTE_FAILED_KEY, LIST_COLUMN_STATUS_KEY, LIST_COLUMN_SEARCH_METHOD_KEY,
  LIST_COLUMN_MAX_CANDIDATES_KEY, LIST_COLUMN_ACTIONS_KEY, LIST_COLUMN_VIEW_CANDIDATES_KEY,
  CANDIDATES_COLUMN_RANK_KEY, CANDIDATES_COLUMN_GRADE_KEY, CANDIDATES_COLUMN_SCORE_KEY,
  CANDIDATES_COLUMN_PARAMETERS_KEY, CANDIDATES_COLUMN_ACTIONS_KEY, CANDIDATES_COLUMN_GENERATE_DRAFT_KEY,
  SUBMIT_FORM_TITLE_KEY, SUBMIT_FORM_BASE_TEMPLATE_KEY, SUBMIT_FORM_BASE_TEMPLATE_PLACEHOLDER_KEY,
  SUBMIT_FORM_PARAMETER_SPACE_KEY, SUBMIT_FORM_SEARCH_METHOD_KEY, SUBMIT_FORM_MAX_CANDIDATES_KEY,
  SUBMIT_FORM_OBJECTIVE_KEY, SUBMIT_FORM_SUBMIT_KEY, LIST_TITLE_KEY,
  CANDIDATES_TITLE_KEY, CANDIDATES_TITLE_WITH_ID_KEY,
} from '@/gen/ant/v1/i18n/strategy_experiment_keys';

const { Text } = Typography;

type SubmitValues = {
  baseTemplateId: string;
  parameterSpace: string;
  searchMethod: string;
  maxCandidates: number;
  objective: string;
};

function defaultParamSpace() {
  return JSON.stringify({ fast_period: [8, 12, 16], slow_period: [24, 30], risk_pct: [0.5, 1] }, null, 2);
}

export default function BatchTuningPanel() {
  const { t } = useTranslation();
  const [selectedId, setSelectedId] = useState('');
  const [loading, setLoading] = useState(false);

  const { data: templates = [] } = useRpcQuery(['strategy', 'templates'], () => strategyApi.listTemplates());
  const { data: experiments = [], refetch } = useRpcQuery(['strategy', 'experiments'], () => strategyExperimentApi.list());
  const { data: candidates = [], isLoading: candLoading } = useRpcQuery(
    ['strategy', 'experiments', selectedId, 'candidates'],
    () => strategyExperimentApi.listCandidates(selectedId),
    { enabled: !!selectedId },
  );

  useEffect(() => {
    if (!selectedId && experiments.length > 0) setSelectedId(experiments[0].id);
  }, [experiments, selectedId]);

  const handleSubmit = async (values: SubmitValues) => {
    setLoading(true);
    try {
      const ps = JSON.parse(values.parameterSpace || '{}') as Record<string, unknown>;
      const res = await strategyExperimentApi.submit({
        baseTemplateId: values.baseTemplateId,
        parameterSpace: ps,
        searchMethod: values.searchMethod,
        maxCandidates: values.maxCandidates,
        objective: values.objective,
      });
      showSuccess(t(MESSAGES_CANDIDATES_GENERATED_KEY));
      await refetch();
      if (res.experiment?.id) setSelectedId(res.experiment.id);
    } catch {
      showError(t(MESSAGES_SUBMIT_FAILED_KEY));
    } finally {
      setLoading(false);
    }
  };

  const promote = async (c: StrategyExperimentCandidate) => {
    try {
      const res = await strategyExperimentApi.promoteCandidateToDraft(c.id, `Batch candidate ${c.rank}`);
      showSuccess(t(MESSAGES_DRAFT_GENERATED_KEY, { templateId: res.templateId }));
    } catch {
      showError(t(MESSAGES_PROMOTE_FAILED_KEY));
    }
  };

  const expColumns: ColumnsType<StrategyExperiment> = [
    { title: t(LIST_COLUMN_STATUS_KEY), dataIndex: 'status', render: v => <Tag color={v === 'SUCCEEDED' ? 'green' : 'blue'}>{v}</Tag> },
    { title: t(LIST_COLUMN_SEARCH_METHOD_KEY), dataIndex: 'searchMethod' },
    { title: t(LIST_COLUMN_MAX_CANDIDATES_KEY), dataIndex: 'maxCandidates' },
    { title: t(LIST_COLUMN_ACTIONS_KEY), render: (_, r) => <Button size="small" onClick={() => setSelectedId(r.id)}>{t(LIST_COLUMN_VIEW_CANDIDATES_KEY)}</Button> },
  ];

  const candColumns: ColumnsType<StrategyExperimentCandidate> = [
    { title: t(CANDIDATES_COLUMN_RANK_KEY), dataIndex: 'rank', width: 60 },
    { title: t(CANDIDATES_COLUMN_GRADE_KEY), dataIndex: 'grade', width: 60, render: v => <Tag color={v === 'A' ? 'gold' : v === 'B' ? 'blue' : 'default'}>{v}</Tag> },
    { title: t(CANDIDATES_COLUMN_SCORE_KEY), dataIndex: 'score', width: 70, render: v => Number(v).toFixed(1) },
    { title: t(CANDIDATES_COLUMN_PARAMETERS_KEY), dataIndex: 'parameters', render: v => <Text code style={{ fontSize: 10 }}>{JSON.stringify(v)}</Text> },
    { title: t(CANDIDATES_COLUMN_ACTIONS_KEY), width: 100, render: (_, r) => <Button size="small" type="primary" onClick={() => promote(r)}>{t(CANDIDATES_COLUMN_GENERATE_DRAFT_KEY)}</Button> },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <Card size="small" title={t(SUBMIT_FORM_TITLE_KEY)}>
        <Form layout="vertical" size="small" initialValues={{ parameterSpace: defaultParamSpace(), searchMethod: 'grid', maxCandidates: 12, objective: 'balanced' }} onFinish={handleSubmit}>
          <Form.Item name="baseTemplateId" label={t(SUBMIT_FORM_BASE_TEMPLATE_KEY)} rules={[{ required: true }]}>
            <Select showSearch placeholder={t(SUBMIT_FORM_BASE_TEMPLATE_PLACEHOLDER_KEY)}
              options={templates.map((t: StrategyTemplate) => ({ value: t.id, label: `${t.name || t.id}` }))} />
          </Form.Item>
          <Form.Item name="parameterSpace" label={t(SUBMIT_FORM_PARAMETER_SPACE_KEY)} rules={[{ required: true }]}>
            <Input.TextArea rows={4} style={{ fontFamily: 'monospace', fontSize: 11 }} />
          </Form.Item>
          <Space size="small" wrap>
            <Form.Item name="searchMethod" label={t(SUBMIT_FORM_SEARCH_METHOD_KEY)}>
              <Select style={{ width: 120 }} options={[
                { value: 'grid', label: t('strategy.tuning.searchMethod.grid', { defaultValue: 'Grid' }) },
                { value: 'random', label: t('strategy.tuning.searchMethod.random', { defaultValue: 'Random' }) },
              ]} />
            </Form.Item>
            <Form.Item name="maxCandidates" label={t(SUBMIT_FORM_MAX_CANDIDATES_KEY)}>
              <InputNumber min={1} max={50} />
            </Form.Item>
            <Form.Item name="objective" label={t(SUBMIT_FORM_OBJECTIVE_KEY)}>
              <Input style={{ width: 140 }} />
            </Form.Item>
          </Space>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} size="small">{t(SUBMIT_FORM_SUBMIT_KEY)}</Button>
          </Form.Item>
        </Form>
      </Card>

      <Card size="small" title={t(LIST_TITLE_KEY)}>
        <Table rowKey="id" size="small" dataSource={experiments} columns={expColumns} pagination={false} scroll={{ x: 400 }} />
      </Card>

      <Card size="small" title={selectedId ? t(CANDIDATES_TITLE_WITH_ID_KEY, { id: selectedId }) : t(CANDIDATES_TITLE_KEY)}>
        <Table rowKey="id" size="small" loading={candLoading} dataSource={candidates} columns={candColumns} pagination={false} scroll={{ x: 600 }} />
      </Card>
    </div>
  );
}
