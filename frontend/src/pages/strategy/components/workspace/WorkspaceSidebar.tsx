import { useState, useCallback } from 'react';
import { Button } from 'antd';
import { PlusOutlined, ImportOutlined, FileTextOutlined, HistoryOutlined, CaretLeftOutlined, DownOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import SidebarStrategyList from './SidebarStrategyList';
import SidebarRunList from './SidebarRunList';

interface StrategyItem {
  id: string;
  name: string;
}

interface BacktestRun {
  id: string;
  startedAt?: string;
  totalReturn?: number;
  totalTrades?: number;
  templateName?: string;
  templateId?: string;
  name?: string;
}

interface Props {
  templates: StrategyItem[];
  loading: boolean;
  selectedId: string;
  onSelect: (id: string) => void;
  onDeleteTemplate?: (id: string) => void;
  onRenameTemplate?: (id: string, name: string) => void;
  onBatchDeleteTemplates?: (ids: string[]) => void;
  backtestRuns: BacktestRun[];
  runsLoading: boolean;
  onOpenHistory: (runId?: string) => void;
  onDeleteRun?: (runId: string) => void;
  onBatchDeleteRuns?: (runIds: string[]) => void;
  onRenameRun?: (runId: string, name: string) => void;
  onImport: () => void;
  onNew: () => void;
  collapsed: boolean;
  onToggle: () => void;
  autoExpandHistory?: boolean;
  width?: number;
  onWidthChange?: (w: number) => void;
}

export default function WorkspaceSidebar({
  templates, loading, selectedId, onSelect, onDeleteTemplate, onRenameTemplate, onBatchDeleteTemplates,
  backtestRuns, runsLoading, onOpenHistory, onDeleteRun, onBatchDeleteRuns, onRenameRun,
  onImport, onNew,
  collapsed, onToggle,
  autoExpandHistory,
  width = 240, onWidthChange,
}: Props) {
  const { t } = useTranslation();
  const [strategiesExpanded, setStrategiesExpanded] = useState(true);
  const [historyExpanded, setHistoryExpanded] = useState(false);
  const [sidebarDragging, setSidebarDragging] = useState(false);

  // Auto-expand history section when triggered by external event (e.g. backtest completion)
  const [prevAutoExpand, setPrevAutoExpand] = useState(false);
  if (autoExpandHistory && !prevAutoExpand) {
    setPrevAutoExpand(true);
    setStrategiesExpanded(false);
    setHistoryExpanded(true);
  } else if (!autoExpandHistory && prevAutoExpand) {
    setPrevAutoExpand(false);
  }

  // Toggle: clicking expanded section collapses it and expands the other
  const toggleStrategies = useCallback(() => {
    setStrategiesExpanded(prev => {
      if (prev) {
        setHistoryExpanded(true);
        return false;
      }
      setHistoryExpanded(false);
      return true;
    });
  }, []);
  const toggleHistory = useCallback(() => {
    setHistoryExpanded(prev => {
      if (prev) {
        setStrategiesExpanded(true);
        return false;
      }
      setStrategiesExpanded(false);
      return true;
    });
  }, []);

  const handleSidebarResize = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    if (!onWidthChange) return;
    setSidebarDragging(true);
    const startX = e.clientX;
    const startW = width;
    const onMove = (ev: MouseEvent) => {
      const delta = ev.clientX - startX;
      onWidthChange(Math.max(180, Math.min(480, startW + delta)));
    };
    const onUp = () => {
      setSidebarDragging(false);
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
    };
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
  }, [width, onWidthChange]);

  return (
    <>
    <div style={{
      width: collapsed ? 36 : width, flexShrink: 0, overflow: 'hidden',
      borderRight: '1px solid var(--ant-color-border)',
      background: 'var(--ant-color-bg-container)',
      display: 'flex', flexDirection: 'column',
      transition: sidebarDragging ? 'none' : 'width 0.2s',
      userSelect: sidebarDragging ? 'none' : 'auto',
    }}>
      {/* Header */}
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: collapsed ? '8px 6px' : '8px 12px',
        borderBottom: '1px solid var(--ant-color-border)', flexShrink: 0,
      }}>
        {!collapsed && (
          <span style={{ fontSize: 12, fontWeight: 600 }}>
            {t('strategy.workspace.sidebar.title', { defaultValue: 'Workspace' })}
          </span>
        )}
        <Button size="small" type="text" icon={<CaretLeftOutlined style={{ transform: collapsed ? 'rotate(180deg)' : undefined }} />}
          onClick={onToggle} style={{ padding: '0 4px' }} />
      </div>

      {!collapsed && (
        <div style={{ flex: '1 1 0', overflow: 'hidden', display: 'flex', flexDirection: 'column', padding: '8px 10px' }}>
          {/* My Strategies — expands to fill, collapses to button only */}
          <div style={{ flex: strategiesExpanded ? '1 1 auto' : '0 0 auto', display: 'flex', flexDirection: 'column', marginBottom: 8, overflow: 'hidden', minHeight: 0 }}>
            <Button
              size="small"
              icon={<FileTextOutlined />}
              onClick={toggleStrategies}
              block
              style={{ justifyContent: 'space-between', display: 'flex', alignItems: 'center', flexShrink: 0 }}
            >
              <span>{t('strategy.workspace.sidebar.myStrategies', { defaultValue: 'My Strategies' })}</span>
              <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                {!strategiesExpanded && templates.length > 0 && <span style={{ fontSize: 10 }}>{templates.length}</span>}
                <DownOutlined style={{ fontSize: 10, transform: strategiesExpanded ? 'rotate(180deg)' : 'none', transition: 'transform 0.15s' }} />
              </span>
            </Button>
            {strategiesExpanded && (
              <SidebarStrategyList
                templates={templates}
                loading={loading}
                selectedId={selectedId}
                onSelect={onSelect}
                onDeleteTemplate={onDeleteTemplate}
                onRenameTemplate={onRenameTemplate}
                onBatchDeleteTemplates={onBatchDeleteTemplates}
              />
            )}
          </div>

          {/* Backtest History — expands to fill, collapses to button only */}
          <div style={{ flex: historyExpanded ? '1 1 auto' : '0 0 auto', display: 'flex', flexDirection: 'column', overflow: 'hidden', minHeight: 0 }}>
            <Button
              size="small"
              icon={<HistoryOutlined />}
              onClick={toggleHistory}
              block
              style={{ justifyContent: 'space-between', display: 'flex', alignItems: 'center', flexShrink: 0 }}
            >
              <span>{t('strategy.workspace.sidebar.backtestHistory', { defaultValue: 'Backtest History' })}</span>
              <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                {!historyExpanded && backtestRuns.length > 0 && <span style={{ fontSize: 10 }}>{backtestRuns.length}</span>}
                <DownOutlined style={{ fontSize: 10, transform: historyExpanded ? 'rotate(180deg)' : 'none', transition: 'transform 0.15s' }} />
              </span>
            </Button>
            {historyExpanded && (
              <SidebarRunList
                runs={backtestRuns}
                loading={runsLoading}
                onOpenHistory={onOpenHistory}
                onDeleteRun={onDeleteRun}
                onBatchDeleteRuns={onBatchDeleteRuns}
                onRenameRun={onRenameRun}
              />
            )}
          </div>
        </div>
      )}

      {/* Action buttons */}
      <div style={{
        padding: collapsed ? '6px 4px' : '8px 10px',
        borderTop: '1px solid var(--ant-color-border)', flexShrink: 0,
        display: 'flex', flexDirection: 'column', gap: 4,
      }}>
        {collapsed ? (
          <>
            <Button size="small" type="text" icon={<PlusOutlined />} onClick={onNew}
              title={t('strategy.workspace.sidebar.newStrategy', { defaultValue: 'New Strategy' })} />
            <Button size="small" type="text" icon={<ImportOutlined />} onClick={onImport}
              title={t('strategy.workspace.importMql', { defaultValue: 'Import MQL' })} />
          </>
        ) : (
          <>
            <Button size="small" icon={<PlusOutlined />} onClick={onNew} block>
              {t('strategy.workspace.sidebar.newStrategy', { defaultValue: 'New Strategy' })}
            </Button>
            <Button size="small" icon={<ImportOutlined />} onClick={onImport} block>
              {t('strategy.workspace.importMql', { defaultValue: 'Import MQL' })}
            </Button>
          </>
        )}
      </div>
    </div>
    {/* Resize handle */}
    {!collapsed && onWidthChange && (
      <div
        onMouseDown={handleSidebarResize}
        style={{
          width: 4, cursor: 'col-resize', flexShrink: 0,
          background: sidebarDragging ? '#58a6ff' : 'transparent',
          transition: 'background 0.15s',
        }}
      />
    )}
    </>
  );
}
