import { useState, useEffect } from 'react';
import { Card, Form, Select, InputNumber, DatePicker, Button, Space, message, Descriptions } from 'antd';
import { PlayCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';
import { create } from '@bufbuild/protobuf';
import { executionAlgoClient } from '@/client/connect';
import { StartAlgoRequestSchema } from '@/gen/ant/v1/execution_algo_pb';
import { useAccount } from '@/hooks/useAccount';

interface Props {
  onStarted?: (executionId: string) => void;
}

const ALGO_OPTIONS = [
  { value: 'twap', labelKey: 'algo.twap' },
  { value: 'vwap', labelKey: 'algo.vwap' },
  { value: 'pov', labelKey: 'algo.pov' },
  { value: 'shortfall', labelKey: 'algo.shortfall' },
];

export default function AlgoSubmitForm({ onStarted }: Props) {
  const { t } = useTranslation();
  const { accounts } = useAccount();
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);
  const [selectedAlgo, setSelectedAlgo] = useState<string>('twap');

  useEffect(() => {
    if (accounts.length > 0 && !form.getFieldValue('accountId')) {
      form.setFieldValue('accountId', accounts[0].id);
    }
  }, [accounts, form]);

  const handleSubmit = async () => {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      const msg = create(StartAlgoRequestSchema, {
        accountId: values.accountId,
        symbol: values.symbol,
        side: values.side,
        totalVolume: values.totalVolume,
        startTime: values.timeRange ? { seconds: BigInt(values.timeRange[0].unix()), nanos: 0 } : undefined,
        endTime: values.timeRange ? { seconds: BigInt(values.timeRange[1].unix()), nanos: 0 } : undefined,
        limitPrice: values.limitPrice || 0,
        algo: values.algo,
        sliceInterval: values.sliceInterval ? { seconds: BigInt(Math.floor(values.sliceInterval)), nanos: 0 } : undefined,
        participationRate: values.participationRate,
        urgency: values.urgency,
      });
      const resp = await executionAlgoClient.startAlgo(msg);
      message.success(t('algo.messages.started', { id: resp.executionId }));
      onStarted?.(resp.executionId);
      form.resetFields();
    } catch (e: unknown) {
      message.error(String((e as { message?: string })?.message || e));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Card title={t('algo.submitForm.title')} size="small">
      <Form form={form} layout="vertical" onFinish={handleSubmit}
        initialValues={{ algo: 'twap', side: 'buy', totalVolume: 1.0 }}>
        <Form.Item name="accountId" label={t('algo.fields.account')} rules={[{ required: true }]}>
          <Select options={accounts.map(a => ({ value: a.id, label: a.alias || a.login }))} />
        </Form.Item>
        <Space style={{ width: '100%' }} size="middle">
          <Form.Item name="symbol" label={t('algo.fields.symbol')} rules={[{ required: true }]} style={{ flex: 1 }}>
            <Select showSearch options={[
              { value: 'EURUSD', label: 'EURUSD' }, { value: 'GBPUSD', label: 'GBPUSD' },
              { value: 'USDJPY', label: 'USDJPY' }, { value: 'XAUUSD', label: 'XAUUSD' },
              { value: 'BTCUSD', label: 'BTCUSD' },
            ]} />
          </Form.Item>
          <Form.Item name="side" label={t('algo.fields.side')} rules={[{ required: true }]}>
            <Select options={[
              { value: 'buy', label: t('algo.side.buy') },
              { value: 'sell', label: t('algo.side.sell') },
            ]} />
          </Form.Item>
          <Form.Item name="totalVolume" label={t('algo.fields.volume')} rules={[{ required: true, type: 'number', min: 0.01 }]}>
            <InputNumber min={0.01} step={0.1} style={{ width: 100 }} />
          </Form.Item>
        </Space>
        <Form.Item name="timeRange" label={t('algo.fields.timeRange')} rules={[{ required: true }]}>
          <DatePicker.RangePicker showTime style={{ width: '100%' }}
            presets={[
              { label: t('algo.timePresets.1h'), value: [dayjs(), dayjs().add(1, 'hour')] },
              { label: t('algo.timePresets.4h'), value: [dayjs(), dayjs().add(1, 'hour')] },
              { label: t('algo.timePresets.EOD'), value: [dayjs(), dayjs().endOf('day')] },
            ]} />
        </Form.Item>
        <Form.Item name="algo" label={t('algo.fields.algo')} rules={[{ required: true }]}>
          <Select options={ALGO_OPTIONS.map(a => ({ value: a.value, label: t(a.labelKey) }))}
            onChange={setSelectedAlgo} />
        </Form.Item>
        <Space wrap>
          <Form.Item name="limitPrice" label={t('algo.fields.limitPrice')}>
            <InputNumber min={0} step={0.0001} style={{ width: 120 }} placeholder="Market" />
          </Form.Item>
          <Form.Item name="sliceInterval" label={t('algo.fields.sliceInterval')}>
            <InputNumber min={1} max={3600} style={{ width: 100 }} placeholder="60s" addonAfter="s" />
          </Form.Item>
          {selectedAlgo === 'pov' && (
            <Form.Item name="participationRate" label={t('algo.fields.participationRate')}>
              <InputNumber min={0.01} max={1} step={0.05} style={{ width: 100 }} placeholder="0.10" />
            </Form.Item>
          )}
          {selectedAlgo === 'shortfall' && (
            <Form.Item name="urgency" label={t('algo.fields.urgency')}>
              <InputNumber min={0} max={1} step={0.1} style={{ width: 100 }} placeholder="0.50" />
            </Form.Item>
          )}
        </Space>
        <Button type="primary" htmlType="submit" icon={<PlayCircleOutlined />} loading={submitting}>
          {t('algo.actions.start')}
        </Button>
        {selectedAlgo && (
          <Descriptions size="small" column={1} style={{ marginTop: 12 }}>
            <Descriptions.Item label={t('algo.info.name')}>{t(`algo.${selectedAlgo}.name`)}</Descriptions.Item>
            <Descriptions.Item label={t('algo.info.description')}>{t(`algo.${selectedAlgo}.description`)}</Descriptions.Item>
          </Descriptions>
        )}
      </Form>
    </Card>
  );
}
