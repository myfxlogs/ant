import { Drawer } from 'antd';
import WorkspaceSidebar, { type WorkspaceSection, type NewSource } from './WorkspaceSidebar';

interface Props {
  open: boolean;
  onClose: () => void;
  templates: { id: string; name: string }[];
  loading: boolean;
  selectedId: string;
  onSelect: (id: string) => void;
  onDeleteTemplate?: (id: string) => void;
  onRenameTemplate?: (id: string, name: string) => void;
  onBatchDeleteTemplates?: (ids: string[]) => void;
  backtestRuns: { id: string; templateName?: string; totalReturn?: number; totalTrades?: number; templateId?: string; name?: string }[];
  runsLoading: boolean;
  onOpenHistory: (runId?: string) => void;
  onDeleteRun?: (runId: string) => void;
  onBatchDeleteRuns?: (runIds: string[]) => void;
  onRenameRun?: (runId: string, name: string) => void;
  onNewSource: (source: NewSource) => void;
  onNew: () => void;
  activeSection: WorkspaceSection;
  onSectionChange: (s: WorkspaceSection) => void;
}

export default function MobileSidebarDrawer({ open, onClose, templates, loading, selectedId, onSelect, onDeleteTemplate, onRenameTemplate, onBatchDeleteTemplates, backtestRuns, runsLoading, onOpenHistory, onDeleteRun, onBatchDeleteRuns, onRenameRun, onNewSource, onNew, activeSection, onSectionChange }: Props) {
  return (
    <Drawer open={open} onClose={onClose} placement="left" width={280} styles={{ body: { padding: 0 } }}>
      <WorkspaceSidebar
        templates={templates} loading={loading} selectedId={selectedId}
        onSelect={(id) => { onSelect(id); onClose(); }}
        onDeleteTemplate={onDeleteTemplate}
        onRenameTemplate={onRenameTemplate}
        onBatchDeleteTemplates={onBatchDeleteTemplates}
        backtestRuns={backtestRuns} runsLoading={runsLoading}
        onOpenHistory={(runId) => { onOpenHistory(runId); onClose(); }}
        onDeleteRun={onDeleteRun}
        onBatchDeleteRuns={onBatchDeleteRuns}
        onRenameRun={onRenameRun}
        onNewSource={(source) => { onNewSource(source); onClose(); }}
        onNew={() => { onNew(); onClose(); }}
        activeSection={activeSection}
        onSectionChange={(s) => { onSectionChange(s); onClose(); }}
        collapsed={false} onToggle={onClose}
      />
    </Drawer>
  );
}
