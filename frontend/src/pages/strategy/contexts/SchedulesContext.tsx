import { createContext, useContext } from 'react';
import type { FormInstance } from 'antd';
import type { ScheduleRow, ScheduleHealthSummary, TriggerResult, TriggerContext, TemplateOption, AccountRow } from '../hooks/libraryTypes';
import type { ScheduleFormValues } from '../components/EditScheduleModal';

export interface SchedulesCtx {
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
}

const Ctx = createContext<SchedulesCtx | null>(null);
export function SchedulesProvider({ value, children }: { value: SchedulesCtx; children: React.ReactNode }) {
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}
export function useSchedulesCtx() {
  const c = useContext(Ctx);
  if (!c) throw new Error('useSchedulesCtx must be used within SchedulesProvider');
  return c;
}
