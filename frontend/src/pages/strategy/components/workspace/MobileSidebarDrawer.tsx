import { Drawer } from 'antd';
import WorkspaceSidebar from './WorkspaceSidebar';

interface Props {
  open: boolean;
  onClose: () => void;
  templates: { id: string; name: string }[];
  loading: boolean;
  selectedId: string;
  onSelect: (id: string) => void;
  backtestRuns: { id: string; templateName?: string; totalReturn?: number; totalTrades?: number; templateId?: string }[];
  runsLoading: boolean;
  onOpenHistory: (tid?: string) => void;
  onImport: () => void;
  onNew: () => void;
}

export default function MobileSidebarDrawer({ open, onClose, templates, loading, selectedId, onSelect, backtestRuns, runsLoading, onOpenHistory, onImport, onNew }: Props) {
  return (
    <Drawer open={open} onClose={onClose} placement="left" width={280} styles={{ body: { padding: 0 } }}>
      <WorkspaceSidebar
        templates={templates} loading={loading} selectedId={selectedId}
        onSelect={(id) => { onSelect(id); onClose(); }}
        backtestRuns={backtestRuns} runsLoading={runsLoading}
        onOpenHistory={(tid) => { onOpenHistory(tid); onClose(); }}
        onImport={() => { onImport(); onClose(); }}
        onNew={() => { onNew(); onClose(); }}
        collapsed={false} onToggle={onClose}
      />
    </Drawer>
  );
}
