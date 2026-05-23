import { useEffect, useState } from 'react';
import { Button, Card, Form, Input, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { strategyApi, type StrategyTemplate } from '@/client/strategy';
import { strategyAssetApi, type StrategyAsset } from '@/client/strategyAsset';
import { showError, showSuccess } from '@/utils/message';

const { Text, Title } = Typography;

export default function StrategyAssetPage() {
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
      showError('加载策略资产失败');
    }
  };

  useEffect(() => { void load(); }, []);

  const submit = async (values: { sourceTemplateId: string; name: string; description: string; visibility: string }) => {
    setLoading(true);
    try {
      await strategyAssetApi.submitReview(values);
      showSuccess('已提交策略资产');
      form.resetFields();
      await load();
    } catch {
      showError('提交策略资产失败');
    } finally {
      setLoading(false);
    }
  };

  const clone = async (row: StrategyAsset) => {
    try {
      const res = await strategyAssetApi.clone(row.id, `${row.name} 副本`);
      showSuccess(`已克隆为模板：${res.templateId}`);
      await load();
    } catch {
      showError('克隆策略资产失败');
    }
  };

  const columns: ColumnsType<StrategyAsset> = [
    { title: '名称', dataIndex: 'name' },
    { title: '可见性', dataIndex: 'visibility', render: v => <Tag>{v}</Tag> },
    { title: '审核状态', dataIndex: 'reviewStatus', render: v => <Tag color={v === 'approved' ? 'green' : 'blue'}>{v}</Tag> },
    { title: '克隆数', dataIndex: 'cloneCount' },
    { title: '版本', dataIndex: 'latestVersion' },
    { title: '说明', dataIndex: 'description', ellipsis: true },
    { title: '操作', render: (_, row) => <Button size="small" onClick={() => void clone(row)}>克隆为草稿</Button> },
  ];

  return (
    <div className="space-y-4">
      <div>
        <Title level={3}>策略资产库</Title>
        <Text type="secondary">资产发布、审核状态和克隆由后端维护，克隆结果是独立用户模板。</Text>
      </div>
      <Card title="提交资产">
        <Form form={form} layout="vertical" onFinish={submit} initialValues={{ visibility: 'private' }}>
          <Form.Item name="sourceTemplateId" label="来源模板" rules={[{ required: true, message: '请选择来源模板' }]}>
            <Select options={templates.map(t => ({ value: t.id, label: t.name || t.id }))} />
          </Form.Item>
          <Form.Item name="name" label="资产名称" rules={[{ required: true, message: '请输入资产名称' }]}><Input /></Form.Item>
          <Form.Item name="description" label="说明"><Input.TextArea rows={3} /></Form.Item>
          <Space wrap>
            <Form.Item name="visibility" label="可见性"><Select style={{ width: 160 }} options={[{ value: 'private', label: 'private' }, { value: 'public', label: 'public' }]} /></Form.Item>
          </Space>
          <Form.Item><Button type="primary" htmlType="submit" loading={loading}>提交</Button></Form.Item>
        </Form>
      </Card>
      <Card title="资产列表">
        <Table rowKey="id" dataSource={assets} columns={columns} size="small" pagination={false} />
      </Card>
    </div>
  );
}
