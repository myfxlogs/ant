import React, { useState, useEffect } from 'react';
import { Modal, Form, InputNumber, Select, DatePicker, Switch, Button, Alert, Space, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { codeAssistApi } from '@/client/codeAssist';
import {
  CAPITAL_KEY, COMMISSION_KEY, DATE_RANGE_KEY, DIRECTION_KEY,
  END_DATE_KEY, LEVERAGE_KEY, SLIPPAGE_KEY, START_DATE_KEY,
  STRICT_MODE_KEY, STRICT_MODE_ON_KEY, STRICT_MODE_OFF_KEY,
  LONG_KEY, SHORT_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import { DATE_PRESETS, dateFromPreset } from '@/pages/strategy/hooks/backtestParamHelpers';
import { FACTORY_DEFAULTS, type StandardParams } from '@/components/backtest/backtestRunnerTypes';
import dayjs from 'dayjs';

export interface BacktestParamsModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (params: BacktestModalResult) => void;
  code: string;
  symbol: string;
}

export interface BacktestModalResult {
  params: StandardParams;
  startDate: string;
  endDate: string;
}

export const BacktestParamsModal: React.FC<BacktestParamsModalProps> = ({ open, onClose, onConfirm, code, symbol }) => {
  const { t } = useTranslation();
  const [validating, setValidating] = useState(false);
  const [validationError, setValidationError] = useState('');
  const [validated, setValidated] = useState(false);
  const [datePreset, setDatePreset] = useState('3M');

  const [params, setParams] = useState<StandardParams>(FACTORY_DEFAULTS);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs | null, dayjs.Dayjs | null]>([
    dayjs().subtract(3, 'month'),
    dayjs(),
  ]);

  useEffect(() => {
    if (open && code && !validated && !validating) {
      setValidationError('');
      doValidate();
    }
    if (!open) {
      setValidated(false);
      setValidationError('');
    }
  }, [open]); // eslint-disable-line react-hooks/exhaustive-deps

  const doValidate = async () => {
    if (!code.trim()) {
      setValidationError(t('strategy.workspace.noCode'));
      return;
    }
    setValidating(true);
    try {
      const result = await codeAssistApi.validateExtended(code);
      if (result.valid) {
        setValidated(true);
        setValidationError('');
      } else {
        setValidationError(result.errors.join('\n') || t('strategy.workspace.compileError'));
      }
    } catch (e: unknown) {
      setValidationError((e as Error)?.message || 'Validation failed');
    } finally {
      setValidating(false);
    }
  };

  const handlePresetClick = (presetKey: string) => {
    setDatePreset(presetKey);
    const { start, end } = dateFromPreset(presetKey);
    setDateRange([dayjs(start), dayjs(end)]);
  };

  const handleConfirm = () => {
    if (!validated) {
      message.warning(t('strategy.workspace.validateFirst'));
      return;
    }
    const startDate = dateRange[0]?.format('YYYY-MM-DD') || '';
    const endDate = dateRange[1]?.format('YYYY-MM-DD') || '';
    onConfirm({ params, startDate, endDate });
    onClose();
  };

  return (
    <Modal
      title={t('strategy.workspace.runBacktest')}
      open={open}
      onCancel={onClose}
      width={520}
      footer={[
        <Button key="cancel" onClick={onClose}>{t('common.cancel')}</Button>,
        <Button key="run" type="primary" loading={validating} disabled={!validated} onClick={handleConfirm}
          style={{ background: '#3fb950', borderColor: '#3fb950' }}>
          {t('strategy.workspace.backtest')}
        </Button>,
      ]}
    >
      {/* Validation status */}
      {validating && (
        <Alert type="info" showIcon message={t('strategy.gen.compiling')} style={{ marginBottom: 12 }} />
      )}
      {validationError && !validating && (
        <Alert type="error" showIcon style={{ marginBottom: 12 }}
          message={t('strategy.gen.compileError')}
          description={<pre style={{ fontSize: 11, whiteSpace: 'pre-wrap', margin: 0 }}>{validationError}</pre>}
          action={<Button size="small" onClick={doValidate}>{t('common.retry')}</Button>}
        />
      )}
      {validated && !validating && (
        <Alert type="success" showIcon message={t('strategy.workspace.codeValid')} style={{ marginBottom: 12 }} />
      )}

      <Form layout="vertical" size="small" disabled={!validated}>
        {/* Date range presets */}
        <Form.Item label={t(DATE_RANGE_KEY)}>
          <Space style={{ marginBottom: 8 }}>
            {DATE_PRESETS.map(p => (
              <Button
                key={p.key}
                size="small"
                type={datePreset === p.key ? 'primary' : 'default'}
                onClick={() => handlePresetClick(p.key)}
              >{p.label}</Button>
            ))}
          </Space>
          <DatePicker.RangePicker
            value={dateRange as [dayjs.Dayjs, dayjs.Dayjs]}
            onChange={(vals) => {
              setDateRange(vals as [dayjs.Dayjs | null, dayjs.Dayjs | null]);
              if (vals) setDatePreset('');
            }}
            style={{ width: '100%' }}
            allowClear={false}
          />
        </Form.Item>

        {/* Capital + Leverage */}
        <Space style={{ width: '100%' }} sizes={['50%', '50%']}>
          <Form.Item label={t(CAPITAL_KEY)} style={{ width: '50%' }}>
            <InputNumber
              value={params.initialCapital}
              onChange={(v) => setParams(p => ({ ...p, initialCapital: v ?? FACTORY_DEFAULTS.initialCapital }))}
              min={100} style={{ width: '100%' }}
              prefix="$"
            />
          </Form.Item>
          <Form.Item label={t(LEVERAGE_KEY)} style={{ width: '50%' }}>
            <InputNumber
              value={params.leverage}
              onChange={(v) => setParams(p => ({ ...p, leverage: v ?? FACTORY_DEFAULTS.leverage }))}
              min={1} max={1000} style={{ width: '100%' }}
            />
          </Form.Item>
        </Space>

        {/* Commission + Slippage */}
        <Space style={{ width: '100%' }} sizes={['50%', '50%']}>
          <Form.Item label={t(COMMISSION_KEY)} style={{ width: '50%' }}>
            <InputNumber
              value={params.commission}
              onChange={(v) => setParams(p => ({ ...p, commission: v ?? FACTORY_DEFAULTS.commission }))}
              min={0} max={0.1} step={0.0005} style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item label={t(SLIPPAGE_KEY)} style={{ width: '50%' }}>
            <InputNumber
              value={params.slippage}
              onChange={(v) => setParams(p => ({ ...p, slippage: v ?? FACTORY_DEFAULTS.slippage }))}
              min={0} max={0.1} step={0.0005} style={{ width: '100%' }}
            />
          </Form.Item>
        </Space>

        {/* Trade direction */}
        <Form.Item label={t(DIRECTION_KEY)}>
          <Select
            value={params.tradeDirection}
            onChange={(v) => setParams(p => ({ ...p, tradeDirection: v }))}
            style={{ width: '100%' }}
            options={[
              { value: 'both', label: t('strategy.backtestParams.both') },
              { value: 'long', label: t(LONG_KEY) },
              { value: 'short', label: t(SHORT_KEY) },
            ]}
          />
        </Form.Item>

        {/* Strict mode */}
        <Form.Item label={t(STRICT_MODE_KEY)}>
          <Switch
            checked={params.strictMode}
            onChange={(v) => setParams(p => ({ ...p, strictMode: v }))}
            checkedChildren={t(STRICT_MODE_ON_KEY)}
            unCheckedChildren={t(STRICT_MODE_OFF_KEY)}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
};
