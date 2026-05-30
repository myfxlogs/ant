import { Form, DatePicker, Select, Button, Space, Input } from 'antd';
import { useTranslation } from 'react-i18next';

const { RangePicker } = DatePicker;

interface Props {
  activeTab: string;
  opRiskCode: string;
  opRequestId: string;
  opTriggerSource: string;
  opResult: string;
  onRiskCodeChange: (v: string) => void;
  onRequestIdChange: (v: string) => void;
  onTriggerSourceChange: (v: string) => void;
  onResultChange: (v: string) => void;
  onSearch: () => void;
  onReset: () => void;
  onQuickRiskFilter: () => void;
}

export default function LogFilterForm({
  activeTab, opRiskCode, opRequestId, opTriggerSource, opResult,
  onRiskCodeChange, onRequestIdChange, onTriggerSourceChange, onResultChange,
  onSearch, onReset, onQuickRiskFilter,
}: Props) {
  const { t } = useTranslation();
  const [form] = Form.useForm();

  return (
    <Form form={form} layout="inline" className="mb-4">
      <Space wrap>
        <Form.Item name="dateRange" label={t('logs.dateRange')}>
          <RangePicker style={{ width: 240 }} />
        </Form.Item>
        {activeTab === 'connection' && (
          <Form.Item name="status" label={t('logs.status')}>
            <Select style={{ width: 120 }} allowClear>
              <Select.Option value="success">{t('logs.success')}</Select.Option>
              <Select.Option value="failed">{t('logs.failed')}</Select.Option>
            </Select>
          </Form.Item>
        )}
        {(activeTab === 'execution' || activeTab === 'orders') && (
          <Form.Item name="symbol" label={t('logs.symbol')}>
            <Input style={{ width: 120 }} placeholder={t('logs.exampleSymbolPlaceholder')} />
          </Form.Item>
        )}
        {activeTab === 'operations' && (
          <>
            <Form.Item name="module" hidden><Input /></Form.Item>
            <Form.Item name="action" hidden><Input /></Form.Item>
            <Form.Item>
              <Button onClick={onQuickRiskFilter}>{t('logs.riskLogQuickFilter')}</Button>
            </Form.Item>
            <Form.Item label={t('logs.riskCode')}>
              <Input style={{ width: 220 }} placeholder="RISK_MARGIN_INSUFFICIENT" value={opRiskCode} onChange={(e) => onRiskCodeChange(e.target.value)} />
            </Form.Item>
            <Form.Item label={t('logs.requestId')}>
              <Input style={{ width: 220 }} placeholder="request_id" value={opRequestId} onChange={(e) => onRequestIdChange(e.target.value)} />
            </Form.Item>
            <Form.Item label={t('logs.triggerSource')}>
              <Select allowClear style={{ width: 130 }} value={opTriggerSource || undefined} onChange={(v) => onTriggerSourceChange(v || '')}
                options={[{ label: 'manual', value: 'manual' }, { label: 'strategy', value: 'strategy' }, { label: 'recovery', value: 'recovery' }]} />
            </Form.Item>
            <Form.Item label={t('logs.result')}>
              <Select allowClear style={{ width: 120 }} value={opResult || undefined} onChange={(v) => onResultChange(v || '')}
                options={[{ label: 'PASS', value: 'pass' }, { label: 'REJECT', value: 'reject' }]} />
            </Form.Item>
          </>
        )}
        <Form.Item name="accountId" label={t('logs.accountId')}>
          <Input style={{ width: 200 }} placeholder={t('logs.accountId')} />
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" onClick={onSearch}>{t('logs.search')}</Button>
            <Button onClick={onReset}>{t('logs.reset')}</Button>
          </Space>
        </Form.Item>
      </Space>
    </Form>
  );
}
