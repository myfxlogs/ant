import { Button, Tooltip } from 'antd';
import { PlayCircleOutlined, SaveOutlined, CopyOutlined, QuestionCircleOutlined, RobotOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import {
  CHART_ERROR_KEY, SELECT_SYMBOL_HINT_KEY,
  SEND_TO_AI_KEY, BROWSE_INDICATORS_KEY,
  CHART_WINDOW_KEY, CODE_KEY, SAVE_KEY, COPY_KEY, RUN_BACKTEST_KEY,
  BACKTEST_KEY as WS_BACKTEST_KEY, QUICK_TRADE_KEY,
} from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { BACKTEST_KEY as GEN_BACKTEST_KEY } from '@/gen/ant/v1/i18n/strategy_gen_keys';
import { COMMON_UNSAVED_KEY, COMMON_SAVED_KEY, COMMON_SAVE_KEY } from '@/gen/ant/v1/i18n/base_keys';
import { useWorkspaceStore, type CenterTab } from '@/stores/workspaceStore';
import PriceChart from '@/components/chart/PriceChart';
import BacktestPanel from '@/components/backtest/BacktestPanel';
import ChartBottomPanel from '@/components/chart/ChartBottomPanel';
import QuickTradePanel from '@/components/chart/QuickTradePanel';
import StrategyCodeEditor from '@/components/strategy/StrategyCodeEditor';
import WorkspaceErrorBoundary from './WorkspaceErrorBoundary';
import { useStrategyWorkspaceState } from '../../hooks/useStrategyWorkspaceState';

type WsState = ReturnType<typeof useStrategyWorkspaceState>;

interface Props {
  ws: WsState;
  btModalOpen: boolean;
  setBtModalOpen: (v: boolean) => void;
  setIndicatorDrawerOpen: (v: boolean) => void;
}

export default function WorkspaceCenterColumn({ ws, btModalOpen, setBtModalOpen, setIndicatorDrawerOpen }: Props) {
  const { t } = useTranslation();
  const centerTab = useWorkspaceStore(s => s.centerTab);
  const setCenterTab = useWorkspaceStore(s => s.setCenterTab);

  const strategyName = ws.templates.list.find((t2: any) => t2.id === ws.templates.selectedId)?.name || ws.code.loadedTemplate?.name || '';
  const saveStatus: 'modified' | 'saved' | 'none' = ws.code.code && ws.code.lastValidatedCode && ws.code.code !== ws.code.lastValidatedCode ? 'modified' : ws.code.lastSavedId ? 'saved' : 'none';

  const CTABS: { key: CenterTab; icon: string; label: string }[] = [
    { key: 'design', icon: '📈', label: t(CHART_WINDOW_KEY) },
    { key: 'code', icon: '📄', label: t(CODE_KEY) },
    { key: 'backtest', icon: '📊', label: t(GEN_BACKTEST_KEY) },
  ];

  const handleCopy = () => {
    if (!ws.code.code) return;
    navigator.clipboard?.writeText(ws.code.code).catch(() => {});
  };

  return (
    <div style={{ flex: '1 1 0', minWidth: 0, position: 'relative', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
      {/* Center tab bar */}
      <div style={{
        display: 'flex', alignItems: 'center', flexShrink: 0, height: 34,
        borderBottom: '1px solid var(--ant-color-border)',
        background: 'var(--ant-color-bg-container)',
      }}>
        {CTABS.map(({ key, icon, label }) => (
          <div
            key={key}
            onClick={() => setCenterTab(key)}
            style={{
              padding: '0 20px', height: '100%', display: 'flex', alignItems: 'center', gap: 6,
              fontSize: 12, fontWeight: 600, cursor: 'pointer',
              color: centerTab === key ? '#58a6ff' : 'var(--ant-color-text-secondary)',
              borderBottom: centerTab === key ? '2px solid #58a6ff' : '2px solid transparent',
            }}
          >
            {key === 'code' && saveStatus === 'modified' && <span style={{ color: '#d29922', fontSize: 14 }}>●</span>}
            {icon} {label}
          </div>
        ))}
        <div style={{ flex: 1 }} />
        {/* Code tab actions */}
        {centerTab === 'code' && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '0 12px', fontSize: 11 }}>
            {strategyName && <span style={{ fontWeight: 600, color: 'var(--ant-color-text)' }}>{strategyName}</span>}
            {saveStatus === 'modified' && <span style={{ color: '#d29922' }}>● {t(COMMON_UNSAVED_KEY)}</span>}
            {saveStatus === 'saved' && <span style={{ color: '#3fb950' }}>✓ {t(COMMON_SAVED_KEY)}</span>}
            <Tooltip title={t(SAVE_KEY)}>
              <Button size="small" icon={<SaveOutlined />}
                onClick={() => ws.code.setSaveModalOpen(true)}
                style={{ background: '#58a6ff', borderColor: '#58a6ff', color: '#fff' }}>
                {t(COMMON_SAVE_KEY)}
              </Button>
            </Tooltip>
            <Tooltip title={t(RUN_BACKTEST_KEY)}>
              <Button size="small" type="primary" icon={<PlayCircleOutlined />}
                onClick={() => setBtModalOpen(true)}
                style={{ background: '#3fb950', borderColor: '#3fb950' }}>
                {t(WS_BACKTEST_KEY)}
              </Button>
            </Tooltip>
            <Tooltip title={t(COPY_KEY)}>
              <Button size="small" icon={<CopyOutlined />} onClick={handleCopy} />
            </Tooltip>
            <Tooltip title={t(SEND_TO_AI_KEY)}>
              <Button size="small" icon={<RobotOutlined />} onClick={() => ws.layout.setRightTab('chat')}
                style={{ background: '#722ed1', borderColor: '#722ed1', color: '#fff' }}>
                {t(SEND_TO_AI_KEY)}
              </Button>
            </Tooltip>
            <Tooltip title={t(BROWSE_INDICATORS_KEY)}>
              <Button size="small" icon={<QuestionCircleOutlined />} onClick={() => setIndicatorDrawerOpen(true)} />
            </Tooltip>
          </div>
        )}
      </div>

      {/* Design = Chart */}
      <div style={{ flex: '1 1 0', minHeight: 0, display: centerTab === 'design' ? 'flex' : 'none', flexDirection: 'column' }}>
        {ws.account.symbol ? (
          <WorkspaceErrorBoundary fallback={<div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--ant-color-text-tertiary)' }}>{t(CHART_ERROR_KEY)}</div>}>
            <PriceChart
              symbol={ws.account.symbol} timeframe={ws.account.timeframe} onTimeframeChange={ws.account.setTimeframe}
              accountId={ws.account.accountId}
              trades={ws.backtest.chartTrades}
            />
          </WorkspaceErrorBoundary>
        ) : (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--ant-color-text-secondary)', fontSize: 14 }}>
            {t(SELECT_SYMBOL_HINT_KEY)}
          </div>
        )}
      </div>

      {/* Code */}
      <div style={{ flex: '1 1 0', minHeight: 0, display: centerTab === 'code' ? 'flex' : 'none', flexDirection: 'column' }}>
        <StrategyCodeEditor
          value={ws.code.code}
          onChange={ws.code.setCode}
          style={{ flex: 1, borderRadius: 0, border: 'none', minHeight: 0 }}
        />
      </div>

      {/* Backtest */}
      <div style={{ flex: '1 1 0', minHeight: 0, display: centerTab === 'backtest' ? 'flex' : 'none', flexDirection: 'column' }}>
        <BacktestPanel
          runner={ws.backtest.runner}
          inputs={{
            strategyCode: ws.code.code,
            accountId: ws.account.accountId,
            symbol: ws.account.symbol,
            timeframe: ws.account.timeframe,
            templateId: ws.templates.selectedId || undefined,
            strategyId: ws.code.strategyId,
          }}
          templates={{
            list: ws.templates.list,
            loading: ws.templates.loading,
            selectedId: ws.templates.selectedId,
            onSelect: ws.templates.onSelect,
          }}
          onOpenHistory={(templateId?: string) => ws.history.open(templateId)}
          onAIOptimize={() => ws.ai.optimize()}
          code={ws.code.code}
          onApplyTunedParams={ws.code.setCode}
          onRunBacktest={() => setBtModalOpen(true)}
          onSaveAs={() => ws.code.setSaveModalOpen(true)}
          hasUnsavedDraft={saveStatus === 'modified'}
          draftName={strategyName}
        />
      </div>

      {/* Bottom panel: Positions | History | Backtest  +  Quick Trade on the right */}
      {ws.layout.bottomPanelCollapsed ? (
        <ChartBottomPanel
          positions={ws.quickTrade.allPositions}
          recentTrades={ws.quickTrade.qtRecentTrades}
          onClosePosition={ws.quickTrade.handleClosePosition}
          collapsed={true}
          onToggleCollapsed={() => ws.layout.setBottomPanelCollapsed(false)}
          backtestMetrics={ws.backtest.metrics}
          backtestStatus={ws.backtest.status}
        />
      ) : (
        <div style={{ flexShrink: 0, display: 'flex', borderTop: '1px solid var(--ant-color-border)' }}>
          <div style={{ flex: '1 1 0', minWidth: 0 }}>
            <ChartBottomPanel
              positions={ws.quickTrade.allPositions}
              recentTrades={ws.quickTrade.qtRecentTrades}
              onClosePosition={ws.quickTrade.handleClosePosition}
              collapsed={false}
              onToggleCollapsed={() => ws.layout.setBottomPanelCollapsed(true)}
              backtestMetrics={ws.backtest.metrics}
              backtestStatus={ws.backtest.status}
            />
          </div>
          {ws.account.symbol && (
            <div style={{
              width: 420, flexShrink: 0,
              borderLeft: '1px solid var(--ant-color-border)',
              background: 'var(--ant-color-bg-elevated)',
              display: 'flex', flexDirection: 'column',
              maxHeight: 200, overflow: 'hidden',
            }}>
              <div style={{
                padding: '4px 10px', fontSize: 11, fontWeight: 700,
                borderBottom: '1px solid var(--ant-color-border)',
                background: 'var(--ant-color-bg-layout)',
                display: 'flex', alignItems: 'center', gap: 4, flexShrink: 0,
              }}>
                ⚡ {t(QUICK_TRADE_KEY)}
              </div>
              <div style={{ flex: 1, overflowY: 'auto', padding: '4px 10px' }}>
                <QuickTradePanel
                  accountId={ws.account.accountId}
                  symbol={ws.account.symbol}
                  accountMeta={ws.account.selectedAccountMeta}
                  allPositions={ws.quickTrade.allPositions}
                  positions={ws.quickTrade.qtPositions}
                  recentTrades={ws.quickTrade.qtRecentTrades}
                  onClosePosition={ws.quickTrade.handleClosePosition}
                  horizontal
                />
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
