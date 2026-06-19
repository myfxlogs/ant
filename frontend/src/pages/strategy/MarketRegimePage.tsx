import { Alert, Button, Card, Descriptions, Form, InputNumber, Select, Space, Tag, Typography } from 'antd';
import { showError, showSuccess } from '@/utils/message';
import { useTranslation } from 'react-i18next'
import { NO_ACCOUNTS_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';

;
import { TIMEFRAMES } from '@/constants/timeframes';
import SymbolPicker from '@/components/chart/SymbolPicker';
import { useMarketRegimeForm } from './hooks/useMarketRegimeForm';

const { Text, Title } = Typography;

export default function MarketRegimePage() {
  const { t } = useTranslation();
  const {
    form, activeAccounts, watchedAccountId,
    result, loading, handleAccountChange, detect,
  } = useMarketRegimeForm();

  const handleSubmit = async () => {
    try {
      const row = await detect();
      if (row) showSuccess(t(DETECT_SUCCESS_KEY));
    } catch {
      showError(t(DETECT_FAILED_KEY));
    }
  };

  return (
    <div className="space-y-4">
      <div>
        <Title level={3}>{t(TITLE_KEY)}</Title>
        <Text type="secondary">{t(SUBTITLE_KEY)}</Text>
      </div>

      <Alert type="info" showIcon message={t(RULE_VERSION_ALERT_KEY)} />

      <Card title={t(FORM_TITLE_KEY)}>
        <Form form={form} layout="vertical" onFinish={handleSubmit} initialValues={{ timeframe: '15m' as const, count: 120 }}>
          <Form.Item name="accountId" label={t(FORM_ACCOUNT_ID_KEY)} rules={[{ required: true, message: t(FORM_ACCOUNT_ID_REQUIRED_KEY) }]}>
            <Select
              showSearch
              placeholder={t(FORM_ACCOUNT_ID_PLACEHOLDER_KEY)}
              onChange={handleAccountChange}
              optionFilterProp="label"
              notFoundContent={t(NO_ACCOUNTS_KEY)}
              options={activeAccounts.map((a) => ({ value: a.id, label: `${a.brokerServer} · ${a.login}` }))}
              style={{ width: '30%' }}
            />
          </Form.Item>
          <Space size="large" wrap>
            <Form.Item name="symbol" label={t(FORM_SYMBOL_KEY)} rules={[{ required: true, message: t(FORM_SYMBOL_REQUIRED_KEY) }]}>
              <SymbolPicker accountId={watchedAccountId ?? ''} placeholder={t(FORM_SYMBOL_PLACEHOLDER_KEY)} style={{ width: 180 }} />
            </Form.Item>
            <Form.Item name="timeframe" label={t(FORM_TIMEFRAME_KEY)}>
              <Select style={{ width: 140 }} options={TIMEFRAMES.map((v) => ({ value: v, label: v }))} />
            </Form.Item>
            <Form.Item name="count" label={t(FORM_KLINE_COUNT_KEY)}>
              <InputNumber min={20} max={500} />
            </Form.Item>
          </Space>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading}>{t(FORM_SUBMIT_KEY)}</Button>
          </Form.Item>
        </Form>
      </Card>

      {result && (
        <Card title={t(RESULT_TITLE_KEY)}>
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label={t(RESULT_STATUS_KEY)}><Tag color="blue">{result.regime}</Tag></Descriptions.Item>
            <Descriptions.Item label={t(RESULT_CONFIDENCE_KEY)}>{(result.confidence * 100).toFixed(1)}%</Descriptions.Item>
            <Descriptions.Item label={t(RESULT_MODEL_VERSION_KEY)}>{result.modelVersion}</Descriptions.Item>
            <Descriptions.Item label={t(RESULT_STRATEGY_FAMILIES_KEY)}>{(result.strategyFamilies ?? []).map((item) => <Tag key={item}>{item}</Tag>)}</Descriptions.Item>
            <Descriptions.Item label={t(RESULT_FEATURES_KEY)}><Text code>{result.features}</Text></Descriptions.Item>
            <Descriptions.Item label={t(RESULT_RECORD_ID_KEY)}><Text copyable>{result.id}</Text></Descriptions.Item>
          </Descriptions>
        </Card>
      )}
    </div>
  );
}
