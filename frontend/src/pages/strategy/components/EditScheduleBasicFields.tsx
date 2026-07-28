import { Col, Form, Input, InputNumber, Row, Select, Switch, Tooltip } from 'antd';
import { useTranslation } from 'react-i18next'
import { SCHEDULE_LAUNCH_FORM_ACCOUNT_KEY, SCHEDULE_LAUNCH_FORM_ENABLE_AFTER_CREATE_KEY, SCHEDULE_LAUNCH_FORM_HF_COOLDOWN_MS_KEY, SCHEDULE_LAUNCH_FORM_HF_COOLDOWN_MS_TIP_KEY, SCHEDULE_LAUNCH_FORM_INTERVAL_MS_KEY, SCHEDULE_LAUNCH_FORM_INTERVAL_MS_TIP_KEY, SCHEDULE_LAUNCH_FORM_SCHEDULE_NAME_KEY, SCHEDULE_LAUNCH_FORM_SCHEDULE_NAME_PLACEHOLDER_KEY, SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPES_HF_QUOTE_KEY, SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPES_INTERVAL_KEY, SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPES_KLINE_CLOSE_KEY, SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPE_KEY, SCHEDULE_LAUNCH_FORM_SYMBOL_KEY, SCHEDULE_LAUNCH_FORM_SYMBOL_PLACEHOLDER_KEY, SCHEDULE_LAUNCH_FORM_TIMEFRAME_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';
import { EDIT_MODAL_FIELDS_TEMPLATE_EXTRA_KEY, EDIT_MODAL_FIELDS_TEMPLATE_KEY, EDIT_MODAL_VALIDATION_NAME_REQUIRED_KEY, EDIT_MODAL_VALIDATION_TEMPLATE_REQUIRED_KEY } from '@/gen/ant/v1/i18n/strategy_schedules_keys';

;
import { isTradingAccountEnabled } from '@/utils/accountStatus';
import { TIMEFRAMES } from '@/constants/timeframes';

interface Props {
  isCreate: boolean;
  accountIdWatch: string | undefined;
  symbols: { value: string; label: string }[];
  symbolsLoading: boolean;
  scheduleTypeWatch: string | undefined;
  templates: any[];
  accounts: any[];
}

export function buildAccountOptions(accounts: any[]) {
  return (accounts || []).filter(isTradingAccountEnabled).map((a: unknown) => ({
    value: a.id,
    label: a.login ? `${a.login} (${a.mtType || ''})` : a.id,
  }));
}

export default function EditScheduleBasicFields({
  isCreate, accountIdWatch, symbols, symbolsLoading,
  scheduleTypeWatch, templates, accounts,
}: Props) {
  const { t } = useTranslation();
  const accountOptions = buildAccountOptions(accounts);
  const symbolDisabled = isCreate && (!accountIdWatch || symbolsLoading);
  const symbolPlaceholder = symbolDisabled
    ? t(symbolsLoading ? 'strategy.schedules.editModal.placeholders.symbolsLoading' : 'strategy.schedules.editModal.placeholders.selectAccountFirst')
    : t(SCHEDULE_LAUNCH_FORM_SYMBOL_PLACEHOLDER_KEY, { defaultValue: 'Search symbol, e.g. EURUSD' });

  return (
    <>
      {isCreate && (
        <Row gutter={12}>
          <Col span={12}>
            <Form.Item label={t(EDIT_MODAL_FIELDS_TEMPLATE_KEY)} name="templateId"
              rules={[{ required: true, message: t(EDIT_MODAL_VALIDATION_TEMPLATE_REQUIRED_KEY) }]}
              extra={t(EDIT_MODAL_FIELDS_TEMPLATE_EXTRA_KEY)}>
              <Select showSearch optionFilterProp="label"
                options={(templates || []).map((tpl: unknown) => ({ value: tpl.id, label: tpl.name }))} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item label={t(SCHEDULE_LAUNCH_FORM_ACCOUNT_KEY, { defaultValue: 'Trading Account' })} name="accountId"
              rules={[{ required: true, message: t('common.required') }]}>
              <Select showSearch optionFilterProp="label" options={accountOptions} />
            </Form.Item>
          </Col>
        </Row>
      )}

      <Form.Item label={t(SCHEDULE_LAUNCH_FORM_SCHEDULE_NAME_KEY, { defaultValue: 'Schedule Name' })} name="name"
        rules={[{ required: true, message: t(EDIT_MODAL_VALIDATION_NAME_REQUIRED_KEY) }, { max: 100 }]}>
        <Input placeholder={t(SCHEDULE_LAUNCH_FORM_SCHEDULE_NAME_PLACEHOLDER_KEY, { defaultValue: 'Optional, used to distinguish in schedule list' })} />
      </Form.Item>

      <Row gutter={12}>
        <Col span={12}>
          <Form.Item label={t(SCHEDULE_LAUNCH_FORM_SYMBOL_KEY, { defaultValue: 'Symbol' })} name="symbol"
            rules={[{ required: true, message: t('common.required') }]}>
            <Select showSearch allowClear loading={symbolsLoading} options={symbols}
              optionFilterProp="label" placeholder={symbolPlaceholder} disabled={symbolDisabled} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item label={t(SCHEDULE_LAUNCH_FORM_TIMEFRAME_KEY, { defaultValue: 'Timeframe' })} name="timeframe"
            rules={[{ required: true, message: t('common.required') }]}>
            <Select options={TIMEFRAMES.map((tf) => ({ value: tf, label: tf }))} />
          </Form.Item>
        </Col>
      </Row>

      <Form.Item label={t(SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPE_KEY, { defaultValue: 'Schedule Type' })} name="scheduleType"
        rules={[{ required: true }]}>
        <Select options={[
          { value: 'interval', label: t(SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPES_INTERVAL_KEY, { defaultValue: 'Fixed Interval' }) },
          { value: 'kline_close', label: t(SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPES_KLINE_CLOSE_KEY, { defaultValue: 'Kline Close Trigger' }) },
          { value: 'hf_quote', label: t(SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPES_HF_QUOTE_KEY, { defaultValue: 'Tick-by-tick Quote (HF)' }) },
        ]} />
      </Form.Item>

      {scheduleTypeWatch === 'interval' && (
        <Form.Item
          label={<Tooltip title={t(SCHEDULE_LAUNCH_FORM_INTERVAL_MS_TIP_KEY, { defaultValue: 'Strategy re-evaluation period in ms. Default 5 min = 300000' })}>
            <span>{t(SCHEDULE_LAUNCH_FORM_INTERVAL_MS_KEY, { defaultValue: 'Interval (ms)' })}</span>
          </Tooltip>}
          name="intervalMs"
          rules={[{ required: true, type: 'number', min: 1000, message: '>= 1000' }]}>
          <InputNumber style={{ width: '100%' }} min={1000} step={1000} />
        </Form.Item>
      )}

      {scheduleTypeWatch === 'hf_quote' && (
        <Form.Item
          label={<Tooltip title={t(SCHEDULE_LAUNCH_FORM_HF_COOLDOWN_MS_TIP_KEY, { defaultValue: 'Minimum interval between evaluations in tick mode to avoid waste.' })}>
            <span>{t(SCHEDULE_LAUNCH_FORM_HF_COOLDOWN_MS_KEY, { defaultValue: 'Cooldown (ms)' })}</span>
          </Tooltip>}
          name="hfCooldownMs"
          rules={[{ required: true, type: 'number', min: 100, message: '>= 100' }]}>
          <InputNumber style={{ width: '100%' }} min={100} step={100} />
        </Form.Item>
      )}

      {isCreate && (
        <Form.Item label={t(SCHEDULE_LAUNCH_FORM_ENABLE_AFTER_CREATE_KEY, { defaultValue: 'Enable after creation' })}
          name="isActive" valuePropName="checked">
          <Switch />
        </Form.Item>
      )}
    </>
  );
}
