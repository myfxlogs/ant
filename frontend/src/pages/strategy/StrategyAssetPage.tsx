import { useEffect, useState } from 'react';
import { Button, Card, Form, Input, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { strategyApi, type StrategyTemplate } from '@/client/strategy';
import { strategyAssetApi, type StrategyAsset } from '@/client/strategyAsset';
import { showError, showSuccess } from '@/utils/message';
import { useTranslation } from 'react-i18next';

const { Text, Title } = Typography;

export default function StrategyAssetPage() {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [templates, setTemplates] = useState<StrategyTemplate[]>([]);
  const [assets, setAssets] = useState<StrategyAsset[]>([]);
  const [loading, setLoading] = useState(false);

  const load = async () => {
    try {
      const [tpls, rows] = await Promise.all([strategyApi.listTemplates(), strategyAssetApi.list()]);
      setTemplates(tpls);
      setAssets(rows);
    } catch {
      showError(t('strategy.asset.messages.loadFailed'));
    }
  };

  useEffect(() => { void load(); }, []);

  const submit = async (values: { sourceTemplateId: string; name: string; description: string; visibility: string }) => {
    setLoading(true);
    try {
      await strategyAssetApi.submitReview(values);
      showSuccess(t('strategy.asset.messages.submitSuccess'));
      form.resetFields();
      await load();
    } catch {
      showError(t('strategy.asset.messages.submitFailed'));
    } finally {
      setLoading(false);
    }
  };

  const clone = async (row: StrategyAsset) => {
    try {
      const res = await strategyAssetApi.clone(row.id, `${row.name} ${t('strategy.templates.copySuffix')}`);
      showSuccess(t('strategy.asset.messages.cloneSuccess', { templateId: res.templateId }));
      await load();
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

  return (
    <div className="space-y-4">
      <div>
        <Title level={3}>{t('strategy.asset.title')}</Title>
        <Text type="secondary">{t('strategy.asset.subtitle')}</Text>
      </div>
      <Card title={t('strategy.asset.submitAsset')}>
        <Form form={form} layout="vertical" onFinish={submit} initialValues={{ visibility: 'private' }}>
          <Form.Item name="sourceTemplateId" label={t('strategy.asset.sourceTemplate')} rules={[{ required: true, message: t('strategy.asset.validation.selectTemplate') }]}>
            <Select options={templates.map(tpl => ({ value: tpl.id, label: tpl.name || tpl.id }))} />
          </Form.Item>
          <Form.Item name="name" label={t('strategy.asset.assetName')} rules={[{ required: true, message: t('strategy.asset.validation.enterName') }]}><Input /></Form.Item>
          <Form.Item name="description" label={t('strategy.asset.description')}><Input.TextArea rows={3} /></Form.Item>
          <Space wrap>
            <Form.Item name="visibility" label={t('strategy.asset.visibility')}><Select style={{ width: 160 }} options={[{ value: 'private', label: 'private' }, { value: 'public', label: 'public' }]} /></Form.Item>
          </Space>
          <Form.Item><Button type="primary" htmlType="submit" loading={loading}>{t('strategy.asset.submit')}</Button></Form.Item>
        </Form>
      </Card>
      <Card title={t('strategy.asset.assetList')}>
        <Table rowKey="id" dataSource={assets} columns={columns} size="small" pagination={false} />
      </Card>
    </div>
  );
}
