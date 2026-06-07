import { Button, Row, Col, InputNumber, DatePicker, Segmented, Dropdown, message, Tooltip } from 'antd';
import type { MenuProps } from 'antd';
import { PlayCircleOutlined, SettingOutlined, CaretUpOutlined, CaretDownOutlined, HistoryOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { DATE_PRESETS } from '../../hooks/useBacktestParams';
import type { StrategyDirective } from '../../hooks/useBacktestParams';
import { StrategyDirectivesCard } from './StrategyDirectivesCard';
import BacktestConfigSection from './BacktestConfigSection';

const TIMEFRAMES = ['1m', '5m', '15m', '30m', '1h', '4h', '1d', '1w'];
const DEFAULTS_KEY = 'ant_backtest_defaults';
// TODO(P3-6): Fetch factory defaults from server (GET /api/user/preferences or proto RPC).
// Currently localStorage takes precedence; FACTORY_DEFAULTS are the fallback.
const FACTORY_DEFAULTS = {
  commission: 0.001, slippage: 0.0, leverage: 1,
  tradeDirection: 'both', strictMode: true,
};

function loadDefaults() {
  try {
    const raw = localStorage.getItem(DEFAULTS_KEY);
    return raw ? JSON.parse(raw) : null;
  } catch { return null; }
}
function saveDefaults(vals: Record<string, unknown>) {
  try { localStorage.setItem(DEFAULTS_KEY, JSON.stringify(vals)); } catch { /* quota exceeded */ }
}
function removeDefaults() {
  try { localStorage.removeItem(DEFAULTS_KEY); } catch { /* ignore */ }
}

interface Props {
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

const fieldLabel: React.CSSProperties = { fontSize: 9, fontWeight: 600, color: '#8c8c8c', textTransform: 'uppercase', marginBottom: 2 };
const paramLabel: React.CSSProperties = { fontSize: 9, fontWeight: 700, color: '#8c8c8c', textTransform: 'uppercase', marginBottom: 4 };
const narrow = { width: '100%' };

export default function BacktestParamsCard(props: Props) {
  const {
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
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <PlayCircleOutlined style={{ color: '#1890ff' }} />
          <span style={{ fontSize: 12, fontWeight: 700, color: '#262626' }}>Backtest Parameters</span>
          {/* Timeframe selector — always visible, even when collapsed */}
          <span onClick={e => e.stopPropagation()}>
            <Segmented size="small" value={timeframe} onChange={v => onTimeframeChange(v as string)}
              options={TIMEFRAMES} style={{ fontSize: 10 }} />
          </span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }} onClick={e => e.stopPropagation()}>
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
            ▶ Run Backtest
          </Button>
          <span style={{ fontSize: 9, color: '#8c8c8c', cursor: 'pointer' }}>
            {expanded ? <CaretUpOutlined /> : <CaretDownOutlined />}
          </span>
        </div>
      </div>

      {expanded && (
        <div style={{ padding: '12px 14px' }}>
          {/* Top row: 3-column grid */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 14 }}>
            {/* Date Range */}
            <div>
              <div style={paramLabel}>Date Range</div>
              <div style={{ display: 'flex', gap: 4, marginBottom: 6 }}>
                {datePresets.map(p => (
                  <Button key={p.key} size="small"
                    type={datePresetKey === p.key ? 'primary' : 'default'}
                    onClick={() => onApplyDatePreset(p)}
                  >{p.label}</Button>
                ))}
              </div>
              <Row gutter={8}>
                <Col span={12}>
                  <DatePicker size="small" style={narrow} value={startDate ? dayjs(startDate) : null}
                    onChange={(d) => d && onStartDateChange(d.format('YYYY-MM-DD'))} placeholder="Start" />
                </Col>
                <Col span={12}>
                  <DatePicker size="small" style={narrow} value={endDate ? dayjs(endDate) : null}
                    onChange={(d) => d && onEndDateChange(d.format('YYYY-MM-DD'))} placeholder="End" />
                </Col>
              </Row>
              {timeframeWarning && (
                <div style={{ fontSize: 9, color: '#fa8c16', marginTop: 4 }}>
                  ⚠ {timeframeWarning}
                </div>
              )}
            </div>

            {/* Capital & Leverage */}
            <div>
              <div style={paramLabel}>Capital & Leverage</div>
              <Row gutter={8}>
                <Col span={12}>
                  <div style={fieldLabel}>Initial Capital</div>
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
              <BacktestConfigSection
                commission={commission} slippage={slippage}
                tradeDirection={tradeDirection} strictMode={strictMode}
                onCommissionChange={onCommissionChange} onSlippageChange={onSlippageChange}
                onTradeDirectionChange={onTradeDirectionChange}
                onStrictModeChange={onStrictModeChange}
                onApplyPreset={onApplyPreset}
              />
            </div>
          </div>

          {strategyDirectives.length > 0 && (
            <StrategyDirectivesCard directives={strategyDirectives} />
          )}
        </div>
      )}
    </div>
  );
}
