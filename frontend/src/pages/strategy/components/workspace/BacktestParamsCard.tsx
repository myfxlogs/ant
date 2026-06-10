import { Button, Row, Col, InputNumber, DatePicker, Segmented, Dropdown, Tooltip, Radio, Switch, Tag, Select } from 'antd';
import { PlayCircleOutlined, SettingOutlined, CaretUpOutlined, CaretDownOutlined, HistoryOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';
import { DATE_PRESETS } from '../../hooks/useBacktestParams';
import type { StrategyDirective } from '../../hooks/useBacktestParams';
import { PRESETS } from '../../hooks/useBacktestParams';
import type { StrategyTemplate } from '@/client/strategy';
import { StrategyDirectivesCard } from './StrategyDirectivesCard';
import { useBacktestDefaults, FACTORY_DEFAULTS } from '../../hooks/useBacktestDefaults';

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
    onApplyPreset, timeframeWarning,
    onOpenHistory, onApplyDefaults,
  } = props;

  const { t } = useTranslation();
  const { settingsItems } = useBacktestDefaults(
    t,
    { commission, slippage, leverage, tradeDirection, strictMode },
    onApplyDefaults,
  );

  const strategyOptions = [
    { value: '__draft__', label: t('strategy.backtestParams.currentDraft') },
    ...(templates.list.length > 0 ? [{ value: '__sep__', label: '──────────────', disabled: true }] : []),
    ...templates.list.map((tpl: StrategyTemplate) => ({ value: tpl.id, label: tpl.name })),
  ];

  const presetI18n: Record<string, string> = {
    live_aligned: 'strategy.backtestParams.presets.liveAligned',
    exploration: 'strategy.backtestParams.presets.exploration',
  };

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
          <span style={{ fontSize: 12, fontWeight: 700, color: '#262626', whiteSpace: 'nowrap', flexShrink: 0 }}>{t('strategy.backtestParams.title')}</span>

          {/* Strategy selector */}
          <span onClick={e => e.stopPropagation()} style={{ flex: 1, minWidth: 120, maxWidth: 220 }}>
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
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0 }} onClick={e => e.stopPropagation()}>
          {onOpenHistory && (
            <Tooltip title={t('strategy.backtestParams.history')}>
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
            {t('strategy.backtestParams.run')}
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
            <div style={sectionLabel}>{t('strategy.backtestParams.dateRange')}</div>
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
              <div style={sectionLabel}>{t('strategy.backtestParams.execution')}</div>
              <Row gutter={8} style={{ marginBottom: 8 }}>
                <Col span={12}>
                  <div style={fieldLabel}>{t('strategy.backtestParams.capital')}</div>
                  <InputNumber size="small" style={narrow} min={100} step={1000}
                    value={initialCapital} onChange={onInitialCapitalChange}
                    formatter={v => `$ ${v}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}
                    parser={v => v!.replace(/\$\s?|(,*)/g, '') as unknown as number} />
                </Col>
                <Col span={12}>
                  <div style={fieldLabel}>{t('strategy.backtestParams.leverage')}</div>
                  <InputNumber size="small" style={narrow} min={1} max={125} step={1}
                    value={leverage} onChange={onLeverageChange}
                    formatter={v => `${v}x`} parser={v => v!.replace('x', '') as unknown as number} />
                </Col>
              </Row>
              <Row gutter={8} style={{ marginBottom: 6 }}>
                <Col span={12}>
                  <div style={fieldLabel}>
                    {t('strategy.backtestParams.commission')}
                    <span style={{ fontSize: 8, color: '#bfbfbf', fontWeight: 400, marginLeft: 3, textTransform: 'none' }}>%</span>
                  </div>
                  <InputNumber size="small" style={narrow} min={0} max={10} step={0.01} precision={4}
                    value={commission} onChange={onCommissionChange}
                    formatter={v => `${v}%`} parser={v => v!.replace('%', '') as unknown as number} />
                </Col>
                <Col span={12}>
                  <div style={fieldLabel}>
                    {t('strategy.backtestParams.slippage')}
                    <span style={{ fontSize: 8, color: '#bfbfbf', fontWeight: 400, marginLeft: 3, textTransform: 'none' }}>
                      {slippagePct}%
                    </span>
                  </div>
                  <InputNumber size="small" style={narrow} min={0} max={10} step={0.0001} precision={4}
                    value={slippage} onChange={onSlippageChange} />
                </Col>
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

            {/* Right: Trade Settings */}
            <div>
              <div style={sectionLabel}>{t('strategy.backtestParams.trade')}</div>
              <div style={{ marginBottom: 14 }}>
                <div style={fieldLabel}>{t('strategy.backtestParams.direction')}</div>
                <Radio.Group value={tradeDirection} onChange={e => onTradeDirectionChange(e.target.value)}
                  size="small" buttonStyle="solid" style={{ display: 'flex' }}>
                  <Radio.Button value="long" style={{ flex: 1, textAlign: 'center', fontSize: 10 }}>{t('strategy.backtestParams.long')}</Radio.Button>
                  <Radio.Button value="short" style={{ flex: 1, textAlign: 'center', fontSize: 10 }}>{t('strategy.backtestParams.short')}</Radio.Button>
                  <Radio.Button value="both" style={{ flex: 1, textAlign: 'center', fontSize: 10 }}>{t('strategy.backtestParams.both')}</Radio.Button>
                </Radio.Group>
              </div>
              <div>
                <div style={fieldLabel}>{t('strategy.backtestParams.strictMode')}</div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <Switch size="small" checked={strictMode} onChange={onStrictModeChange} />
                  <Tooltip title={strictMode
                    ? t('strategy.backtestParams.strictModeOnTooltip')
                    : t('strategy.backtestParams.strictModeOffTooltip')}>
                    <Tag color={strictMode ? 'blue' : 'orange'} style={{ fontSize: 8, margin: 0, lineHeight: '14px', cursor: 'help' }}>
                      {strictMode ? t('strategy.backtestParams.strictModeOn') : t('strategy.backtestParams.strictModeOff')}
                    </Tag>
                  </Tooltip>
                </div>
                <div style={{ fontSize: 8, color: '#8c8c8c', marginTop: 2, lineHeight: '12px' }}>
                  {strictMode ? t('strategy.backtestParams.strictModeOnDesc') : t('strategy.backtestParams.strictModeOffDesc')}
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
