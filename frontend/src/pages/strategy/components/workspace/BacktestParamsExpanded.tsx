import { Row, Col, InputNumber, DatePicker, Segmented, Radio, Switch, Tag, Tooltip, Button } from 'antd';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';
import {
  CAPITAL_KEY, COMMISSION_KEY, DATE_RANGE_KEY, DIRECTION_KEY, END_DATE_KEY,
  EXECUTION_KEY, LEVERAGE_KEY, LONG_KEY, SHORT_KEY, BOTH_KEY,
  SLIPPAGE_KEY, START_DATE_KEY, STRICT_MODE_KEY, STRICT_MODE_OFF_DESC_KEY,
  STRICT_MODE_OFF_KEY, STRICT_MODE_OFF_TOOLTIP_KEY, STRICT_MODE_ON_DESC_KEY,
  STRICT_MODE_ON_KEY, STRICT_MODE_ON_TOOLTIP_KEY, TRADE_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import { PRESETS } from '../../hooks/useBacktestParams';
import type { StrategyDirective } from '../../hooks/useBacktestParams';
import { StrategyDirectivesCard } from './StrategyDirectivesCard';

const sectionLabel: React.CSSProperties = { fontSize: 10, fontWeight: 700, color: '#8c8c8c', textTransform: 'uppercase', marginBottom: 6 };
const fieldLabel: React.CSSProperties = { fontSize: 9, fontWeight: 600, color: '#8c8c8c', textTransform: 'uppercase', marginBottom: 2 };
const narrow: React.CSSProperties = { width: '100%' };

const presetI18n: Record<string, string> = {
  live_aligned: 'strategy.backtestParams.presets.liveAligned',
  exploration: 'strategy.backtestParams.presets.exploration',
};

interface ExpandedProps {
  initialCapital: number; onInitialCapitalChange: (v: number | null) => void;
  leverage: number; onLeverageChange: (v: number | null) => void;
  commission: number; onCommissionChange: (v: number | null) => void;
  slippage: number; onSlippageChange: (v: number | null) => void;
  lotSize: number; onLotSizeChange: (v: number | null) => void;
  startDate: string; onStartDateChange: (v: string) => void;
  endDate: string; onEndDateChange: (v: string) => void;
  tradeDirection: string; onTradeDirectionChange: (d: string) => void;
  strictMode: boolean; onStrictModeChange: (v: boolean) => void;
  datePresets: { key: string; label: string }[];
  datePresetKey: string; onApplyDatePreset: (p: { key: string; months: number }) => void;
  strategyDirectives: StrategyDirective[];
  strategyParams: Array<{ name: string; type: string; default: string; label: string }>;
  strategyParamValues: Record<string, string>;
  onStrategyParamChange: (name: string, value: string) => void;
  onApplyPreset: (key: 'live_aligned' | 'exploration') => void;
  timeframeWarning: string | null;
}

export default function BacktestParamsExpanded(props: ExpandedProps) {
  const { t } = useTranslation();
  const {
    initialCapital, onInitialCapitalChange, leverage, onLeverageChange,
    commission, onCommissionChange, slippage, onSlippageChange,
    lotSize, onLotSizeChange,
    startDate, onStartDateChange, endDate, onEndDateChange,
    tradeDirection, onTradeDirectionChange, strictMode, onStrictModeChange,
    datePresets = [], datePresetKey, onApplyDatePreset,
    strategyDirectives = [], strategyParams = [], strategyParamValues = {},
    onStrategyParamChange, onApplyPreset, timeframeWarning,
  } = props;

  const slippagePct = (slippage * 100).toFixed(4).replace(/\.?0+$/, '');

  return (
    <div style={{ padding: '12px 14px' }}>
      <div style={{ marginBottom: 14 }}>
        <div style={sectionLabel}>{t(DATE_RANGE_KEY)}</div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
          <Segmented
            size="small"
            value={datePresetKey}
            onChange={(v) => {
              const p = datePresets.find(d => d.key === v);
              if (p) onApplyDatePreset(p as unknown as { key: string; months: number });
            }}
            options={datePresets.map(p => ({ value: p.key, label: p.label }))}
            style={{ fontSize: 10 }}
          />
          <DatePicker size="small" style={{ width: 130 }} value={startDate ? dayjs(startDate) : null}
            onChange={(d) => d && onStartDateChange(d.format('YYYY-MM-DD'))} placeholder={t(START_DATE_KEY)} />
          <DatePicker size="small" style={{ width: 130 }} value={endDate ? dayjs(endDate) : null}
            onChange={(d) => d && onEndDateChange(d.format('YYYY-MM-DD'))} placeholder={t(END_DATE_KEY)} />
        </div>
        {timeframeWarning && (
          <div style={{ fontSize: 9, color: '#fa8c16', marginTop: 4 }}>⚠ {timeframeWarning}</div>
        )}
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
        <div>
          <div style={sectionLabel}>{t(EXECUTION_KEY)}</div>
          <Row gutter={8} style={{ marginBottom: 8 }}>
            <Col span={12}>
              <div style={fieldLabel}>{t(CAPITAL_KEY)}</div>
              <InputNumber size="small" style={narrow} min={100} step={1000}
                value={initialCapital} onChange={onInitialCapitalChange}
                formatter={v => `$ ${v}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}
                parser={v => v!.replace(/\$\s?|(,*)/g, '') as unknown as number} />
            </Col>
            <Col span={12}>
              <div style={fieldLabel}>{t(LEVERAGE_KEY)}</div>
              <InputNumber size="small" style={narrow} min={1} step={1}
                value={leverage} onChange={onLeverageChange}
                formatter={v => `${v}x`} parser={v => v!.replace('x', '') as unknown as number} />
            </Col>
          </Row>
          <Row gutter={8} style={{ marginBottom: 8 }}>
            <Col span={12}>
              <div style={fieldLabel}>Lot Size</div>
              <InputNumber size="small" style={narrow} min={0.01} max={100} step={0.01}
                value={lotSize} onChange={onLotSizeChange} />
            </Col>
            <Col span={12}>
              <div style={fieldLabel}>
                {t(COMMISSION_KEY)}
                <span style={{ fontSize: 8, color: '#bfbfbf', fontWeight: 400, marginLeft: 3, textTransform: 'none' }}>%</span>
              </div>
              <InputNumber size="small" style={narrow} min={0} max={10} step={0.01} precision={4}
                value={commission} onChange={onCommissionChange}
                formatter={v => `${v}%`} parser={v => v!.replace('%', '') as unknown as number} />
            </Col>
          </Row>
          <Row gutter={8} style={{ marginBottom: 6 }}>
            <Col span={12}>
              <div style={fieldLabel}>
                {t(SLIPPAGE_KEY)}
                <span style={{ fontSize: 8, color: '#bfbfbf', fontWeight: 400, marginLeft: 3, textTransform: 'none' }}>
                  {slippagePct}%
                </span>
              </div>
              <InputNumber size="small" style={narrow} min={0} max={10} step={0.0001} precision={4}
                value={slippage} onChange={onSlippageChange} />
            </Col>
            <Col span={12} />
          </Row>
          <div style={{ display: 'flex', gap: 4 }}>
            {Object.entries(PRESETS).map(([key]) => (
              <Button key={key} size="small"
                onClick={() => onApplyPreset(key as 'live_aligned' | 'exploration')}
                style={{ fontSize: 9, padding: '0 8px', height: 22 }}
              >{t(presetI18n[key])}</Button>
            ))}
          </div>
        </div>

        <div>
          <div style={sectionLabel}>{t(TRADE_KEY)}</div>
          <div style={{ marginBottom: 14 }}>
            <div style={fieldLabel}>{t(DIRECTION_KEY)}</div>
            <Radio.Group value={tradeDirection} onChange={e => onTradeDirectionChange(e.target.value)}
              size="small" buttonStyle="solid" style={{ display: 'flex' }}>
              <Radio.Button value="long" style={{ flex: 1, textAlign: 'center', fontSize: 10 }}>{t(LONG_KEY)}</Radio.Button>
              <Radio.Button value="short" style={{ flex: 1, textAlign: 'center', fontSize: 10 }}>{t(SHORT_KEY)}</Radio.Button>
              <Radio.Button value="both" style={{ flex: 1, textAlign: 'center', fontSize: 10 }}>{t(BOTH_KEY)}</Radio.Button>
            </Radio.Group>
          </div>
          <div>
            <div style={fieldLabel}>{t(STRICT_MODE_KEY)}</div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Switch size="small" checked={strictMode} onChange={onStrictModeChange} />
              <Tooltip title={strictMode
                ? t(STRICT_MODE_ON_TOOLTIP_KEY)
                : t(STRICT_MODE_OFF_TOOLTIP_KEY)}>
                <Tag color={strictMode ? 'blue' : 'orange'} style={{ fontSize: 8, margin: 0, lineHeight: '14px', cursor: 'help' }}>
                  {strictMode ? t(STRICT_MODE_ON_KEY) : t(STRICT_MODE_OFF_KEY)}
                </Tag>
              </Tooltip>
            </div>
            <div style={{ fontSize: 8, color: '#8c8c8c', marginTop: 2, lineHeight: '12px' }}>
              {strictMode ? t(STRICT_MODE_ON_DESC_KEY) : t(STRICT_MODE_OFF_DESC_KEY)}
            </div>
          </div>
        </div>
      </div>

      {strategyDirectives.length > 0 && (
        <div style={{ marginTop: 12 }}>
          <StrategyDirectivesCard directives={strategyDirectives} />
        </div>
      )}

      {strategyParams.length > 0 && (
        <div style={{ marginTop: 12, borderTop: '1px solid #f0f0f0', paddingTop: 12 }}>
          <div style={sectionLabel}>Strategy Parameters</div>
          <Row gutter={8}>
            {strategyParams.map((p) => {
              const value = strategyParamValues[p.name] ?? p.default;
              if (p.type === 'bool') {
                return (
                  <Col span={8} key={p.name} style={{ marginBottom: 6 }}>
                    <div style={fieldLabel}>{p.label || p.name}</div>
                    <Switch size="small" checked={value === 'True' || value === 'true'}
                      onChange={(v) => onStrategyParamChange(p.name, v ? 'True' : 'False')} />
                  </Col>
                );
              }
              if (p.type === 'int') {
                return (
                  <Col span={8} key={p.name} style={{ marginBottom: 6 }}>
                    <div style={fieldLabel}>{p.label || p.name}</div>
                    <InputNumber size="small" style={narrow} step={1}
                      value={Number(value)} onChange={(v) => onStrategyParamChange(p.name, String(v ?? p.default))} />
                  </Col>
                );
              }
              return (
                <Col span={8} key={p.name} style={{ marginBottom: 6 }}>
                  <div style={fieldLabel}>{p.label || p.name}</div>
                  <InputNumber size="small" style={narrow} step={p.type === 'float' ? 0.01 : 1}
                    value={Number(value)} onChange={(v) => onStrategyParamChange(p.name, String(v ?? p.default))} />
                </Col>
              );
            })}
          </Row>
        </div>
      )}
    </div>
  );
}
