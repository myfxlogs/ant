import { lazy, Suspense, useCallback } from 'react';
import { Form, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { useStrategyLibrary } from './hooks/useStrategyLibrary';
import LibraryLeftPanel from './components/library/LibraryLeftPanel';
import LibraryRightPanel from './components/library/LibraryRightPanel';
import type { StrategyTemplate } from '@/client/strategy';

const StrategyTemplateEditModal = lazy(() => import('./StrategyTemplateEditModal').then(m => ({ default: m.StrategyTemplateEditModal })));
const BacktestRunDrawer = lazy(() => import('@/components/strategy/BacktestRunDrawer'));

export default function StrategyLibraryPage() {
  const { t } = useTranslation();
  const lib = useStrategyLibrary();
  const { templates: tCtx, schedules: sCtx, runs: rCtx } = lib;
  const [editForm] = Form.useForm();

  // ── Schedule count by template ──
  const scheduleCountByTemplate = useCallback((templateId: string): number => {
    return sCtx.allSchedules.filter((s: any) => String(s.templateId || '') === templateId && s.isActive).length;
  }, [sCtx.allSchedules]);

  // ── Edit form handlers ──
  const handleEditOpen = useCallback((tpl: StrategyTemplate) => {
    tCtx.openEdit(tpl);
    editForm.setFieldsValue({
      name: tpl.name, description: tpl.description,
      code: (tpl as any).code, isPublic: (tpl as any).isPublic,
    });
  }, [tCtx.openEdit, editForm]);

  const handleEditSave = useCallback(async (values: Record<string, unknown>) => {
    await tCtx.handleSave(values);
    editForm.resetFields();
  }, [tCtx.handleSave, editForm]);

  // ── Code view ──
  const handleViewCode = useCallback(async (code: string) => {
    lib.setViewingCode(code); lib.setCodeViewOpen(true);
  }, [lib.setViewingCode, lib.setCodeViewOpen]);

  // ── Backtest (opens workspace-style modal) ──
  const handleRunBacktest = useCallback((tpl: StrategyTemplate) => {
    lib.setBacktestModalOpen(true);
  }, [lib.setBacktestModalOpen]);

  // ── Create schedule (opens from overview tab) ──
  const handleOpenCreateSchedule = useCallback(() => {
    sCtx.openCreate();
  }, [sCtx.openCreate]);

  return (
    <div style={{ display: 'flex', height: 'calc(100vh - 112px)', background: '#fff' }}>
      {/* Left Panel */}
      <LibraryLeftPanel
        templates={tCtx.filtered}
        loading={tCtx.loading}
        error={tCtx.error}
        filter={tCtx.filter}
        onFilterChange={tCtx.setFilter}
        search={tCtx.search}
        onSearchChange={tCtx.setSearch}
        selectedId={tCtx.selectedId}
        onSelect={lib.selectTemplate}
        onEdit={handleEditOpen}
        onDelete={tCtx.handleDelete}
        onPublish={tCtx.handlePublish}
        onUnpublish={tCtx.handleUnpublish}
        publishing={tCtx.publishing}
        scheduleCountByTemplate={scheduleCountByTemplate}
        onCreate={tCtx.openCreate}
      />

      {/* Right Panel */}
      <LibraryRightPanel
        selectedTemplate={tCtx.selected}
        activeTab={lib.activeTab}
        onTabChange={lib.setActiveTab}
        scheduleCount={tCtx.selectedId ? scheduleCountByTemplate(tCtx.selectedId) : 0}
        publishing={tCtx.publishing}
        onEdit={handleEditOpen}
        onDelete={tCtx.handleDelete}
        onPublish={tCtx.handlePublish}
        onUnpublish={tCtx.handleUnpublish}
        onViewCode={handleViewCode}
        onRunBacktest={handleRunBacktest}
        onOpenCreateSchedule={handleOpenCreateSchedule}
        scheduleProps={{
          schedules: sCtx.filteredSchedules,
          allSchedules: sCtx.allSchedules,
          loading: sCtx.loading,
          templates: sCtx.templates,
          accounts: sCtx.accounts,
          symbols: sCtx.symbols,
          symbolsLoading: sCtx.symbolsLoading,
          formatTime: sCtx.formatTime,
          openEdit: sCtx.openEdit, setOpenEdit: sCtx.setOpenEdit,
          editing: sCtx.editing, setEditing: sCtx.setEditing,
          form: sCtx.form, accountIdWatch: sCtx.accountIdWatch,
          loadSymbols: sCtx.loadSymbols, submitEdit: sCtx.submitEdit,
          onToggleActive: sCtx.onToggleActive, onDelete: sCtx.onDelete,
          onManualTrigger: sCtx.onManualTrigger, loadScheduleHealth: sCtx.loadScheduleHealth,
          healthOpen: sCtx.healthOpen, setHealthOpen: sCtx.setHealthOpen,
          healthLoading: sCtx.healthLoading, healthTarget: sCtx.healthTarget, setHealthTarget: sCtx.setHealthTarget,
          healthSummary: sCtx.healthSummary, setHealthSummary: sCtx.setHealthSummary,
          triggering: sCtx.triggering, openTrigger: sCtx.openTrigger, setOpenTrigger: sCtx.setOpenTrigger,
          triggerResult: sCtx.triggerResult, triggerContext: sCtx.triggerContext,
          setTriggerContext: sCtx.setTriggerContext, setTriggerResult: sCtx.setTriggerResult,
          openCreate: sCtx.openCreate, openUpdate: sCtx.openUpdate,
        }}
        backtestProps={{
          runs: rCtx.runs, loading: rCtx.loading,
          page: rCtx.page, pageSize: rCtx.pageSize, total: rCtx.total,
          deleting: rCtx.deleting,
          onPageChange: rCtx.onPageChange,
          onViewRun: rCtx.onViewRun,
          onDeleteRun: rCtx.onDeleteRun,
          onBatchDelete: rCtx.onBatchDelete,
          onRefresh: () => rCtx.fetchRuns(rCtx.page, rCtx.pageSize),
        }}
      />

      {/* Modals */}
      <Suspense fallback={null}>
        {tCtx.editOpen && (
          <StrategyTemplateEditModal
            open={tCtx.editOpen}
            editingTemplate={tCtx.editing}
            form={editForm}
            codeValidating={tCtx.codeValidating}
            lastValidatedCode={tCtx.lastValidatedCode}
            onCancel={() => { tCtx.setEditOpen(false); editForm.resetFields(); }}
            onValidate={() => {
              // Validation is handled inside handleSave
            }}
            onSubmit={handleEditSave}
          />
        )}
      </Suspense>

      {/* Code view modal — simple inline pre */}
      {lib.codeViewOpen && (
        <CodeModal
          open={lib.codeViewOpen}
          code={lib.viewingCode}
          onClose={() => { lib.setCodeViewOpen(false); lib.setViewingCode(''); }}
          onCopy={async (c: string) => {
            const { copyToClipboard } = await import('@/utils/clipboard');
            const ok = await copyToClipboard(c);
            if (ok) message.success(t('strategy.templates.messages.codeCopied'));
            else message.error(t('strategy.templates.messages.copyFailed'));
          }}
        />
      )}

      {/* Backtest run drawer */}
      <Suspense fallback={null}>
        {rCtx.drawerOpen && rCtx.selectedRunId && (
          <BacktestRunDrawer
            open={rCtx.drawerOpen}
            runId={rCtx.selectedRunId}
            onClose={() => rCtx.setDrawerOpen(false)}
            onCancel={() => rCtx.setDrawerOpen(false)}
          />
        )}
      </Suspense>
    </div>
  );
}

// Simple code view modal
function CodeModal({ open, code, onClose, onCopy }: { open: boolean; code: string; onClose: () => void; onCopy: (c: string) => void }) {
  const { t } = useTranslation();
  if (!open) return null;
  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 2000, background: 'rgba(0,0,0,0.45)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
    }} onClick={onClose}>
      <div style={{
        background: '#fff', borderRadius: 8, width: 800, maxHeight: '80vh',
        display: 'flex', flexDirection: 'column', overflow: 'hidden',
      }} onClick={e => e.stopPropagation()}>
        <div style={{ padding: '16px 20px', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span style={{ fontWeight: 600, fontSize: 15 }}>{t('strategy.library.viewCode', '查看策略代码')}</span>
          <button onClick={onClose} style={{ border: 'none', background: 'transparent', cursor: 'pointer', fontSize: 18, color: '#8c8c8c' }}>✕</button>
        </div>
        <pre style={{
          flex: 1, overflow: 'auto', padding: 16, margin: 0,
          background: '#1e1e1e', color: '#d4d4d4', fontSize: 13, lineHeight: 1.5,
        }}>
          {code}
        </pre>
        <div style={{ padding: '10px 20px', borderTop: '1px solid #f0f0f0', textAlign: 'right' }}>
          <button onClick={() => onCopy(code)} style={{
            padding: '4px 16px', border: '1px solid #d9d9d9', borderRadius: 4,
            background: '#fff', cursor: 'pointer', fontSize: 13,
          }}>
            {t('strategy.templates.actions.copyCode', '复制')}
          </button>
        </div>
      </div>
    </div>
  );
}
