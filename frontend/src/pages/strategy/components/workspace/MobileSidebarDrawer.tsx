import { Drawer } from 'antd';
import WorkspaceSidebar from './WorkspaceSidebar';

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
  backtestRuns: { id: string; templateName?: string; totalReturn?: number; totalTrades?: number; templateId?: string }[];
  runsLoading: boolean;
  onOpenHistory: (tid?: string) => void;
  onDeleteRun?: (runId: string) => void;
  onBatchDeleteRuns?: (runIds: string[]) => void;
  onImport: () => void;
  onNew: () => void;
  autoExpandHistory?: boolean;
}

export default function MobileSidebarDrawer({ open, onClose, templates, loading, selectedId, onSelect, onDeleteTemplate, onRenameTemplate, onBatchDeleteTemplates, backtestRuns, runsLoading, onOpenHistory, onDeleteRun, onBatchDeleteRuns, onImport, onNew, autoExpandHistory }: Props) {
  return (
    <Drawer open={open} onClose={onClose} placement="left" width={280} styles={{ body: { padding: 0 } }}>
      <WorkspaceSidebar
        templates={templates} loading={loading} selectedId={selectedId}
        onSelect={(id) => { onSelect(id); onClose(); }}
        onDeleteTemplate={onDeleteTemplate}
        onRenameTemplate={onRenameTemplate}
        onBatchDeleteTemplates={onBatchDeleteTemplates}
        backtestRuns={backtestRuns} runsLoading={runsLoading}
        onOpenHistory={(tid) => { onOpenHistory(tid); onClose(); }}
        onDeleteRun={onDeleteRun}
        onBatchDeleteRuns={onBatchDeleteRuns}
        onImport={() => { onImport(); onClose(); }}
        onNew={() => { onNew(); onClose(); }}
        collapsed={false} onToggle={onClose}
        autoExpandHistory={autoExpandHistory}
      />
    </Drawer>
  );
}
