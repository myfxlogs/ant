import { useCallback, useRef, useState } from 'react';
import { Button, Tabs, Tooltip, Dropdown, Radio } from 'antd';
import {
  PlayCircleOutlined, SettingOutlined, CaretDownOutlined,
  HistoryOutlined, DoubleRightOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import {
  RUN_KEY, EXECUTION_KEY, HISTORY_KEY, TITLE_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import { RUN_DISABLED_HINT_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import {
  BACKTEST_TAB_KEY, GATE_TAB_KEY, TUNING_TAB_KEY,
} from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import {
  BACKTEST_RECORDS_KEY, MAX_DRAWDOWN_KEY, PROFIT_FACTOR_KEY, SHARPE_KEY,
  TOTAL_RETURN_KEY, TOTAL_TRADES_KEY, WIN_RATE_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_keys';
import SmartTuningPanel from '@/pages/strategy/components/workspace/SmartTuningPanel';
import BatchTuningPanel from '@/pages/strategy/components/workspace/BatchTuningPanel';
import GatePanel from '@/pages/strategy/components/workspace/GatePanel';
import BacktestResultsTab from './BacktestResultsTab';
import BacktestTradesTab from './BacktestTradesTab';
import BacktestParamsTab from './BacktestParamsTab';
import type { useBacktestRunner, BacktestRunnerInputs } from './useBacktestRunner';
import type { StrategyTemplate } from '@/client/strategy';

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
  const { t } = useTranslation();
  const [tuningMode, setTuningMode] = useState<'interactive' | 'batch'>('interactive');

  const handleRun = () => runner.run(inputs);
  const canRun = Boolean(inputs.strategyCode && inputs.symbol) && !runner.submitting;

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
          <BacktestParamsTab runner={runner} inputs={inputs} templates={templates} />
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
          <>
            <Radio.Group value={tuningMode} onChange={e => setTuningMode(e.target.value)} size="small"
              buttonStyle="solid" style={{ marginBottom: 8 }}>
              <Radio.Button value="interactive">{t('strategy.workspace.tuningInteractive')}</Radio.Button>
              <Radio.Button value="batch">{t('strategy.workspace.tuningBatch')}</Radio.Button>
            </Radio.Group>
            {tuningMode === 'interactive' ? (
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
            ) : (
              <BatchTuningPanel />
            )}
          </>
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
