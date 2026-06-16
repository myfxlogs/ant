import { useState, useCallback } from 'react';
import { useLibraryTemplates } from './useLibraryTemplates';
import { useLibrarySchedules } from './useLibrarySchedules';
import { useLibraryRuns } from './useLibraryRuns';
import type { TemplateFilter } from './useLibraryTemplates';

export type LibraryTab = 'overview' | 'schedules' | 'backtest' | 'logs';

export function useStrategyLibrary() {
  const templates = useLibraryTemplates();
  const schedules = useLibrarySchedules(templates.selectedId);
  const runs = useLibraryRuns(templates.selectedId);
  const [activeTab, setActiveTab] = useState<LibraryTab>('overview');

  // Reset tab when selection changes and tab is no longer relevant
  const selectTemplate = useCallback((id: string) => {
    templates.setSelectedId(id);
    if (id && activeTab === 'overview') { /* keep overview */ }
    else if (!id) { setActiveTab('overview'); }
  }, [templates.setSelectedId, activeTab]);

  // Code view modal state
  const [codeViewOpen, setCodeViewOpen] = useState(false);
  const [viewingCode, setViewingCode] = useState('');

  // Backtest modal state (shared with template backtest modal)
  const [backtestModalOpen, setBacktestModalOpen] = useState(false);

  // Schedule flow state (for launch from backtest score)
  const [scheduleFlow, setScheduleFlow] = useState({
    publishing: false,
    creating: false,
    enableAfterCreate: true,
    templateId: '',
    templateDraftId: undefined as string | undefined,
  });

  return {
    // Templates
    templates,
    // Schedules
    schedules,
    // Runs
    runs,
    // UI state
    activeTab, setActiveTab,
    selectTemplate,
    codeViewOpen, setCodeViewOpen, viewingCode, setViewingCode,
    backtestModalOpen, setBacktestModalOpen,
    scheduleFlow, setScheduleFlow,
  };
}
