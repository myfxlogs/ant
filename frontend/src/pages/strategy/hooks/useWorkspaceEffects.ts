import { useEffect } from 'react';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import { DATE_PRESETS } from '@/components/backtest/useBacktestRunner';

interface WorkspaceEffectsDeps {
  code: string;
  setCode: (code: string) => void;
  loadedTemplate: any;
  resetBacktestStatus: () => void;
  activeAccounts: any[];
  accountId: string;
  setAccountId: (v: string) => void;
  setSymbol: (v: string) => void;
  fetchAccounts: () => void;
  loadTemplates: () => void;
  datePreset: string;
  applyDatePreset: (preset: any) => void;
  financialsReady: boolean;
  fetchTradeHistory: () => void;
}

function useCodeSync(code: string) {
  const setCurrentCode = useWorkspaceStore(s => s.setCurrentCode);
  useEffect(() => { setCurrentCode(code); }, [code]); // eslint-disable-line react-hooks/exhaustive-deps
}

function useTemplateNameSync(loadedTemplate: any) {
  const setCurrentCodeName = useWorkspaceStore(s => s.setCurrentCodeName);
  useEffect(() => {
    setCurrentCodeName(loadedTemplate?.name || '');
  }, [loadedTemplate]); // eslint-disable-line react-hooks/exhaustive-deps
}

function useCodeRestore(code: string, setCode: (v: string) => void) {
  const hasHydrated = useWorkspaceStore(s => s._hasHydrated);
  const currentCode = useWorkspaceStore(s => s.currentCode);
  useEffect(() => {
    if (hasHydrated && currentCode && !code) setCode(currentCode);
  }, [hasHydrated]); // eslint-disable-line react-hooks/exhaustive-deps
}

function useBacktestReset(code: string, resetBacktestStatus: () => void) {
  useEffect(() => { resetBacktestStatus(); }, [code]); // eslint-disable-line react-hooks/exhaustive-deps
}

function useStaleAccountCleanup(
  accountId: string,
  activeAccounts: any[],
  setAccountId: (v: string) => void,
  setSymbol: (v: string) => void,
) {
  useEffect(() => {
    if (!accountId) return;
    if (activeAccounts.length === 0 || !activeAccounts.some(a => a.id === accountId)) {
      setAccountId(''); setSymbol('');
    }
  }, [activeAccounts, accountId, setAccountId, setSymbol]);
}

function useWorkspaceInit(fetchAccounts: () => void, loadTemplates: () => void) {
  useEffect(() => { fetchAccounts(); loadTemplates(); }, []); // eslint-disable-line react-hooks/exhaustive-deps
}

function useTradeHistory(accountId: string, financialsReady: boolean, fetchTradeHistory: () => void) {
  useEffect(() => {
    if (accountId && financialsReady) fetchTradeHistory();
  }, [accountId, financialsReady, fetchTradeHistory]);
}

function useDatePresetInit(datePreset: string, applyDatePreset: (preset: any) => void) {
  useEffect(() => {
    const preset = DATE_PRESETS.find(p => p.key === datePreset);
    if (preset) applyDatePreset(preset);
  }, []); // eslint-disable-line react-hooks/exhaustive-deps
}

export function useWorkspaceEffects(deps: WorkspaceEffectsDeps) {
  useCodeSync(deps.code);
  useTemplateNameSync(deps.loadedTemplate);
  useCodeRestore(deps.code, deps.setCode);
  useBacktestReset(deps.code, deps.resetBacktestStatus);
  useStaleAccountCleanup(deps.accountId, deps.activeAccounts, deps.setAccountId, deps.setSymbol);
  useWorkspaceInit(deps.fetchAccounts, deps.loadTemplates);
  useTradeHistory(deps.accountId, deps.financialsReady, deps.fetchTradeHistory);
  useDatePresetInit(deps.datePreset, deps.applyDatePreset);
}
