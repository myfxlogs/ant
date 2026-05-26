import { useState } from 'react';
import { Alert, Button, Card, Descriptions, Form, Input, InputNumber, Select, Space, Tag, Typography } from 'antd';
import { marketRegimeApi, type MarketRegime } from '@/client/marketRegime';
import { showError, showSuccess } from '@/utils/message';
import { useTranslation } from 'react-i18next';

const { Text, Title } = Typography;

export default function MarketRegimePage() {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [result, setResult] = useState<MarketRegime | null>(null);
  const [loading, setLoading] = useState(false);

  const detect = async (values: { accountId: string; symbol: string; timeframe: string; count: number }) => {
    setLoading(true);
    try {
      const row = await marketRegimeApi.detect(values);
      setResult(row);
      showSuccess(t('strategy.marketRegime.detectSuccess'));
    } catch {
      showError(t('strategy.marketRegime.detectFailed'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-4">
      <div>
        <Title level={3}>{t('strategy.marketRegime.title')}</Title>
        <Text type="secondary">{t('strategy.marketRegime.subtitle')}</Text>
      </div>

      <Alert type="info" showIcon message={t('strategy.marketRegime.ruleVersionAlert')} />

      <Card title={t('strategy.marketRegime.form.title')}>
        <Form form={form} layout="vertical" onFinish={detect} initialValues={{ timeframe: 'M15', count: 120 }}>
          <Form.Item name="accountId" label={t('strategy.marketRegime.form.accountId')} rules={[{ required: true, message: t('strategy.marketRegime.form.accountIdRequired') }]}>
            <Input placeholder={t('strategy.marketRegime.form.accountIdPlaceholder')} />
          </Form.Item>
          <Space size="large" wrap>
            <Form.Item name="symbol" label={t('strategy.marketRegime.form.symbol')} rules={[{ required: true, message: t('strategy.marketRegime.form.symbolRequired') }]}>
              <Input style={{ width: 180 }} placeholder={t('strategy.marketRegime.form.symbolPlaceholder')} />
            </Form.Item>
            <Form.Item name="timeframe" label={t('strategy.marketRegime.form.timeframe')}>
              <Select style={{ width: 140 }} options={['M1', 'M5', 'M15', 'M30', 'H1', 'H4', 'D1'].map(v => ({ value: v, label: v }))} />
            </Form.Item>
            <Form.Item name="count" label={t('strategy.marketRegime.form.klineCount')}>
              <InputNumber min={20} max={500} />
            </Form.Item>
          </Space>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading}>{t('strategy.marketRegime.form.submit')}</Button>
          </Form.Item>
        </Form>
      </Card>

      {result && (
        <Card title={t('strategy.marketRegime.result.title')}>
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label={t('strategy.marketRegime.result.status')}><Tag color="blue">{result.regime}</Tag></Descriptions.Item>
            <Descriptions.Item label={t('strategy.marketRegime.result.confidence')}>{(result.confidence * 100).toFixed(1)}%</Descriptions.Item>
            <Descriptions.Item label={t('strategy.marketRegime.result.modelVersion')}>{result.modelVersion}</Descriptions.Item>
            <Descriptions.Item label={t('strategy.marketRegime.result.strategyFamilies')}>{result.strategyFamilies.map(item => <Tag key={item}>{item}</Tag>)}</Descriptions.Item>
            <Descriptions.Item label={t('strategy.marketRegime.result.features')}><Text code>{JSON.stringify(result.features)}</Text></Descriptions.Item>
            <Descriptions.Item label={t('strategy.marketRegime.result.recordId')}><Text copyable>{result.id}</Text></Descriptions.Item>
          </Descriptions>
        </Card>
      )}
    </div>
  );
}
