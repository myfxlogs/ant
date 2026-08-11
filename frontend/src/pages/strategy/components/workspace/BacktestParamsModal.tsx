import React, { useState, useEffect } from 'react';
import { Modal, Form, InputNumber, DatePicker, Button, Alert, Space, Row, Col, Dropdown, message } from 'antd';
import { SettingOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { codeAssistApi } from '@/client/codeAssist';
import {
  CAPITAL_KEY, COMMISSION_KEY, DATE_RANGE_KEY,
  LEVERAGE_KEY, SLIPPAGE_KEY,
  TIMEFRAME_KEY,
  SETTINGS_SAVE_KEY, SETTINGS_LOAD_KEY, SETTINGS_RESET_KEY,
  DEFAULTS_SAVED_KEY, DEFAULTS_LOADED_KEY, DEFAULTS_RESET_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import {
  NO_CODE_KEY, COMPILE_ERROR_KEY, VALIDATE_FIRST_KEY, CODE_VALID_KEY,
  VALIDATE_FAILED_KEY, RUN_BACKTEST_KEY, BACKTEST_KEY as WS_BACKTEST_KEY,
} from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { COMPILING_KEY as GEN_COMPILING_KEY, COMPILE_ERROR_KEY as GEN_COMPILE_ERROR_KEY } from '@/gen/ant/v1/i18n/strategy_gen_keys';
import { COMMON_CANCEL_KEY, COMMON_RETRY_KEY } from '@/gen/ant/v1/i18n/base_keys';
import { DATE_PRESETS, dateFromPreset } from '@/pages/strategy/hooks/backtestParamHelpers';
import { FACTORY_DEFAULTS, loadSavedDefaults, saveDefaults, removeDefaults, type StandardParams } from '@/components/backtest/backtestRunnerTypes';
import { TIMEFRAMES } from '@/constants/timeframes';
import ExecutionAssumptionsSelectors from './ExecutionAssumptionsSelectors';
import StrategyParamsSection from './StrategyParamsSection';
import dayjs from 'dayjs';

export interface BacktestParamsModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (params: BacktestModalResult) => void;
  code: string;
  symbol?: string;
  timeframe?: string;
}

export interface BacktestModalResult {
  params: StandardParams;
  startDate: string;
  endDate: string;
  timeframe: string;
  strategyParams?: Record<string, string>;
  signalTiming: 'next_bar_open' | 'same_bar_close';
  fillRule: 'bar_close' | 'market' | 'limit';
  simulationMode: 'KLINE_RANGE' | 'OHLC_PATH';
}

export const BacktestParamsModal: React.FC<BacktestParamsModalProps> = ({ open, onClose, onConfirm, code, symbol: _symbol, timeframe: initialTimeframe }) => {
  const { t, i18n } = useTranslation();
  const [validating, setValidating] = useState(false);
  const [validationError, setValidationError] = useState('');
  const [validated, setValidated] = useState(false);
  const [datePreset, setDatePreset] = useState('3M');
  const [extractedParams, setExtractedParams] = useState<Array<{ name: string; type: string; default: string; label: string }>>([]);
  const [strategyParamValues, setStrategyParamValues] = useState<Record<string, string>>({});

  const [params, setParams] = useState<StandardParams>(() => loadSavedDefaults() ?? FACTORY_DEFAULTS);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs | null, dayjs.Dayjs | null]>([
    dayjs().subtract(3, 'month'),
    dayjs(),
  ]);
  const [timeframe, setTimeframe] = useState(initialTimeframe || '1h');
  const [signalTiming, setSignalTiming] = useState<'next_bar_open' | 'same_bar_close'>('next_bar_open');
  const [fillRule, setFillRule] = useState<'bar_close' | 'market' | 'limit'>('bar_close');
  const [simulationMode, setSimulationMode] = useState<'KLINE_RANGE' | 'OHLC_PATH'>('KLINE_RANGE');

  useEffect(() => {
    if (open) setTimeframe(initialTimeframe || '1h');
  }, [open, initialTimeframe]);

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
    onConfirm({
      params: { ...params, strictMode: signalTiming === 'next_bar_open' },
      startDate, endDate, timeframe,
      strategyParams: strategyParamValues,
      signalTiming,
      fillRule,
      simulationMode,
    });
    onClose();
  };

  const defaultsMenuItems = [
    { key: 'save', label: t(SETTINGS_SAVE_KEY), onClick: () => { saveDefaults(params); message.success(t(DEFAULTS_SAVED_KEY)); } },
    ...(loadSavedDefaults() ? [{ key: 'load', label: t(SETTINGS_LOAD_KEY), onClick: () => { const d = loadSavedDefaults()!; setParams(d); message.success(t(DEFAULTS_LOADED_KEY)); } }] : []),
    { key: 'reset', label: t(SETTINGS_RESET_KEY), onClick: () => { removeDefaults(); setParams(FACTORY_DEFAULTS); message.success(t(DEFAULTS_RESET_KEY)); } },
  ];

  return (
    <Modal
      title={t(RUN_BACKTEST_KEY)}
      open={open}
      onCancel={onClose}
      width="min(620px, 92vw)"
      footer={[
        <Button key="cancel" onClick={onClose}>{t(COMMON_CANCEL_KEY)}</Button>,
        <Button key="run" type="primary" loading={validating} disabled={!validated} onClick={handleConfirm}
          style={{ background: '#3fb950', borderColor: '#3fb950' }}>
          {t(WS_BACKTEST_KEY)}
        </Button>,
      ]}
    >
      {/* Validation status */}
      {validating && <Alert type="info" showIcon message={t(GEN_COMPILING_KEY)} style={{ marginBottom: 12 }} />}
      {validationError && !validating && (
        <Alert type="error" showIcon style={{ marginBottom: 12 }} message={t(GEN_COMPILE_ERROR_KEY)}
          description={<pre style={{ fontSize: 11, whiteSpace: 'pre-wrap', margin: 0 }}>{validationError}</pre>}
          action={<Button size="small" onClick={doValidate}>{t(COMMON_RETRY_KEY)}</Button>} />
      )}
      {validated && !validating && <Alert type="success" showIcon message={t(CODE_VALID_KEY)} style={{ marginBottom: 12 }} />}

      <Form layout="vertical" size="small" disabled={!validated}>
        {/* Date range presets */}
        <Form.Item label={t(DATE_RANGE_KEY)}>
          <Space style={{ marginBottom: 8 }}>
            {DATE_PRESETS.map(p => (
              <Button key={p.key} size="small" type={datePreset === p.key ? 'primary' : 'default'} onClick={() => handlePresetClick(p.key)}>{p.label}</Button>
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

        {/* Timeframe selector — inline buttons, only 8 options */}
        <Form.Item label={t(TIMEFRAME_KEY)}>
          <Space wrap>
            {TIMEFRAMES.map(tf => (
              <Button key={tf} size="small" type={timeframe === tf ? 'primary' : 'default'} onClick={() => setTimeframe(tf)}>{tf}</Button>
            ))}
          </Space>
        </Form.Item>

        {/* Capital + Leverage + Commission + Slippage — 4 in a row with save/load defaults */}
        <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', marginBottom: 0 }}>
          <span style={{ fontSize: 12, color: 'var(--ant-color-text-secondary)' }}>{t(CAPITAL_KEY)} / {t(LEVERAGE_KEY)} / {t(COMMISSION_KEY)} / {t(SLIPPAGE_KEY)}</span>
          <Dropdown menu={{ items: defaultsMenuItems }}>
            <Button size="small" type="text" icon={<SettingOutlined />} />
          </Dropdown>
        </div>
        <Row gutter={16}>
          <Col xs={12} sm={12} md={6}>
            <Form.Item label={t(CAPITAL_KEY)}>
              <InputNumber value={params.initialCapital} onChange={(v) => setParams(p => ({ ...p, initialCapital: v ?? FACTORY_DEFAULTS.initialCapital }))} min={100} style={{ width: '100%' }} prefix="$" />
            </Form.Item>
          </Col>
          <Col xs={12} sm={12} md={6}>
            <Form.Item label={t(LEVERAGE_KEY)}>
              <InputNumber value={params.leverage} onChange={(v) => setParams(p => ({ ...p, leverage: v ?? FACTORY_DEFAULTS.leverage }))} min={1} max={1000} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={12} md={6}>
            <Form.Item label={t(COMMISSION_KEY)}>
              <InputNumber value={params.commission} onChange={(v) => setParams(p => ({ ...p, commission: v ?? FACTORY_DEFAULTS.commission }))} min={0} max={0.1} step={0.0005} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={12} md={6}>
            <Form.Item label={t(SLIPPAGE_KEY)}>
              <InputNumber value={params.slippage} onChange={(v) => setParams(p => ({ ...p, slippage: v ?? FACTORY_DEFAULTS.slippage }))} min={0} max={0.1} step={0.0005} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        {/* Execution assumptions: Direction + Mode + Signal Timing + Fill Rule (2x2 grid) */}
        <ExecutionAssumptionsSelectors
          simulationMode={simulationMode}
          signalTiming={signalTiming}
          fillRule={fillRule}
          tradeDirection={params.tradeDirection}
          onSimulationModeChange={setSimulationMode}
          onSignalTimingChange={setSignalTiming}
          onFillRuleChange={setFillRule}
          onTradeDirectionChange={(v) => setParams(p => ({ ...p, tradeDirection: v }))}
        />

        {/* Strategy params (extracted from code) */}
        <StrategyParamsSection
          extractedParams={extractedParams}
          strategyParamValues={strategyParamValues}
          onChange={(name, value) => setStrategyParamValues(prev => ({ ...prev, [name]: value }))}
          language={i18n.language}
        />
      </Form>
    </Modal>
  );
};
