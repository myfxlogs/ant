import React, { useState, useEffect } from 'react';
import { Modal, Form, InputNumber, Select, DatePicker, Switch, Button, Alert, Space, message, Row, Col } from 'antd';
import { useTranslation } from 'react-i18next';
import { codeAssistApi } from '@/client/codeAssist';
import {
  CAPITAL_KEY, COMMISSION_KEY, DATE_RANGE_KEY, DIRECTION_KEY,
  LEVERAGE_KEY, SLIPPAGE_KEY,
  STRICT_MODE_KEY, STRICT_MODE_ON_KEY, STRICT_MODE_OFF_KEY,
  LONG_KEY, SHORT_KEY, BOTH_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import {
  NO_CODE_KEY, COMPILE_ERROR_KEY, VALIDATE_FIRST_KEY, CODE_VALID_KEY,
  VALIDATE_FAILED_KEY, RUN_BACKTEST_KEY, BACKTEST_KEY as WS_BACKTEST_KEY,
} from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { STRATEGY_PARAMS_KEY } from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import { COMPILING_KEY as GEN_COMPILING_KEY, COMPILE_ERROR_KEY as GEN_COMPILE_ERROR_KEY } from '@/gen/ant/v1/i18n/strategy_gen_keys';
import { COMMON_CANCEL_KEY, COMMON_RETRY_KEY } from '@/gen/ant/v1/i18n/base_keys';
import { DATE_PRESETS, dateFromPreset } from '@/pages/strategy/hooks/backtestParamHelpers';
import { FACTORY_DEFAULTS, type StandardParams } from '@/components/backtest/backtestRunnerTypes';
import { paramLabel } from '@/utils/paramLabel';
import dayjs from 'dayjs';

export interface BacktestParamsModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (params: BacktestModalResult) => void;
  code: string;
  symbol: string;
  timeframe?: string;
}

export interface BacktestModalResult {
  params: StandardParams;
  startDate: string;
  endDate: string;
  strategyParams?: Record<string, string>;
}

export const BacktestParamsModal: React.FC<BacktestParamsModalProps> = ({ open, onClose, onConfirm, code, _symbol, timeframe: _timeframe }) => {
  const { t, i18n } = useTranslation();
  const [validating, setValidating] = useState(false);
  const [validationError, setValidationError] = useState('');
  const [validated, setValidated] = useState(false);
  const [datePreset, setDatePreset] = useState('3M');
  const [extractedParams, setExtractedParams] = useState<Array<{ name: string; type: string; default: string; label: string }>>([]);
  const [strategyParamValues, setStrategyParamValues] = useState<Record<string, string>>({});

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
      setExtractedParams([]);
    }
  }, [open]); // eslint-disable-line react-hooks/exhaustive-deps

  const doValidate = async () => {
    if (!code.trim()) {
      setValidationError(t(NO_CODE_KEY));
      return;
    }
    setValidating(true);
    try {
      const result = await codeAssistApi.validateExtended(code);
      if (result.valid) {
        setValidated(true);
        setValidationError('');
        // Parse extracted strategy params from validation result
        try {
          const list = (result.parameterEntries || []).map(e => ({ name: e.name, type: e.type, default: e.default, label: e.label || '' }));
          setExtractedParams(list || []);
          const vals: Record<string, string> = {};
          for (const p of (list || [])) vals[p.name] = strategyParamValues[p.name] ?? p.default;
          setStrategyParamValues(vals);
        } catch { setExtractedParams([]); }
      } else {
        setValidationError(result.errors.join('\n') || t(COMPILE_ERROR_KEY));
      }
    } catch (e: unknown) {
      setValidationError((e as Error)?.message || t(VALIDATE_FAILED_KEY));
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
      message.warning(t(VALIDATE_FIRST_KEY));
      return;
    }
    const startDate = dateRange[0]?.format('YYYY-MM-DD') || '';
    const endDate = dateRange[1]?.format('YYYY-MM-DD') || '';
    onConfirm({ params, startDate, endDate, strategyParams: strategyParamValues });
    onClose();
  };

  return (
    <Modal
      title={t(RUN_BACKTEST_KEY)}
      open={open}
      onCancel={onClose}
      width={520}
      footer={[
        <Button key="cancel" onClick={onClose}>{t(COMMON_CANCEL_KEY)}</Button>,
        <Button key="run" type="primary" loading={validating} disabled={!validated} onClick={handleConfirm}
          style={{ background: '#3fb950', borderColor: '#3fb950' }}>
          {t(WS_BACKTEST_KEY)}
        </Button>,
      ]}
    >
      {/* Validation status */}
      {validating && (
        <Alert type="info" showIcon message={t(GEN_COMPILING_KEY)} style={{ marginBottom: 12 }} />
      )}
      {validationError && !validating && (
        <Alert type="error" showIcon style={{ marginBottom: 12 }}
          message={t(GEN_COMPILE_ERROR_KEY)}
          description={<pre style={{ fontSize: 11, whiteSpace: 'pre-wrap', margin: 0 }}>{validationError}</pre>}
          action={<Button size="small" onClick={doValidate}>{t(COMMON_RETRY_KEY)}</Button>}
        />
      )}
      {validated && !validating && (
        <Alert type="success" showIcon message={t(CODE_VALID_KEY)} style={{ marginBottom: 12 }} />
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
              { value: 'both', label: t(BOTH_KEY) },
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

        {/* Strategy params (extracted from code) */}
        {extractedParams.length > 0 && (
          <>
            <div style={{ borderTop: '1px solid var(--ant-color-border)', marginTop: 12, paddingTop: 12 }}>
              <div style={{ fontSize: 12, fontWeight: 700, color: '#595959', marginBottom: 8 }}>
                {t(STRATEGY_PARAMS_KEY)} ({extractedParams.length})
              </div>
              <Row gutter={[12, 8]}>
                {extractedParams.map((p) => {
                  const label = paramLabel(p.name, i18n.language, null) || p.label || p.name;
                  const value = strategyParamValues[p.name] ?? p.default;
                  if (p.type === 'bool') {
                    return (
                      <Col span={8} key={p.name}>
                        <Form.Item label={label} style={{ marginBottom: 0 }}>
                          <Switch size="small" checked={value === 'True' || value === 'true'}
                            onChange={(v) => setStrategyParamValues(prev => ({ ...prev, [p.name]: v ? 'True' : 'False' }))} />
                        </Form.Item>
                      </Col>
                    );
                  }
                  const step = p.type === 'float' ? 0.01 : 1;
                  return (
                    <Col span={8} key={p.name}>
                      <Form.Item label={label} style={{ marginBottom: 0 }}>
                        <InputNumber size="small" style={{ width: '100%' }} step={step}
                          value={Number(value)} onChange={(v) => setStrategyParamValues(prev => ({ ...prev, [p.name]: String(v ?? p.default) }))} />
                      </Form.Item>
                    </Col>
                  );
                })}
              </Row>
            </div>
          </>
        )}
      </Form>
    </Modal>
  );
};
