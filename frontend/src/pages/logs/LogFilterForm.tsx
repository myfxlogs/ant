import { Form, DatePicker, Select, Button, Space, Input } from 'antd';
import { useTranslation } from 'react-i18next'
import { ACCOUNT_ID_KEY, DATE_RANGE_KEY, EXAMPLE_SYMBOL_PLACEHOLDER_KEY, FAILED_KEY, REQUEST_ID_KEY, RESET_KEY, RESULT_KEY, RISK_CODE_KEY, RISK_LOG_QUICK_FILTER_KEY, SEARCH_KEY, STATUS_KEY, SUCCESS_KEY, SYMBOL_KEY, TRIGGER_SOURCE_KEY } from '@/gen/ant/v1/i18n/logs_keys';
;

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
        <Form.Item name="dateRange" label={t(DATE_RANGE_KEY)}>
          <RangePicker style={{ width: 240 }} />
        </Form.Item>
        {activeTab === 'connection' && (
          <Form.Item name="status" label={t(STATUS_KEY)}>
            <Select style={{ width: 120 }} allowClear>
              <Select.Option value="success">{t(SUCCESS_KEY)}</Select.Option>
              <Select.Option value="failed">{t(FAILED_KEY)}</Select.Option>
            </Select>
          </Form.Item>
        )}
        {(activeTab === 'execution' || activeTab === 'orders') && (
          <Form.Item name="symbol" label={t(SYMBOL_KEY)}>
            <Input style={{ width: 120 }} placeholder={t(EXAMPLE_SYMBOL_PLACEHOLDER_KEY)} />
          </Form.Item>
        )}
        {activeTab === 'operations' && (
          <>
            <Form.Item name="module" hidden><Input /></Form.Item>
            <Form.Item name="action" hidden><Input /></Form.Item>
            <Form.Item>
              <Button onClick={onQuickRiskFilter}>{t(RISK_LOG_QUICK_FILTER_KEY)}</Button>
            </Form.Item>
            <Form.Item label={t(RISK_CODE_KEY)}>
              <Input style={{ width: 220 }} placeholder="RISK_MARGIN_INSUFFICIENT" value={opRiskCode} onChange={(e) => onRiskCodeChange(e.target.value)} />
            </Form.Item>
            <Form.Item label={t(REQUEST_ID_KEY)}>
              <Input style={{ width: 220 }} placeholder="request_id" value={opRequestId} onChange={(e) => onRequestIdChange(e.target.value)} />
            </Form.Item>
            <Form.Item label={t(TRIGGER_SOURCE_KEY)}>
              <Select allowClear style={{ width: 130 }} value={opTriggerSource || undefined} onChange={(v) => onTriggerSourceChange(v || '')}
                options={[{ label: 'manual', value: 'manual' }, { label: 'strategy', value: 'strategy' }, { label: 'recovery', value: 'recovery' }]} />
            </Form.Item>
            <Form.Item label={t(RESULT_KEY)}>
              <Select allowClear style={{ width: 120 }} value={opResult || undefined} onChange={(v) => onResultChange(v || '')}
                options={[{ label: 'PASS', value: 'pass' }, { label: 'REJECT', value: 'reject' }]} />
            </Form.Item>
          </>
        )}
        <Form.Item name="accountId" label={t(ACCOUNT_ID_KEY)}>
          <Input style={{ width: 200 }} placeholder={t(ACCOUNT_ID_KEY)} />
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" onClick={onSearch}>{t(SEARCH_KEY)}</Button>
            <Button onClick={onReset}>{t(RESET_KEY)}</Button>
          </Space>
        </Form.Item>
      </Space>
    </Form>
  );
}
