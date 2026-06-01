import { Select, Button, Space } from 'antd';
import { useTranslation } from 'react-i18next';
import type { StrategyTemplate } from '@/client/strategy';

interface Props {
  templates: StrategyTemplate[];
  loading: boolean;
  loadedTemplate: StrategyTemplate | null;
  onLoad: (id: string) => void;
  onSaveAs: () => void;
}

export default function WorkspaceTemplateManager({ templates, loading, loadedTemplate, onLoad, onSaveAs }: Props) {
  const { t } = useTranslation();
  return (
    <Space direction="vertical" style={{ width: '100%' }}>
      <Select
        style={{ width: '100%' }}
        loading={loading}
        showSearch
        allowClear
        optionFilterProp="label"
        value={loadedTemplate?.id}
        placeholder={t('strategy.workspace.template.selectPlaceholder', 'Select a template...')}
        options={templates.map((tpl) => ({ value: tpl.id, label: tpl.name }))}
        onChange={(val) => { if (val) onLoad(val); }}
      />
      <Space>
        <Button size="small" onClick={onSaveAs}>
          {t('strategy.workspace.template.saveAs', 'Save As New')}
        </Button>
        {loadedTemplate && (
          <span style={{ fontSize: 12, color: '#8c8c8c' }}>
            {t('strategy.workspace.template.loaded', 'Loaded')}: {loadedTemplate.name}
          </span>
        )}
      </Space>
    </Space>
  );
}
