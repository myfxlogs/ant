import { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Tooltip, Modal, Drawer } from 'antd';
import { PlayCircleOutlined, SaveOutlined, CopyOutlined, QuestionCircleOutlined, RobotOutlined, HistoryOutlined, ImportOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import {
  SEND_TO_AI_KEY, BROWSE_INDICATORS_KEY,
  CODE_KEY, SAVE_KEY, COPY_KEY, RUN_BACKTEST_KEY,
  BACKTEST_KEY as WS_BACKTEST_KEY, AI_ASSISTANT_KEY,
} from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { COMMON_UNSAVED_KEY, COMMON_SAVED_KEY, COMMON_SAVE_KEY } from '@/gen/ant/v1/i18n/base_keys';
import { useWorkspaceStore, type CenterTab } from '@/stores/workspaceStore';
import BacktestPanel from '@/components/backtest/BacktestPanel';
import StrategyCodeEditor from '@/components/strategy/StrategyCodeEditor';
import StrategyChat from '@/components/strategy/StrategyChat';
import ImportEAPanel from '../editor/ImportEAPanel';
import WorkspaceSidebar from './WorkspaceSidebar';
import WorkspaceAIPanel from './WorkspaceAIPanel';
import BottomPanelSection from './BottomPanelSection';
import { useWsAccount, useWsCode, useWsTemplates, useWsBacktest, useWsQuickTrade, useWsLayout, useWsHistory, useWsAI } from '../../WorkspaceContext';

interface Props {
  isMobile?: boolean;
  btModalOpen: boolean;
  setBtModalOpen: (v: boolean) => void;
  setIndicatorDrawerOpen: (v: boolean) => void;
  onShowVersionHistory?: () => void;
}

export default function WorkspaceCenterColumn({ isMobile = false, _btModalOpen, setBtModalOpen, setIndicatorDrawerOpen, onShowVersionHistory }: Props) {
  const { t } = useTranslation();
  const centerTab = useWorkspaceStore(s => s.centerTab);
  const setCenterTab = useWorkspaceStore(s => s.setCenterTab);

  const account = useWsAccount();
  const code = useWsCode();
  const templates = useWsTemplates();
  const backtest = useWsBacktest();
  const quickTrade = useWsQuickTrade();
  const layout = useWsLayout();
  const history = useWsHistory();
  const ai = useWsAI();

  // ── Sidebar ──────────────────────────────────────────────────────────
  const leftSidebarCollapsed = useWorkspaceStore(s => s.leftSidebarCollapsed);
  const setLeftSidebarCollapsed = useWorkspaceStore(s => s.setLeftSidebarCollapsed);
  const [sidebarDrawerOpen, setSidebarDrawerOpen] = useState(false);

  // ── Right panel tab: 'ai' | 'backtest' | null — mode-driven layout ──
  const [rightPanelTab, setRightPanelTab] = useState<'ai' | 'backtest' | null>(null);
  // Auto-switch to backtest tab when backtest starts
  const prevBtStatusRef = useRef(backtest.status);
  useEffect(() => {
    if (backtest.status === 'running' && prevBtStatusRef.current !== 'running') {
      setRightPanelTab('backtest');
    }
    prevBtStatusRef.current = backtest.status;
  }, [backtest.status]);
  // Mode-driven: collapse sidebars when right panel is active
  useEffect(() => {
    if (rightPanelTab === 'backtest') {
      layout.setBottomPanelCollapsed(true);
      setLeftSidebarCollapsed(true);
    } else if (rightPanelTab === 'ai') {
      setLeftSidebarCollapsed(true);
    }
  }, [rightPanelTab, layout, setLeftSidebarCollapsed]);

  // ── Bottom panel resize ──────────────────────────────────────────────
  const [bpDragging, setBpDragging] = useState(false);
  const handleBpResize = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setBpDragging(true);
    layout.setBottomPanelUserResized(true);
    const startY = e.clientY;
    const startH = layout.bottomPanelHeight;
    const onMove = (ev: MouseEvent) => {
      const delta = startY - ev.clientY;
      layout.setBottomPanelHeight(Math.max(80, Math.min(500, startH + delta)));
    };
    const onUp = () => {
      setBpDragging(false);
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
    };
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
  }, [layout]);

  // ── Auto-restore last backtest on first visit ────────────────────────
  const restoreRef = useRef(false);
  useEffect(() => {
    if (restoreRef.current) return;
    if (!account.accountId) return;
    if (backtest.runner.status !== 'idle') return;
    restoreRef.current = true;
    backtest.runner.restoreLastRun(account.accountId, templates.selectedId || undefined);
  }, [account.accountId, backtest.runner, templates.selectedId]);

  // ── Backtest drawer (replaces backtest tab) ──────────────────────────
  const [btDrawerOpen, setBtDrawerOpen] = useState(false);

  // ── Import MQL inline (replaces empty state area, never modal) ────────
  const [importMode, setImportMode] = useState(false);

  // ── Backtest context for AI ───────────────────────────────────────────
  const btSummary = backtest.metrics?.totalTrades != null
    ? { totalReturn: backtest.metrics.totalReturn, maxDrawdown: backtest.metrics.maxDrawdown, sharpeRatio: backtest.metrics.sharpeRatio, winRate: backtest.metrics.winRate, totalTrades: backtest.metrics.totalTrades }
    : undefined;
  const recentSummaries = (history.runs as Array<{ templateName?: string; totalReturn?: number; totalTrades?: number; startedAt?: string }>)
    ?.slice(0, 10).map(r => ({ templateName: r.templateName || '', totalReturn: r.totalReturn ?? 0, totalTrades: r.totalTrades ?? 0, startedAt: r.startedAt || '' })) || [];

  // Redirect legacy tab values from localStorage (import/strategies/backtest tabs removed)
  useEffect(() => {
    if (centerTab === 'import' || centerTab === 'strategies' || centerTab === 'backtest') {
      setCenterTab('code');
    }
  }, [centerTab, setCenterTab]);

  const strategyName = templates.list.find((t2: { id: string; name?: string }) => t2.id === templates.selectedId)?.name || code.loadedTemplate?.name || '';
  const saveStatus: 'modified' | 'saved' | 'none' = code.code && code.lastValidatedCode && code.code !== code.lastValidatedCode ? 'modified' : code.lastSavedId ? 'saved' : 'none';

  // Desktop: AI is a right panel, not a tab. Mobile: keep chat tab for full-width overlay.
  const CTABS: { key: CenterTab; icon: string; label: string }[] = isMobile
    ? [
        { key: 'chat', icon: '🤖', label: t(AI_ASSISTANT_KEY) },
        { key: 'code', icon: '📝', label: t(CODE_KEY) },
      ]
    : [
        { key: 'code', icon: '📝', label: t(CODE_KEY) },
      ];

  // Desktop: redirect chat tab clicks to AI panel toggle
  const aiPanelOpen = useWorkspaceStore(s => s.aiPanelOpen);
  const setAiPanelOpen = useWorkspaceStore(s => s.setAiPanelOpen);
  const aiPanelWidth = useWorkspaceStore(s => s.aiPanelWidth);
  const setAiPanelWidth = useWorkspaceStore(s => s.setAiPanelWidth);

  // Desktop: if centerTab is 'chat', open AI panel instead
  useEffect(() => {
    if (!isMobile && centerTab === 'chat') {
      setAiPanelOpen(true);
      setCenterTab('code');
    }
  }, [isMobile, centerTab, setCenterTab, setAiPanelOpen]);

  const handleCopy = () => {
    if (!code.code) return;
    navigator.clipboard?.writeText(code.code).catch(() => {});
  };

  // ── AI panel resize ──────────────────────────────────────────────────
  const [aiDragging, setAiDragging] = useState(false);
  const handleAiResize = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setAiDragging(true);
    const startX = e.clientX;
    const startW = aiPanelWidth;
    const onMove = (ev: MouseEvent) => {
      const delta = startX - ev.clientX;
      setAiPanelWidth(Math.max(300, Math.min(600, startW + delta)));
    };
    const onUp = () => {
      setAiDragging(false);
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
    };
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
  }, [aiPanelWidth, setAiPanelWidth]);

  return (
    <div data-tour="code-editor" style={{ flex: '1 1 0', minWidth: 0, position: 'relative', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
      {/* Center tab bar */}
      <div style={{
        display: 'flex', alignItems: 'center', flexShrink: 0, height: 34,
        borderBottom: '1px solid var(--ant-color-border)',
        background: 'var(--ant-color-bg-container)',
      }}>
        {/* Mobile: sidebar toggle button */}
        {isMobile && (
          <Button size="small" type="text" icon={<span style={{ fontSize: 16 }}>☰</span>}
            onClick={() => setSidebarDrawerOpen(true)}
            style={{ marginLeft: 4, padding: '0 6px' }} />
        )}
        {CTABS.map(({ key, icon, label }) => (
          <div
            key={key}
            data-tour={key === 'backtest' ? 'backtest' : undefined}
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
                disabled={!code.code}
                onClick={() => code.setSaveModalOpen(true)}
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
              <Button size="small" icon={<RobotOutlined />}
                disabled={!code.code}
                onClick={() => isMobile ? setCenterTab('chat') : setRightPanelTab(prev => prev === 'ai' ? null : 'ai')}
                style={rightPanelTab === 'ai'
                  ? { background: '#531dab', borderColor: '#531dab', color: '#fff' }
                  : { background: '#722ed1', borderColor: '#722ed1', color: '#fff' }}>
                {t(SEND_TO_AI_KEY)}
              </Button>
            </Tooltip>
            <Tooltip title={t(BROWSE_INDICATORS_KEY)}>
              <Button size="small" icon={<QuestionCircleOutlined />} onClick={() => setIndicatorDrawerOpen(true)} />
            </Tooltip>
            {onShowVersionHistory && code.strategyId && (
              <Tooltip title={t('strategy.version.history', { defaultValue: 'Version History' })}>
                <Button size="small" icon={<HistoryOutlined />} onClick={onShowVersionHistory} />
              </Tooltip>
            )}
          </div>
        )}
      </div>

      {/* Main area: sidebar + content */}
      <div style={{ flex: '1 1 0', minHeight: 0, display: 'flex', flexDirection: 'row' }}>
        {/* Sidebar: persistent on desktop, drawer overlay on mobile */}
        {!isMobile && (
          <WorkspaceSidebar
            templates={templates.list}
            loading={templates.loading}
            selectedId={templates.selectedId || ''}
            onSelect={(id) => templates.onSelect(id)}
            backtestRuns={(history.runs as Array<{ id: string; startedAt?: string; totalReturn?: number; totalTrades?: number; templateName?: string; templateId?: string }>) || []}
            runsLoading={history.loading}
            onOpenHistory={(tid) => history.open(tid)}
            onImport={() => setImportMode(true)}
            onNew={() => { templates.onSelect(''); }}
            collapsed={leftSidebarCollapsed}
            onToggle={() => setLeftSidebarCollapsed(!leftSidebarCollapsed)}
          />
        )}

        {/* Content */}
        <div style={{ flex: '1 1 0', minWidth: 0, display: 'flex', flexDirection: 'column' }}>
          {/* AI Chat — full width tab (mobile only; desktop uses right panel) */}
          {isMobile && (
            <div style={{ flex: '1 1 0', minHeight: 0, display: centerTab === 'chat' ? 'flex' : 'none', flexDirection: 'column' }}>
              <StrategyChat
                symbol={account.symbol}
                timeframe={account.timeframe}
                accountId={account.accountId}
                onApplyCode={c => { code.setCode(c); setCenterTab('code'); }}
                currentCode={code.code}
                lastBacktest={btSummary}
                recentBacktests={recentSummaries}
              />
            </div>
          )}

          {/* Code + optional AI right panel (desktop) */}
          <div style={{ flex: '1 1 0', minHeight: 0, display: centerTab === 'code' ? 'flex' : 'none', flexDirection: 'row' }}>
        <div style={{ flex: '1 1 0', minWidth: 0, display: 'flex', flexDirection: 'column' }}>
          {/* Backtest status bar — shows last run result + quick re-run */}
          {code.code && backtest.metrics && backtest.metrics.totalTrades != null && (
            <div style={{
              display: 'flex', alignItems: 'center', gap: 16, padding: '6px 16px', flexShrink: 0,
              background: 'linear-gradient(90deg, #f6ffed, #f0f5ff)', borderBottom: '1px solid #d9f7be',
              fontSize: 12,
            }}>
              <span style={{ fontWeight: 600 }}>📊 {t('strategy.workspace.lastBacktest', { defaultValue: 'Last Backtest' })}</span>
              <span style={{ color: (backtest.metrics.totalReturn ?? 0) >= 0 ? '#3fb950' : '#f85149', fontWeight: 700 }}>
                {backtest.metrics.totalReturn != null ? `${backtest.metrics.totalReturn.toFixed(1)}%` : '—'}
              </span>
              <span style={{ color: 'var(--ant-color-text-secondary)' }}>
                {backtest.metrics.totalTrades} {t('strategy.workspace.tradesLabel', { defaultValue: 'trades' })}
                {backtest.metrics.winRate != null ? ` · ${backtest.metrics.winRate.toFixed(0)}% win` : ''}
                {backtest.metrics.maxDrawdown != null ? ` · DD ${backtest.metrics.maxDrawdown.toFixed(1)}%` : ''}
              </span>
              <div style={{ flex: 1 }} />
              <Button size="small" icon={<PlayCircleOutlined />} onClick={() => setBtModalOpen(true)}
                style={{ background: '#3fb950', borderColor: '#3fb950', color: '#fff' }}>
                {t('strategy.workspace.reRun', { defaultValue: 'Re-run' })}
              </Button>
            </div>
          )}
          {/* Code editor — or import mode — or empty state guidance */}
          {importMode ? (
            <div style={{ flex: 1, overflow: 'auto', padding: '12px 16px' }}>
              <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
                <Button size="small" type="text" onClick={() => setImportMode(false)}>← {t('strategy.workspace.backToEditor', { defaultValue: 'Back' })}</Button>
              </div>
              <ImportEAPanel
                onApplyCode={(c) => { code.setCode(c); setCenterTab('code'); setImportMode(false); }}
                onStrategyIdChange={(id) => { if (id) code.setStrategyId(id); }}
              />
            </div>
          ) : code.code ? (
            <StrategyCodeEditor
              value={code.code}
              onChange={code.setCode}
              style={{ flex: 1, borderRadius: 0, border: 'none', minHeight: 0 }}
            />
          ) : (
            <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <div style={{ textAlign: 'center', maxWidth: 420, padding: 40 }}>
                <div style={{ fontSize: 48, marginBottom: 16 }}>📝</div>
                <div style={{ fontSize: 16, fontWeight: 600, marginBottom: 8, color: 'var(--ant-color-text)' }}>
                  {t('strategy.workspace.emptyTitle', { defaultValue: 'Start building your strategy' })}
                </div>
                <div style={{ fontSize: 13, color: 'var(--ant-color-text-secondary)', marginBottom: 24, lineHeight: 1.6 }}>
                  {t('strategy.workspace.emptyDesc', { defaultValue: 'Import an existing MQL EA, pick a template, or let AI generate one for you. All backtesting and deployment happens right here.' })}
                </div>
                <div style={{ display: 'flex', gap: 10, justifyContent: 'center', flexWrap: 'wrap' }}>
                  <Button type="primary" icon={<ImportOutlined />} onClick={() => setImportMode(true)}>
                    {t('strategy.workspace.importMql', { defaultValue: 'Import MQL EA' })}
                  </Button>
                  <Button icon={<RobotOutlined />} onClick={() => isMobile ? setCenterTab('chat') : setRightPanelTab('ai')}
                    style={{ background: '#722ed1', borderColor: '#722ed1', color: '#fff' }}>
                    {t('strategy.workspace.aiGenerate', { defaultValue: 'AI Generate' })}
                  </Button>
                  <Button icon={<HistoryOutlined />} onClick={() => templates.onSelect(templates.list[0]?.id || '')}
                    disabled={!templates.list.length}>
                    {t('strategy.workspace.useTemplate', { defaultValue: 'Use Template' })}
                  </Button>
                </div>
              </div>
            </div>
          )}
        </div>
        {/* Right panel: AI / Backtest tabs (desktop only) */}
        {!isMobile && rightPanelTab && (
          <WorkspaceAIPanel
            activeTab={rightPanelTab}
            onTabChange={setRightPanelTab}
            onClose={() => setRightPanelTab(null)}
            btSummary={btSummary}
            recentSummaries={recentSummaries}
            backtestStatus={backtest.status}
            backtestMetrics={backtest.metrics}
            onRunBacktest={() => setBtModalOpen(true)}
            onOpenAdvanced={() => setBtDrawerOpen(true)}
          />
        )}
          </div>
        </div>
      </div>

      {/* Backtest drawer (replaces backtest tab) */}
      <Modal
        title={t(WS_BACKTEST_KEY)}
        open={btDrawerOpen}
        onCancel={() => setBtDrawerOpen(false)}
        footer={null}
        width="95vw"
        style={{ top: 20 }}
        destroyOnClose
      >
        <div style={{ height: 'calc(95vh - 120px)', overflow: 'auto' }}>
          <BacktestPanel
            runner={backtest.runner}
            inputs={{
              strategyCode: code.code,
              accountId: account.accountId,
              symbol: account.symbol,
              timeframe: account.timeframe,
              templateId: templates.selectedId || undefined,
              strategyId: code.strategyId,
            }}
            onOpenHistory={(templateId?: string) => history.open(templateId)}
            onAIOptimize={() => ai.optimize()}
            code={code.code}
            onApplyTunedParams={code.setCode}
          />
        </div>
      </Modal>

      {/* Mobile sidebar drawer */}
      {isMobile && (
        <Drawer
          open={sidebarDrawerOpen}
          onClose={() => setSidebarDrawerOpen(false)}
          placement="left"
          width={280}
          styles={{ body: { padding: 0 } }}
        >
          <WorkspaceSidebar
            templates={templates.list}
            loading={templates.loading}
            selectedId={templates.selectedId || ''}
            onSelect={(id) => { templates.onSelect(id); setSidebarDrawerOpen(false); }}
            backtestRuns={(history.runs as Array<{ id: string; startedAt?: string; totalReturn?: number; totalTrades?: number; templateName?: string; templateId?: string }>) || []}
            runsLoading={history.loading}
            onOpenHistory={(tid) => { history.open(tid); setSidebarDrawerOpen(false); }}
            onImport={() => { setImportMode(true); setSidebarDrawerOpen(false); }}
            onNew={() => { templates.onSelect(''); setSidebarDrawerOpen(false); }}
            collapsed={false}
            onToggle={() => setSidebarDrawerOpen(false)}
          />
        </Drawer>
      )}

      {/* Bottom panel: Positions | History | Backtest  +  Quick Trade on the right (desktop only) */}
      <BottomPanelSection
        isMobile={!!isMobile}
        collapsed={layout.bottomPanelCollapsed}
        onToggleCollapsed={() => layout.setBottomPanelCollapsed(!layout.bottomPanelCollapsed)}
        positions={quickTrade.allPositions}
        recentTrades={quickTrade.qtRecentTrades}
        onClosePosition={quickTrade.handleClosePosition}
        panelHeight={layout.bottomPanelUserResized ? layout.bottomPanelHeight : undefined}
        onResizeStart={handleBpResize}
        dragging={bpDragging}
        accountId={account.accountId}
        symbol={account.symbol}
        accountMeta={account.selectedAccountMeta}
        qtPositions={quickTrade.qtPositions}
      />
    </div>
  );
}
