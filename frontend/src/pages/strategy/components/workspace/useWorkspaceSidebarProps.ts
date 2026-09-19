// useWorkspaceSidebarProps — sidebarProps useMemo + 新建策略/来源选择回调抽取
// （LOWPRI-SWEEP-2 CQ-12：自 WorkspaceCenterColumn 移出以控主文件行数）。
import { useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal } from 'antd';
import { useWsCode, useWsTemplates, useWsBacktest, useWsHistory } from '../../../WorkspaceContext';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import { useSidebarActions } from './useSidebarActions';
import { COMMON_CANCEL_KEY, COMMON_CONFIRM_KEY, COMMON_UNSAVED_KEY } from '@/gen/ant/v1/i18n/base_keys';
import { SIDEBAR_NEW_STRATEGY_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import type { WorkspaceSection, NewSource } from './WorkspaceSidebar';

interface Params {
  activeSection: WorkspaceSection;
  onSectionChange: (s: WorkspaceSection) => void;
  onSetCenterView: (v: 'sources' | 'editor' | 'history') => void;
  onSetDock: (d: 'ai' | 'backtest' | null) => void;
  onSetImportMode: (v: boolean) => void;
  onSetCode: (code: string) => void;
}

export function useWorkspaceSidebarProps(p: Params) {
  const { t } = useTranslation();
  const { onSetCenterView, onSetDock, onSetImportMode, onSetCode } = p;
  const setCenterTab = useWorkspaceStore(s => s.setCenterTab);
  const code = useWsCode();
  const templates = useWsTemplates();
  const backtest = useWsBacktest();
  const history = useWsHistory();
  const sidebarActions = useSidebarActions(code, history);

  // 新建策略（带未保存确认）：sidebar 与主区大卡/菜单共用
  const handleNewStrategy = useCallback(() => {
    const hasUnsaved = code.code && code.lastValidatedCode && code.code !== code.lastValidatedCode;
    const doNew = () => {
      templates.onSelect('');
      code.setCode('');
      code.setStrategyId(undefined);
      code.setValidationResult(null);
      code.setLastValidatedCode('');
      code.setLoadedTemplate(null);
      backtest.runner.resetStatus();
      onSetImportMode(false);
      onSetDock(null);
      setCenterTab('code');
    };
    if (hasUnsaved) {
      Modal.confirm({
        title: t(SIDEBAR_NEW_STRATEGY_KEY),
        content: t(COMMON_UNSAVED_KEY),
        okText: t(COMMON_CONFIRM_KEY),
        cancelText: t(COMMON_CANCEL_KEY),
        onOk: doNew,
      });
    } else {
      doNew();
    }
  }, [templates, code, backtest, setCenterTab, onSetDock, onSetImportMode, t]);

  // 来源选择（侧栏菜单项与主区大卡共用）：全部落在编辑器视图，分区保持展开
  const onNewSource = useCallback((source: NewSource) => {
    handleNewStrategy();
    if (source === 'ai') { onSetDock('ai'); return; }
    if (source === 'import') onSetImportMode(true);
    if (source === 'manual') {
      // 最小脚手架（<20 字符不触发审计），让用户直接落进空白编辑器
      onSetCode('# 新策略\n');
    }
    onSetCenterView('editor');
  }, [handleNewStrategy, onSetCode, onSetDock, onSetImportMode, onSetCenterView]);

  const sidebarProps = useMemo(() => ({
    templates: templates.list,
    loading: templates.loading,
    selectedId: templates.selectedId || '',
    onSelect: (id: string) => { templates.onSelect(id); onSetImportMode(false); onSetDock(null); onSetCenterView('editor'); },
    onDeleteTemplate: sidebarActions.onDeleteTemplate,
    onRenameTemplate: sidebarActions.onRenameTemplate,
    onBatchDeleteTemplates: sidebarActions.onBatchDeleteTemplates,
    backtestRuns: ((history.runs || []) as Array<{ id: string; startedAt?: string; totalReturn?: number; totalTrades?: number; templateName?: string; templateId?: string; name?: string }>),
    runsLoading: history.loading,
    onOpenHistory: (runId?: string) => { if (runId) backtest.loadRunById(runId, code.setCode); onSetCenterView('history'); onSetDock('backtest'); },
    onDeleteRun: history.onDeleteRun,
    onBatchDeleteRuns: sidebarActions.onBatchDeleteRuns,
    onRenameRun: sidebarActions.onRenameRun,
    onNew: handleNewStrategy,
    onNewSource,
    activeSection: p.activeSection,
    onSectionChange: p.onSectionChange,
    autoExpandHistory: history.autoExpandHistory,
  }), [templates, sidebarActions, history, handleNewStrategy, onNewSource, p.activeSection, p.onSectionChange, backtest, code.setCode, onSetCenterView, onSetDock, onSetImportMode]);

  return { sidebarProps, handleNewStrategy, onNewSource };
}
