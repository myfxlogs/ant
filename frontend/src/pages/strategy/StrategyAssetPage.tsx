import { useMemo, useState } from 'react';
import { Button, Card, Form, Input, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { strategyApi, type StrategyTemplate } from '@/client/strategy';
import { strategyAssetApi, type StrategyAsset } from '@/client/strategyAsset';
import { showError, showSuccess } from '@/utils/message';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import { StatusResult } from '@/components/common/StatusResult';
import { useTranslation } from 'react-i18next'
import { COPY_SUFFIX_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

;

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
      showSuccess(t(MESSAGES_SUBMIT_SUCCESS_KEY));
      form.resetFields();
      refetch();
    } catch {
      showError(t(MESSAGES_SUBMIT_FAILED_KEY));
    } finally {
      setSubmitting(false);
    }
  };

  const clone = async (row: StrategyAsset) => {
    try {
      const res = await strategyAssetApi.clone(row.id, `${row.name} ${t(COPY_SUFFIX_KEY)}`);
      showSuccess(t(MESSAGES_CLONE_SUCCESS_KEY, { templateId: res.templateId }));
      refetch();
    } catch {
      showError(t(MESSAGES_CLONE_FAILED_KEY));
    }
  };

  const columns: ColumnsType<StrategyAsset> = [
    { title: t(NAME_KEY), dataIndex: 'name' },
    { title: t(VISIBILITY_KEY), dataIndex: 'visibility', render: v => <Tag>{v}</Tag> },
    { title: t(REVIEW_STATUS_KEY), dataIndex: 'reviewStatus', render: v => <Tag color={v === 'approved' ? 'green' : 'blue'}>{v}</Tag> },
    { title: t(CLONE_COUNT_KEY), dataIndex: 'cloneCount' },
    { title: t(VERSION_KEY), dataIndex: 'latestVersion' },
    { title: t(DESCRIPTION_KEY), dataIndex: 'description', ellipsis: true },
    { title: t(ACTIONS_KEY), render: (_, row) => <Button size="small" onClick={() => void clone(row)}>{t(CLONE_AS_DRAFT_KEY)}</Button> },
  ];

  const templateOptions = useMemo(
    () => (templates as StrategyTemplate[]).map(tpl => ({ value: tpl.id, label: tpl.name || tpl.id })),
    [templates],
  );

  return (
    <div className="space-y-4">
      <div>
        <Title level={3}>{t(TITLE_KEY)}</Title>
        <Text type="secondary">{t(SUBTITLE_KEY)}</Text>
      </div>
      <Card title={t(SUBMIT_ASSET_KEY)}>
        <Form form={form} layout="vertical" onFinish={submit} initialValues={{ visibility: 'private' }}>
          <Form.Item name="sourceTemplateId" label={t(SOURCE_TEMPLATE_KEY)} rules={[{ required: true, message: t(VALIDATION_SELECT_TEMPLATE_KEY) }]}>
            <Select options={templateOptions} loading={templatesLoading} />
          </Form.Item>
          <Form.Item name="name" label={t(ASSET_NAME_KEY)} rules={[{ required: true, message: t(VALIDATION_ENTER_NAME_KEY) }]}><Input /></Form.Item>
          <Form.Item name="description" label={t(DESCRIPTION_KEY)}><Input.TextArea rows={3} /></Form.Item>
          <Space wrap>
            <Form.Item name="visibility" label={t(VISIBILITY_KEY)}><Select style={{ width: 160 }} options={[{ value: 'private', label: 'private' }, { value: 'public', label: 'public' }]} /></Form.Item>
          </Space>
          <Form.Item><Button type="primary" htmlType="submit" loading={submitting}>{t(SUBMIT_KEY)}</Button></Form.Item>
        </Form>
      </Card>
      <Card title={t(ASSET_LIST_KEY)}>
        <StatusResult
          loading={isLoading}
          error={error instanceof Error ? error.message : null}
          empty={!isLoading && !error && (assets as StrategyAsset[]).length === 0}
          emptyText={t(EMPTY_KEY, { defaultValue: 'No strategy assets yet' })}
          onRetry={refetch}
        >
          <Table rowKey="id" dataSource={assets as StrategyAsset[]} columns={columns} size="small" pagination={false} />
        </StatusResult>
      </Card>
    </div>
  );
}
