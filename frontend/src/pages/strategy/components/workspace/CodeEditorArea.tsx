import { Button } from 'antd';
import { ImportOutlined, RobotOutlined, HistoryOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import StrategyCodeEditor from '@/components/strategy/StrategyCodeEditor';
import ImportEAPanel from '../editor/ImportEAPanel';

interface Props {
  code: string;
  importMode: boolean;
  isMobile: boolean;
  templateCount: number;
  onSetImportMode: (v: boolean) => void;
  onSetCode: (c: string) => void;
  onSetCenterTab: (tab: string) => void;
  onSetRightPanelTab: (tab: 'ai') => void;
  onSelectFirstTemplate: () => void;
  onStrategyIdChange?: (id: string | undefined) => void;
}

export default function CodeEditorArea({ code, importMode, isMobile, templateCount, onSetImportMode, onSetCode, onSetCenterTab, onSetRightPanelTab, onSelectFirstTemplate, onStrategyIdChange }: Props) {
  const { t } = useTranslation();

  if (importMode) {
    return (
      <div style={{ flex: 1, overflow: 'auto', padding: '12px 16px' }}>
        <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
          <Button size="small" type="text" onClick={() => onSetImportMode(false)}>
            ← {t('strategy.workspace.backToEditor', { defaultValue: 'Back' })}
          </Button>
        </div>
        <ImportEAPanel
          onApplyCode={(c) => { onSetCode(c); onSetCenterTab('code'); onSetImportMode(false); }}
          onStrategyIdChange={onStrategyIdChange}
        />
      </div>
    );
  }

  if (code) {
    return (
      <StrategyCodeEditor
        value={code}
        onChange={onSetCode}
        style={{ flex: 1, borderRadius: 0, border: 'none', minHeight: 0 }}
      />
    );
  }

  return (
    <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <div style={{ textAlign: 'center', maxWidth: 420, padding: 40 }}>
        <div style={{ fontSize: 48, marginBottom: 16 }}>📝</div>
        <div style={{ fontSize: 16, fontWeight: 600, marginBottom: 8, color: 'var(--ant-color-text)' }}>
          {t('strategy.workspace.emptyTitle', { defaultValue: 'Start building your strategy' })}
        </div>
        <div style={{ fontSize: 13, color: 'var(--ant-color-text-secondary)', marginBottom: 24, lineHeight: 1.6 }}>
          {t('strategy.workspace.emptyDesc', { defaultValue: 'Import an existing MQL EA, pick a template, or let AI generate one for you. All backtesting and deployment happens right here.' })}
        </div>
        <div style={{ display: 'flex', gap: 10, justifyContent: 'center', flexWrap: 'wrap' }}>
          <Button type="primary" icon={<ImportOutlined />} onClick={() => onSetImportMode(true)}>
            {t('strategy.workspace.importMql', { defaultValue: 'Import MQL EA' })}
          </Button>
          <Button icon={<RobotOutlined />} onClick={() => isMobile ? onSetCenterTab('chat') : onSetRightPanelTab('ai')}
            style={{ background: '#722ed1', borderColor: '#722ed1', color: '#fff' }}>
            {t('strategy.workspace.aiGenerate', { defaultValue: 'AI Generate' })}
          </Button>
          <Button icon={<HistoryOutlined />} onClick={onSelectFirstTemplate}
            disabled={templateCount === 0}>
            {t('strategy.workspace.useTemplate', { defaultValue: 'Use Template' })}
          </Button>
        </div>
      </div>
    </div>
  );
}
