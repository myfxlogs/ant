import { useState, useEffect, useCallback } from 'react';
import { Modal, Form, Input, Select, InputNumber, Spin, message } from 'antd';
import { useTranslation } from 'react-i18next';
import type { ScheduleRow, AccountRow } from '../../hooks/libraryTypes';
import { strategyScheduleV2Api, strategyTemplateApi } from '@/client/strategy-schedules';
import { codeAssistApi } from '@/client/codeAssist';
import { parseParametersToForm, buildParametersFromForm, type CommonFields } from '../../StrategyScheduleParams';
import StrategyParamsSection from '../workspace/StrategyParamsSection';

interface Props {
  open: boolean;
  schedule: ScheduleRow | null;
  accounts: AccountRow[];
  onClose: () => void;
  onUpdated: () => void;
}

interface ExtractedParam {
  name: string;
  type: string;
  default: string;
  label: string;
}

export default function EditParamsModal({ open, schedule, accounts, onClose, onUpdated }: Props) {
  const { t, i18n } = useTranslation();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [extractedParams, setExtractedParams] = useState<ExtractedParam[]>([]);
  const [strategyParamValues, setStrategyParamValues] = useState<Record<string, string>>({});
  const [paramsLoading, setParamsLoading] = useState(false);

  const loadStrategyParams = useCallback(async (templateId: string) => {
    if (!templateId) return;
    setParamsLoading(true);
    try {
      const tpl = await strategyTemplateApi.get(templateId);
      const code = String(tpl?.code || '');
      if (!code) { setParamsLoading(false); return; }
      const result = await codeAssistApi.validateExtended(code);
      if (result.valid && result.parameterEntries) {
        setExtractedParams(result.parameterEntries.map(e => ({
          name: e.name, type: e.type, default: e.default, label: e.label || '',
        })));
      }
    } catch { /* template not found or validation failed */ }
    setParamsLoading(false);
  }, []);

  useEffect(() => {
    if (open && schedule) {
      const parsed = parseParametersToForm(schedule.parameters || {});
      form.setFieldsValue({
        name: schedule.name,
        symbol: schedule.symbol,
        timeframe: schedule.timeframe,
        accountId: schedule.accountId,
        defaultVolume: parsed.defaultVolume,
        maxPositions: parsed.maxPositions,
        stopLossPriceOffset: parsed.stopLossPriceOffset,
        takeProfitPriceOffset: parsed.takeProfitPriceOffset,
        maxDrawdownPct: parsed.maxDrawdownPct,
      });
      const vals: Record<string, string> = {};
      const params = schedule.parameters || {};
      for (const p of extractedParams) {
        vals[p.name] = params[p.name] ?? p.default;
      }
      setStrategyParamValues(vals);
      if (schedule.templateId) void loadStrategyParams(schedule.templateId);
    }
  }, [open, schedule, form, extractedParams, loadStrategyParams]);

  useEffect(() => {
    if (!open) {
      setExtractedParams([]);
      setStrategyParamValues({});
    }
  }, [open]);

  const handleOk = async () => {
    if (!schedule) return;
    const v = await form.validateFields();
    setLoading(true);
    try {
      const formFields: CommonFields = {
        defaultVolume: v.defaultVolume,
        maxPositions: v.maxPositions,
        stopLossPriceOffset: v.stopLossPriceOffset,
        takeProfitPriceOffset: v.takeProfitPriceOffset,
        maxDrawdownPct: v.maxDrawdownPct,
      };
      const riskParams = buildParametersFromForm(formFields);
      const merged = { ...strategyParamValues, ...riskParams };
      await strategyScheduleV2Api.update({
        id: schedule.id,
        name: v.name,
        symbol: v.symbol,
        timeframe: v.timeframe,
        accountId: v.accountId,
        parameters: merged,
      });
      message.success(t('common.updated', { defaultValue: 'Updated' }));
      onUpdated();
      onClose();
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t('common.updateFailed', { defaultValue: 'Update failed' }));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      open={open}
      title={t('strategy.live.editParams', { defaultValue: 'Edit Parameters' })}
      onOk={handleOk}
      onCancel={onClose}
      okText={t('common.save', { defaultValue: 'Save' })}
      confirmLoading={loading}
      width={560}
    >
      <Form form={form} layout="vertical" size="small">
        <Form.Item name="name" label={t('strategy.live.strategyName', { defaultValue: 'Strategy Name' })}>
          <Input />
        </Form.Item>
        <Form.Item name="symbol" label={t('strategy.live.symbol', { defaultValue: 'Symbol' })}>
          <Input />
        </Form.Item>
        <Form.Item name="timeframe" label={t('strategy.live.timeframe', { defaultValue: 'Timeframe' })}>
          <Select options={[
            { value: '1m', label: '1m' },
            { value: '5m', label: '5m' },
            { value: '15m', label: '15m' },
            { value: '30m', label: '30m' },
            { value: '1h', label: '1h' },
            { value: '4h', label: '4h' },
            { value: '1d', label: '1d' },
          ]} />
        </Form.Item>
        <Form.Item name="accountId" label={t('strategy.live.account', { defaultValue: 'Account' })}>
          <Select options={accounts.map(a => ({ value: a.id, label: a.login ? `${a.login}${a.brokerCompany ? ' - ' + a.brokerCompany : ''}` : a.id }))} />
        </Form.Item>

        <div style={{ borderTop: '1px solid var(--ant-color-border)', marginTop: 12, paddingTop: 12 }}>
          <div style={{ fontSize: 12, fontWeight: 700, color: '#595959', marginBottom: 8 }}>
            {t('strategy.live.riskParams', { defaultValue: 'Risk Parameters' })}
          </div>
          <Form.Item name="defaultVolume" label={t('strategy.live.defaultVolume', { defaultValue: 'Default Volume' })}>
            <InputNumber min={0} step={0.01} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="maxPositions" label={t('strategy.live.maxPositions', { defaultValue: 'Max Positions' })}>
            <InputNumber min={0} step={1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="stopLossPriceOffset" label={t('strategy.live.stopLossOffset', { defaultValue: 'Stop Loss Offset' })}>
            <InputNumber min={0} step={0.01} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="takeProfitPriceOffset" label={t('strategy.live.takeProfitOffset', { defaultValue: 'Take Profit Offset' })}>
            <InputNumber min={0} step={0.01} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="maxDrawdownPct" label={t('strategy.live.maxDrawdown', { defaultValue: 'Max Drawdown %' })}>
            <InputNumber min={0} max={1} step={0.01} style={{ width: '100%' }} />
          </Form.Item>
        </div>

        <Spin spinning={paramsLoading}>
          <StrategyParamsSection
            extractedParams={extractedParams}
            strategyParamValues={strategyParamValues}
            onChange={(name, value) => setStrategyParamValues(prev => ({ ...prev, [name]: value }))}
            language={i18n.language}
          />
        </Spin>
      </Form>
    </Modal>
  );
}
