import { Select, Button, Space } from 'antd';
import { useTranslation } from 'react-i18next'
import { TEMPLATE_LOADED_KEY, TEMPLATE_SAVE_AS_KEY, TEMPLATE_SELECT_PLACEHOLDER_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';

;
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
        placeholder={t(TEMPLATE_SELECT_PLACEHOLDER_KEY, 'Select a template...')}
        options={templates.map((tpl) => ({ value: tpl.id, label: tpl.name }))}
        onChange={(val) => { if (val) onLoad(val); }}
      />
      <Space>
        <Button size="small" onClick={onSaveAs}>
          {t(TEMPLATE_SAVE_AS_KEY, 'Save As New')}
        </Button>
        {loadedTemplate && (
          <span style={{ fontSize: 12, color: '#8c8c8c' }}>
            {t(TEMPLATE_LOADED_KEY, 'Loaded')}: {loadedTemplate.name}
          </span>
        )}
      </Space>
    </Space>
  );
}
