import { useCallback, useRef } from 'react';
import { Button, Tabs, InputNumber, Row, Col, Segmented, DatePicker, Radio, Switch, Tag, Tooltip, Dropdown, Select, Card, Statistic, Table, Empty, Spin } from 'antd';
import {
  PlayCircleOutlined, SettingOutlined, CaretDownOutlined,
  HistoryOutlined, DoubleRightOutlined, RiseOutlined, FallOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';
import { LineChart, Line, XAxis, YAxis, Tooltip as RechartsTooltip, ResponsiveContainer } from 'recharts';
import {
  BOTH_KEY, CAPITAL_KEY, COMMISSION_KEY, DATE_RANGE_KEY, DIRECTION_KEY,
  END_DATE_KEY, EXECUTION_KEY, HISTORY_KEY, LEVERAGE_KEY, LONG_KEY,
  RUN_KEY, SHORT_KEY, SLIPPAGE_KEY, START_DATE_KEY, STRICT_MODE_KEY,
  STRICT_MODE_OFF_DESC_KEY, STRICT_MODE_OFF_KEY, STRICT_MODE_OFF_TOOLTIP_KEY,
  STRICT_MODE_ON_DESC_KEY, STRICT_MODE_ON_KEY, STRICT_MODE_ON_TOOLTIP_KEY,
  TITLE_KEY, TRADE_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import { RUN_DISABLED_HINT_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import {
  BACKTEST_COMPLETED_KEY, BACKTEST_EMPTY_KEY, BACKTEST_ERROR_KEY, BACKTEST_RUNNING_KEY,
  BACKTEST_TAB_KEY, EXEC_ASSUMPTIONS_KEY, EXEC_ASSUMPTIONS_FIELDS_COMMISSION_KEY,
  EXEC_ASSUMPTIONS_FIELDS_DIRECTION_KEY, EXEC_ASSUMPTIONS_FIELDS_FILL_RULE_KEY,
  EXEC_ASSUMPTIONS_FIELDS_LEVERAGE_KEY, EXEC_ASSUMPTIONS_FIELDS_MODE_KEY,
  EXEC_ASSUMPTIONS_FIELDS_MTF_FALLBACK_KEY, EXEC_ASSUMPTIONS_FIELDS_SLIPPAGE_KEY,
  EXEC_ASSUMPTIONS_FIELDS_TIMING_KEY, GATE_TAB_KEY, TUNING_TAB_KEY,
} from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import {
  ANNUAL_RETURN_KEY, EQUITY_CURVE_KEY, MAX_DRAWDOWN_KEY, SHARPE_KEY,
  TOTAL_RETURN_KEY, TOTAL_TRADES_KEY, TRADE_LOG_KEY, TRADE_PRICE_KEY,
  TRADE_SIDE_KEY, TRADE_TIME_KEY, TRADE_VOLUME_KEY, WIN_RATE_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_keys';
import SmartTuningPanel from '@/pages/strategy/components/workspace/SmartTuningPanel';
import GatePanel from '@/pages/strategy/components/workspace/GatePanel';
import type { useBacktestRunner, BacktestRunnerInputs } from './useBacktestRunner';
import { DATE_PRESETS, PRESETS } from '@/pages/strategy/hooks/backtestParamHelpers';
import type { StrategyTemplate } from '@/client/strategy';

const S = {
  sectionLabel: { fontSize: 10, fontWeight: 700, color: '#8c8c8c', textTransform: 'uppercase' as const, marginBottom: 6 },
  fieldLabel: { fontSize: 9, fontWeight: 600, color: '#8c8c8c', textTransform: 'uppercase' as const, marginBottom: 2 },
  narrow: { width: '100%' },
  metricStyle: { fontSize: 16, fontFamily: 'monospace' as const },
};

function pct(v: number | undefined): string { if (v == null) return '-'; return (v * 100).toFixed(2) + '%'; }
function num(v: number | undefined, d = 2): string { if (v == null) return '-'; return v.toFixed(d); }

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

function MetricsRow({ m }: { m: any }) {
  if (!m) return null;
  return (
    <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', fontSize: 11, color: '#595959' }}>
      <span>Return: <b style={{ color: (m.totalReturn ?? 0) >= 0 ? '#26a69a' : '#e57373' }}>{(m.totalReturn ?? 0).toFixed(2)}%</b></span>
      <span>Trades: <b>{m.totalTrades ?? 0}</b></span>
      <span>Win: <b>{((m.winRate ?? 0) * 100).toFixed(1)}%</b></span>
      <span>PF: <b>{m.profitFactor?.toFixed(2) ?? '—'}</b></span>
      <span>Sharpe: <b>{m.sharpeRatio?.toFixed(2) ?? '—'}</b></span>
      <span>MaxDD: <b style={{ color: '#e57373' }}>{(m.maxDrawdown ?? 0).toFixed(2)}%</b></span>
    </div>
  );
}

export default function BacktestPanel(props: Props) {
  const {
    runner, inputs, templates, collapsed, onToggleCollapsed,
    onOpenHistory, onAIOptimize, code, onApplyTunedParams,
  } = props;
  const { t } = useTranslation();

  const tplList = templates?.list || [];
  const strategyOptions = [
    { value: '__draft__', label: 'Current Draft' },
    ...(tplList.length > 0 ? [{ value: '__sep__', label: '──────────────', disabled: true }] : []),
    ...tplList.map((tpl: StrategyTemplate) => ({ value: tpl.id, label: tpl.name })),
  ];

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
          <MetricsRow m={runner.metrics} />
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
              { key: 'params', label: t('strategy.backtestParams.title', 'Params') },
              { key: 'results', label: t(BACKTEST_TAB_KEY, 'Results') },
              { key: 'tuning', label: t(TUNING_TAB_KEY, 'Tuning') },
              { key: 'gate', label: t(GATE_TAB_KEY, 'Gate') },
              { key: 'trades', label: `Trades${runner.metrics?.totalTrades ? ` (${runner.metrics.totalTrades})` : ''}` },
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
            style={{ fontSize: 9, color: '#8c8c8c', cursor: 'pointer' }}>
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
                <div style={S.fieldLabel}>Strategy</div>
                <Select size="small" style={{ width: 180 }}
                  loading={templates.loading}
                  value={templates.selectedId || '__draft__'}
                  options={strategyOptions}
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
                  <Button key={key} size="small" type="link" onClick={() => runner.applyPreset(key)}
                    style={{ fontSize: 10, padding: '0 4px', height: 22 }}>
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
                <div style={S.fieldLabel}>Lot Size</div>
                <InputNumber size="small" style={S.narrow} min={0.01} max={100} step={0.01}
                  value={runner.lotSize} onChange={(v) => runner.setLotSize(v ?? 0.01)} />
              </Col>
              <Col span={5}>
                <div style={S.fieldLabel}>{t(COMMISSION_KEY)}
                  <span style={{ fontSize: 8, color: '#bfbfbf', fontWeight: 400, marginLeft: 3 }}>%</span>
                </div>
                <InputNumber size="small" style={S.narrow} min={0} max={10} step={0.01} precision={4}
                  value={runner.commission} onChange={(v) => runner.setCommission(v ?? 0.001)}
                  formatter={v => `${v}%`} parser={v => v!.replace('%', '') as unknown as number} />
              </Col>
              <Col span={5}>
                <div style={S.fieldLabel}>{t(SLIPPAGE_KEY)}
                  <span style={{ fontSize: 8, color: '#bfbfbf', fontWeight: 400, marginLeft: 3 }}>{slippagePct}%</span>
                </div>
                <InputNumber size="small" style={S.narrow} min={0} max={10} step={0.0001} precision={4}
                  value={runner.slippage} onChange={(v) => runner.setSlippage(v ?? 0)} />
              </Col>
              <Col span={4}>
                <div style={S.fieldLabel}>{t(DIRECTION_KEY)}</div>
                <Radio.Group value={runner.tradeDirection} onChange={e => runner.setTradeDirection(e.target.value)}
                  size="small" buttonStyle="solid" style={{ display: 'flex' }}>
                  <Radio.Button value="long" style={{ flex: 1, textAlign: 'center', fontSize: 9 }}>{t(LONG_KEY)}</Radio.Button>
                  <Radio.Button value="short" style={{ flex: 1, textAlign: 'center', fontSize: 9 }}>{t(SHORT_KEY)}</Radio.Button>
                  <Radio.Button value="both" style={{ flex: 1, textAlign: 'center', fontSize: 9 }}>{t(BOTH_KEY)}</Radio.Button>
                </Radio.Group>
              </Col>
            </Row>
            {/* Strategy-specific params */}
            {runner.extractedParams.length > 0 && (
              <div style={{ marginTop: 10, borderTop: '1px solid #f0f0f0', paddingTop: 8 }}>
                <div style={S.sectionLabel}>Strategy Parameters ({runner.extractedParams.length})</div>
                <Row gutter={8}>
                  {runner.extractedParams.map((p) => {
                    const value = runner.strategyParamValues[p.name] ?? p.default;
                    if (p.type === 'bool') {
                      return (
                        <Col span={8} key={p.name} style={{ marginBottom: 6 }}>
                          <div style={S.fieldLabel}>{p.label || p.name}</div>
                          <Switch size="small" checked={value === 'True' || value === 'true'}
                            onChange={(v) => runner.setParam(p.name, v ? 'True' : 'False')} />
                        </Col>
                      );
                    }
                    const step = p.type === 'float' ? 0.01 : 1;
                    return (
                      <Col span={8} key={p.name} style={{ marginBottom: 6 }}>
                        <div style={S.fieldLabel}>{p.label || p.name}</div>
                        <InputNumber size="small" style={S.narrow} step={step}
                          value={Number(value)} onChange={(v) => runner.setParam(p.name, String(v ?? p.default))} />
                      </Col>
                    );
                  })}
                </Row>
              </div>
            )}
          </div>
        )}

        {/* ── Results Tab ─────────────────────────────────────────────── */}
        {runner.activeTab === 'results' && (
          <div>
            {runner.status === 'idle' && (
              <Empty description={t(BACKTEST_EMPTY_KEY, 'Run a backtest to see results')} style={{ padding: 24 }} />
            )}

            <div style={{ marginBottom: 8 }}>
              {runner.status === 'running' && (
                <Tag color="processing" icon={<Spin size="small" />}>{t(BACKTEST_RUNNING_KEY)}</Tag>
              )}
              {runner.status === 'completed' && (
                <Tag color="success">{t(BACKTEST_COMPLETED_KEY)}</Tag>
              )}
              {runner.status === 'completed' && onAIOptimize && runner.metrics && (
                <Button size="small" type="dashed" onClick={onAIOptimize} style={{ marginLeft: 8, fontSize: 11 }}>
                  🤖 AI Optimize
                </Button>
              )}
              {runner.status === 'error' && (
                <Tag color="error">{runner.errorMsg || t(BACKTEST_ERROR_KEY, 'Backtest failed')}</Tag>
              )}
            </div>

            {/* Execution Assumptions */}
            {runner.executionAssumptions && runner.status === 'completed' && (
              <div style={{
                marginBottom: 12, padding: '8px 12px', border: '1px solid #e6f4ff', borderRadius: 8,
                background: 'linear-gradient(180deg, #f8fbff 0%, #f4f9ff 100%)',
              }}>
                <div style={{ fontSize: 10, fontWeight: 600, color: '#1677ff', marginBottom: 6 }}>{t(EXEC_ASSUMPTIONS_KEY)}</div>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: '4px 12px', fontSize: 11 }}>
                  <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_MODE_KEY)}:</span> <strong>{runner.executionAssumptions.simulationMode || '-'}</strong></div>
                  <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_TIMING_KEY)}:</span> <strong>{runner.executionAssumptions.signalTiming || '-'}</strong></div>
                  <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_FILL_RULE_KEY)}:</span> <strong>{runner.executionAssumptions.fillRule || '-'}</strong></div>
                  <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_DIRECTION_KEY)}:</span> <strong>{runner.executionAssumptions.tradeDirection || '-'}</strong></div>
                  <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_COMMISSION_KEY)}:</span> <strong>{runner.executionAssumptions.actualCommission != null ? (runner.executionAssumptions.actualCommission * 100).toFixed(4) + '%' : '-'}</strong></div>
                  <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_SLIPPAGE_KEY)}:</span> <strong>{runner.executionAssumptions.actualSlippage != null ? (runner.executionAssumptions.actualSlippage * 100).toFixed(4) + '%' : '-'}</strong></div>
                  <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_LEVERAGE_KEY)}:</span> <strong>{runner.executionAssumptions.actualLeverage || '-'}x</strong></div>
                  {runner.executionAssumptions.mtfFallbackReason && (
                    <div style={{ gridColumn: '1 / -1' }}><span style={{ color: '#fa8c16' }}>{t(EXEC_ASSUMPTIONS_FIELDS_MTF_FALLBACK_KEY)}:</span> <strong>{runner.executionAssumptions.mtfFallbackReason}</strong></div>
                  )}
                </div>
              </div>
            )}

            {runner.metrics && (
              <>
                <Row gutter={[12, 12]}>
                  <Col span={8}>
                    <Card size="small">
                      <Statistic title={t(TOTAL_RETURN_KEY, 'Total Return')} value={pct(runner.metrics.totalReturn)}
                        prefix={runner.metrics.totalReturn != null && runner.metrics.totalReturn >= 0
                          ? <RiseOutlined style={{ color: '#26a69a' }} /> : <FallOutlined style={{ color: '#ef5350' }} />}
                        valueStyle={S.metricStyle} />
                    </Card>
                  </Col>
                  <Col span={8}>
                    <Card size="small">
                      <Statistic title={t(ANNUAL_RETURN_KEY, 'Annual Return')} value={pct(runner.metrics.annualReturn)} valueStyle={S.metricStyle} />
                    </Card>
                  </Col>
                  <Col span={8}>
                    <Card size="small">
                      <Statistic title={t(MAX_DRAWDOWN_KEY, 'Max Drawdown')} value={pct(runner.metrics.maxDrawdown)} valueStyle={{ ...S.metricStyle, color: '#ef5350' }} />
                    </Card>
                  </Col>
                  <Col span={8}>
                    <Card size="small">
                      <Statistic title={t(SHARPE_KEY, 'Sharpe')} value={num(runner.metrics.sharpeRatio)} valueStyle={S.metricStyle} />
                    </Card>
                  </Col>
                  <Col span={8}>
                    <Card size="small">
                      <Statistic title={t(WIN_RATE_KEY, 'Win Rate')} value={pct(runner.metrics.winRate)} valueStyle={S.metricStyle} />
                    </Card>
                  </Col>
                  <Col span={8}>
                    <Card size="small">
                      <Statistic title={t(TOTAL_TRADES_KEY, 'Total Trades')} value={runner.metrics.totalTrades ?? '-'} valueStyle={S.metricStyle} />
                    </Card>
                  </Col>
                </Row>

                {runner.metrics.equityCurve && runner.metrics.equityCurve.length > 0 && (
                  <Card size="small" title={t(EQUITY_CURVE_KEY, 'Equity Curve')} style={{ marginTop: 12 }}>
                    <ResponsiveContainer width="100%" height={150}>
                      <LineChart data={runner.metrics.equityCurve}>
                        <XAxis dataKey="time" hide />
                        <YAxis width={60} tick={{ fontSize: 11 }} />
                        <RechartsTooltip />
                        <Line type="monotone" dataKey="equity" stroke="#1890ff" dot={false} strokeWidth={1.5} />
                      </LineChart>
                    </ResponsiveContainer>
                  </Card>
                )}
              </>
            )}
          </div>
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
          <div>
            {runner.chartTrades.length === 0 ? (
              <Empty description={t(BACKTEST_EMPTY_KEY, 'Run a backtest to see trades')} style={{ padding: 24 }} />
            ) : (
              <Table dataSource={runner.chartTrades.map((t, i) => ({ ...t, key: i }))} size="small"
                pagination={{ pageSize: 30, size: 'small' }} scroll={{ y: runner.panelHeight - 140 }}
                columns={[
                  { title: '#', dataIndex: 'key', width: 40 },
                  { title: t(TRADE_SIDE_KEY, 'Side'), dataIndex: 'side', width: 60,
                    render: (v: string) => <span style={{ color: v === 'buy' ? '#26a69a' : '#e57373' }}>{v?.toUpperCase()}</span> },
                  { title: t(TRADE_VOLUME_KEY, 'Volume'), dataIndex: 'volume', width: 70,
                    render: (v: number) => v?.toFixed(2) },
                  { title: t(TRADE_PRICE_KEY, 'Price'), dataIndex: 'openPrice', width: 80,
                    render: (v: number) => v?.toFixed(2) },
                  { title: 'Close', dataIndex: 'closePrice', width: 80,
                    render: (v: number) => v?.toFixed(2) ?? '—' },
                  { title: 'PnL', dataIndex: 'pnl', width: 80,
                    render: (v: number) => v != null ? (
                      <span style={{ color: v >= 0 ? '#26a69a' : '#ef5350' }}>{v >= 0 ? '+' : ''}{v.toFixed(2)}</span>
                    ) : '-' },
                ]} />
            )}
          </div>
        )}
      </div>
    </div>
  );
}
