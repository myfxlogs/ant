import { useState, useEffect, useMemo } from 'react';
import { Modal, Form, Input, Select, InputNumber, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { useAccount } from '@/hooks/useAccount';
import { strategyScheduleApi } from '@/client/strategy-schedules';
import type { ScheduleConfig } from '@/gen/ant/v1/strategy_schedule_entity_pb';
import type { PartialMessage } from '@bufbuild/protobuf';
import SymbolPicker from '@/components/chart/SymbolPicker';
import { TIMEFRAMES } from '@/constants/timeframes';
import {
  SCHEDULE_LAUNCH_FORM_SCHEDULE_NAME_KEY,
  SCHEDULE_LAUNCH_FORM_SCHEDULE_NAME_PLACEHOLDER_KEY,
  SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPE_KEY,
  SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPES_HF_QUOTE_KEY,
  SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPES_INTERVAL_KEY,
  SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPES_KLINE_CLOSE_KEY,
  SCHEDULE_LAUNCH_FORM_INTERVAL_MS_KEY,
  SCHEDULE_LAUNCH_FORM_HF_COOLDOWN_MS_KEY,
  SCHEDULE_LAUNCH_FORM_ACCOUNT_KEY,
  SCHEDULE_LAUNCH_FORM_ACCOUNT_PLACEHOLDER_KEY,
  SCHEDULE_LAUNCH_FORM_SYMBOL_KEY,
  SCHEDULE_LAUNCH_FORM_TIMEFRAME_KEY,
  SCHEDULE_LAUNCH_TITLE_KEY,
  SCHEDULE_LAUNCH_ACTIONS_CREATE_KEY,
  SCHEDULE_LAUNCH_NO_ACCOUNT_BODY_KEY,
  MESSAGES_SCHEDULE_CREATED_KEY,
} from '@/gen/ant/v1/i18n/strategy_templates_keys';

interface Props {
  open: boolean;
  templateId: string;
  templateName: string;
  onClose: () => void;
  onCreated?: () => void;
}

export default function DeployScheduleModal({ open, templateId, templateName, onClose, onCreated }: Props) {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const { accounts, fetchAccounts } = useAccount();
  const activeAccounts = useMemo(() => (accounts || []).filter(a => !a.isDisabled), [accounts]);
  const scheduleType = Form.useWatch('scheduleType', form);
  const accountIdWatch = Form.useWatch('accountId', form);
  const symbolWatch = Form.useWatch('symbol', form);

  useEffect(() => { if (open) fetchAccounts(); }, [open, fetchAccounts]);

  useEffect(() => {
    if (open) {
      form.setFieldsValue({
        name: `${templateName} - Live`,
        scheduleType: 'kline_close',
        intervalMs: 300000,
        hfCooldownMs: 1000,
        timeframe: '1h',
      });
    }
  }, [open, templateName, form]);

  const handleSubmit = async () => {
    try {
      const v = await form.validateFields();
      setLoading(true);
      const scheduleConfig: PartialMessage<ScheduleConfig> = {
        cronExpression: '', intervalMs: 0n, eventTrigger: '',
        triggerMode: v.scheduleType === 'hf_quote' ? 'hf_quote_stream' : 'stable_kline',
        hfCooldownMs: 0n,
      };
      if (v.scheduleType === 'interval') {
        scheduleConfig.intervalMs = BigInt(Math.max(1000, Math.floor(Number(v.intervalMs || 300000))));
      }
      if (v.scheduleType === 'hf_quote') {
        scheduleConfig.hfCooldownMs = BigInt(Math.max(100, Math.floor(Number(v.hfCooldownMs || 1000))));
      }
      const backendType = v.scheduleType === 'interval' ? 'interval' : 'event';
      await strategyScheduleApi.createSchedule({
        templateId,
        accountId: v.accountId,
        name: v.name,
        symbol: v.symbol,
        timeframe: v.timeframe,
        scheduleType: backendType,
        scheduleConfig,
      });
      message.success(t(MESSAGES_SCHEDULE_CREATED_KEY));
      onCreated?.();
      onClose();
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'errorFields' in e) return;
      const msg = e instanceof Error ? e.message : String(e);
      message.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title={t(SCHEDULE_LAUNCH_TITLE_KEY)}
      open={open}
      onOk={handleSubmit}
      onCancel={onClose}
      confirmLoading={loading}
      okText={t(SCHEDULE_LAUNCH_ACTIONS_CREATE_KEY)}
      destroyOnClose
    >
      <Form form={form} layout="vertical">
        <Form.Item name="name" label={t(SCHEDULE_LAUNCH_FORM_SCHEDULE_NAME_KEY)} rules={[{ required: true }]}>
          <Input placeholder={t(SCHEDULE_LAUNCH_FORM_SCHEDULE_NAME_PLACEHOLDER_KEY)} />
        </Form.Item>
        <Form.Item name="accountId" label={t(SCHEDULE_LAUNCH_FORM_ACCOUNT_KEY)} rules={[{ required: true }]}>
          <Select
            placeholder={t(SCHEDULE_LAUNCH_FORM_ACCOUNT_PLACEHOLDER_KEY)}
            notFoundContent={t(SCHEDULE_LAUNCH_NO_ACCOUNT_BODY_KEY)}
            showSearch optionFilterProp="label"
            options={activeAccounts.map(a => ({ value: a.id, label: `${a.brokerServer} · ${a.login}` }))}
          />
        </Form.Item>
        <Form.Item name="symbol" label={t(SCHEDULE_LAUNCH_FORM_SYMBOL_KEY)} rules={[{ required: true }]}>
          <SymbolPicker accountId={accountIdWatch} value={symbolWatch} onChange={(v) => form.setFieldValue('symbol', v)} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item name="timeframe" label={t(SCHEDULE_LAUNCH_FORM_TIMEFRAME_KEY)} rules={[{ required: true }]}>
          <Select
            options={TIMEFRAMES.map(tf => ({ value: tf, label: tf }))}
          />
        </Form.Item>
        <Form.Item name="scheduleType" label={t(SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPE_KEY)}>
          <Select options={[
            { value: 'kline_close', label: t(SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPES_KLINE_CLOSE_KEY) },
            { value: 'interval', label: t(SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPES_INTERVAL_KEY) },
            { value: 'hf_quote', label: t(SCHEDULE_LAUNCH_FORM_SCHEDULE_TYPES_HF_QUOTE_KEY) },
          ]} />
        </Form.Item>
        {scheduleType === 'interval' && (
          <Form.Item name="intervalMs" label={t(SCHEDULE_LAUNCH_FORM_INTERVAL_MS_KEY)}>
            <InputNumber min={1000} step={1000} style={{ width: '100%' }} addonAfter="ms" />
          </Form.Item>
        )}
        {scheduleType === 'hf_quote' && (
          <Form.Item name="hfCooldownMs" label={t(SCHEDULE_LAUNCH_FORM_HF_COOLDOWN_MS_KEY)}>
            <InputNumber min={100} step={100} style={{ width: '100%' }} addonAfter="ms" />
          </Form.Item>
        )}
      </Form>
    </Modal>
  );
}
