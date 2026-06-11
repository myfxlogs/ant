import React from 'react';
import { Alert, Button, Divider, Form, Input, InputNumber, Select, Space, Switch, Tag, Tooltip, message } from 'antd';
import { LockOutlined, SafetyCertificateOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { RequiredParamSpec } from '@/client/codeAssist';
import { TIMEFRAMES } from '@/constants/timeframes';
import { RequiredParamsForm } from '@/components/strategy/CodeAssist';
import { useScheduleLaunchForm } from './useScheduleLaunchForm';
import TradePasswordModal from './components/TradePasswordModal';

export type ScheduleLaunchFormValues = {
  scheduleName?: string;
  accountId: string;
  symbol: string;
  timeframe: string;
  scheduleType: 'interval' | 'kline_close' | 'hf_quote';
  intervalMs?: number;
  hfCooldownMs?: number;
  defaultVolume?: number;
  maxPositions?: number;
  stopLossPriceOffset?: number;
  takeProfitPriceOffset?: number;
  maxDrawdownPct?: number;
  enableAfterCreate?: boolean;
};

export interface SubmitParams {
  form: ScheduleLaunchFormValues;
  buildParameters: () => Record<string, string>;
}

type Props = {
  open: boolean;
  accounts: any[];
  symbols: { value: string; label: string }[];
  symbolsLoading: boolean;
  defaults?: Partial<ScheduleLaunchFormValues>;
  requiredParams?: RequiredParamSpec[];
  paramValues?: Record<string, unknown>;
  onParamValuesChange?: (v: Record<string, unknown>) => void;
  onAccountChange?: (accountId: string) => void | Promise<void>;
  onSubmit: (params: SubmitParams) => void;
  submitting: boolean;
  disabled: boolean;
};

export const StrategyTemplateScheduleLaunchForm: React.FC<Props> = ({
  open, accounts, symbols, symbolsLoading, defaults,
  requiredParams = [], paramValues = {}, onParamValuesChange,
  onAccountChange, onSubmit, submitting, disabled,
}) => {
  const { t } = useTranslation();
  const ctx = useScheduleLaunchForm({ open, accounts, defaults, requiredParams, paramValues, onParamValuesChange, onAccountChange });

  const noAccountBanner = ctx.accountOptions.length === 0 ? (
    <Alert type="warning" showIcon icon={<ExclamationCircleOutlined />} className="mb-3"
      message={t('strategy.templates.scheduleLaunch.noAccountTitle', '还没有可用的交易账号')}
      description={
        <div>
          {t('strategy.templates.scheduleLaunch.noAccountBody', '请先在"账户管理"中添加并绑定 MT4/MT5 账号，账号联机成功后才能上线调度。')}
          <div className="mt-2"><Button size="small" type="primary" onClick={() => window.open('/accounts/bind', '_blank')}>{t('strategy.templates.scheduleLaunch.actions.addAccount', '去添加交易账号')}</Button></div>
        </div>
      }
    />
  ) : null;

  const investorBanner = ctx.tradePermission.isInvestor ? (
    <Alert type="error" showIcon icon={<ExclamationCircleOutlined />} className="mb-3"
      message={t('strategy.templates.scheduleLaunch.investorWarningTitle', '此账户无交易权限（投资者只读模式）')}
      description={
        <div>
          {t('strategy.templates.scheduleLaunch.investorWarningBody', '请填写该账户的交易密码以启用自动下单。')}
          <div className="mt-2"><Button size="small" icon={<LockOutlined />} onClick={() => ctx.setPasswordModalOpen(true)}>{t('strategy.templates.scheduleLaunch.actions.updateTradingPassword', '填写交易密码')}</Button></div>
        </div>
      }
    />
  ) : ctx.tradePermission.verified && ctx.tradePermission.hasTradePermission ? (
    <Alert type="success" showIcon icon={<SafetyCertificateOutlined />} className="mb-3" message={t('strategy.templates.scheduleLaunch.tradePermissionOk', '账户已验证有交易权限')} />
  ) : ctx.tradePermission.loading ? (
    <Alert type="info" showIcon className="mb-3" message={t('strategy.templates.scheduleLaunch.verifyingPermission', '正在验证交易权限…')} />
  ) : null;

  return (
    <>
      {noAccountBanner}
      {investorBanner}
      <Form form={ctx.form} layout="vertical" disabled={disabled || ctx.accountOptions.length === 0}
        onFinish={() => void ctx.handleFinish(onSubmit)}>
        <Form.Item label={t('strategy.templates.scheduleLaunch.form.scheduleName', '调度名称')} name="scheduleName"
          rules={[{ required: false }, { max: 100, message: t('strategy.templates.scheduleLaunch.form.scheduleNameMax', '名称长度需在 100 字以内') }]}>
          <Input placeholder={t('strategy.templates.scheduleLaunch.form.scheduleNamePlaceholder', '可选，用于在调度列表中区分')} />
        </Form.Item>

        <Form.Item label={t('strategy.templates.scheduleLaunch.form.account', '交易账户')} name="accountId"
          rules={[{ required: true, message: t('common.required', '必填') }]}>
          <Select showSearch placeholder={t('strategy.templates.scheduleLaunch.form.accountPlaceholder', '选择账户')}
            onChange={(v) => { const id = String(v || ''); ctx.setSelectedAccountId(id); ctx.form.setFieldsValue({ symbol: '' }); void onAccountChange?.(id); }}
            optionLabelProp="labelText"
            filterOption={(input, option) => { const labelText = String((option as { labelText?: string })?.labelText || '').toLowerCase(); return labelText.includes(input.toLowerCase()); }}
            options={ctx.accountOptions.map((a) => ({
              value: a.id, labelText: a.label + (a.isInvestor ? ' [投资者只读]' : ''),
              label: <Space><span>{a.label}</span>{a.isInvestor && <Tag color="red">{t('strategy.templates.scheduleLaunch.form.investorTag', '投资者只读')}</Tag>}</Space>,
            }))}
          />
        </Form.Item>

        <Form.Item label={t('strategy.templates.scheduleLaunch.form.symbol', '交易品种')} name="symbol"
          rules={[{ required: true, message: t('common.required', '必填') }]}>
          <Select showSearch allowClear loading={symbolsLoading}
            placeholder={symbols.length === 0 && !symbolsLoading ? t('strategy.templates.scheduleLaunch.form.symbolPlaceholderEmpty', '请先选择账户') : t('strategy.templates.scheduleLaunch.form.symbolPlaceholder', '搜索品种，如 EURUSD')}
            options={symbols} optionFilterProp="label" notFoundContent={symbolsLoading ? t('common.loading', '加载中...') : null} />
        </Form.Item>

        <Form.Item label={t('strategy.templates.scheduleLaunch.form.timeframe', '周期')} name="timeframe"
          rules={[{ required: true, message: t('common.required', '必填') }]}>
          <Select options={TIMEFRAMES.map((tf) => ({ value: tf, label: tf }))} />
        </Form.Item>

        <Form.Item label={t('strategy.templates.scheduleLaunch.form.scheduleType', '调度类型')} name="scheduleType" rules={[{ required: true }]}>
          <Select options={[
            { value: 'interval', label: t('strategy.templates.scheduleLaunch.form.scheduleTypes.interval', '固定间隔') },
            { value: 'kline_close', label: t('strategy.templates.scheduleLaunch.form.scheduleTypes.klineClose', 'K线收盘触发') },
            { value: 'hf_quote', label: t('strategy.templates.scheduleLaunch.form.scheduleTypes.hfQuote', '逐笔报价（高频）') },
          ]} />
        </Form.Item>

        {ctx.watchedScheduleType === 'interval' && (
          <Form.Item label={<Tooltip title={t('strategy.templates.scheduleLaunch.form.intervalMsTip', '策略重新评估的周期，单位 ms。默认 5 分钟 = 300000')}><span>{t('strategy.templates.scheduleLaunch.form.intervalMs', '间隔（ms）')}</span></Tooltip>}
            name="intervalMs" rules={[{ required: true, type: 'number', min: 1000, message: '>= 1000' }]}>
            <InputNumber style={{ width: '100%' }} min={1000} step={1000} />
          </Form.Item>
        )}
        {ctx.watchedScheduleType === 'hf_quote' && (
          <Form.Item label={<Tooltip title={t('strategy.templates.scheduleLaunch.form.hfCooldownMsTip', '逐笔报价模式下连续两次 evaluate 的最短间隔，避免算力浪费。')}><span>{t('strategy.templates.scheduleLaunch.form.hfCooldownMs', '冷却时间（ms）')}</span></Tooltip>}
            name="hfCooldownMs" rules={[{ required: true, type: 'number', min: 100, message: '>= 100' }]}>
            <InputNumber style={{ width: '100%' }} min={100} step={100} />
          </Form.Item>
        )}

        <Divider orientation="left" plain>{t('strategy.templates.scheduleLaunch.form.riskSection', '风控参数（可选）')}</Divider>

        <Form.Item label={<Tooltip title={t('strategy.templates.scheduleLaunch.form.defaultVolumeTip', '策略信号里 volume=0 时默认下单量。手数单位。')}><span>{t('strategy.templates.scheduleLaunch.form.defaultVolume', '默认手数')}</span></Tooltip>} name="defaultVolume">
          <InputNumber style={{ width: '100%' }} min={0} step={0.01} placeholder="0.01" />
        </Form.Item>
        <Form.Item label={<Tooltip title={t('strategy.templates.scheduleLaunch.form.maxPositionsTip', '同一品种上允许同时持有的最多持仓数；达到后本次信号跳过。')}><span>{t('strategy.templates.scheduleLaunch.form.maxPositions', '最大持仓数')}</span></Tooltip>} name="maxPositions">
          <InputNumber style={{ width: '100%' }} min={1} step={1} placeholder="不限" />
        </Form.Item>

        <Space style={{ width: '100%' }} size="large">
          <Form.Item label={<Tooltip title={t('strategy.templates.scheduleLaunch.form.stopLossOffsetTip', '策略信号没给 SL 时使用；单位是价格（不是点）。')}><span>{t('strategy.templates.scheduleLaunch.form.stopLossOffset', '止损距离（价格）')}</span></Tooltip>} name="stopLossPriceOffset" style={{ flex: 1 }}>
            <InputNumber style={{ width: '100%' }} min={0} step={0.0001} placeholder="0.0020" />
          </Form.Item>
          <Form.Item label={<Tooltip title={t('strategy.templates.scheduleLaunch.form.takeProfitOffsetTip', '同上，止盈。')}><span>{t('strategy.templates.scheduleLaunch.form.takeProfitOffset', '止盈距离（价格）')}</span></Tooltip>} name="takeProfitPriceOffset" style={{ flex: 1 }}>
            <InputNumber style={{ width: '100%' }} min={0} step={0.0001} placeholder="0.0040" />
          </Form.Item>
        </Space>

        <Form.Item label={<Tooltip title={t('strategy.templates.scheduleLaunch.form.maxDrawdownPctTip', '自峰值权益的最大回撤比例，0.2 = 20%；触发后调度自动停用。')}><span>{t('strategy.templates.scheduleLaunch.form.maxDrawdownPct', '最大回撤比例（0~1）')}</span></Tooltip>} name="maxDrawdownPct" rules={[{ type: 'number', min: 0, max: 1 }]}>
          <InputNumber style={{ width: '100%' }} min={0} max={1} step={0.01} placeholder="0.2" />
        </Form.Item>

        {requiredParams.length > 0 && onParamValuesChange && (
          <>
            <Divider orientation="left" plain>{t('strategy.templates.scheduleLaunch.form.strategyParamsSection', '策略参数')}</Divider>
            <RequiredParamsForm parameters={requiredParams} values={paramValues} onChange={onParamValuesChange} />
          </>
        )}
        <Divider />

        <Form.Item label={t('strategy.templates.scheduleLaunch.form.enableAfterCreate', '创建后立即启用')} name="enableAfterCreate" valuePropName="checked">
          <Switch />
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit" loading={submitting} block
            disabled={disabled || (ctx.tradePermission.verified && ctx.tradePermission.isInvestor)}>
            {t('strategy.templates.scheduleLaunch.actions.create', '创建调度')}
          </Button>
        </Form.Item>
      </Form>

      <TradePasswordModal open={ctx.passwordModalOpen} accountId={ctx.selectedAccountId}
        onCancel={() => ctx.setPasswordModalOpen(false)}
        onSuccess={(res) => {
          ctx.setPasswordModalOpen(false);
          ctx.setTradePermission({ loading: false, verified: true, hasTradePermission: res.hasTradePermission, isInvestor: res.isInvestor, message: res.message });
        }}
      />
    </>
  );
};
