import { useCallback, useState } from 'react';
import { Button } from 'antd';
import { PlusOutlined, ImportOutlined, FileTextOutlined, HistoryOutlined, CaretLeftOutlined, DownOutlined, RobotOutlined, EditOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import SidebarStrategyList from './SidebarStrategyList';
import SidebarRunList from './SidebarRunList';
import { IMPORT_MQL_KEY, AI_GENERATE_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';

export type WorkspaceSection = 'new' | 'strategies' | 'history';
export type NewSource = 'ai' | 'manual' | 'import' | 'template';

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
  onNewSource: (source: NewSource) => void;
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
  onNew, onNewSource,
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
      content: (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          {([
            { key: 'ai' as const, icon: <RobotOutlined />, label: t(AI_GENERATE_KEY, { defaultValue: 'AI 生成' }) },
            { key: 'manual' as const, icon: <EditOutlined />, label: t('strategy.workspace.new.manual', { defaultValue: '手动编写' }) },
            { key: 'import' as const, icon: <ImportOutlined />, label: t(IMPORT_MQL_KEY, { defaultValue: '导入 MQL' }) },
          ]).map(({ key, icon, label }) => (
            <button key={key} type="button" className="sidebar-item"
              style={{ padding: '6px 10px', borderRadius: 6, cursor: 'pointer', fontSize: 12, display: 'flex', alignItems: 'center', gap: 6, border: '1px solid transparent', background: 'transparent', textAlign: 'left' }}
              onClick={() => onNewSource(key)}>
              {icon}
              <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{label}</span>
            </button>
          ))}
        </div>
      ),
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
