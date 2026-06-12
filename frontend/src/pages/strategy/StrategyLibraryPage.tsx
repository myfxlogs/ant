import { lazy, Suspense, useCallback } from 'react';
import { Form, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { useStrategyLibrary } from './hooks/useStrategyLibrary';
import { LibraryProvider } from './LibraryContext';
import LibraryLeftPanel from './components/library/LibraryLeftPanel';
import LibraryRightPanel from './components/library/LibraryRightPanel';
import WorkspaceErrorBoundary from './components/workspace/WorkspaceErrorBoundary';
import type { StrategyTemplate } from '@/client/strategy';

const StrategyTemplateEditModal = lazy(() => import('./StrategyTemplateEditModal').then(m => ({ default: m.StrategyTemplateEditModal })));
const BacktestRunDrawer = lazy(() => import('@/components/strategy/BacktestRunDrawer'));

function LibraryUI() {
  const { t } = useTranslation();
  const lib = useStrategyLibrary();
  const { templates: tCtx, schedules: sCtx, runs: rCtx } = lib;
  const [editForm] = Form.useForm();

  const scheduleCountByTemplate = useCallback((templateId: string): number =>
    sCtx.allSchedules.filter(s => String(s.templateId || '') === templateId && s.isActive).length,
  [sCtx.allSchedules]);

  const ctxValue = {
    templates: tCtx.filtered, allTemplates: tCtx.allTemplates,
    templatesLoading: tCtx.loading, templatesError: tCtx.error,
    selectedId: tCtx.selectedId, selected: tCtx.selected,
    selectTemplate: lib.selectTemplate,
    filter: tCtx.filter, setFilter: tCtx.setFilter,
    search: tCtx.search, setSearch: tCtx.setSearch,
    publishing: tCtx.publishing,
    scheduleCountByTemplate,
    openCreate: tCtx.openCreate, openEdit: (tpl: StrategyTemplate) => { tCtx.openEdit(tpl); editForm.setFieldsValue({ name: tpl.name, description: tpl.description, code: (tpl as any).code, isPublic: (tpl as any).isPublic }); },
    handleDelete: tCtx.handleDelete, handlePublish: tCtx.handlePublish, handleUnpublish: tCtx.handleUnpublish,
    editOpen: tCtx.editOpen, setEditOpen: tCtx.setEditOpen,
    editing: tCtx.editing, setEditing: tCtx.setEditing,
    codeValidating: tCtx.codeValidating, lastValidatedCode: tCtx.lastValidatedCode, setLastValidatedCode: tCtx.setLastValidatedCode,
    handleSave: async (values: Record<string, unknown>) => { await tCtx.handleSave(values); editForm.resetFields(); },
    codeViewOpen: lib.codeViewOpen, setCodeViewOpen: lib.setCodeViewOpen,
    viewingCode: lib.viewingCode, setViewingCode: lib.setViewingCode,
    activeTab: lib.activeTab, setActiveTab: lib.setActiveTab,
    scheduleProps: {
      schedules: sCtx.filteredSchedules, allSchedules: sCtx.allSchedules,
      loading: sCtx.loading, error: sCtx.error,
      templates: sCtx.templates, accounts: sCtx.accounts,
      symbols: sCtx.symbols, symbolsLoading: sCtx.symbolsLoading,
      formatTime: sCtx.formatTime,
      openEdit: sCtx.openEdit, setOpenEdit: sCtx.setOpenEdit,
      editing: sCtx.editing, setEditing: sCtx.setEditing,
      form: sCtx.form, accountIdWatch: sCtx.accountIdWatch,
      loadSymbols: sCtx.loadSymbols, submitEdit: sCtx.submitEdit,
      openUpdate: sCtx.openUpdate,
      onToggleActive: sCtx.onToggleActive, onDelete: sCtx.onDelete,
      onManualTrigger: sCtx.onManualTrigger, loadScheduleHealth: sCtx.loadScheduleHealth,
      healthOpen: sCtx.healthOpen, setHealthOpen: sCtx.setHealthOpen,
      healthLoading: sCtx.healthLoading, healthTarget: sCtx.healthTarget, setHealthTarget: sCtx.setHealthTarget,
      healthSummary: sCtx.healthSummary, setHealthSummary: sCtx.setHealthSummary,
      triggering: sCtx.triggering, openTrigger: sCtx.openTrigger, setOpenTrigger: sCtx.setOpenTrigger,
      triggerResult: sCtx.triggerResult, triggerContext: sCtx.triggerContext,
      setTriggerContext: sCtx.setTriggerContext, setTriggerResult: sCtx.setTriggerResult,
      doOrderSend: sCtx.doOrderSend, openCreate: sCtx.openCreate,
    },
    backtestProps: {
      runs: rCtx.runs, loading: rCtx.loading, error: rCtx.error,
      page: rCtx.page, pageSize: rCtx.pageSize, total: rCtx.total,
      deleting: rCtx.deleting, drawerOpen: rCtx.drawerOpen, selectedRunId: rCtx.selectedRunId,
      onPageChange: rCtx.onPageChange, onViewRun: rCtx.onViewRun,
      onDeleteRun: rCtx.onDeleteRun, onBatchDelete: rCtx.onBatchDelete,
      onRefresh: () => rCtx.fetchRuns(rCtx.page, rCtx.pageSize),
      setDrawerOpen: rCtx.setDrawerOpen,
    },
  };

  return (
    <LibraryProvider value={ctxValue}>
      <div style={{ display: 'flex', height: 'calc(100vh - 112px)', background: '#fff' }}>
        <WorkspaceErrorBoundary fallback={<div style={{ width: 340, padding: 20, color: '#8c8c8c' }}>{t('common.loadingFailed')}</div>}>
          <LibraryLeftPanel />
        </WorkspaceErrorBoundary>
        <WorkspaceErrorBoundary fallback={<div style={{ flex: 1, padding: 20, color: '#8c8c8c', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>{t('common.loadingFailed')}</div>}>
          <LibraryRightPanel />
        </WorkspaceErrorBoundary>
      </div>

      <Suspense fallback={null}>
        {tCtx.editOpen && (
          <StrategyTemplateEditModal open={tCtx.editOpen} editingTemplate={tCtx.editing} form={editForm}
            codeValidating={tCtx.codeValidating} lastValidatedCode={tCtx.lastValidatedCode}
            onCancel={() => { tCtx.setEditOpen(false); editForm.resetFields(); }}
            onValidate={() => {}}
            onSubmit={ctxValue.handleSave} />
        )}
      </Suspense>

      {lib.codeViewOpen && (
        <CodeModal open={lib.codeViewOpen} code={lib.viewingCode}
          onClose={() => { lib.setCodeViewOpen(false); lib.setViewingCode(''); }}
          onCopy={async (c: string) => {
            const { copyToClipboard } = await import('@/utils/clipboard');
            const ok = await copyToClipboard(c);
            if (ok) message.success(t('strategy.templates.messages.codeCopied'));
            else message.error(t('strategy.templates.messages.copyFailed'));
          }} />
      )}

      <Suspense fallback={null}>
        {rCtx.drawerOpen && rCtx.selectedRunId && (
          <BacktestRunDrawer open={rCtx.drawerOpen} runId={rCtx.selectedRunId}
            onClose={() => rCtx.setDrawerOpen(false)} onCancel={() => rCtx.setDrawerOpen(false)} />
        )}
      </Suspense>
    </LibraryProvider>
  );
}

function CodeModal({ open, code, onClose, onCopy }: { open: boolean; code: string; onClose: () => void; onCopy: (c: string) => void }) {
  const { t } = useTranslation();
  if (!open) return null;
  return (
    <div style={{ position: 'fixed', inset: 0, zIndex: 2000, background: 'rgba(0,0,0,0.45)', display: 'flex', alignItems: 'center', justifyContent: 'center' }} onClick={onClose}>
      <div style={{ background: '#fff', borderRadius: 8, width: 800, maxHeight: '80vh', display: 'flex', flexDirection: 'column', overflow: 'hidden' }} onClick={e => e.stopPropagation()}>
        <div style={{ padding: '16px 20px', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span style={{ fontWeight: 600, fontSize: 15 }}>{t('strategy.library.viewCode')}</span>
          <button onClick={onClose} style={{ border: 'none', background: 'transparent', cursor: 'pointer', fontSize: 18, color: '#8c8c8c' }}>✕</button>
        </div>
        <pre style={{ flex: 1, overflow: 'auto', padding: 16, margin: 0, background: '#1e1e1e', color: '#d4d4d4', fontSize: 13, lineHeight: 1.5 }}>{code}</pre>
        <div style={{ padding: '10px 20px', borderTop: '1px solid #f0f0f0', textAlign: 'right' }}>
          <button onClick={() => onCopy(code)} style={{ padding: '4px 16px', border: '1px solid #d9d9d9', borderRadius: 4, background: '#fff', cursor: 'pointer', fontSize: 13 }}>{t('strategy.templates.actions.copy')}</button>
        </div>
      </div>
    </div>
  );
}

export default function StrategyLibraryPage() {
  return <LibraryUI />;
}
