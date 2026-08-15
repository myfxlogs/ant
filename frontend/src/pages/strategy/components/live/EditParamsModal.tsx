import { useState, useEffect } from 'react';
import { Modal, Form, Input, Select, Spin, message } from 'antd';
import { useTranslation } from 'react-i18next';
import type { ScheduleRow, AccountRow } from '../../hooks/libraryTypes';
import { strategyScheduleV2Api } from '@/client/strategy-schedules';
import StrategyParamsSection from '../workspace/StrategyParamsSection';
import { useStrategyParams } from './useStrategyParams';

interface Props {
  open: boolean;
  schedule: ScheduleRow | null;
  accounts: AccountRow[];
  onClose: () => void;
  onUpdated: () => void;
}

export default function EditParamsModal({ open, schedule, accounts, onClose, onUpdated }: Props) {
  const { t, i18n } = useTranslation();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  const { extractedParams, strategyParamValues, setStrategyParamValues, paramsLoading } =
    useStrategyParams({
      open,
      templateId: schedule?.templateId,
      initialValues: schedule?.parameters,
    });

  // Load form fields when modal opens or schedule changes (by id).
  useEffect(() => {
    if (open && schedule) {
      form.setFieldsValue({
        name: schedule.name,
        symbol: schedule.symbol,
        timeframe: schedule.timeframe,
        accountId: schedule.accountId,
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, schedule?.id, form]);

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
