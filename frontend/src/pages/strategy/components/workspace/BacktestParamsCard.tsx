import { Button, Row, Col, InputNumber, DatePicker, Segmented, Dropdown, message, Tooltip, Radio, Switch, Tag, Select } from 'antd';
import type { MenuProps } from 'antd';
import { PlayCircleOutlined, SettingOutlined, CaretUpOutlined, CaretDownOutlined, HistoryOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { DATE_PRESETS } from '../../hooks/useBacktestParams';
import type { StrategyDirective } from '../../hooks/useBacktestParams';
import { PRESETS } from '../../hooks/useBacktestParams';
import type { StrategyTemplate } from '@/client/strategy';
import { StrategyDirectivesCard } from './StrategyDirectivesCard';

const TIMEFRAMES = ['1m', '5m', '15m', '30m', '1h', '4h', '1d', '1w'];
const DEFAULTS_KEY = 'ant_backtest_defaults';
const FACTORY_DEFAULTS = {
  commission: 0.001, slippage: 0.0, leverage: 1,
  tradeDirection: 'both', strictMode: true,
};

function loadDefaults() {
  try { const raw = localStorage.getItem(DEFAULTS_KEY); return raw ? JSON.parse(raw) : null; }
  catch { return null; }
}
function saveDefaults(vals: Record<string, unknown>) {
  try { localStorage.setItem(DEFAULTS_KEY, JSON.stringify(vals)); } catch { /* quota exceeded */ }
}
function removeDefaults() {
  try { localStorage.removeItem(DEFAULTS_KEY); } catch { /* ignore */ }
}

interface TemplatesProp {
  list: StrategyTemplate[];
  loading: boolean;
  selectedId: string;
  onSelect: (id: string | null) => void;
}

interface Props {
  templates: TemplatesProp;
  initialCapital: number; onInitialCapitalChange: (v: number | null) => void;
  leverage: number; onLeverageChange: (v: number | null) => void;
  commission: number; onCommissionChange: (v: number | null) => void;
  slippage: number; onSlippageChange: (v: number | null) => void;
  startDate: string; onStartDateChange: (v: string) => void;
  endDate: string; onEndDateChange: (v: string) => void;
  tradeDirection: string; onTradeDirectionChange: (d: string) => void;
  strictMode: boolean; onStrictModeChange: (v: boolean) => void;
  canRun: boolean; running: boolean; onRunBacktest: () => void;
  datePresets: typeof DATE_PRESETS;
  datePresetKey: string; onApplyDatePreset: (p: { key: string; months: number }) => void;
  expanded: boolean; onExpandedChange: (v: boolean) => void;
  strategyDirectives: StrategyDirective[];
  onApplyPreset: (key: 'live_aligned' | 'exploration') => void;
  timeframeWarning: string | null;
  timeframe: string; onTimeframeChange: (tf: string) => void;
  onOpenHistory?: () => void;
  onApplyDefaults?: (defaults: {
    commission: number; slippage: number; leverage: number;
    tradeDirection: string; strictMode: boolean;
  }) => void;
}

const sectionLabel: React.CSSProperties = { fontSize: 10, fontWeight: 700, color: '#8c8c8c', textTransform: 'uppercase', marginBottom: 6 };
const fieldLabel: React.CSSProperties = { fontSize: 9, fontWeight: 600, color: '#8c8c8c', textTransform: 'uppercase', marginBottom: 2 };
const narrow: React.CSSProperties = { width: '100%' };

export default function BacktestParamsCard(props: Props) {
  const {
    templates,
    initialCapital, onInitialCapitalChange, leverage, onLeverageChange,
    commission, onCommissionChange, slippage, onSlippageChange,
    startDate, onStartDateChange, endDate, onEndDateChange,
    tradeDirection, onTradeDirectionChange, strictMode, onStrictModeChange,
    canRun, running, onRunBacktest, datePresets = [], datePresetKey, onApplyDatePreset,
    expanded, onExpandedChange, strategyDirectives = [],
    onApplyPreset, timeframeWarning, timeframe, onTimeframeChange,
    onOpenHistory, onApplyDefaults,
  } = props;

  const saved = loadDefaults();
  const settingsItems: MenuProps['items'] = [
    {
      key: 'save', label: 'Save as My Defaults',
      onClick: () => {
        saveDefaults({ commission, slippage, leverage, tradeDirection, strictMode });
        message.success('Defaults saved');
      },
    },
    ...(saved ? [{
      key: 'load', label: 'Load My Defaults',
      onClick: () => {
        onApplyDefaults?.(saved);
        message.success('Defaults loaded');
      },
    }] : []),
    {
      key: 'reset', label: 'Reset to Factory',
      onClick: () => {
        removeDefaults();
        onApplyDefaults?.(FACTORY_DEFAULTS);
        message.success('Reset to factory defaults');
      },
    },
  ];

  const strategyOptions = [
    { value: '__draft__', label: '📝 Current Draft' },
    ...(templates.list.length > 0 ? [{ value: '__sep__', label: '──────────────', disabled: true }] : []),
    ...templates.list.map((tpl: StrategyTemplate) => ({ value: tpl.id, label: tpl.name })),
  ];

  const slippagePct = (slippage * 100).toFixed(4).replace(/\.?0+$/, '');

  return (
    <div style={{
      borderBottom: '1px solid #e8e8e8', background: '#fafbfc',
      borderTop: '1px solid #e8e8e8',
    }}>
      {/* Header */}
      <div onClick={() => onExpandedChange(!expanded)} style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '9px 14px', cursor: 'pointer', userSelect: 'none',
        background: 'linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%)',
      }} onKeyUp={e => e.key === 'Enter' && onExpandedChange(!expanded)} role="button" tabIndex={0}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flex: 1, minWidth: 0 }}>
          <PlayCircleOutlined style={{ color: '#1890ff', flexShrink: 0 }} />
          <span style={{ fontSize: 12, fontWeight: 700, color: '#262626', whiteSpace: 'nowrap', flexShrink: 0 }}>Backtest</span>

          {/* Strategy selector */}
          <span onClick={e => e.stopPropagation()} style={{ flex: 1, minWidth: 120, maxWidth: 200 }}>
            <Select
              size="small"
              style={{ width: '100%' }}
              loading={templates.loading}
              value={templates.selectedId || '__draft__'}
              options={strategyOptions}
              onChange={(val) => {
                if (val === '__draft__') { templates.onSelect(null); }
                else if (val !== '__sep__') { templates.onSelect(val); }
              }}
              popupMatchSelectWidth={false}
            />
          </span>

          {/* Timeframe selector */}
          <span onClick={e => e.stopPropagation()}>
            <Segmented size="small" value={timeframe} onChange={v => onTimeframeChange(v as string)}
              options={TIMEFRAMES} style={{ fontSize: 10 }} />
          </span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0 }} onClick={e => e.stopPropagation()}>
          {onOpenHistory && (
            <Tooltip title="Backtest History">
              <Button size="small" type="text" icon={<HistoryOutlined />}
                onClick={onOpenHistory} style={{ borderRadius: 6 }} />
            </Tooltip>
          )}
          <Dropdown menu={{ items: settingsItems }} trigger={['click']} placement="bottomRight">
            <Button size="small" type="text" icon={<SettingOutlined />} style={{ borderRadius: 6 }} />
          </Dropdown>
          <Button type="primary" size="small" loading={running} disabled={!canRun}
            onClick={onRunBacktest}
            style={{ borderRadius: 6, fontWeight: 600, boxShadow: '0 2px 8px rgba(24,144,255,0.25)' }}>
            ▶ Run
          </Button>
          <span style={{ fontSize: 9, color: '#8c8c8c', cursor: 'pointer' }}>
            {expanded ? <CaretUpOutlined /> : <CaretDownOutlined />}
          </span>
        </div>
      </div>

      {expanded && (
        <div style={{ padding: '12px 14px' }}>
          {/* ── Row 1: Date Range (full width) ── */}
          <div style={{ marginBottom: 14 }}>
            <div style={sectionLabel}>Date Range</div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
              <Segmented
                size="small"
                value={datePresetKey}
                onChange={(v) => {
                  const p = datePresets.find(d => d.key === v);
                  if (p) onApplyDatePreset(p);
                }}
                options={datePresets.map(p => ({ value: p.key, label: p.label }))}
                style={{ fontSize: 10 }}
              />
              <DatePicker size="small" style={{ width: 130 }} value={startDate ? dayjs(startDate) : null}
                onChange={(d) => d && onStartDateChange(d.format('YYYY-MM-DD'))} placeholder="Start" />
              <DatePicker size="small" style={{ width: 130 }} value={endDate ? dayjs(endDate) : null}
                onChange={(d) => d && onEndDateChange(d.format('YYYY-MM-DD'))} placeholder="End" />
            </div>
            {timeframeWarning && (
              <div style={{ fontSize: 9, color: '#fa8c16', marginTop: 4 }}>
                ⚠ {timeframeWarning}
              </div>
            )}
          </div>

          {/* ── Row 2: 2-column grid ── */}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
            {/* Left: Execution Parameters */}
            <div>
              <div style={sectionLabel}>Execution</div>
              <Row gutter={8} style={{ marginBottom: 8 }}>
                <Col span={12}>
                  <div style={fieldLabel}>Capital</div>
                  <InputNumber size="small" style={narrow} min={100} step={1000}
                    value={initialCapital} onChange={onInitialCapitalChange}
                    formatter={v => `$ ${v}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}
                    parser={v => v!.replace(/\$\s?|(,*)/g, '') as unknown as number} />
                </Col>
                <Col span={12}>
                  <div style={fieldLabel}>Leverage</div>
                  <InputNumber size="small" style={narrow} min={1} max={125} step={1}
                    value={leverage} onChange={onLeverageChange}
                    formatter={v => `${v}x`} parser={v => v!.replace('x', '') as unknown as number} />
                </Col>
              </Row>
              <Row gutter={8} style={{ marginBottom: 6 }}>
                <Col span={12}>
                  <div style={fieldLabel}>
                    Commission
                    <span style={{ fontSize: 8, color: '#bfbfbf', fontWeight: 400, marginLeft: 3, textTransform: 'none' }}>%</span>
                  </div>
                  <InputNumber size="small" style={narrow} min={0} max={10} step={0.01} precision={4}
                    value={commission} onChange={onCommissionChange}
                    formatter={v => `${v}%`} parser={v => v!.replace('%', '') as unknown as number} />
                </Col>
                <Col span={12}>
                  <div style={fieldLabel}>
                    Slippage
                    <span style={{ fontSize: 8, color: '#bfbfbf', fontWeight: 400, marginLeft: 3, textTransform: 'none' }}>
                      {slippagePct}%
                    </span>
                  </div>
                  <InputNumber size="small" style={narrow} min={0} max={10} step={0.0001} precision={4}
                    value={slippage} onChange={onSlippageChange} />
                </Col>
              </Row>
              <div style={{ display: 'flex', gap: 4 }}>
                {Object.entries(PRESETS).map(([key, p]) => (
                  <Button key={key} size="small"
                    onClick={() => onApplyPreset(key as 'live_aligned' | 'exploration')}
                    style={{ fontSize: 9, padding: '0 8px', height: 22 }}
                  >{p.label}</Button>
                ))}
              </div>
            </div>

            {/* Right: Trade Settings */}
            <div>
              <div style={sectionLabel}>Trade</div>
              <div style={{ marginBottom: 14 }}>
                <div style={fieldLabel}>Direction</div>
                <Radio.Group value={tradeDirection} onChange={e => onTradeDirectionChange(e.target.value)}
                  size="small" buttonStyle="solid" style={{ display: 'flex' }}>
                  <Radio.Button value="long" style={{ flex: 1, textAlign: 'center', fontSize: 10 }}>↑ Long</Radio.Button>
                  <Radio.Button value="short" style={{ flex: 1, textAlign: 'center', fontSize: 10 }}>↓ Short</Radio.Button>
                  <Radio.Button value="both" style={{ flex: 1, textAlign: 'center', fontSize: 10 }}>Both</Radio.Button>
                </Radio.Group>
              </div>
              <div>
                <div style={fieldLabel}>Strict Mode</div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <Switch size="small" checked={strictMode} onChange={onStrictModeChange} />
                  <Tooltip title={strictMode
                    ? 'ON: signals confirmed at bar close, executed next bar open'
                    : 'OFF: same-bar close execution with 1m sub-resolution'}>
                    <Tag color={strictMode ? 'blue' : 'orange'} style={{ fontSize: 8, margin: 0, lineHeight: '14px', cursor: 'help' }}>
                      {strictMode ? 'ON' : 'OFF'}
                    </Tag>
                  </Tooltip>
                </div>
                <div style={{ fontSize: 8, color: '#8c8c8c', marginTop: 2, lineHeight: '12px' }}>
                  {strictMode ? 'Next-bar-open. Standard, conservative.' : 'Same-bar-close + MTF 1m. Higher precision.'}
                </div>
              </div>
            </div>
          </div>

          {strategyDirectives.length > 0 && (
            <div style={{ marginTop: 12 }}>
              <StrategyDirectivesCard directives={strategyDirectives} />
            </div>
          )}
        </div>
      )}
    </div>
  );
}
