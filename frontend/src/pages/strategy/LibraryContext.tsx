import { createContext, useContext } from 'react';
import type { FormInstance } from 'antd';
import type { StrategyTemplate } from '@/client/strategy';
import type { TemplateFilter } from './hooks/useLibraryTemplates';
import type { LibraryTab } from './hooks/useStrategyLibrary';
import type { ScheduleRow, BacktestRunRow, ScheduleHealthSummary, TriggerResult, TriggerContext, TemplateOption, AccountRow } from './hooks/libraryTypes';
import type { ScheduleFormValues } from './components/EditScheduleModal';

export interface LibraryCtx {
  // Templates
  templates: StrategyTemplate[];
  allTemplates: StrategyTemplate[];
  templatesLoading: boolean;
  templatesError: string | null;
  selectedId: string;
  selected: StrategyTemplate | null;
  selectTemplate: (id: string) => void;
  filter: TemplateFilter; setFilter: (f: TemplateFilter) => void;
  search: string; setSearch: (s: string) => void;
  publishing: boolean;
  scheduleCountByTemplate: (id: string) => number;
  // Template actions
  openCreate: () => void;
  openEdit: (tpl: StrategyTemplate) => void;
  handleSaveAsMine: (tpl: StrategyTemplate) => void;
  handleDelete: (id: string) => void;
  handlePublish: (id: string) => void;
  handleUnpublish: (id: string) => void;
  // CRUD modal
  editOpen: boolean; setEditOpen: (v: boolean) => void;
  editing: StrategyTemplate | null; setEditing: (tpl: StrategyTemplate | null) => void;
  codeValidating: boolean; lastValidatedCode: string; setLastValidatedCode: (s: string) => void;
  handleSave: (values: Record<string, unknown>) => void;
  // Code view
  codeViewOpen: boolean; setCodeViewOpen: (v: boolean) => void;
  viewingCode: string; setViewingCode: (s: string) => void;
  // UI
  activeTab: LibraryTab; setActiveTab: (t: LibraryTab) => void;
  // Schedules
  scheduleProps: {
    schedules: ScheduleRow[]; allSchedules: ScheduleRow[];
    loading: boolean; error: string | null;
    templates: TemplateOption[]; accounts: AccountRow[];
    symbols: { value: string; label: string }[]; symbolsLoading: boolean;
    formatTime: (v: unknown) => string;
    openEdit: boolean; setOpenEdit: (v: boolean) => void;
    editing: ScheduleRow | null; setEditing: (v: ScheduleRow | null) => void;
    form: FormInstance<ScheduleFormValues>; accountIdWatch: string | undefined;
    loadSymbols: (accountId: string, symbol?: string) => void;
    submitEdit: () => void; openUpdate: (row: ScheduleRow) => void;
    onToggleActive: (row: ScheduleRow, next: boolean) => void;
    onDelete: (row: ScheduleRow) => void;
    onManualTrigger: (row: ScheduleRow) => void;
    loadScheduleHealth: (row: ScheduleRow) => void;
    healthOpen: boolean; setHealthOpen: (v: boolean) => void;
    healthLoading: boolean; healthTarget: ScheduleRow | null; setHealthTarget: (v: ScheduleRow | null) => void;
    healthSummary: ScheduleHealthSummary | null; setHealthSummary: (v: ScheduleHealthSummary | null) => void;
    triggering: boolean; openTrigger: boolean; setOpenTrigger: (v: boolean) => void;
    triggerResult: TriggerResult | null; triggerContext: TriggerContext | null;
    setTriggerContext: (v: TriggerContext | null) => void; setTriggerResult: (v: TriggerResult | null) => void;
    doOrderSend: () => void; openCreate: () => void;
  };
  // Backtest
  backtestProps: {
    runs: BacktestRunRow[]; loading: boolean; error: string | null;
    page: number; pageSize: number; total: number;
    deleting: boolean; drawerOpen: boolean; selectedRunId: string;
    onPageChange: (p: number, ps: number) => void;
    onViewRun: (id: string) => void;
    onDeleteRun: (id: string) => void;
    onBatchDelete: () => void;
    onRefresh: () => void;
    setDrawerOpen: (v: boolean) => void;
  };
}

const Ctx = createContext<LibraryCtx | null>(null);

export function LibraryProvider({ value, children }: { value: LibraryCtx; children: React.ReactNode }) {
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useLibraryCtx() {
  const c = useContext(Ctx);
  if (!c) throw new Error('useLibraryCtx must be used within LibraryProvider');
  return c;
}
