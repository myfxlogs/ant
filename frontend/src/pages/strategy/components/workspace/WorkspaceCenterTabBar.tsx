import { Button, Tooltip, Modal } from 'antd';
import { PlayCircleOutlined, SaveOutlined, CopyOutlined, QuestionCircleOutlined, RobotOutlined, HistoryOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import {
  SEND_TO_AI_KEY, BROWSE_INDICATORS_KEY,
  CODE_KEY, SAVE_KEY, COPY_KEY, RUN_BACKTEST_KEY,
  BACKTEST_KEY as WS_BACKTEST_KEY, AI_ASSISTANT_KEY,
} from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { COMMON_UNSAVED_KEY, COMMON_SAVED_KEY, COMMON_SAVE_KEY } from '@/gen/ant/v1/i18n/base_keys';
import { type CenterTab } from '@/stores/workspaceStore';
import type { WsCode, WsAccount, WsTemplates } from '../../WorkspaceContext';

interface Props {
  isMobile?: boolean;
  centerTab: CenterTab;
  setCenterTab: (tab: CenterTab) => void;
  setSidebarDrawerOpen: (v: boolean) => void;
  setBtModalOpen: (v: boolean) => void;
  setIndicatorDrawerOpen: (v: boolean) => void;
  onShowVersionHistory?: () => void;
  rightPanelTab: 'ai' | 'backtest' | null;
  setRightPanelTab: (tab: 'ai' | 'backtest' | null) => void;
  code: WsCode;
  account: WsAccount;
  templates: WsTemplates;
}

export default function WorkspaceCenterTabBar({
  isMobile = false,
  centerTab,
  setCenterTab,
  setSidebarDrawerOpen,
  setBtModalOpen,
  setIndicatorDrawerOpen,
  onShowVersionHistory,
  rightPanelTab,
  setRightPanelTab,
  code,
  account,
  templates,
}: Props) {
  const { t } = useTranslation();

  const CTABS: { key: CenterTab; icon: string; label: string }[] = isMobile
    ? [
        { key: 'chat', icon: '🤖', label: t(AI_ASSISTANT_KEY) },
        { key: 'code', icon: '📝', label: t(CODE_KEY) },
      ]
    : [
        { key: 'code', icon: '📝', label: t(CODE_KEY) },
      ];

  const strategyName = templates.list.find((t2: { id: string; name?: string }) => t2.id === templates.selectedId)?.name || code.loadedTemplate?.name || '';
  const saveStatus: 'modified' | 'saved' | 'none' = code.code && code.lastValidatedCode && code.code !== code.lastValidatedCode ? 'modified' : code.lastSavedId ? 'saved' : 'none';

  const handleBacktestClick = () => {
    if (!account.accountId) {
      Modal.warning({ title: t('strategy.workspace.selectAccountFirst', { defaultValue: 'Please select a trading account first' }) });
      return;
    }
    if (!account.symbol) {
      Modal.warning({ title: t('strategy.workspace.selectSymbolFirst', { defaultValue: 'Please select a trading symbol first' }) });
      return;
    }
    setBtModalOpen(true);
  };

  const handleCopy = () => {
    if (!code.code) return;
    navigator.clipboard?.writeText(code.code).catch(() => {});
  };

  return (
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
            <Button size="small" icon={<SaveOutlined />} data-tour="save"
              disabled={!code.code}
              onClick={() => code.setSaveModalOpen(true)}
              style={{ background: '#58a6ff', borderColor: '#58a6ff', color: '#fff' }}>
              {t(COMMON_SAVE_KEY)}
            </Button>
          </Tooltip>
          <Tooltip title={t(RUN_BACKTEST_KEY)}>
            <Button size="small" type="primary" icon={<PlayCircleOutlined />} data-tour="backtest"
              onClick={handleBacktestClick}
              style={{ background: '#3fb950', borderColor: '#3fb950' }}>
              {t(WS_BACKTEST_KEY)}
            </Button>
          </Tooltip>
          <Tooltip title={t(COPY_KEY)}>
            <Button size="small" icon={<CopyOutlined />} onClick={handleCopy} />
          </Tooltip>
          <Tooltip title={t(SEND_TO_AI_KEY)}>
            <Button size="small" icon={<RobotOutlined />} data-tour="ai-assistant"
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
  );
}
