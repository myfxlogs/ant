import { useState } from 'react';
import { Modal, Form, Input, Select, InputNumber, Button, Space, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';

const ASSET_CLASSES = ['forex', 'crypto', 'commodity', 'index', 'stock', 'other'] as const;

interface Props {
  open: boolean;
  userId: string;
  onClose: () => void;
  onPublished: () => void;
}

export default function PublishStrategyModal({ open, userId, onClose, onPublished }: Props) {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  const handlePublish = async (values: Record<string, unknown>) => {
    setLoading(true);
    try {
      await marketplaceClient.publishStrategy({
        userId,
        strategyId: values.strategyId as string,
        title: (values.title as string) || '',
        description: (values.description as string) || '',
        priceModel: (values.priceModel as string) || 'free',
        priceAmount: Number(values.priceAmount || 0),
        assetClass: (values.assetClass as string) || 'forex',
        symbols: values.symbols ? (values.symbols as string).split(',').map((s: string) => s.trim()) : [],
        timeframe: (values.timeframe as string) || 'H1',
        riskLevel: (values.riskLevel as string) || 'medium',
        tags: values.tags ? (values.tags as string).split(',').map((s: string) => s.trim()) : [],
      });
      message.success(t('marketplace.messages.published'));
      onClose();
      form.resetFields();
      onPublished();
    } catch {
      message.error(t('marketplace.messages.publishFailed'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal title={t('marketplace.publishModal.title')} open={open} onCancel={onClose} footer={null} destroyOnClose>
      <Form form={form} layout="vertical" onFinish={handlePublish}
        initialValues={{ priceModel: 'free', assetClass: 'forex', riskLevel: 'medium', timeframe: 'H1' }}>
        <Form.Item name="strategyId" label={t('marketplace.publishModal.strategyId')} rules={[{ required: true }]}>
          <Input placeholder="e.g. abc123-def4-..." />
        </Form.Item>
        <Form.Item name="title" label={t('marketplace.publishModal.titleField')} rules={[{ required: true }]}>
          <Input placeholder={t('marketplace.publishModal.titlePlaceholder')} />
        </Form.Item>
        <Form.Item name="description" label={t('marketplace.publishModal.description')}>
          <Input.TextArea rows={3} />
        </Form.Item>
        <Space style={{ width: '100%' }} size="middle">
          <Form.Item name="assetClass" label={t('marketplace.publishModal.assetClass')}>
            <Select options={ASSET_CLASSES.map(c => ({ value: c, label: t(`marketplace.assetClass.${c}`, { defaultValue: c }) }))} style={{ width: 140 }} />
          </Form.Item>
          <Form.Item name="riskLevel" label={t('marketplace.publishModal.riskLevel')}>
            <Select options={[
              { value: 'low', label: t('marketplace.risk.low') },
              { value: 'medium', label: t('marketplace.risk.medium') },
              { value: 'high', label: t('marketplace.risk.high') },
            ]} style={{ width: 120 }} />
          </Form.Item>
          <Form.Item name="timeframe" label={t('marketplace.publishModal.timeframe')}>
            <Select options={['M1','M5','M15','M30','H1','H4','D1'].map(v => ({ value: v, label: v }))} style={{ width: 90 }} />
          </Form.Item>
        </Space>
        <Space style={{ width: '100%' }} size="middle">
          <Form.Item name="priceModel" label={t('marketplace.publishModal.priceModel')}>
            <Select options={[
              { value: 'free', label: t('marketplace.priceModel.free') },
              { value: 'subscription', label: t('marketplace.priceModel.subscription') },
              { value: 'performance_fee', label: t('marketplace.priceModel.performanceFee') },
            ]} style={{ width: 160 }} />
          </Form.Item>
          <Form.Item name="priceAmount" label={t('marketplace.publishModal.priceAmount')}>
            <InputNumber min={0} step={1} style={{ width: 100 }} />
          </Form.Item>
        </Space>
        <Form.Item name="symbols" label={t('marketplace.publishModal.symbols')}>
          <Input placeholder="EURUSD, GBPUSD, XAUUSD" />
        </Form.Item>
        <Form.Item name="tags" label={t('marketplace.publishModal.tags')}>
          <Input placeholder="trend, scalping, mean-reversion" />
        </Form.Item>
        <Form.Item style={{ marginBottom: 0 }}>
          <Button type="primary" htmlType="submit" loading={loading} block>
            {t('marketplace.publishModal.submit')}
          </Button>
        </Form.Item>
      </Form>
    </Modal>
  );
}
