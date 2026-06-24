import { useCallback, useRef, useEffect } from 'react';
import { Button, Tabs, InputNumber, Row, Col, Segmented, DatePicker, Radio, Switch, Tag, Tooltip, Dropdown, Select } from 'antd';
import {
  PlayCircleOutlined, SettingOutlined, CaretUpOutlined, CaretDownOutlined,
  HistoryOutlined, DoubleRightOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';
import {
  BOTH_KEY, CAPITAL_KEY, COMMISSION_KEY, DATE_RANGE_KEY, DIRECTION_KEY,
  END_DATE_KEY, EXECUTION_KEY, HISTORY_KEY, LEVERAGE_KEY, LONG_KEY,
  RUN_KEY, SHORT_KEY, SLIPPAGE_KEY, START_DATE_KEY, STRICT_MODE_KEY,
  STRICT_MODE_OFF_DESC_KEY, STRICT_MODE_OFF_KEY, STRICT_MODE_OFF_TOOLTIP_KEY,
  STRICT_MODE_ON_DESC_KEY, STRICT_MODE_ON_KEY, STRICT_MODE_ON_TOOLTIP_KEY,
  TITLE_KEY, TRADE_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import { RUN_DISABLED_HINT_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { BACKTEST_RESULTS_LABEL_KEY, COMPLETED_STATUS_KEY, RUNNING_STATUS_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import type { useBacktestRunner, BacktestRunnerInputs } from './useBacktestRunner';
import { DATE_PRESETS, PRESETS } from '@/pages/strategy/hooks/backtestParamHelpers';
import type { StrategyTemplate } from '@/client/strategy';
import type { BacktestMetrics } from './useBacktestRunner';

const S = {
  sectionLabel: { fontSize: 10, fontWeight: 700, color: '#8c8c8c', textTransform: 'uppercase' as const, marginBottom: 6 },
  fieldLabel: { fontSize: 9, fontWeight: 600, color: '#8c8c8c', textTransform: 'uppercase' as const, marginBottom: 2 },
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
  accountId: string; onAccountChange: (v: string) => void;
  accounts: Array<{ id: string; name: string; login?: string }>;
  symbol: string; onSymbolChange: (v: string) => void;
  timeframe: string; onTimeframeChange: (v: string) => void;
  collapsed: boolean; onToggleCollapsed: () => void;
  onOpenHistory?: () => void;
}

function MetricsRow({ m }: { m: BacktestMetrics | null }) {
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
    runner, inputs, templates,
    accountId, onAccountChange, accounts, symbol, onSymbolChange, timeframe, onTimeframeChange,
    collapsed, onToggleCollapsed, onOpenHistory,
  } = props;
  const { t } = useTranslation();

  const strategyOptions = [
    { value: '__draft__', label: 'Current Draft' },
    ...(templates.list.length > 0 ? [{ value: '__sep__', label: '──────────────', disabled: true }] : []),
    ...templates.list.map((tpl: StrategyTemplate) => ({ value: tpl.id, label: tpl.name })),
  ];

  const accountsForSelect = accounts.map(a => ({
    value: a.id, label: a.login || a.name || a.id.slice(0, 8),
  }));

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
        height: 5, cursor: 'row-resize', background: runner.dragging ? '#1890ff' : 'transparent',
        flexShrink: 0,
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
              { key: 'params', label: 'Params' },
              { key: 'results', label: 'Results' },
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
            {/* Selectors */}
            <Row gutter={8} style={{ marginBottom: 10 }}>
              <Col span={6}>
                <div style={S.fieldLabel}>Strategy</div>
                <Select size="small" style={{ width: '100%' }}
                  loading={templates.loading}
                  value={templates.selectedId || '__draft__'}
                  options={strategyOptions}
                  onChange={(val) => {
                    if (val === '__draft__') templates.onSelect(null);
                    else if (val !== '__sep__') templates.onSelect(val);
                  }}
                />
              </Col>
              <Col span={6}>
                <div style={S.fieldLabel}>Account</div>
                <Select size="small" style={{ width: '100%' }}
                  value={accountId} options={accountsForSelect}
                  onChange={onAccountChange}
                />
              </Col>
              <Col span={6}>
                <div style={S.fieldLabel}>Symbol</div>
                <Select size="small" style={{ width: '100%' }}
                  value={symbol} onChange={onSymbolChange}
                  mode="tags" maxCount={1} options={[]}
                  tokenSeparators={[',', ' ']}
                />
              </Col>
              <Col span={6}>
                <div style={S.fieldLabel}>Timeframe</div>
                <Segmented size="small" value={timeframe}
                  onChange={(v) => onTimeframeChange(v as string)}
                  options={['M1', 'M5', 'M15', 'M30', 'H1', 'H4', 'D1', 'W1']}
                  style={{ width: '100%' }}
                />
              </Col>
            </Row>

            {/* Date range */}
            <div style={{ marginBottom: 10 }}>
              <div style={S.fieldLabel}>{t(DATE_RANGE_KEY)}</div>
              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                <Segmented size="small" value={runner.datePreset}
                  onChange={(v) => {
                    const p = DATE_PRESETS.find(d => d.key === v);
                    if (p) runner.applyDatePreset(p);
                  }}
                  options={DATE_PRESETS.map(p => ({ value: p.key, label: p.label }))}
                />
                <DatePicker size="small" style={{ width: 130 }}
                  value={runner.startDate ? dayjs(runner.startDate) : null}
                  onChange={(d) => d && runner.setStartDate(d.format('YYYY-MM-DD'))}
                  placeholder={t(START_DATE_KEY)} />
                <DatePicker size="small" style={{ width: 130 }}
                  value={runner.endDate ? dayjs(runner.endDate) : null}
                  onChange={(d) => d && runner.setEndDate(d.format('YYYY-MM-DD'))}
                  placeholder={t(END_DATE_KEY)} />
              </div>
            </div>

            {/* Standard params */}
            <div style={S.sectionLabel}>{t(EXECUTION_KEY)}</div>
            <Row gutter={8} style={{ marginBottom: 8 }}>
              <Col span={8}>
                <div style={S.fieldLabel}>{t(CAPITAL_KEY)}</div>
                <InputNumber size="small" style={S.narrow} min={100} step={1000}
                  value={runner.initialCapital} onChange={(v) => runner.setInitialCapital(v ?? FACTORY_DEFAULTS.initialCapital)}
                  formatter={v => `$ ${v}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}
                  parser={v => v!.replace(/\$\s?|(,*)/g, '') as unknown as number} />
              </Col>
              <Col span={8}>
                <div style={S.fieldLabel}>{t(LEVERAGE_KEY)}</div>
                <InputNumber size="small" style={S.narrow} min={1} step={1}
                  value={runner.leverage} onChange={(v) => runner.setLeverage(v ?? FACTORY_DEFAULTS.leverage)}
                  formatter={v => `${v}x`} parser={v => v!.replace('x', '') as unknown as number} />
              </Col>
              <Col span={8}>
                <div style={S.fieldLabel}>Lot Size</div>
                <InputNumber size="small" style={S.narrow} min={0.01} max={100} step={0.01}
                  value={runner.lotSize} onChange={(v) => runner.setLotSize(v ?? FACTORY_DEFAULTS.lotSize)} />
              </Col>
            </Row>
            <Row gutter={8} style={{ marginBottom: 6 }}>
              <Col span={8}>
                <div style={S.fieldLabel}>{t(COMMISSION_KEY)}
                  <span style={{ fontSize: 8, color: '#bfbfbf', fontWeight: 400, marginLeft: 3 }}>%</span>
                </div>
                <InputNumber size="small" style={S.narrow} min={0} max={10} step={0.01} precision={4}
                  value={runner.commission} onChange={(v) => runner.setCommission(v ?? FACTORY_DEFAULTS.commission)}
                  formatter={v => `${v}%`} parser={v => v!.replace('%', '') as unknown as number} />
              </Col>
              <Col span={8}>
                <div style={S.fieldLabel}>{t(SLIPPAGE_KEY)}
                  <span style={{ fontSize: 8, color: '#bfbfbf', fontWeight: 400, marginLeft: 3 }}>{slippagePct}%</span>
                </div>
                <InputNumber size="small" style={S.narrow} min={0} max={10} step={0.0001} precision={4}
                  value={runner.slippage} onChange={(v) => runner.setSlippage(v ?? FACTORY_DEFAULTS.slippage)} />
              </Col>
              <Col span={8}>
                <div style={S.fieldLabel}>{t(DIRECTION_KEY)}</div>
                <Radio.Group value={runner.tradeDirection} onChange={e => runner.setTradeDirection(e.target.value)}
                  size="small" buttonStyle="solid" style={{ display: 'flex' }}>
                  <Radio.Button value="long" style={{ flex: 1, textAlign: 'center', fontSize: 10 }}>{t(LONG_KEY)}</Radio.Button>
                  <Radio.Button value="short" style={{ flex: 1, textAlign: 'center', fontSize: 10 }}>{t(SHORT_KEY)}</Radio.Button>
                  <Radio.Button value="both" style={{ flex: 1, textAlign: 'center', fontSize: 10 }}>{t(BOTH_KEY)}</Radio.Button>
                </Radio.Group>
              </Col>
            </Row>
            <Row gutter={8}>
              <Col span={12}>
                <div style={S.fieldLabel}>{t(STRICT_MODE_KEY)}</div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <Switch size="small" checked={runner.strictMode} onChange={runner.setStrictMode} />
                  <Tooltip title={runner.strictMode ? t(STRICT_MODE_ON_TOOLTIP_KEY) : t(STRICT_MODE_OFF_TOOLTIP_KEY)}>
                    <Tag color={runner.strictMode ? 'blue' : 'orange'}
                      style={{ fontSize: 8, margin: 0, lineHeight: '14px', cursor: 'help' }}>
                      {runner.strictMode ? t(STRICT_MODE_ON_KEY) : t(STRICT_MODE_OFF_KEY)}
                    </Tag>
                  </Tooltip>
                </div>
                <div style={{ fontSize: 8, color: '#8c8c8c', marginTop: 2 }}>
                  {runner.strictMode ? t(STRICT_MODE_ON_DESC_KEY) : t(STRICT_MODE_OFF_DESC_KEY)}
                </div>
              </Col>
            </Row>
          </div>
        )}

        {/* ── Results Tab ─────────────────────────────────────────────── */}
        {runner.activeTab === 'results' && (
          <div>
            {runner.status === 'idle' && (
              <div style={{ textAlign: 'center', padding: 40, color: '#8c8c8c' }}>
                Press "{t(RUN_KEY)}" to start backtest
              </div>
            )}
            {runner.status === 'running' && (
              <div style={{ textAlign: 'center', padding: 40, color: '#1890ff' }}>
                Running backtest...
              </div>
            )}
            {runner.status === 'error' && (
              <div style={{ textAlign: 'center', padding: 40, color: '#e57373' }}>
                Backtest failed: {runner.errorMsg}
              </div>
            )}
            {runner.status === 'completed' && runner.metrics && (
              <div>
                <MetricsRow m={runner.metrics} />
                {runner.executionAssumptions && (
                  <div style={{ marginTop: 12 }}>
                    <div style={S.fieldLabel}>Execution Assumptions</div>
                    <div style={{ fontSize: 10, color: '#8c8c8c', maxHeight: 120, overflowY: 'auto' }}>
                      <pre style={{ margin: 0, fontSize: 10 }}>
                        {JSON.stringify(runner.executionAssumptions, null, 2)}
                      </pre>
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {/* ── Trades Tab ──────────────────────────────────────────────── */}
        {runner.activeTab === 'trades' && (
          <div>
            {runner.chartTrades.length === 0 ? (
              <div style={{ textAlign: 'center', padding: 40, color: '#8c8c8c' }}>
                Run a backtest to see trades
              </div>
            ) : (
              <div style={{ maxHeight: runner.panelHeight - 100, overflowY: 'auto' }}>
                <table style={{ width: '100%', fontSize: 10, borderCollapse: 'collapse' }}>
                  <thead>
                    <tr style={{ borderBottom: '1px solid #e8e8e8', textAlign: 'left' }}>
                      <th style={{ padding: '4px 8px' }}>#</th>
                      <th style={{ padding: '4px 8px' }}>Side</th>
                      <th style={{ padding: '4px 8px', textAlign: 'right' }}>Open</th>
                      <th style={{ padding: '4px 8px', textAlign: 'right' }}>Close</th>
                      <th style={{ padding: '4px 8px', textAlign: 'right' }}>PnL</th>
                    </tr>
                  </thead>
                  <tbody>
                    {runner.chartTrades.map((t, i) => (
                      <tr key={i} style={{ borderBottom: '1px solid #f0f0f0' }}>
                        <td style={{ padding: '3px 8px' }}>{i + 1}</td>
                        <td style={{ padding: '3px 8px', color: t.side === 'buy' ? '#26a69a' : '#e57373' }}>
                          {t.side?.toUpperCase()}
                        </td>
                        <td style={{ padding: '3px 8px', textAlign: 'right' }}>
                          {t.openPrice?.toFixed(2)}
                        </td>
                        <td style={{ padding: '3px 8px', textAlign: 'right' }}>
                          {t.closePrice?.toFixed(2) ?? '—'}
                        </td>
                        <td style={{
                          padding: '3px 8px', textAlign: 'right',
                          color: (t.pnl ?? 0) >= 0 ? '#26a69a' : '#e57373',
                        }}>
                          {t.pnl?.toFixed(2) ?? '—'}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

const FACTORY_DEFAULTS = { initialCapital: 10000, leverage: 1, lotSize: 0.01, commission: 0.001, slippage: 0.0 };
