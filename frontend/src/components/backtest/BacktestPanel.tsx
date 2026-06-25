import { useCallback, useMemo, useRef } from 'react';
import { Button, Tabs, InputNumber, Row, Col, Segmented, DatePicker, Radio, Switch, Tag, Tooltip, Dropdown, Select } from 'antd';
import {
  PlayCircleOutlined, SettingOutlined, CaretDownOutlined,
  HistoryOutlined, DoubleRightOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';
import {
  BOTH_KEY, CAPITAL_KEY, COMMISSION_KEY, CURRENT_DRAFT_KEY, DATE_RANGE_KEY, DIRECTION_KEY,
  END_DATE_KEY, EXECUTION_KEY, HISTORY_KEY, LEVERAGE_KEY, LONG_KEY, LOT_SIZE_KEY,
  MORE_KEY, PNL_KEY, RUN_KEY, SHORT_KEY, SLIPPAGE_KEY, START_DATE_KEY, STRATEGY_KEY, STRATEGY_PARAMS_KEY, STRICT_MODE_KEY,
  STRICT_MODE_OFF_DESC_KEY, STRICT_MODE_OFF_KEY, STRICT_MODE_OFF_TOOLTIP_KEY,
  STRICT_MODE_ON_DESC_KEY, STRICT_MODE_ON_KEY, STRICT_MODE_ON_TOOLTIP_KEY,
  TITLE_KEY, TRADE_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import { RUN_DISABLED_HINT_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { VALIDATE_TO_SEE_PARAMS_KEY } from '@/gen/ant/v1/i18n/strategy_code_assist_keys';
import {
  BACKTEST_COMPLETED_KEY, BACKTEST_TAB_KEY, GATE_TAB_KEY, TUNING_TAB_KEY,
} from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import {
  BACKTEST_RECORDS_KEY, TOTAL_RETURN_KEY, TOTAL_TRADES_KEY, PROFIT_FACTOR_KEY, WIN_RATE_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_keys';
import SmartTuningPanel from '@/pages/strategy/components/workspace/SmartTuningPanel';
import GatePanel from '@/pages/strategy/components/workspace/GatePanel';
import StrategyParamsModal from './StrategyParamsModal';
import BacktestResultsTab from './BacktestResultsTab';
import { paramLabel } from '@/utils/paramLabel';
import BacktestTradesTab from './BacktestTradesTab';
import type { useBacktestRunner, BacktestRunnerInputs } from './useBacktestRunner';
import { DATE_PRESETS, PRESETS } from '@/pages/strategy/hooks/backtestParamHelpers';
import type { StrategyTemplate } from '@/client/strategy';

const S = {
  sectionLabel: { fontSize: 12, fontWeight: 700, color: '#595959', marginBottom: 6 },
  fieldLabel: { fontSize: 12, fontWeight: 500, color: '#8c8c8c', marginBottom: 4 },
  narrow: { width: '100%' },
};

interface TemplatesProp {
  list: StrategyTemplate[];
  loading: boolean;
  selectedId: string;
  onSelect: (id: string | null) => void;
}

interface Props {
  runner: ReturnType<typeof useBacktestRunner>;
  inputs: BacktestRunnerInputs;
  templates: TemplatesProp;
  collapsed: boolean; onToggleCollapsed: () => void;
  onOpenHistory?: () => void;
  onAIOptimize?: () => void;
  code?: string;
  onApplyTunedParams?: (code: string) => void;
}

function MetricsRow({ m, t }: { m: any; t: any }) {
  if (!m) return null;
  return (
    <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', fontSize: 12, color: '#595959' }}>
      <span>{t(TOTAL_RETURN_KEY)} <b style={{ color: (m.totalReturn ?? 0) >= 0 ? '#26a69a' : '#e57373' }}>{(m.totalReturn ?? 0).toFixed(2)}%</b></span>
      <span>{t(TOTAL_TRADES_KEY)} <b>{m.totalTrades ?? 0}</b></span>
      <span>{t(WIN_RATE_KEY)} <b>{((m.winRate ?? 0) * 100).toFixed(1)}%</b></span>
      <span>{t(PROFIT_FACTOR_KEY)}: <b>{m.profitFactor?.toFixed(2) ?? '—'}</b></span>
      <span>{t(SHARPE_KEY)} <b>{m.sharpeRatio?.toFixed(2) ?? '—'}</b></span>
      <span>{t(MAX_DRAWDOWN_KEY)} <b style={{ color: '#e57373' }}>{(m.maxDrawdown ?? 0).toFixed(2)}%</b></span>
    </div>
  );
}

export default function BacktestPanel(props: Props) {
  const {
    runner, inputs, templates, collapsed, onToggleCollapsed,
    onOpenHistory, onAIOptimize, code, onApplyTunedParams,
  } = props;
  const { t, i18n } = useTranslation();
  const loc = i18n.language;

  const tplList = templates?.list || [];
  const selectedTpl = tplList.find((t: StrategyTemplate) => t.id === templates?.selectedId);
  const i18nData = useMemo(() => {
    if (!selectedTpl?.i18n) return null;
    try { return JSON.parse(selectedTpl.i18n as string); } catch { return null; }
  }, [selectedTpl?.i18n]);

  const handleRun = () => runner.run(inputs);
  const canRun = Boolean(inputs.strategyCode && inputs.symbol) && !runner.submitting;
  const slippagePct = (runner.slippage * 100).toFixed(4).replace(/\.?0+$/, '');

  // ── Resize handle ─────────────────────────────────────────────────────
  const resizeRef = useRef<HTMLDivElement>(null);
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    runner.setDragging(true);
    const startY = e.clientY;
    const startH = runner.panelHeight;
    const onMove = (ev: MouseEvent) => {
      const delta = startY - ev.clientY;
      runner.setPanelHeight(Math.max(160, Math.min(600, startH + delta)));
    };
    const onUp = () => {
      runner.setDragging(false);
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
    };
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
  }, [runner]);

  // ── Collapsed state ───────────────────────────────────────────────────
  if (collapsed) {
    return (
      <div style={{
        borderTop: '2px solid #e8e8e8', background: '#fafbfc',
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '6px 14px', cursor: 'pointer', userSelect: 'none',
      }} onClick={onToggleCollapsed}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <DoubleRightOutlined style={{ fontSize: 12, color: '#1890ff', transform: 'rotate(-90deg)' }} />
          <span style={{ fontSize: 12, fontWeight: 700, color: '#262626' }}>{t(TITLE_KEY)}</span>
          <MetricsRow m={runner.metrics} t={t} />
        </div>
        <Button size="small" type="primary" loading={runner.submitting}
          disabled={!canRun} onClick={(e) => { e.stopPropagation(); handleRun(); }}
          style={{ borderRadius: 6, fontWeight: 600 }}>{t(RUN_KEY)}</Button>
      </div>
    );
  }

  // ── Expanded state ────────────────────────────────────────────────────
  return (
    <div style={{
      borderTop: '2px solid #e8e8e8', background: '#fafbfc',
      display: 'flex', flexDirection: 'column',
      height: runner.panelHeight, minHeight: 160,
      userSelect: runner.dragging ? 'none' : 'auto',
    }}>
      {/* Resize handle */}
      <div ref={resizeRef} onMouseDown={handleMouseDown} style={{
        height: 5, cursor: 'row-resize', background: runner.dragging ? '#1890ff' : 'transparent', flexShrink: 0,
      }} />

      {/* Tab bar */}
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '4px 14px 0', flexShrink: 0,
        background: 'linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%)',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <PlayCircleOutlined style={{ color: '#1890ff' }} />
          <span style={{ fontSize: 12, fontWeight: 700, color: '#262626' }}>{t(TITLE_KEY)}</span>
          <Tabs size="small" activeKey={runner.activeTab} onChange={runner.setActiveTab}
            tabBarStyle={{ marginBottom: 0, borderBottom: 'none' }}
            items={[
              { key: 'params', label: t(EXECUTION_KEY) },
              { key: 'results', label: t(BACKTEST_TAB_KEY, 'Results') },
              { key: 'tuning', label: t(TUNING_TAB_KEY, 'Tuning') },
              { key: 'gate', label: t(GATE_TAB_KEY, 'Gate') },
              { key: 'trades', label: `${t(BACKTEST_RECORDS_KEY, 'Records')}${runner.metrics?.totalTrades ? ` (${runner.metrics.totalTrades})` : ''}` },
            ]}
          />
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0 }}>
          {onOpenHistory && (
            <Tooltip title={t(HISTORY_KEY)}>
              <Button size="small" type="text" icon={<HistoryOutlined />} onClick={onOpenHistory} style={{ borderRadius: 6 }} />
            </Tooltip>
          )}
          <Dropdown menu={{ items: runner.settingsItems }} trigger={['click']} placement="bottomRight">
            <Button size="small" type="text" icon={<SettingOutlined />} style={{ borderRadius: 6 }} />
          </Dropdown>
          <Tooltip title={!canRun ? t(RUN_DISABLED_HINT_KEY) : undefined}>
            <Button type="primary" size="small" loading={runner.submitting} disabled={!canRun}
              onClick={handleRun} style={{ borderRadius: 6, fontWeight: 600, boxShadow: '0 2px 8px rgba(24,144,255,0.25)' }}>
              {t(RUN_KEY)}
            </Button>
          </Tooltip>
          <span onClick={(e) => { e.stopPropagation(); onToggleCollapsed(); }}
            style={{ fontSize: 12, color: '#8c8c8c', cursor: 'pointer' }}>
            <CaretDownOutlined />
          </span>
        </div>
      </div>

      {/* Tab content */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '8px 14px' }}>
        {/* ── Params Tab ──────────────────────────────────────────────── */}
        {runner.activeTab === 'params' && (
          <div>
            {/* Strategy + Date + Strict + Presets — single row */}
            <div style={{ marginBottom: 8, display: 'flex', alignItems: 'flex-end', gap: 8, flexWrap: 'wrap' }}>
              <div>
                <div style={S.fieldLabel}>{t(STRATEGY_KEY)}</div>
                <Select size="small" style={{ width: 180 }}
                  loading={templates.loading}
                  value={templates.selectedId || '__draft__'}
                  options={[
                    { value: '__draft__', label: t(CURRENT_DRAFT_KEY) },
                    ...(tplList.length > 0 ? [{ value: '__sep__', label: '──────────────', disabled: true }] : []),
                    ...tplList.map((tpl: StrategyTemplate) => ({ value: tpl.id, label: tpl.name })),
                  ]}
                  onChange={(val) => {
                    if (val === '__draft__') templates.onSelect(null);
                    else if (val !== '__sep__') templates.onSelect(val);
                  }}
                />
              </div>
              <div style={{ borderLeft: '1px solid #e8e8e8', paddingLeft: 8 }}>
                <div style={S.fieldLabel}>{t(DATE_RANGE_KEY)}</div>
                <div style={{ display: 'flex', gap: 4 }}>
                  <Segmented size="small" value={runner.datePreset}
                    onChange={(v) => {
                      const p = DATE_PRESETS.find(d => d.key === v);
                      if (p) runner.applyDatePreset(p);
                    }}
                    options={DATE_PRESETS.map(p => ({ value: p.key, label: p.label }))}
                  />
                  <DatePicker size="small" style={{ width: 110 }}
                    value={runner.startDate ? dayjs(runner.startDate) : null}
                    onChange={(d) => d && runner.setStartDate(d.format('YYYY-MM-DD'))}
                    placeholder={t(START_DATE_KEY)} />
                  <DatePicker size="small" style={{ width: 110 }}
                    value={runner.endDate ? dayjs(runner.endDate) : null}
                    onChange={(d) => d && runner.setEndDate(d.format('YYYY-MM-DD'))}
                    placeholder={t(END_DATE_KEY)} />
                </div>
              </div>
              <div style={{ borderLeft: '1px solid #e8e8e8', paddingLeft: 8 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <span style={S.fieldLabel}>{t(STRICT_MODE_KEY)}</span>
                  <Switch size="small" checked={runner.strictMode} onChange={runner.setStrictMode} />
                </div>
              </div>
              <div style={{ borderLeft: '1px solid #e8e8e8', paddingLeft: 8 }}>
                {(Object.keys(PRESETS) as Array<keyof typeof PRESETS>).map((key) => (
                  <Button key={key} size="small" onClick={() => runner.applyPreset(key)}
                    style={{ fontSize: 12, padding: '0 8px', height: 26 }}>
                    {t(`strategy.backtestParams.presets.${key === 'live_aligned' ? 'liveAligned' : 'exploration'}`)}
                  </Button>
                ))}
              </div>
            </div>

            {/* Standard params — compact single row */}
            <div style={S.sectionLabel}>{t(EXECUTION_KEY)}</div>
            <Row gutter={6} style={{ marginBottom: 6 }}>
              <Col span={4}>
                <div style={S.fieldLabel}>{t(CAPITAL_KEY)}</div>
                <InputNumber size="small" style={S.narrow} min={100} step={1000}
                  value={runner.initialCapital} onChange={(v) => runner.setInitialCapital(v ?? 10000)}
                  formatter={v => `$ ${v}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}
                  parser={v => v!.replace(/\$\s?|(,*)/g, '') as unknown as number} />
              </Col>
              <Col span={3}>
                <div style={S.fieldLabel}>{t(LEVERAGE_KEY)}</div>
                <InputNumber size="small" style={S.narrow} min={1} step={1}
                  value={runner.leverage} onChange={(v) => runner.setLeverage(v ?? 1)}
                  formatter={v => `${v}x`} parser={v => v!.replace('x', '') as unknown as number} />
              </Col>
              <Col span={3}>
                <div style={S.fieldLabel}>{t(LOT_SIZE_KEY)}</div>
                <InputNumber size="small" style={S.narrow} min={0.01} max={100} step={0.01}
                  value={runner.lotSize} onChange={(v) => runner.setLotSize(v ?? 0.01)} />
              </Col>
              <Col span={5}>
                <div style={S.fieldLabel}>{t(COMMISSION_KEY)}
                  <span style={{ fontSize: 12, color: '#bfbfbf', fontWeight: 400, marginLeft: 3 }}>%</span>
                </div>
                <InputNumber size="small" style={S.narrow} min={0} max={10} step={0.01} precision={4}
                  value={runner.commission} onChange={(v) => runner.setCommission(v ?? 0.001)}
                  formatter={v => `${v}%`} parser={v => v!.replace('%', '') as unknown as number} />
              </Col>
              <Col span={5}>
                <div style={S.fieldLabel}>{t(SLIPPAGE_KEY)}
                  <span style={{ fontSize: 12, color: '#bfbfbf', fontWeight: 400, marginLeft: 3 }}>{slippagePct}%</span>
                </div>
                <InputNumber size="small" style={S.narrow} min={0} max={10} step={0.0001} precision={4}
                  value={runner.slippage} onChange={(v) => runner.setSlippage(v ?? 0)} />
              </Col>
              <Col span={4}>
                <div style={S.fieldLabel}>{t(DIRECTION_KEY)}</div>
                <Radio.Group value={runner.tradeDirection} onChange={e => runner.setTradeDirection(e.target.value)}
                  size="small" buttonStyle="solid" style={{ display: 'flex' }}>
                  <Radio.Button value="long" style={{ flex: 1, textAlign: 'center', fontSize: 12 }}>{t(LONG_KEY)}</Radio.Button>
                  <Radio.Button value="short" style={{ flex: 1, textAlign: 'center', fontSize: 12 }}>{t(SHORT_KEY)}</Radio.Button>
                  <Radio.Button value="both" style={{ flex: 1, textAlign: 'center', fontSize: 12 }}>{t(BOTH_KEY)}</Radio.Button>
                </Radio.Group>
              </Col>
            </Row>
            {/* Strategy params — title row + tag preview (TradingView Settings style) */}
            <div style={{ marginTop: 10, borderTop: '1px solid #f0f0f0', paddingTop: 8 }}>
              {runner.extractedParams.length > 0 ? (
                <>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                    <span style={S.sectionLabel}>{t(STRATEGY_PARAMS_KEY)} ({runner.extractedParams.length})</span>
                    <Button size="small" onClick={() => runner.setStrategyParamsModalOpen(true)}>
                      {t('common.edit')}
                    </Button>
                  </div>
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: '2px 12px' }}>
                    {runner.extractedParams.slice(0, 8).map((p) => {
                      const v = runner.strategyParamValues[p.name] ?? p.default;
                      const label = paramLabel(p.name, loc, i18nData) || p.label || p.name;
                      return (
                        <span key={p.name} style={{ fontSize: 11, color: '#595959', whiteSpace: 'nowrap' }}>
                          <span style={{ color: '#8c8c8c' }}>{label}</span>
                          <span style={{ fontWeight: 500 }}>={v}</span>
                        </span>
                      );
                    })}
                    {runner.extractedParams.length > 8 && (
                      <span style={{ fontSize: 11, color: '#bfbfbf' }}>
                        +{runner.extractedParams.length - 8} {t(MORE_KEY)}
                      </span>
                    )}
                  </div>
                </>
              ) : (
                <span style={{ fontSize: 11, color: '#bfbfbf' }}>
                  {t(VALIDATE_TO_SEE_PARAMS_KEY)}
                </span>
              )}
            </div>
            <StrategyParamsModal
              open={runner.strategyParamsModalOpen}
              params={runner.extractedParams}
              values={runner.strategyParamValues}
              i18nData={i18nData}
              onClose={() => runner.setStrategyParamsModalOpen(false)}
              onChange={runner.setParam}
            />
          </div>
        )}

        {/* ── Results Tab ─────────────────────────────────────────────── */}
        {runner.activeTab === 'results' && (
          <BacktestResultsTab
            status={runner.status}
            metrics={runner.metrics}
            executionAssumptions={runner.executionAssumptions}
            errorMsg={runner.errorMsg}
            onAIOptimize={onAIOptimize}
          />
        )}

        {/* ── Tuning Tab ──────────────────────────────────────────────── */}
        {runner.activeTab === 'tuning' && runner.tuning && (
          <SmartTuningPanel
            tuneMethod={runner.tuning.method || 'grid'} onTuneMethodChange={runner.tuning.setMethod || (() => {})}
            sweepDimensions={runner.tuning.sweepDimensions || []} onToggleDimension={runner.tuning.toggleDimension || (() => {})}
            enabledSweepDims={runner.tuning.enabledDims || []} cartesianSize={runner.tuning.cartesianSize || 0}
            tuningRunning={runner.tuning.running || false} canRun={Boolean(inputs.strategyCode && inputs.symbol)}
            onRunTuning={() => runner.tuning.run?.({
              code: inputs.strategyCode, symbol: inputs.symbol, timeframe: inputs.timeframe,
              startDate: runner.startDate, endDate: runner.endDate,
              templateId: inputs.templateId,
            })}
            code={code} onApplyToCode={onApplyTunedParams}
          />
        )}

        {/* ── Gate Tab ────────────────────────────────────────────────── */}
        {runner.activeTab === 'gate' && runner.gate && (
          <GatePanel
            loading={runner.gate.loading || false} gates={runner.gate.gates || []} summary={runner.gate.summary || null}
            error={runner.gate.error || ''} status={runner.status} canRun={runner.status === 'completed'}
            onRun={runner.gate.run || (() => {})}
          />
        )}

        {/* ── Trades Tab ──────────────────────────────────────────────── */}
        {runner.activeTab === 'trades' && (
          <BacktestTradesTab trades={runner.chartTrades} panelHeight={runner.panelHeight} />
        )}
      </div>
    </div>
  );
}
