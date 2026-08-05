import { useCallback, useEffect, useRef, useState } from 'react';
import { Tabs, Radio, Badge } from 'antd';
import {
  PlayCircleOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import {
  TITLE_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import {
  BACKTEST_TAB_KEY, GATE_TAB_KEY, TUNING_TAB_KEY,
  TUNING_INTERACTIVE_KEY, TUNING_BATCH_KEY,
} from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import SmartTuningPanel from '@/pages/strategy/components/workspace/SmartTuningPanel';
import BatchTuningPanel from '@/pages/strategy/components/workspace/BatchTuningPanel';
import GatePanel from '@/pages/strategy/components/workspace/GatePanel';
import BacktestResultsTab from './BacktestResultsTab';
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
  onOpenHistory?: (templateId?: string) => void;
  onAIOptimize?: () => void;
  code?: string;
  onApplyTunedParams?: (code: string) => void;
  onRunBacktest?: () => void;
  onSaveAs?: () => void;
  hasUnsavedDraft?: boolean;
  draftName?: string;
}

export default function BacktestPanel(props: Props) {
  const {
    runner, inputs, templates,
    onOpenHistory, onAIOptimize, code, onApplyTunedParams,
    onRunBacktest, onSaveAs, hasUnsavedDraft, draftName,
  } = props;
  const { t } = useTranslation();
  const [tuningMode, setTuningMode] = useState<'interactive' | 'batch'>('interactive');

  const _handleRun = () => runner.run(inputs);
  const _canRun = Boolean(inputs.strategyCode && inputs.symbol) && !runner.submitting;

  // ── Measure actual content height for table scroll sizing ─────────────
  const contentRef = useRef<HTMLDivElement>(null);
  const [contentHeight, setContentHeight] = useState(300);
  useEffect(() => {
    if (!contentRef.current) return;
    const ro = new ResizeObserver(([entry]) => {
      setContentHeight(entry.contentRect.height);
    });
    ro.observe(contentRef.current);
    return () => ro.disconnect();
  }, []);

  // ── Resize handle ─────────────────────────────────────────────────────
  const resizeRef = useRef<HTMLDivElement>(null);
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    runner.setDragging(true);
    runner.setUserResized(true);
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

  // ── Expanded state ────────────────────────────────────────────────────
  return (
    <div style={{
      borderTop: '2px solid #e8e8e8', background: '#fafbfc',
      display: 'flex', flexDirection: 'column',
      ...(runner.userResized ? { height: runner.panelHeight } : { flex: 1 }),
      minHeight: 160,
      userSelect: runner.dragging ? 'none' : 'auto',
    }}>
      {/* Resize handle */}
      <div ref={resizeRef} onMouseDown={handleMouseDown} style={{
        height: 5, cursor: 'row-resize', background: runner.dragging ? '#1890ff' : 'transparent', flexShrink: 0,
      }} />

      {/* Tab bar */}
      <div style={{
        display: 'flex', alignItems: 'center',
        padding: '4px 14px 0', flexShrink: 0,
        background: 'linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%)',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <PlayCircleOutlined style={{ color: '#1890ff' }} />
          <span style={{ fontSize: 12, fontWeight: 700, color: '#262626' }}>{t(TITLE_KEY)}</span>
          <Tabs size="small" activeKey={runner.activeTab} onChange={runner.setActiveTab}
            tabBarStyle={{ marginBottom: 0, borderBottom: 'none' }}
            items={[
              { key: 'results', label: t(BACKTEST_TAB_KEY, 'Results') },
              { key: 'tuning', label: t(TUNING_TAB_KEY, 'Tuning') },
              { key: 'gate', label: t(GATE_TAB_KEY, 'Gate') },
            ].map(item => ({
              ...item,
              label: item.key === 'results' && runner.status === 'running'
                ? <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                    {item.label}
                    <Badge status="processing" />
                  </span>
                : item.key === 'results' && runner.status === 'completed'
                  ? <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                      {item.label}
                      <Badge status="success" />
                    </span>
                : item.key === 'results' && runner.status === 'error'
                  ? <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                      {item.label}
                      <Badge status="error" />
                    </span>
                : item.label,
            }))}
          />
        </div>
      </div>

      {/* Tab content */}
      <div ref={contentRef} style={{ flex: 1, overflowY: 'auto', padding: '8px 14px' }}>
        {/* ── Results Tab ─────────────────────────────────────────────── */}
        {runner.activeTab === 'results' && (
          <BacktestResultsTab
            status={runner.status}
            metrics={runner.metrics}
            executionAssumptions={runner.executionAssumptions}
            errorMsg={runner.errorMsg}
            onAIOptimize={onAIOptimize}
            onOpenHistory={() => onOpenHistory?.()}
            trades={runner.chartTrades}
            panelHeight={contentHeight}
            onCancel={runner.cancelRun}
            gateUpdate={runner.gateUpdate}
            gateResults={runner.gateResults}
            qualityPreview={runner.qualityPreview}
          />
        )}

        {/* ── Tuning Tab ──────────────────────────────────────────────── */}
        {runner.activeTab === 'tuning' && runner.tuning && (
          <>
            <Radio.Group value={tuningMode} onChange={e => setTuningMode(e.target.value)} size="small"
              buttonStyle="solid" style={{ marginBottom: 8 }}>
              <Radio.Button value="interactive">{t(TUNING_INTERACTIVE_KEY)}</Radio.Button>
              <Radio.Button value="batch">{t(TUNING_BATCH_KEY)}</Radio.Button>
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
            fixDepth={runner.fixDepth || 0}
          />
        )}

        {/* ── Trades Tab ──────────────────────────────────────────────── */}
      </div>
    </div>
  );
}
