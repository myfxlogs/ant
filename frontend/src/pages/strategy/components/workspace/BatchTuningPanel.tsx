import { useEffect, useMemo, useState } from 'react';
import { Button, Card, Form, Input, InputNumber, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { strategyApi, type StrategyTemplate } from '@/client/strategy';
import { strategyExperimentApi, type StrategyExperiment, type StrategyExperimentCandidate } from '@/client/strategyExperiment';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import { showError, showSuccess } from '@/utils/message';
import { useTranslation } from 'react-i18next';

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
      showSuccess(t('strategy.experiment.messages.candidatesGenerated'));
      await refetch();
      if (res.experiment?.id) setSelectedId(res.experiment.id);
    } catch {
      showError(t('strategy.experiment.messages.submitFailed'));
    } finally {
      setLoading(false);
    }
  };

  const promote = async (c: StrategyExperimentCandidate) => {
    try {
      const res = await strategyExperimentApi.promoteCandidateToDraft(c.id, `Batch candidate ${c.rank}`);
      showSuccess(t('strategy.experiment.messages.draftGenerated', { templateId: res.templateId }));
    } catch {
      showError(t('strategy.experiment.messages.promoteFailed'));
    }
  };

  const expColumns: ColumnsType<StrategyExperiment> = [
    { title: t('strategy.experiment.list.status'), dataIndex: 'status', render: v => <Tag color={v === 'SUCCEEDED' ? 'green' : 'blue'}>{v}</Tag> },
    { title: t('strategy.experiment.list.searchMethod'), dataIndex: 'searchMethod' },
    { title: t('strategy.experiment.list.maxCandidates'), dataIndex: 'maxCandidates' },
    { title: t('strategy.experiment.list.actions'), render: (_, r) => <Button size="small" onClick={() => setSelectedId(r.id)}>{t('strategy.experiment.list.viewCandidates')}</Button> },
  ];

  const candColumns: ColumnsType<StrategyExperimentCandidate> = [
    { title: t('strategy.experiment.candidates.rank'), dataIndex: 'rank', width: 60 },
    { title: t('strategy.experiment.candidates.grade'), dataIndex: 'grade', width: 60, render: v => <Tag color={v === 'A' ? 'gold' : v === 'B' ? 'blue' : 'default'}>{v}</Tag> },
    { title: t('strategy.experiment.candidates.score'), dataIndex: 'score', width: 70, render: v => Number(v).toFixed(1) },
    { title: t('strategy.experiment.candidates.parameters'), dataIndex: 'parameters', render: v => <Text code style={{ fontSize: 10 }}>{JSON.stringify(v)}</Text> },
    { title: t('strategy.experiment.candidates.actions'), width: 100, render: (_, r) => <Button size="small" type="primary" onClick={() => promote(r)}>{t('strategy.experiment.candidates.generateDraft')}</Button> },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <Card size="small" title={t('strategy.experiment.submit.title')}>
        <Form layout="vertical" size="small" initialValues={{ parameterSpace: defaultParamSpace(), searchMethod: 'grid', maxCandidates: 12, objective: 'balanced' }} onFinish={handleSubmit}>
          <Form.Item name="baseTemplateId" label={t('strategy.experiment.submit.baseTemplate')} rules={[{ required: true }]}>
            <Select showSearch placeholder={t('strategy.experiment.submit.baseTemplatePlaceholder')}
              options={templates.map((t: StrategyTemplate) => ({ value: t.id, label: `${t.name || t.id}` }))} />
          </Form.Item>
          <Form.Item name="parameterSpace" label={t('strategy.experiment.submit.parameterSpace')} rules={[{ required: true }]}>
            <Input.TextArea rows={4} style={{ fontFamily: 'monospace', fontSize: 11 }} />
          </Form.Item>
          <Space size="small" wrap>
            <Form.Item name="searchMethod" label={t('strategy.experiment.submit.searchMethod')}>
              <Select style={{ width: 120 }} options={[{ value: 'grid', label: 'Grid' }, { value: 'random', label: 'Random' }]} />
            </Form.Item>
            <Form.Item name="maxCandidates" label={t('strategy.experiment.submit.maxCandidates')}>
              <InputNumber min={1} max={50} />
            </Form.Item>
            <Form.Item name="objective" label={t('strategy.experiment.submit.objective')}>
              <Input style={{ width: 140 }} />
            </Form.Item>
          </Space>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} size="small">{t('strategy.experiment.submit.submit')}</Button>
          </Form.Item>
        </Form>
      </Card>

      <Card size="small" title={t('strategy.experiment.list.title')}>
        <Table rowKey="id" size="small" dataSource={experiments} columns={expColumns} pagination={false} scroll={{ x: 400 }} />
      </Card>

      <Card size="small" title={selectedId ? t('strategy.experiment.candidates.titleWithId', { id: selectedId }) : t('strategy.experiment.candidates.title')}>
        <Table rowKey="id" size="small" loading={candLoading} dataSource={candidates} columns={candColumns} pagination={false} scroll={{ x: 600 }} />
      </Card>
    </div>
  );
}
