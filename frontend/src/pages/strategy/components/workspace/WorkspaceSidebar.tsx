import { useCallback, useState } from 'react';
import { Button } from 'antd';
import { PlusOutlined, FileTextOutlined, HistoryOutlined, CaretLeftOutlined, DownOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import SidebarStrategyList from './SidebarStrategyList';
import SidebarRunList from './SidebarRunList';

export type WorkspaceSection = 'new' | 'strategies' | 'history';

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
  onNew: () => void;
  // 导航语义：展开哪个分区，主内容区就切换到对应视图。
  activeSection: WorkspaceSection;
  onSectionChange: (s: WorkspaceSection) => void;
  collapsed: boolean;
  onToggle: () => void;
  width?: number;
  onWidthChange?: (w: number) => void;
}

export default function WorkspaceSidebar({
  templates, loading, selectedId, onSelect, onDeleteTemplate, onRenameTemplate, onBatchDeleteTemplates,
  backtestRuns, runsLoading, onOpenHistory, onDeleteRun, onBatchDeleteRuns, onRenameRun,
  onNew,
  activeSection, onSectionChange,
  collapsed, onToggle,
  width = 240, onWidthChange,
}: Props) {
  const { t } = useTranslation();
  const [sidebarDragging, setSidebarDragging] = useState(false);

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

  const sections = [
    {
      key: 'new' as const,
      icon: <PlusOutlined />,
      label: t('strategy.workspace.sidebar.newStrategy', { defaultValue: 'New Strategy' }),
      count: 0,
      fill: false,
      content: null, // center shows the source-selection panel for this section
    },
    {
      key: 'strategies' as const,
      icon: <FileTextOutlined />,
      label: t('strategy.workspace.sidebar.myStrategies', { defaultValue: 'My Strategies' }),
      count: templates.length,
      fill: activeSection === 'strategies',
      content: (
        <SidebarStrategyList
          templates={templates}
          loading={loading}
          selectedId={selectedId}
          onSelect={onSelect}
          onDeleteTemplate={onDeleteTemplate}
          onRenameTemplate={onRenameTemplate}
          onBatchDeleteTemplates={onBatchDeleteTemplates}
        />
      ),
    },
    {
      key: 'history' as const,
      icon: <HistoryOutlined />,
      label: t('strategy.workspace.sidebar.backtestHistory', { defaultValue: 'Backtest History' }),
      count: backtestRuns.length,
      fill: activeSection === 'history',
      content: (
        <SidebarRunList
          runs={backtestRuns}
          loading={runsLoading}
          onOpenHistory={onOpenHistory}
          onDeleteRun={onDeleteRun}
          onBatchDeleteRuns={onBatchDeleteRuns}
          onRenameRun={onRenameRun}
        />
      ),
    },
  ];

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
          {sections.map(({ key, icon, label, count, fill, content }) => {
            const active = activeSection === key;
            return (
              <div key={key}
                style={{ flex: fill ? '1 1 auto' : '0 0 auto', display: 'flex', flexDirection: 'column', marginBottom: 8, overflow: 'hidden', minHeight: 0 }}>
                <Button
                  size="small"
                  icon={icon}
                  onClick={() => onSectionChange(key)}
                  block
                  type={active ? 'primary' : 'default'}
                  ghost={active}
                  style={{ justifyContent: 'space-between', display: 'flex', alignItems: 'center', flexShrink: 0 }}
                >
                  <span>{label}</span>
                  <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                    {!active && count > 0 && <span style={{ fontSize: 10 }}>{count}</span>}
                    <DownOutlined style={{ fontSize: 10, transform: active ? 'rotate(180deg)' : 'none', transition: 'transform 0.15s' }} />
                  </span>
                </Button>
                {active && content}
              </div>
            );
          })}
        </div>
      )}

      {/* Collapsed rail: single quick action (section navigation lives in the
          expanded sidebar) */}
      {collapsed && (
        <div style={{ padding: '6px 4px', borderTop: '1px solid var(--ant-color-border)', flexShrink: 0 }}>
          <Button size="small" type="text" icon={<PlusOutlined />} onClick={onNew}
            title={t('strategy.workspace.sidebar.newStrategy', { defaultValue: 'New Strategy' })} block />
        </div>
      )}
    </div>
    {/* Resize handle */}
    {!collapsed && onWidthChange && (
      <div
        onMouseDown={handleSidebarResize}
        style={{
          width: 4, cursor: 'col-resize', flexShrink: 0,
          background: sidebarDragging ? 'var(--color-info)' : 'transparent',
          transition: 'background 0.15s',
        }}
      />
    )}
    </>
  );
}
