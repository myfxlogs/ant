import StrategiesTab from '@/components/backtest/StrategiesTab';
import type { StrategyTemplate } from '@/client/strategy';

interface Props {
  templates: StrategyTemplate[];
  loading: boolean;
  selectedId: string;
  hasUnsavedDraft: boolean;
  draftName: string;
  onSelect: (id: string | null) => void;
  onRunBacktest: () => void;
  onOpenHistory: (templateId?: string) => void;
  onSaveAs: () => void;
  visible: boolean;
}

export default function StrategiesTabPanel({
  templates, loading, selectedId, hasUnsavedDraft, draftName,
  onSelect, onRunBacktest, onOpenHistory, onSaveAs, visible,
}: Props) {
  return (
    <div style={{ flex: '1 1 0', minHeight: 0, display: visible ? 'flex' : 'none', flexDirection: 'column', overflow: 'auto', padding: '8px 14px' }}>
      <StrategiesTab
        templates={templates}
        loading={loading}
        selectedId={selectedId}
        hasUnsavedDraft={hasUnsavedDraft}
        draftName={draftName}
        onSelect={onSelect}
        onRunBacktest={onRunBacktest}
        onOpenHistory={onOpenHistory}
        onSaveAs={onSaveAs}
      />
    </div>
  );
}
