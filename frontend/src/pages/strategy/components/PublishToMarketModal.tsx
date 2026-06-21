import { useState, useEffect } from 'react';
import { Modal, Form, Input, Select, InputNumber, Switch, Tag, Typography, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { create } from '@bufbuild/protobuf';
import { PublishStrategyRequestSchema } from '@/gen/ant/v1/marketplace_service_pb';
import type { BacktestSnapshot } from '@/gen/ant/v1/marketplace_service_pb';
import type { StrategyTemplate } from '@/client/strategy';

const { TextArea } = Input;
const { Text } = Typography;

interface Props {
  open: boolean;
  template: StrategyTemplate | null;
  /** Latest backtest metrics to optionally include as snapshot. */
  backtestSnapshot?: BacktestSnapshot | null;
  onClose: () => void;
  onPublished: () => void;
}

const ASSET_CLASSES = ['forex', 'crypto', 'commodity', 'index', 'stock'];
const RISK_LEVELS = ['low', 'medium', 'high'];
const PRICE_MODELS = [
  { value: 'free', label: 'Free' },
  { value: 'monthly', label: 'Monthly Subscription' },
  { value: 'once', label: 'One-Time Purchase' },
];

export default function PublishToMarketModal({ open, template, backtestSnapshot, onClose, onPublished }: Props) {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);
  const [includeSnapshot, setIncludeSnapshot] = useState(false);
  const [tags, setTags] = useState<string[]>([]);

  useEffect(() => {
    if (open && template) {
      form.setFieldsValue({
        title: String(template.name || ''),
        description: String(template.description || ''),
        assetClass: 'forex',
        riskLevel: 'medium',
        priceModel: 'free',
        priceAmount: undefined as number | undefined,
        codeSnippet: '',
      });
      setTags([]);
      setIncludeSnapshot(!!backtestSnapshot);
    }
  }, [open, template, backtestSnapshot, form]);

  const handleSubmit = async () => {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      const msg = create(PublishStrategyRequestSchema, {
        userId: '', // filled by backend from auth context
        strategyId: String(template?.id || ''),
        title: values.title,
        description: values.description,
        priceModel: values.priceModel || 'free',
        priceAmount: values.priceAmount || 0,
        assetClass: values.assetClass || 'forex',
        symbols: [],
        timeframe: '',
        riskLevel: values.riskLevel || 'medium',
        tags,
        codeSnippet: values.codeSnippet || '',
        backtestSnapshot: includeSnapshot && backtestSnapshot ? backtestSnapshot : undefined,
      });
      await marketplaceClient.publishStrategy(msg);
      message.success(t('marketplace.messages.published', 'Strategy published to marketplace!'));
      onPublished();
      onClose();
    } catch (e: any) {
      message.error(e?.message || t('marketplace.messages.publishFailed', 'Failed to publish strategy'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title={t('marketplace.publish.title', 'Publish to Marketplace')}
      open={open}
      onCancel={onClose}
      onOk={handleSubmit}
      confirmLoading={submitting}
      width={560}
      destroyOnClose
    >
      <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
        <Form.Item name="title" label={t('marketplace.publish.titleLabel', 'Title')} rules={[{ required: true }]}>
          <Input maxLength={120} placeholder={t('marketplace.publish.titlePlaceholder', 'e.g. Golden Cross Strategy')} />
        </Form.Item>

        <Form.Item name="description" label={t('marketplace.publish.descriptionLabel', 'Description')} rules={[{ required: true }]}>
          <TextArea rows={3} maxLength={1000} placeholder={t('marketplace.publish.descriptionPlaceholder', 'Describe your strategy logic, entry/exit rules...')} />
        </Form.Item>

        <div style={{ display: 'flex', gap: 12 }}>
          <Form.Item name="assetClass" label={t('marketplace.publish.assetClass', 'Asset Class')} style={{ flex: 1 }}>
            <Select options={ASSET_CLASSES.map(v => ({ value: v, label: v }))} />
          </Form.Item>
          <Form.Item name="riskLevel" label={t('marketplace.publish.riskLevel', 'Risk Level')} style={{ flex: 1 }}>
            <Select options={RISK_LEVELS.map(v => ({ value: v, label: v }))} />
          </Form.Item>
        </div>

        <div style={{ display: 'flex', gap: 12 }}>
          <Form.Item name="priceModel" label={t('marketplace.publish.priceModel', 'Pricing')} style={{ flex: 1 }}>
            <Select options={PRICE_MODELS} />
          </Form.Item>
          <Form.Item name="priceAmount" label={t('marketplace.publish.priceAmount', 'Amount')} style={{ flex: 1 }}
            dependencies={['priceModel']}>
            <InputNumber min={0} step={1} style={{ width: '100%' }}
              disabled={Form.useWatch('priceModel', form) === 'free'} />
          </Form.Item>
        </div>

        <Form.Item label={t('marketplace.publish.tags', 'Tags')}>
          <Select
            mode="tags"
            placeholder={t('marketplace.publish.tagsPlaceholder', 'Type and press enter to add tags')}
            value={tags}
            onChange={setTags}
            style={{ width: '100%' }}
          />
        </Form.Item>

        <Form.Item name="codeSnippet" label={t('marketplace.publish.codeSnippet', 'Strategy Preview (public)')}>
          <TextArea rows={2} maxLength={500}
            placeholder={t('marketplace.publish.codeSnippetPlaceholder', 'Optional: share a snippet or high-level idea of your strategy (visible to all)')} />
        </Form.Item>

        {backtestSnapshot && (
          <div style={{ marginBottom: 16 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Switch checked={includeSnapshot} onChange={setIncludeSnapshot} size="small" />
              <Text>{t('marketplace.publish.includeBacktestSnapshot', 'Include latest backtest results')}</Text>
            </div>
            {includeSnapshot && (
              <div style={{ display: 'flex', gap: 12, marginTop: 8, flexWrap: 'wrap' }}>
                <Tag color="blue">Return: {((backtestSnapshot.totalReturn ?? 0) * 100).toFixed(2)}%</Tag>
                <Tag>Win Rate: {((backtestSnapshot.winRate ?? 0) * 100).toFixed(0)}%</Tag>
                <Tag>Trades: {backtestSnapshot.totalTrades}</Tag>
              </div>
            )}
          </div>
        )}
      </Form>
    </Modal>
  );
}
