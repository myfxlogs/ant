import { useState, useEffect, useCallback } from 'react';
import { Card, Table, Button, Modal, Form, Input, InputNumber, Space, Typography, Tag, Popconfirm } from 'antd';
import { PlusOutlined, ReloadOutlined, ExperimentOutlined, DeleteOutlined } from '@ant-design/icons';
import { sreApi, type CanaryConfig } from './sreApi';

const { Text, Title } = Typography;

export default function CanaryPage() {
  const [canaries, setCanaries] = useState<CanaryConfig[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [submitLoading, setSubmitLoading] = useState(false);
  const [form] = Form.useForm();

  const fetchCanaries = useCallback(async () => {
    setLoading(true);
    try { setCanaries(await sreApi.canaryList()); } catch { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchCanaries(); }, [fetchCanaries]);

  const handleSet = async () => {
    const values = await form.validateFields();
    setSubmitLoading(true);
    try {
      const accountIds = (values.accountIds || '').split(/[\n,]+/).map((s: string) => s.trim()).filter(Boolean);
      await sreApi.canarySet({ ...values, account_ids: accountIds });
      setModalOpen(false); form.resetFields(); fetchCanaries();
    } finally { setSubmitLoading(false); }
  };

  const handleDelete = async (strategyId: string) => {
    await sreApi.canaryDelete(strategyId);
    fetchCanaries();
  };

  const columns = [
    { title: '策略 ID', dataIndex: 'strategy_id', key: 'id', width: 180, render: (v: string) => <Text code>{v}</Text> },
    { title: '版本 Tag', dataIndex: 'version_tag', key: 'version', width: 100 },
    {
      title: '灰度账户', dataIndex: 'account_ids', key: 'accounts',
      render: (v: string[]) => v?.length ? v.map(id => <Tag key={id}>{id}</Tag>) : '-',
    },
    { title: '开始时间', dataIndex: 'start_at', key: 'start', width: 160 },
    { title: '天数', dataIndex: 'duration_days', key: 'days', width: 70, render: (v: number) => `${v}d` },
    {
      title: '状态', dataIndex: 'promoted', key: 'promoted', width: 90,
      render: (v: boolean) => v ? <Tag color="green">已推广</Tag> : <Tag color="blue">灰度中</Tag>,
    },
    {
      title: '', key: 'actions', width: 80,
      render: (_: unknown, record: CanaryConfig) => (
        <Popconfirm title="确认删除此灰度配置？" onConfirm={() => handleDelete(record.strategy_id)} okText="确认" cancelText="取消">
          <Button size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ];

  return (
    <div style={{ maxWidth: 960 }}>
      <Title level={4}><ExperimentOutlined style={{ marginRight: 8 }} />Canary 灰度配置</Title>
      <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
        新策略版本在少量账户上试运行 N 天后推广至全量
      </Text>

      <Card size="small" extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={fetchCanaries} loading={loading}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>新建灰度</Button>
        </Space>
      }>
        <Table dataSource={canaries} columns={columns} rowKey="strategy_id"
          loading={loading} size="small" pagination={false}
          locale={{ emptyText: '暂无灰度配置' }}
        />
      </Card>

      <Modal title="新建 Canary 灰度" open={modalOpen}
        onOk={handleSet} onCancel={() => setModalOpen(false)} confirmLoading={submitLoading}
      >
        <Form form={form} layout="vertical" size="small">
          <Form.Item name="strategy_id" label="策略 ID" rules={[{ required: true }]}>
            <Input placeholder="strategy-uuid" />
          </Form.Item>
          <Form.Item name="version_tag" label="版本 Tag" rules={[{ required: true }]}>
            <Input placeholder="v1.2.0-canary" />
          </Form.Item>
          <Form.Item name="accountIds" label="灰度账户 ID (逗号或换行分隔)" rules={[{ required: true }]}>
            <Input.TextArea rows={2} placeholder="account-1, account-2" />
          </Form.Item>
          <Form.Item name="duration_days" label="灰度天数" rules={[{ required: true }]}>
            <InputNumber min={1} max={90} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
