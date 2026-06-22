import { createContext, useContext } from 'react';
import type { StrategyTemplate } from '@/client/strategy';
import type { TemplateFilter } from '../hooks/useLibraryTemplates';

export interface TemplatesCtx {
  templates: StrategyTemplate[];
  allTemplates: StrategyTemplate[];
  loading: boolean;
  error: string | null;
  selectedId: string;
  selected: StrategyTemplate | null;
  selectTemplate: (id: string) => void;
  filter: TemplateFilter; setFilter: (f: TemplateFilter) => void;
  search: string; setSearch: (s: string) => void;
  publishing: boolean;
  scheduleCountByTemplate: (id: string) => number;
  openCreate: () => void;
  openEdit: (tpl: StrategyTemplate) => void;
  handleDelete: (id: string) => void;
  handlePublish: (id: string) => void;
  handleUnpublish: (id: string) => void;
  handleSaveAsMine: (tpl: StrategyTemplate) => void;
  handleValidate: (code: string) => void;
  handleSave: (values: Record<string, unknown>) => void;
  publishModalOpen: boolean; setPublishModalOpen: (v: boolean) => void;
  publishingTemplate: StrategyTemplate | null;
  openPublishModal: (tpl: StrategyTemplate) => void;
  closePublishModal: () => void;
  editOpen: boolean; setEditOpen: (v: boolean) => void;
  editing: StrategyTemplate | null; setEditing: (tpl: StrategyTemplate | null) => void;
  codeValidating: boolean; lastValidatedCode: string; setLastValidatedCode: (s: string) => void;
  validationResult: any;
}

const Ctx = createContext<TemplatesCtx | null>(null);
export function TemplatesProvider({ value, children }: { value: TemplatesCtx; children: React.ReactNode }) {
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}
export function useTemplatesCtx() {
  const c = useContext(Ctx);
  if (!c) throw new Error('useTemplatesCtx must be used within TemplatesProvider');
  return c;
}
