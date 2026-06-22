import { createContext, useContext } from 'react';
import type { BacktestRunRow } from '../hooks/libraryTypes';

export interface RunsCtx {
  runs: BacktestRunRow[]; loading: boolean; error: string | null;
  page: number; pageSize: number; total: number;
  deleting: boolean; drawerOpen: boolean; selectedRunId: string;
  onPageChange: (p: number, ps: number) => void;
  onViewRun: (id: string) => void;
  onDeleteRun: (id: string) => void;
  onBatchDelete: () => void;
  onRefresh: () => void;
  setDrawerOpen: (v: boolean) => void;
}

const Ctx = createContext<RunsCtx | null>(null);
export function RunsProvider({ value, children }: { value: RunsCtx; children: React.ReactNode }) {
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}
export function useRunsCtx() {
  const c = useContext(Ctx);
  if (!c) throw new Error('useRunsCtx must be used within RunsProvider');
  return c;
}
