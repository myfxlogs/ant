import { createContext, useContext } from 'react';
import type { LibraryTab } from './hooks/useStrategyLibrary';

export interface LibraryCtx {
  activeTab: LibraryTab; setActiveTab: (t: LibraryTab) => void;
  codeViewOpen: boolean; setCodeViewOpen: (v: boolean) => void;
  viewingCode: string; setViewingCode: (s: string) => void;
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

// Re-export domain contexts
export { TemplatesProvider, useTemplatesCtx } from './contexts/TemplatesContext';
export type { TemplatesCtx } from './contexts/TemplatesContext';
export { SchedulesProvider, useSchedulesCtx } from './contexts/SchedulesContext';
export type { SchedulesCtx } from './contexts/SchedulesContext';
export { RunsProvider, useRunsCtx } from './contexts/RunsContext';
export type { RunsCtx } from './contexts/RunsContext';
