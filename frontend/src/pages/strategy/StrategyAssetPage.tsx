import { useMemo, useState } from 'react';
import { Button, Card, Form, Input, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { strategyApi, type StrategyTemplate } from '@/client/strategy';
import { strategyAssetApi, type StrategyAsset } from '@/client/strategyAsset';
import { showError, showSuccess } from '@/utils/message';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import { StatusResult } from '@/components/common/StatusResult';
import { useTranslation } from 'react-i18next';

const { Text, Title } = Typography;

export default function StrategyAssetPage() {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);

  const {
    data: templates = [],
    isLoading: templatesLoading,
    error: templatesError,
    refetch: refetchTemplates,
  } = useRpcQuery(['strategy', 'templates'], () => strategyApi.listTemplates());

  const {
    data: assets = [],
    isLoading: assetsLoading,
    error: assetsError,
    refetch: refetchAssets,
  } = useRpcQuery(['strategy', 'assets'], () => strategyAssetApi.list());

  const isLoading = templatesLoading || assetsLoading;
  const error = templatesError || assetsError;
  const refetch = () => { refetchTemplates(); refetchAssets(); };

  const submit = async (values: { sourceTemplateId: string; name: string; description: string; visibility: string }) => {
    setSubmitting(true);
    try {
      await strategyAssetApi.submitReview(values);
      showSuccess(t('strategy.asset.messages.submitSuccess'));
      form.resetFields();
      refetch();
    } catch {
      showError(t('strategy.asset.messages.submitFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  const clone = async (row: StrategyAsset) => {
    try {
      const res = await strategyAssetApi.clone(row.id, `${row.name} ${t('strategy.templates.copySuffix')}`);
      showSuccess(t('strategy.asset.messages.cloneSuccess', { templateId: res.templateId }));
      refetch();
    } catch {
      showError(t('strategy.asset.messages.cloneFailed'));
    }
  };

  const columns: ColumnsType<StrategyAsset> = [
    { title: t('strategy.asset.name'), dataIndex: 'name' },
    { title: t('strategy.asset.visibility'), dataIndex: 'visibility', render: v => <Tag>{v}</Tag> },
    { title: t('strategy.asset.reviewStatus'), dataIndex: 'reviewStatus', render: v => <Tag color={v === 'approved' ? 'green' : 'blue'}>{v}</Tag> },
    { title: t('strategy.asset.cloneCount'), dataIndex: 'cloneCount' },
    { title: t('strategy.asset.version'), dataIndex: 'latestVersion' },
    { title: t('strategy.asset.description'), dataIndex: 'description', ellipsis: true },
    { title: t('strategy.asset.actions'), render: (_, row) => <Button size="small" onClick={() => void clone(row)}>{t('strategy.asset.cloneAsDraft')}</Button> },
  ];

  const templateOptions = useMemo(
    () => (templates as StrategyTemplate[]).map(tpl => ({ value: tpl.id, label: tpl.name || tpl.id })),
    [templates],
  );

  return (
    <div className="space-y-4">
      <div>
        <Title level={3}>{t('strategy.asset.title')}</Title>
        <Text type="secondary">{t('strategy.asset.subtitle')}</Text>
      </div>
      <Card title={t('strategy.asset.submitAsset')}>
        <Form form={form} layout="vertical" onFinish={submit} initialValues={{ visibility: 'private' }}>
          <Form.Item name="sourceTemplateId" label={t('strategy.asset.sourceTemplate')} rules={[{ required: true, message: t('strategy.asset.validation.selectTemplate') }]}>
            <Select options={templateOptions} loading={templatesLoading} />
          </Form.Item>
          <Form.Item name="name" label={t('strategy.asset.assetName')} rules={[{ required: true, message: t('strategy.asset.validation.enterName') }]}><Input /></Form.Item>
          <Form.Item name="description" label={t('strategy.asset.description')}><Input.TextArea rows={3} /></Form.Item>
          <Space wrap>
            <Form.Item name="visibility" label={t('strategy.asset.visibility')}><Select style={{ width: 160 }} options={[{ value: 'private', label: 'private' }, { value: 'public', label: 'public' }]} /></Form.Item>
          </Space>
          <Form.Item><Button type="primary" htmlType="submit" loading={submitting}>{t('strategy.asset.submit')}</Button></Form.Item>
        </Form>
      </Card>
      <Card title={t('strategy.asset.assetList')}>
        <StatusResult
          loading={isLoading}
          error={error instanceof Error ? error.message : null}
          empty={!isLoading && !error && (assets as StrategyAsset[]).length === 0}
          emptyText={t('strategy.asset.empty', { defaultValue: 'No strategy assets yet' })}
          onRetry={refetch}
        >
          <Table rowKey="id" dataSource={assets as StrategyAsset[]} columns={columns} size="small" pagination={false} />
        </StatusResult>
      </Card>
    </div>
  );
}
