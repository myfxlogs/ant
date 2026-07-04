import { Grid, Select } from 'antd';
import { useTranslation } from 'react-i18next';
import { CHART_ERROR_KEY, SELECT_SYMBOL_HINT_KEY, TITLE_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { useStrategyWorkspaceState } from './hooks/useStrategyWorkspaceState';
import WorkspaceTemplateManager from './components/workspace/WorkspaceTemplateManager';
import WorkspaceToolbar from './components/workspace/WorkspaceToolbar';
import StrategyChat from '@/components/strategy/StrategyChat';
import BacktestResultsCard from '@/components/strategy/BacktestResultsCard';
import QuickTradePanel from '@/components/chart/QuickTradePanel';
import ChartBottomPanel from '@/components/chart/ChartBottomPanel';
import SymbolPicker from '@/components/chart/SymbolPicker';
import { useAuthStore } from '@/stores/authStore';
import PriceChart from '@/components/chart/PriceChart';
import BacktestRunDrawer from '@/components/strategy/BacktestRunDrawer';
import BacktestHistoryModal from './components/workspace/BacktestHistoryModal';
import WorkspaceErrorBoundary from './components/workspace/WorkspaceErrorBoundary';
import MobileGuard from './components/workspace/MobileGuard';
import { useWorkspaceSession } from './hooks/useWorkspaceSession';
import { SaveTemplateWrapper } from './WorkspaceLayout';

const COL_BORDER = '1px solid var(--ant-color-border)';
const TIMEFRAMES = ['1m', '5m', '15m', '30m', '1h', '4h', '1d', '1w'];

export default function StrategyWorkspacePage() {
  const { t } = useTranslation();
  const screens = Grid.useBreakpoint();
  const ws = useStrategyWorkspaceState();
  const userId = useAuthStore(s => s.user?.id);
  const { sessionId } = useWorkspaceSession(
    userId,
    ws.account.symbol,
    ws.account.timeframe,
    ws.code.lastSavedId,
  );

  if (!screens.lg) return <MobileGuard />;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 112px)' }}>
      {/* Title bar */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 0 12px' }}>
        <h2 style={{ margin: 0, fontSize: 16, fontWeight: 600 }}>{t(TITLE_KEY, 'Strategy Workspace')}</h2>
      </div>

      {/* ═══ TOP TOOLBAR ═══ */}
      <WorkspaceToolbar
        accounts={ws.account.activeAccounts} accountId={ws.account.accountId} onAccountChange={ws.account.handleAccountChange}
        symbol={ws.account.symbol} onSymbolChange={ws.account.setSymbol}
        accountInfo={ws.account.accountInfo} positionCount={ws.quickTrade.positionCount}
        busy={ws.backtest.submitting || ws.tuning.running}
        positionsCount={ws.quickTrade.allPositions.length}
        onTogglePositionsPanel={() => ws.layout.setPositionsPanelVisible(!ws.layout.positionsPanelVisible)}
      />

      {/* ═══ BODY: Two-column + AI drawer overlay ═══ */}
      <div style={{ display: 'flex', flex: '1 1 auto', overflow: 'hidden', minHeight: 0, position: 'relative' }}>

        {/* ── LEFT COLUMN: slide-out drawer (like AI chat) ── */}
        <div style={{
          width: 260, minWidth: 260, flexShrink: 0, borderRight: COL_BORDER,
          display: 'flex', flexDirection: 'column', overflow: 'hidden',
          marginLeft: ws.layout.leftSidebarCollapsed ? -260 : 0,
          transition: 'margin-left 0.25s ease',
        }}>
          <div style={{ padding: 12, flexShrink: 0 }}>
            <WorkspaceTemplateManager
              templates={ws.templates.list}
              loading={ws.templates.loading}
              loadedTemplate={ws.templates.list.find((t: any) => t.id === ws.templates.selectedId) || null}
              onLoad={ws.templates.onSelect}
              onSaveAs={() => ws.code.setSaveModalOpen(true)}
            />
          </div>

          <div style={{ flexShrink: 0 }}>
            <BacktestResultsCard metrics={ws.backtest.metrics} status={ws.backtest.status} />
          </div>

          <div style={{ flex: 1 }} />

          <div style={{ borderTop: COL_BORDER, padding: 10, flexShrink: 0 }}>
            <div style={{ fontSize: 10, fontWeight: 700, color: 'var(--ant-color-text-tertiary)', textTransform: 'uppercase', marginBottom: 6 }}>
              {t('strategy.workspace.symbol', 'Symbol')}
            </div>
            <SymbolPicker accountId={ws.account.accountId} value={ws.account.symbol} onChange={ws.account.setSymbol} style={{ width: '100%' }} />
            <div style={{ fontSize: 10, fontWeight: 700, color: 'var(--ant-color-text-tertiary)', textTransform: 'uppercase', margin: '8px 0 4px' }}>
              {t('strategy.workspace.timeframe', 'Timeframe')}
            </div>
            <Select size="small" style={{ width: '100%' }} value={ws.account.timeframe} onChange={ws.account.setTimeframe}
              options={TIMEFRAMES.map(tf => ({ value: tf, label: tf.toUpperCase() }))} />
          </div>
        </div>

        {/* Left sidebar toggle button (vertical, left edge of chart) */}
        <div
          onClick={() => ws.layout.setLeftSidebarCollapsed(!ws.layout.leftSidebarCollapsed)}
          style={{
            width: 28, height: 80, flexShrink: 0,
            background: 'var(--ant-color-bg-elevated)',
            border: '1px solid var(--ant-color-border)', borderLeft: 'none',
            borderRadius: '0 8px 8px 0', cursor: 'pointer', zIndex: 20,
            display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
            color: ws.layout.leftSidebarCollapsed ? 'var(--ant-color-text-tertiary)' : '#58a6ff',
            fontSize: 11, writingMode: 'vertical-rl', letterSpacing: 2,
            alignSelf: 'center',
          }}
        >
          Strategies
        </div>

        {/* ── CENTER COLUMN: Chart (flex:1) + Bottom Panel (fixed) ── */}
        <div style={{ flex: '1 1 0', minWidth: 0, position: 'relative', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
          {/* Chart area */}
          <div style={{ flex: '1 1 0', minHeight: 0, position: 'relative', display: 'flex', flexDirection: 'column' }}>
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
                {t(SELECT_SYMBOL_HINT_KEY, 'Select a trading account and symbol to view chart')}
              </div>
            )}

            {/* Quick Trade floating overlay (top-right, 210px) — collapsible */}
            {ws.account.symbol && (
              <div style={{
                position: 'absolute', top: 48, right: ws.layout.aiDrawerOpen ? 496 : 40, zIndex: 10,
                width: 210, transition: 'right 0.25s ease',
              }}>
                <div style={{
                  background: 'var(--ant-color-bg-elevated)', border: '1px solid var(--ant-color-border)',
                  borderRadius: 8, boxShadow: '0 4px 16px rgba(0,0,0,0.4)', overflow: 'hidden',
                }}>
                  {/* Title bar — always visible, click to toggle */}
                  <div
                    onClick={() => ws.layout.setQuickTradeCollapsed(!ws.layout.quickTradeCollapsed)}
                    style={{
                      padding: '8px 12px', cursor: 'pointer', userSelect: 'none',
                      display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                      fontSize: 12, fontWeight: 700, color: 'var(--ant-color-text)',
                      borderBottom: ws.layout.quickTradeCollapsed ? 'none' : '1px solid var(--ant-color-border)',
                      background: 'var(--ant-color-bg-layout)',
                    }}
                  >
                    <span>⚡ Quick Trade</span>
                    <span style={{ fontSize: 10, color: 'var(--ant-color-text-tertiary)' }}>
                      {ws.layout.quickTradeCollapsed ? '▼' : '▲'}
                    </span>
                  </div>
                  {/* Order form — only when expanded */}
                  {!ws.layout.quickTradeCollapsed && (
                    <div style={{ padding: 10 }}>
                      <QuickTradePanel
                        accountId={ws.account.accountId} symbol={ws.account.symbol}
                        accountMeta={ws.account.selectedAccountMeta}
                        allPositions={[]}
                        positions={[]}
                        recentTrades={[]}
                        onClosePosition={ws.quickTrade.handleClosePosition}
                        onToggleAllPositions={() => ws.layout.setBottomPanelCollapsed(false)}
                      />
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* AI Chat toggle button (vertical, right edge) */}
            <div
              onClick={() => ws.layout.setAiDrawerOpen(!ws.layout.aiDrawerOpen)}
              style={{
                position: 'absolute', right: 0, top: '50%', transform: 'translateY(-50%)',
                width: 28, height: 80, background: 'var(--ant-color-bg-elevated)',
                border: '1px solid var(--ant-color-border)', borderRight: 'none',
                borderRadius: '8px 0 0 8px', cursor: 'pointer', zIndex: 20,
                display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
                color: ws.layout.aiDrawerOpen ? '#58a6ff' : 'var(--ant-color-text-tertiary)',
                fontSize: 11, writingMode: 'vertical-rl', letterSpacing: 2,
              }}
            >
              AI Chat
            </div>

            {/* AI Chat Drawer (480px slide-out, overlays chart) */}
            <div style={{
              position: 'absolute', right: 0, top: 0, bottom: 0, width: 480, zIndex: 15,
              background: 'var(--ant-color-bg-container)', borderLeft: COL_BORDER,
              display: 'flex', flexDirection: 'column',
              transform: ws.layout.aiDrawerOpen ? 'translateX(0)' : 'translateX(100%)',
              transition: 'transform 0.25s ease',
              boxShadow: ws.layout.aiDrawerOpen ? '-8px 0 24px rgba(0,0,0,0.3)' : 'none',
            }}>
              <StrategyChat
                symbol={ws.account.symbol}
                timeframe={ws.account.timeframe}
                sessionId={sessionId}
                accountId={ws.account.accountId}
                onApplyCode={(code) => ws.code.setCode(code)}
                onValidateResult={(result) => ws.backtest.runner.handleValidationResult(result)}
                onRunBacktest={ws.backtest.run}
                backtestStatus={ws.backtest.status}
              />
            </div>
          </div>

          {/* Bottom panel: Positions + History (MT-style) */}
          <ChartBottomPanel
            positions={ws.quickTrade.allPositions}
            recentTrades={ws.quickTrade.qtRecentTrades}
            onClosePosition={ws.quickTrade.handleClosePosition}
            collapsed={ws.layout.bottomPanelCollapsed}
            onToggleCollapsed={() => ws.layout.setBottomPanelCollapsed(!ws.layout.bottomPanelCollapsed)}
          />
        </div>
      </div>

      <SaveTemplateWrapper ws={ws} />
      <BacktestHistoryModal
        open={ws.history.modalOpen}
        runs={ws.history.runs}
        loading={ws.history.loading}
        page={ws.history.page}
        pageSize={ws.history.pageSize}
        total={ws.history.total}
        selectedRowKeys={ws.history.selectedRowKeys}
        deleting={ws.history.deleting}
        onPageChange={ws.history.onPageChange}
        onSelectionChange={ws.history.setSelectedRowKeys}
        onViewRun={ws.history.onViewRun}
        onDeleteRun={ws.history.onDeleteRun}
        onBatchDelete={ws.history.onBatchDelete}
        onRefresh={ws.history.onRefresh}
        onClose={ws.history.closeModal}
      />
      <BacktestRunDrawer
        open={ws.history.drawerOpen} runId={ws.history.runId}
        onClose={ws.history.close}
        onCancel={ws.history.close}
      />
    </div>
  );
}
