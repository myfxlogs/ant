import { useState, useEffect, Suspense, lazy } from 'react';
import { Modal, Select, Spin } from 'antd';
import { SettingOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useSystemAIConfigsQuery } from '@/queries/useSystemAIConfigsQuery';

const SystemAI = lazy(() => import('@/pages/ai/SystemAI'));

interface Props {
  open: boolean;
  onClose: () => void;
}

const STORAGE_KEY = 'workspace_ai_model';
function loadModel(): string { try { return localStorage.getItem(STORAGE_KEY) || ''; } catch { return ''; } }
function saveModel(v: string) { localStorage.setItem(STORAGE_KEY, v); }

export default function AISettingsModal({ open, onClose }: Props) {
  const { t } = useTranslation();
  const { data, isLoading } = useSystemAIConfigsQuery();
  const configs = data?.items ?? [];

  const [workspaceModel, setWorkspaceModel] = useState(loadModel);
  useEffect(() => { if (open) setWorkspaceModel(loadModel()); }, [open]);

  const modelOptions = configs
    .filter(c => c.enabled && c.has_secret)
    .flatMap(c => {
      const models = c.models?.length ? c.models : c.default_model ? [c.default_model] : [];
      return models.map(m => ({
        value: `${c.provider_id}|${m}`,
        label: `${c.name || c.provider_id} · ${m}`,
      }));
    });

  return (
    <Modal title={<><SettingOutlined style={{ marginRight: 8 }} />{t('ai.settings.pageTitle', 'AI Settings')}</>}
      open={open} onCancel={onClose} footer={null} width={680} style={{ top: 16 }}>
      <div style={{ maxHeight: '75vh', overflowY: 'auto', paddingRight: 4 }}>
        {/* Inline workspace model selector */}
        <div style={{ marginBottom: 12, padding: '8px 12px', background: '#f0f5ff', borderRadius: 6, border: '1px solid #d6e4ff' }}>
          <div style={{ fontSize: 11, fontWeight: 600, color: '#262626', marginBottom: 4 }}>
            {t('strategy.ai.workspaceModel', 'Workspace Model')}
          </div>
          <Select value={workspaceModel || undefined} onChange={v => { setWorkspaceModel(v); saveModel(v); }}
            loading={isLoading} placeholder="GPT-4o (default)" style={{ width: '100%' }}
            options={modelOptions} allowClear showSearch optionFilterProp="label" />
        </div>

        {/* Full SystemAI page embedded — providers, keys, base URL, models, advanced */}
        <Suspense fallback={<div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>}>
          {open && <SystemAI />}
        </Suspense>
      </div>
    </Modal>
  );
}
