import { useState, useEffect, useCallback } from 'react';
import { Modal, Form, Input, Select, Spin, message } from 'antd';
import { useTranslation } from 'react-i18next';
import type { ScheduleRow, AccountRow } from '../../hooks/libraryTypes';
import { strategyScheduleV2Api, strategyTemplateApi } from '@/client/strategy-schedules';
import { codeAssistApi } from '@/client/codeAssist';
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

  // E1: Load effect — only fires when modal opens or schedule changes (by id).
  // Uses schedule?.id as dep to avoid object-reference retrigger (row data updates every tick).
  useEffect(() => {
    if (open && schedule) {
      form.setFieldsValue({
        name: schedule.name,
        symbol: schedule.symbol,
        timeframe: schedule.timeframe,
        accountId: schedule.accountId,
      });
      // Seed strategy param values from schedule's stored parameters.
      const params = schedule.parameters || {};
      const vals: Record<string, string> = {};
      for (const [k, v] of Object.entries(params)) {
        if (!k.startsWith('__risk.') && !k.startsWith('__schedule.')) vals[k] = String(v);
      }
      setStrategyParamValues(vals);
      if (schedule.templateId) void loadStrategyParams(schedule.templateId);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, schedule?.id, form, loadStrategyParams]);

  // E1: Merge effect — when extracted params arrive, fill in defaults for keys
  // the user hasn't set yet. Uses functional update to preserve user edits.
  useEffect(() => {
    if (extractedParams.length === 0) return;
    setStrategyParamValues(prev => {
      const next = { ...prev };
      for (const p of extractedParams) {
        if (!(p.name in next)) next[p.name] = p.default;
      }
      return next;
    });
  }, [extractedParams]);

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
      await strategyScheduleV2Api.update({
        id: schedule.id,
        name: v.name,
        symbol: v.symbol,
        timeframe: v.timeframe,
        accountId: v.accountId,
        parameters: strategyParamValues,
      });
      // E2: If schedule was running, backend auto-restarts with new params.
      if (schedule.isRunning) {
        message.success(t('strategy.live.savedAndRestarted', { defaultValue: 'Saved — strategy restarted with new parameters' }));
      } else {
        message.success(t('common.updated', { defaultValue: 'Updated' }));
      }
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
