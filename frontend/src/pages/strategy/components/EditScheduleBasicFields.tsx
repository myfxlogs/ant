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
  return (accounts || []).filter(isTradingAccountEnabled).map((a: any) => ({
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
    : t(SCHEDULE_LAUNCH_FORM_SYMBOL_PLACEHOLDER_KEY, '搜索品种，如 EURUSD');

  return (
    <>
      {isCreate && (
        <Row gutter={12}>
          <Col span={12}>
            <Form.Item label={t(EDIT_MODAL_FIELDS_TEMPLATE_KEY)} name="templateId"
              rules={[{ required: true, message: t(EDIT_MODAL_VALIDATION_TEMPLATE_REQUIRED_KEY) }]}
              extra={t(EDIT_MODAL_FIELDS_TEMPLATE_EXTRA_KEY)}>
              <Select showSearch optionFilterProp="label"
                options={(templates || []).map((tpl: any) => ({ value: tpl.id, label: tpl.name }))} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item label={t(SCHEDULE_LAUNCH_FORM_ACCOUNT_KEY, '交易账户')} name="accountId"
              rules={[{ required: true, message: t('common.required', '必填') }]}>
              <Select showSearch optionFilterProp="label" options={accountOptions} />
            </Form.Item>
          </Col>
        </Row>
      )}

      <Form.Item label={t(SCHEDULE_LAUNCH_FORM_SCHEDULE_NAME_KEY, '调度名称')} name="name"
        rules={[{ required: true, message: t(EDIT_MODAL_VALIDATION_NAME_REQUIRED_KEY) }, { max: 100 }]}>
        <Input placeholder={t(SCHEDULE_LAUNCH_FORM_SCHEDULE_NAME_PLACEHOLDER_KEY, '可选，用于在调度列表中区分')} />
      </Form.Item>

      <Row gutter={12}>
        <Col span={12}>
          <Form.Item label={t(SCHEDULE_LAUNCH_FORM_SYMBOL_KEY, '交易品种')} name="symbol"
            rules={[{ required: true, message: t('common.required', '必填') }]}>
            <Select showSearch allowClear loading={symbolsLoading} options={symbols}
              optionFilterProp="label" placeholder={symbolPlaceholder} disabled={symbolDisabled} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item label={t(SCHEDULE_LAUNCH_FORM_TIMEFRAME_KEY, '周期')} name="timeframe"
            rules={[{ required: true, message: t('common.required', '必填') }]}>
            <Select options={TIMEFRAMES.map((tf) => ({ value: tf, label: tf }))} />
          </Form.Item>
        </Col>
      </Row>

      <Form.Item label={t(SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPE_KEY, '调度类型')} name="scheduleType"
        rules={[{ required: true }]}>
        <Select options={[
          { value: 'interval', label: t(SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPES_INTERVAL_KEY, '固定间隔') },
          { value: 'kline_close', label: t(SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPES_KLINE_CLOSE_KEY, 'K线收盘触发') },
          { value: 'hf_quote', label: t(SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPES_HF_QUOTE_KEY, '逐笔报价（高频）') },
        ]} />
      </Form.Item>

      {scheduleTypeWatch === 'interval' && (
        <Form.Item
          label={<Tooltip title={t(SCHEDULE_LAUNCH_FORM_INTERVAL_MS_TIP_KEY, '策略重新评估的周期，单位 ms。默认 5 分钟 = 300000')}>
            <span>{t(SCHEDULE_LAUNCH_FORM_INTERVAL_MS_KEY, '间隔（ms）')}</span>
          </Tooltip>}
          name="intervalMs"
          rules={[{ required: true, type: 'number', min: 1000, message: '>= 1000' }]}>
          <InputNumber style={{ width: '100%' }} min={1000} step={1000} />
        </Form.Item>
      )}

      {scheduleTypeWatch === 'hf_quote' && (
        <Form.Item
          label={<Tooltip title={t(SCHEDULE_LAUNCH_FORM_HF_COOLDOWN_MS_TIP_KEY, '逐笔报价模式下连续两次 evaluate 的最短间隔，避免算力浪费。')}>
            <span>{t(SCHEDULE_LAUNCH_FORM_HF_COOLDOWN_MS_KEY, '冷却时间（ms）')}</span>
          </Tooltip>}
          name="hfCooldownMs"
          rules={[{ required: true, type: 'number', min: 100, message: '>= 100' }]}>
          <InputNumber style={{ width: '100%' }} min={100} step={100} />
        </Form.Item>
      )}

      {isCreate && (
        <Form.Item label={t(SCHEDULE_LAUNCH_FORM_ENABLE_AFTER_CREATE_KEY, '创建后立即启用')}
          name="isActive" valuePropName="checked">
          <Switch />
        </Form.Item>
      )}
    </>
  );
}
