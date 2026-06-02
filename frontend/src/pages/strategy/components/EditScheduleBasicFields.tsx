import { Col, Form, Input, InputNumber, Row, Select, Switch, Tooltip } from 'antd';
import { useTranslation } from 'react-i18next';
import { isTradingAccountEnabled } from '@/utils/accountStatus';

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
    : t('strategy.templates.scheduleLaunch.form.symbolPlaceholder', '搜索品种，如 EURUSD');

  return (
    <>
      {isCreate && (
        <Row gutter={12}>
          <Col span={12}>
            <Form.Item label={t('strategy.schedules.editModal.fields.template')} name="templateId"
              rules={[{ required: true, message: t('strategy.schedules.editModal.validation.templateRequired') }]}
              extra={t('strategy.schedules.editModal.fields.templateExtra')}>
              <Select showSearch optionFilterProp="label"
                options={(templates || []).map((tpl: any) => ({ value: tpl.id, label: tpl.name }))} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item label={t('strategy.templates.scheduleLaunch.form.account', '交易账户')} name="accountId"
              rules={[{ required: true, message: t('common.required', '必填') }]}>
              <Select showSearch optionFilterProp="label" options={accountOptions} />
            </Form.Item>
          </Col>
        </Row>
      )}

      <Form.Item label={t('strategy.templates.scheduleLaunch.form.scheduleName', '调度名称')} name="name"
        rules={[{ required: true, message: t('strategy.schedules.editModal.validation.nameRequired') }, { max: 100 }]}>
        <Input placeholder={t('strategy.templates.scheduleLaunch.form.scheduleNamePlaceholder', '可选，用于在调度列表中区分')} />
      </Form.Item>

      <Row gutter={12}>
        <Col span={12}>
          <Form.Item label={t('strategy.templates.scheduleLaunch.form.symbol', '交易品种')} name="symbol"
            rules={[{ required: true, message: t('common.required', '必填') }]}>
            <Select showSearch allowClear loading={symbolsLoading} options={symbols}
              optionFilterProp="label" placeholder={symbolPlaceholder} disabled={symbolDisabled} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item label={t('strategy.templates.scheduleLaunch.form.timeframe', '周期')} name="timeframe"
            rules={[{ required: true, message: t('common.required', '必填') }]}>
            <Select options={['M1', 'M5', 'M15', 'M30', 'H1', 'H4', 'D1'].map((tf) => ({ value: tf, label: tf }))} />
          </Form.Item>
        </Col>
      </Row>

      <Form.Item label={t('strategy.templates.scheduleLaunch.form.scheduleType', '调度类型')} name="scheduleType"
        rules={[{ required: true }]}>
        <Select options={[
          { value: 'interval', label: t('strategy.templates.scheduleLaunch.form.scheduleTypes.interval', '固定间隔') },
          { value: 'kline_close', label: t('strategy.templates.scheduleLaunch.form.scheduleTypes.klineClose', 'K线收盘触发') },
          { value: 'hf_quote', label: t('strategy.templates.scheduleLaunch.form.scheduleTypes.hfQuote', '逐笔报价（高频）') },
        ]} />
      </Form.Item>

      {scheduleTypeWatch === 'interval' && (
        <Form.Item
          label={<Tooltip title={t('strategy.templates.scheduleLaunch.form.intervalMsTip', '策略重新评估的周期，单位 ms。默认 5 分钟 = 300000')}>
            <span>{t('strategy.templates.scheduleLaunch.form.intervalMs', '间隔（ms）')}</span>
          </Tooltip>}
          name="intervalMs"
          rules={[{ required: true, type: 'number', min: 1000, message: '>= 1000' }]}>
          <InputNumber style={{ width: '100%' }} min={1000} step={1000} />
        </Form.Item>
      )}

      {scheduleTypeWatch === 'hf_quote' && (
        <Form.Item
          label={<Tooltip title={t('strategy.templates.scheduleLaunch.form.hfCooldownMsTip', '逐笔报价模式下连续两次 evaluate 的最短间隔，避免算力浪费。')}>
            <span>{t('strategy.templates.scheduleLaunch.form.hfCooldownMs', '冷却时间（ms）')}</span>
          </Tooltip>}
          name="hfCooldownMs"
          rules={[{ required: true, type: 'number', min: 100, message: '>= 100' }]}>
          <InputNumber style={{ width: '100%' }} min={100} step={100} />
        </Form.Item>
      )}

      {isCreate && (
        <Form.Item label={t('strategy.templates.scheduleLaunch.form.enableAfterCreate', '创建后立即启用')}
          name="isActive" valuePropName="checked">
          <Switch />
        </Form.Item>
      )}
    </>
  );
}
