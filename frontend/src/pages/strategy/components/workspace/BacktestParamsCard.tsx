import { Button, Dropdown, Tooltip, Select } from 'antd';
import { PlayCircleOutlined, SettingOutlined, CaretUpOutlined, CaretDownOutlined, HistoryOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { CURRENT_DRAFT_KEY, HISTORY_KEY, RUN_KEY, TITLE_KEY } from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import { RUN_DISABLED_HINT_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
;
import { DATE_PRESETS } from '../../hooks/useBacktestParams';
import type { StrategyDirective } from '../../hooks/useBacktestParams';
import type { StrategyTemplate } from '@/client/strategy';
import { useBacktestDefaults } from '../../hooks/useBacktestDefaults';
import BacktestParamsExpanded from './BacktestParamsExpanded';

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
  lotSize: number; onLotSizeChange: (v: number | null) => void;
  startDate: string; onStartDateChange: (v: string) => void;
  endDate: string; onEndDateChange: (v: string) => void;
  tradeDirection: string; onTradeDirectionChange: (d: string) => void;
  strictMode: boolean; onStrictModeChange: (v: boolean) => void;
  canRun: boolean; running: boolean; onRunBacktest: () => void;
  datePresets: typeof DATE_PRESETS;
  datePresetKey: string; onApplyDatePreset: (p: { key: string; months: number }) => void;
  expanded: boolean; onExpandedChange: (v: boolean) => void;
  strategyDirectives: StrategyDirective[];
  strategyParams: Array<{ name: string; type: string; default: string; label: string }>;
  strategyParamValues: Record<string, string>;
  onStrategyParamChange: (name: string, value: string) => void;
  onApplyPreset: (key: 'live_aligned' | 'exploration') => void;
  timeframeWarning: string | null;
  onOpenHistory?: () => void;
  onApplyDefaults?: (defaults: {
    commission: number; slippage: number; leverage: number;
    tradeDirection: string; strictMode: boolean;
  }) => void;
}

export default function BacktestParamsCard(props: Props) {
  const {
    templates,
    initialCapital, onInitialCapitalChange, leverage, onLeverageChange,
    commission, onCommissionChange, slippage, onSlippageChange,
    lotSize, onLotSizeChange,
    startDate, onStartDateChange, endDate, onEndDateChange,
    tradeDirection, onTradeDirectionChange, strictMode, onStrictModeChange,
    canRun, running, onRunBacktest, datePresets = [], datePresetKey, onApplyDatePreset,
    expanded, onExpandedChange, strategyDirectives = [],
    strategyParams = [], strategyParamValues = {}, onStrategyParamChange,
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
    { value: '__draft__', label: t(CURRENT_DRAFT_KEY) },
    ...(templates.list.length > 0 ? [{ value: '__sep__', label: '──────────────', disabled: true }] : []),
    ...templates.list.map((tpl: StrategyTemplate) => ({ value: tpl.id, label: tpl.name })),
  ];

  return (
    <div style={{
      borderBottom: '1px solid #e8e8e8', background: '#fafbfc',
      borderTop: '1px solid #e8e8e8',
    }}>
      <div onClick={() => onExpandedChange(!expanded)} style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '9px 14px', cursor: 'pointer', userSelect: 'none',
        background: 'linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%)',
      }} onKeyUp={e => e.key === 'Enter' && onExpandedChange(!expanded)} role="button" tabIndex={0}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flex: 1, minWidth: 0 }}>
          <PlayCircleOutlined style={{ color: '#1890ff', flexShrink: 0 }} />
          <span style={{ fontSize: 12, fontWeight: 700, color: '#262626', whiteSpace: 'nowrap', flexShrink: 0 }}>{t(TITLE_KEY)}</span>
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
            <Tooltip title={t(HISTORY_KEY)}>
              <Button size="small" type="text" icon={<HistoryOutlined />}
                onClick={onOpenHistory} style={{ borderRadius: 6 }} />
            </Tooltip>
          )}
          <Dropdown menu={{ items: settingsItems }} trigger={['click']} placement="bottomRight">
            <Button size="small" type="text" icon={<SettingOutlined />} style={{ borderRadius: 6 }} />
          </Dropdown>
          <Tooltip title={!canRun ? t(RUN_DISABLED_HINT_KEY) : undefined}>
            <Button type="primary" size="small" loading={running} disabled={!canRun}
              onClick={onRunBacktest}
              style={{ borderRadius: 6, fontWeight: 600, boxShadow: '0 2px 8px rgba(24,144,255,0.25)' }}>
              {t(RUN_KEY)}
            </Button>
          </Tooltip>
          <span style={{ fontSize: 9, color: '#8c8c8c', cursor: 'pointer' }}>
            {expanded ? <CaretUpOutlined /> : <CaretDownOutlined />}
          </span>
        </div>
      </div>

      {expanded && (
        <BacktestParamsExpanded
          initialCapital={initialCapital} onInitialCapitalChange={onInitialCapitalChange}
          leverage={leverage} onLeverageChange={onLeverageChange}
          commission={commission} onCommissionChange={onCommissionChange}
          slippage={slippage} onSlippageChange={onSlippageChange}
          lotSize={lotSize} onLotSizeChange={onLotSizeChange}
          startDate={startDate} onStartDateChange={onStartDateChange}
          endDate={endDate} onEndDateChange={onEndDateChange}
          tradeDirection={tradeDirection} onTradeDirectionChange={onTradeDirectionChange}
          strictMode={strictMode} onStrictModeChange={onStrictModeChange}
          datePresets={datePresets} datePresetKey={datePresetKey} onApplyDatePreset={onApplyDatePreset}
          strategyDirectives={strategyDirectives}
          strategyParams={strategyParams}
          strategyParamValues={strategyParamValues}
          onStrategyParamChange={onStrategyParamChange}
          onApplyPreset={onApplyPreset}
          timeframeWarning={timeframeWarning}
        />
      )}
    </div>
  );
}
