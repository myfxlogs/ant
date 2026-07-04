import { useState } from 'react';
import { Grid, Button, Tooltip, Input, Empty, message } from 'antd';
import { PlayCircleOutlined, SaveOutlined, CopyOutlined, EditOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { CHART_ERROR_KEY, SELECT_SYMBOL_HINT_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { useStrategyWorkspaceState } from './hooks/useStrategyWorkspaceState';
import WorkspaceTemplateManager from './components/workspace/WorkspaceTemplateManager';
import WorkspaceToolbar from './components/workspace/WorkspaceToolbar';
import BacktestResultsCard from '@/components/strategy/BacktestResultsCard';
import BacktestPanel from '@/components/backtest/BacktestPanel';
import ChartBottomPanel from '@/components/chart/ChartBottomPanel';
import { useAuthStore } from '@/stores/authStore';
import { useWorkspaceStore, type CenterTab } from '@/stores/workspaceStore';
import PriceChart from '@/components/chart/PriceChart';
import BacktestHistoryDrawer from './components/workspace/BacktestHistoryDrawer';
import WorkspaceErrorBoundary from './components/workspace/WorkspaceErrorBoundary';
import MobileGuard from './components/workspace/MobileGuard';
import RightPanel from './components/workspace/RightPanel';
import QuickTradePanel from '@/components/chart/QuickTradePanel';
import { useWorkspaceSession } from './hooks/useWorkspaceSession';
import { SaveTemplateWrapper } from './WorkspaceLayout';

const COL_BORDER = '1px solid var(--ant-color-border)';

export default function StrategyWorkspacePage() {
  const { t } = useTranslation();
  const screens = Grid.useBreakpoint();
  const ws = useStrategyWorkspaceState();
  const userId = useAuthStore(s => s.user?.id) ?? '';
  const { sessionId } = useWorkspaceSession(
    userId,
    ws.account.symbol,
    ws.account.timeframe,
    ws.code.lastSavedId,
  );

  const centerTab = useWorkspaceStore(s => s.centerTab);
  const setCenterTab = useWorkspaceStore(s => s.setCenterTab);
  const [editable, setEditable] = useState(false);

  if (!screens.lg) return <MobileGuard />;

  const handleCopy = () => {
    if (!ws.code.code) return;
    navigator.clipboard?.writeText(ws.code.code)
      .then(() => message.success(t('common.copied', '已复制')))
      .catch(() => message.error(t('common.copyFailed', '复制失败')));
  };

  const strategyName = ws.templates.list.find((t2: any) => t2.id === ws.templates.selectedId)?.name || ws.code.loadedTemplate?.name || '';
  const saveStatus: 'modified' | 'saved' | 'none' = ws.code.code && ws.code.lastValidatedCode && ws.code.code !== ws.code.lastValidatedCode ? 'modified' : ws.code.lastSavedId ? 'saved' : 'none';

  const CTABS: { key: CenterTab; icon: string; label: string }[] = [
    { key: 'design', icon: '📈', label: t('strategy.workspace.design', 'Design') },
    { key: 'code', icon: '📄', label: 'Code' },
    { key: 'backtest', icon: '📊', label: t('strategy.gen.backtest', 'Backtest') },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 112px)' }}>
      {/* ═══ TOP TOOLBAR ═══ */}
      <WorkspaceToolbar
        accounts={ws.account.activeAccounts} accountId={ws.account.accountId} onAccountChange={ws.account.handleAccountChange}
        symbol={ws.account.symbol} onSymbolChange={ws.account.setSymbol}
        accountInfo={ws.account.accountInfo} positionCount={ws.quickTrade.positionCount}
        busy={ws.backtest.submitting || ws.tuning.running}
        onTogglePositionsPanel={() => ws.layout.setPositionsPanelVisible(!ws.layout.positionsPanelVisible)}
        strategyName={strategyName}
        saveStatus={saveStatus}
      />

      {/* ═══ BODY: Three-column layout ═══ */}
      <div style={{ display: 'flex', flex: '1 1 auto', overflow: 'hidden', minHeight: 0 }}>

        {/* ── LEFT COLUMN: Strategy list + backtest summary (260px, collapsible) ── */}
        <div style={{
          width: 260, minWidth: 260, flexShrink: 0, borderRight: COL_BORDER,
          display: 'flex', flexDirection: 'column', overflow: 'hidden',
          marginLeft: ws.layout.leftSidebarCollapsed ? -260 : 0,
          transition: 'margin-left 0.25s ease',
          background: 'var(--ant-color-bg-container)',
        }}>
          <div style={{ padding: '8px 12px 2px', fontSize: 10, fontWeight: 700, color: 'var(--ant-color-text-tertiary)', textTransform: 'uppercase', letterSpacing: 0.5, flexShrink: 0 }}>
            📋 {t('strategy.workspace.templates', 'Strategies')}
          </div>
          <div style={{ padding: '4px 12px', flexShrink: 0 }}>
            <WorkspaceTemplateManager
              templates={ws.templates.list}
              loading={ws.templates.loading}
              loadedTemplate={ws.templates.list.find((t2: any) => t2.id === ws.templates.selectedId) || null}
              onLoad={ws.templates.onSelect}
              onSaveAs={() => ws.code.setSaveModalOpen(true)}
            />
          </div>

          <div style={{ flexShrink: 0 }}>
            <BacktestResultsCard metrics={ws.backtest.metrics} status={ws.backtest.status} />
          </div>

          <div style={{ flex: 1 }} />
        </div>

        {/* Left sidebar toggle button */}
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

        {/* ── CENTER COLUMN: Design/Code/Backtest tabs ── */}
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
                {saveStatus === 'modified' && <span style={{ color: '#d29922' }}>● {t('common.unsaved', 'Unsaved')}</span>}
                {saveStatus === 'saved' && <span style={{ color: '#3fb950' }}>✓ {t('common.saved', 'Saved')}</span>}
                <Tooltip title={t('strategy.workspace.save', 'Save Strategy')}>
                  <Button size="small" icon={<SaveOutlined />}
                    onClick={() => ws.code.setSaveModalOpen(true)}
                    style={{ background: '#58a6ff', borderColor: '#58a6ff', color: '#fff' }}>
                    {t('common.save', 'Save')}
                  </Button>
                </Tooltip>
                <Tooltip title={t('strategy.workspace.runBacktest', 'Run Backtest')}>
                  <Button size="small" type="primary" icon={<PlayCircleOutlined />}
                    onClick={ws.backtest.run}
                    style={{ background: '#3fb950', borderColor: '#3fb950' }}>
                    {t('strategy.gen.backtest', 'Backtest')}
                  </Button>
                </Tooltip>
                <Tooltip title={t('common.copy', 'Copy Code')}>
                  <Button size="small" icon={<CopyOutlined />} onClick={handleCopy} />
                </Tooltip>
                <Tooltip title={editable ? t('common.readOnly', 'Read Only') : t('common.edit', 'Edit')}>
                  <Button size="small" type={editable ? 'primary' : 'default'} icon={<EditOutlined />}
                    onClick={() => setEditable(e => !e)} />
                </Tooltip>
              </div>
            )}
          </div>

          {/* Tab content — always mounted, hidden via display:none */}
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
                {t(SELECT_SYMBOL_HINT_KEY, 'Select a trading account and symbol to view chart')}
              </div>
            )}
          </div>

          {/* Code */}
          <div style={{ flex: '1 1 0', minHeight: 0, display: centerTab === 'code' ? 'flex' : 'none', flexDirection: 'column' }}>
            {ws.code.code ? (
              editable ? (
                <Input.TextArea
                  value={ws.code.code}
                  onChange={(e) => ws.code.setCode(e.target.value)}
                  style={{
                    flex: 1, borderRadius: 0, resize: 'none',
                    fontFamily: '"SF Mono", "Fira Code", monospace',
                    fontSize: 12, lineHeight: 1.6,
                  }}
                />
              ) : (
                <div style={{
                  flex: 1, overflowY: 'auto', padding: 16,
                  fontFamily: '"SF Mono", "Fira Code", monospace',
                  fontSize: 12, lineHeight: 1.7, whiteSpace: 'pre-wrap',
                  color: 'var(--ant-color-text)',
                }}>
                  {ws.code.code}
                </div>
              )
            ) : (
              <div style={{ display: 'flex', justifyContent: 'center', paddingTop: 40 }}>
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description={t('strategy.workspace.noCode', 'No code generated yet')} />
              </div>
            )}
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
              collapsed={false}
              onToggleCollapsed={() => {}}
              onOpenHistory={() => ws.history.open()}
              onAIOptimize={() => ws.ai.optimize()}
              code={ws.code.code}
              onApplyTunedParams={ws.code.setCode}
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
              {/* Left: tabbed tables */}
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
              {/* Right: Quick Trade */}
              {ws.account.symbol && (
                <div style={{
                  width: 280, flexShrink: 0,
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
                    ⚡ Quick Trade
                  </div>
                  <div style={{ flex: 1, overflowY: 'auto', padding: '4px 8px' }}>
                    <QuickTradePanel
                      accountId={ws.account.accountId}
                      symbol={ws.account.symbol}
                      accountMeta={ws.account.selectedAccountMeta}
                      allPositions={ws.quickTrade.allPositions}
                      positions={ws.quickTrade.qtPositions}
                      recentTrades={ws.quickTrade.qtRecentTrades}
                      onClosePosition={ws.quickTrade.handleClosePosition}
                    />
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* ── RIGHT COLUMN: Chat only (380px, no tabs) ── */}
        <RightPanel
          symbol={ws.account.symbol}
          timeframe={ws.account.timeframe}
          sessionId={sessionId}
          accountId={ws.account.accountId}
          onApplyCode={(code) => { ws.code.setCode(code); setCenterTab('code'); }}
          onValidateResult={(result) => ws.backtest.runner.handleValidationResult(result)}
        />
      </div>

      <SaveTemplateWrapper ws={ws} />
      <BacktestHistoryDrawer
        open={ws.history.modalOpen || ws.history.drawerOpen}
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
        onClose={ws.history.runId ? ws.history.close : ws.history.closeModal}
        runId={ws.history.runId}
      />
    </div>
  );
}
