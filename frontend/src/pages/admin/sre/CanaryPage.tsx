import { useState, useEffect, useCallback } from 'react';
import { Card, Table, Button, Modal, Form, Input, InputNumber, Space, Typography, Tag, Popconfirm } from 'antd';
import { PlusOutlined, ReloadOutlined, ExperimentOutlined, DeleteOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { sreApi, type CanaryConfig } from './sreApi';

const { Text, Title } = Typography;

export default function CanaryPage() {
  const { t } = useTranslation();
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
      await sreApi.canarySet(values.strategyId || '', values.versionTag || '', values.durationDays || 0);
      setModalOpen(false); form.resetFields(); fetchCanaries();
    } finally { setSubmitLoading(false); }
  };

  const handleDelete = async (strategyId: string) => {
    await sreApi.canaryDelete(strategyId);
    fetchCanaries();
  };

  const columns = [
    { title: t('sre.canary.columns.strategyId', { defaultValue: 'Strategy ID' }), dataIndex: 'strategy_id', key: 'id', width: 180, render: (v: string) => <Text code>{v}</Text> },
    { title: t('sre.canary.columns.versionTag', { defaultValue: 'Version Tag' }), dataIndex: 'version_tag', key: 'version', width: 100 },
    {
      title: t('sre.canary.columns.accounts', { defaultValue: 'Canary Accounts' }), dataIndex: 'account_ids', key: 'accounts',
      render: (v: string[]) => v?.length ? v.map(id => <Tag key={id}>{id}</Tag>) : '-',
    },
    { title: t('sre.canary.columns.startAt', { defaultValue: 'Start At' }), dataIndex: 'start_at', key: 'start', width: 160 },
    { title: t('sre.canary.columns.days', { defaultValue: 'Days' }), dataIndex: 'duration_days', key: 'days', width: 70, render: (v: number) => `${v}d` },
    {
      title: t('sre.canary.columns.status', { defaultValue: 'Status' }), dataIndex: 'promoted', key: 'promoted', width: 90,
      render: (v: boolean) => v ? <Tag color="green">{t('sre.canary.promoted', { defaultValue: 'Promoted' })}</Tag> : <Tag color="blue">{t('sre.canary.canarying', { defaultValue: 'Canary' })}</Tag>,
    },
    {
      title: '', key: 'actions', width: 80,
      render: (_: unknown, record: CanaryConfig) => (
        <Popconfirm title={t('sre.canary.confirmDelete', { defaultValue: 'Delete this canary config?' })} onConfirm={() => handleDelete(record.strategy_id)} okText={t('common.confirm', { defaultValue: 'Confirm' })} cancelText={t('common.cancel', { defaultValue: 'Cancel' })}>
          <Button size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ];

  return (
    <div style={{ maxWidth: 960 }}>
      <Title level={4}><ExperimentOutlined style={{ marginRight: 8 }} />{t('sre.canary.title', { defaultValue: 'Canary Configuration' })}</Title>
      <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
        {t('sre.canary.description', { defaultValue: 'New strategy versions run on a few accounts for N days before promotion to all' })}
      </Text>

      <Card size="small" extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={fetchCanaries} loading={loading}>{t('common.refresh', { defaultValue: 'Refresh' })}</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>{t('sre.canary.newCanary', { defaultValue: 'New Canary' })}</Button>
        </Space>
      }>
        <Table dataSource={canaries} columns={columns} rowKey="strategy_id"
          loading={loading} size="small" pagination={false}
          locale={{ emptyText: t('sre.canary.noCanaries', { defaultValue: 'No canary configs' }) }}
        />
      </Card>

      <Modal title={t('sre.canary.newCanaryTitle', { defaultValue: 'New Canary' })} open={modalOpen}
        onOk={handleSet} onCancel={() => setModalOpen(false)} confirmLoading={submitLoading}
      >
        <Form form={form} layout="vertical" size="small">
          <Form.Item name="strategy_id" label={t('sre.canary.columns.strategyId', { defaultValue: 'Strategy ID' })} rules={[{ required: true }]}>
            <Input placeholder="strategy-uuid" />
          </Form.Item>
          <Form.Item name="version_tag" label={t('sre.canary.columns.versionTag', { defaultValue: 'Version Tag' })} rules={[{ required: true }]}>
            <Input placeholder="v1.2.0-canary" />
          </Form.Item>
          <Form.Item name="accountIds" label={t('sre.canary.accountIdsLabel', { defaultValue: 'Canary Account IDs (comma or newline separated)' })} rules={[{ required: true }]}>
            <Input.TextArea rows={2} placeholder={t('sre.canary.accountIdsPlaceholder', { defaultValue: 'account-1, account-2' })} />
          </Form.Item>
          <Form.Item name="duration_days" label={t('sre.canary.durationDays', { defaultValue: 'Canary Days' })} rules={[{ required: true }]}>
            <InputNumber min={1} max={90} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
