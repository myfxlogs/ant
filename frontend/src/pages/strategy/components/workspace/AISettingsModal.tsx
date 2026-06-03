import { useState, useEffect } from 'react';
import { Modal, Select, InputNumber, Form, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { useSystemAIConfigsQuery } from '@/queries/useSystemAIConfigsQuery';

interface Props {
  open: boolean;
  onClose: () => void;
}

const STORAGE_KEY = 'workspace_ai_settings';

interface SavedSettings {
  modelKey?: string;
  temperature?: number;
  maxTokens?: number;
}

function loadSettings(): SavedSettings {
  try { return JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}'); } catch { return {}; }
}
function saveSettings(s: SavedSettings) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(s));
}

export default function AISettingsModal({ open, onClose }: Props) {
  const { t } = useTranslation();
  const { data: configs = [], isLoading } = useSystemAIConfigsQuery();
  const [saved, setSaved] = useState<SavedSettings>(loadSettings);
  const [modelKey, setModelKey] = useState(saved.modelKey || '');
  const [temperature, setTemperature] = useState(saved.temperature ?? 0.3);
  const [maxTokens, setMaxTokens] = useState(saved.maxTokens ?? 4096);

  // Build model options from system configs
  const options = (configs || [])
    .filter(c => c.enabled && c.has_secret)
    .flatMap(c => {
      const models = (c.models?.length ? c.models : c.default_model ? [c.default_model] : []);
      return models.map(m => ({
        value: `${c.provider_id}|${m}`,
        label: `${c.name || c.provider_id} · ${m}`,
      }));
    });

  useEffect(() => setSaved(loadSettings()), [open]);

  const handleOk = () => {
    const s: SavedSettings = { modelKey, temperature, maxTokens };
    saveSettings(s);
    setSaved(s);
    message.success(t('strategy.ai.settingsSaved', 'AI settings saved'));
    onClose();
  };

  return (
    <Modal title={t('strategy.ai.settings', 'AI Settings')} open={open} onOk={handleOk} onCancel={onClose}
      okText={t('common.save', 'Save')} cancelText={t('common.cancel', 'Cancel')} width={420}>
      <Form layout="vertical" size="small" style={{ marginTop: 12 }}>
        <Form.Item label={t('strategy.ai.model', 'Model')}>
          <Select value={modelKey || undefined} onChange={setModelKey}
            loading={isLoading} placeholder="GPT-4o (default)"
            options={options} allowClear showSearch optionFilterProp="label" />
        </Form.Item>
        <Form.Item label={t('strategy.ai.temperature', 'Temperature')}>
          <InputNumber value={temperature} onChange={v => setTemperature(v ?? 0.3)}
            min={0} max={2} step={0.1} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item label={t('strategy.ai.maxTokens', 'Max Tokens')}>
          <InputNumber value={maxTokens} onChange={v => setMaxTokens(v ?? 4096)}
            min={256} max={32768} step={256} style={{ width: '100%' }} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
