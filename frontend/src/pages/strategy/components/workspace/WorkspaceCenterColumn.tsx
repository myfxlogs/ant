import { useCallback, useState } from 'react';
import { Button, Tooltip } from 'antd';
import { PlayCircleOutlined, SaveOutlined, CopyOutlined, QuestionCircleOutlined, RobotOutlined, HistoryOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import {
  SEND_TO_AI_KEY, BROWSE_INDICATORS_KEY,
  CODE_KEY, SAVE_KEY, COPY_KEY, RUN_BACKTEST_KEY,
  BACKTEST_KEY as WS_BACKTEST_KEY, AI_ASSISTANT_KEY,
} from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { BACKTEST_KEY as GEN_BACKTEST_KEY } from '@/gen/ant/v1/i18n/strategy_gen_keys';
import { COMMON_UNSAVED_KEY, COMMON_SAVED_KEY, COMMON_SAVE_KEY } from '@/gen/ant/v1/i18n/base_keys';
import { useWorkspaceStore, type CenterTab } from '@/stores/workspaceStore';
import BacktestPanel from '@/components/backtest/BacktestPanel';
import ChartBottomPanel from '@/components/chart/ChartBottomPanel';
import StrategyCodeEditor from '@/components/strategy/StrategyCodeEditor';
import StrategyChat from '@/components/strategy/StrategyChat';
import QuickTradeSidePanel from './QuickTradeSidePanel';
import ImportEAPanel from '../editor/ImportEAPanel';
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

  const strategyName = templates.list.find((t2: { id: string; name?: string }) => t2.id === templates.selectedId)?.name || code.loadedTemplate?.name || '';
  const saveStatus: 'modified' | 'saved' | 'none' = code.code && code.lastValidatedCode && code.code !== code.lastValidatedCode ? 'modified' : code.lastSavedId ? 'saved' : 'none';

  const CTABS: { key: CenterTab; icon: string; label: string }[] = [
    { key: 'chat', icon: '🤖', label: t(AI_ASSISTANT_KEY) },
    { key: 'code', icon: '�', label: t(CODE_KEY) },
    { key: 'import', icon: '📥', label: t('strategy.workspace.importMql') },
    { key: 'backtest', icon: '📊', label: t(GEN_BACKTEST_KEY) },
  ];

  const handleCopy = () => {
    if (!code.code) return;
    navigator.clipboard?.writeText(code.code).catch(() => {});
  };

  return (
    <div data-tour="code-editor" style={{ flex: '1 1 0', minWidth: 0, position: 'relative', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
      {/* Center tab bar */}
      <div style={{
        display: 'flex', alignItems: 'center', flexShrink: 0, height: 34,
        borderBottom: '1px solid var(--ant-color-border)',
        background: 'var(--ant-color-bg-container)',
      }}>
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
              <Button size="small" icon={<RobotOutlined />} onClick={() => setCenterTab('chat')}
                style={{ background: '#722ed1', borderColor: '#722ed1', color: '#fff' }}>
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

      {/* AI Chat — full width center tab for both desktop and mobile */}
      <div style={{ flex: '1 1 0', minHeight: 0, display: centerTab === 'chat' ? 'flex' : 'none', flexDirection: 'column' }}>
        <StrategyChat
          symbol={account.symbol}
          timeframe={account.timeframe}
          accountId={account.accountId}
          onApplyCode={c => { code.setCode(c); setCenterTab('code'); }}
          currentCode={code.code}
        />
      </div>

      {/* Code */}
      <div style={{ flex: '1 1 0', minHeight: 0, display: centerTab === 'code' ? 'flex' : 'none', flexDirection: 'column' }}>
        <StrategyCodeEditor
          value={code.code}
          onChange={code.setCode}
          style={{ flex: 1, borderRadius: 0, border: 'none', minHeight: 0 }}
        />
      </div>

      {/* Import MQL */}
      <div style={{ flex: '1 1 0', minHeight: 0, display: centerTab === 'import' ? 'flex' : 'none', flexDirection: 'column', overflow: 'auto' }}>
        <ImportEAPanel
          onApplyCode={(c) => { code.setCode(c); setCenterTab('code'); }}
          onStrategyIdChange={(id) => { if (id) code.setStrategyId(id); }}
        />
      </div>

      {/* Backtest */}
      <div style={{ flex: '1 1 0', minHeight: 0, display: centerTab === 'backtest' ? 'flex' : 'none', flexDirection: 'column' }}>
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
          templates={{
            list: templates.list,
            loading: templates.loading,
            selectedId: templates.selectedId,
            onSelect: templates.onSelect,
          }}
          onOpenHistory={(templateId?: string) => history.open(templateId)}
          onAIOptimize={() => ai.optimize()}
          code={code.code}
          onApplyTunedParams={code.setCode}
          onRunBacktest={() => setBtModalOpen(true)}
          onSaveAs={() => code.setSaveModalOpen(true)}
          hasUnsavedDraft={saveStatus === 'modified'}
          draftName={strategyName}
        />
      </div>

      {/* Bottom panel: Positions | History | Backtest  +  Quick Trade on the right (desktop only) */}
      {!isMobile && (
        layout.bottomPanelCollapsed ? (
          <ChartBottomPanel
            positions={quickTrade.allPositions}
            recentTrades={quickTrade.qtRecentTrades}
            onClosePosition={quickTrade.handleClosePosition}
            collapsed={true}
            onToggleCollapsed={() => layout.setBottomPanelCollapsed(false)}
            backtestMetrics={backtest.metrics}
            backtestStatus={backtest.status}
          />
        ) : (
          <div style={{ flexShrink: 0, display: 'flex', borderTop: '1px solid var(--ant-color-border)' }}>
            <div style={{ flex: '1 1 0', minWidth: 0 }}>
              <ChartBottomPanel
                positions={quickTrade.allPositions}
                recentTrades={quickTrade.qtRecentTrades}
                onClosePosition={quickTrade.handleClosePosition}
                collapsed={false}
                onToggleCollapsed={() => layout.setBottomPanelCollapsed(true)}
                backtestMetrics={backtest.metrics}
                backtestStatus={backtest.status}
                panelHeight={layout.bottomPanelUserResized ? layout.bottomPanelHeight : undefined}
                onResizeStart={handleBpResize}
                dragging={bpDragging}
              />
            </div>
            {account.symbol && (
              <QuickTradeSidePanel
                accountId={account.accountId}
                symbol={account.symbol}
                accountMeta={account.selectedAccountMeta}
                allPositions={quickTrade.allPositions}
                positions={quickTrade.qtPositions}
                recentTrades={quickTrade.qtRecentTrades}
                onClosePosition={quickTrade.handleClosePosition}
              />
            )}
          </div>
        )
      )}
    </div>
  );
}
