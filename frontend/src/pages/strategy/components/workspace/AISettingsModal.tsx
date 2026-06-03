import { useState, useCallback } from 'react';
import { Modal, Select, Input, Button, Space, Tag, message, Divider } from 'antd';
import { SettingOutlined, KeyOutlined, CheckCircleOutlined, ExclamationCircleOutlined, ExportOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { useSystemAIConfigsQuery } from '@/queries/useSystemAIConfigsQuery';
import { queryKeys } from '@/queries/queryKeys';
import { updateSystemAISecret, clearSystemAISecret } from '@/pages/ai/systemai/api';
import type { AIConfig } from '@/pages/ai/systemai/model';

interface Props {
  open: boolean;
  onClose: () => void;
}

const STORAGE_KEY = 'workspace_ai_model';

function loadModel(): string {
  try { return localStorage.getItem(STORAGE_KEY) || ''; } catch { return ''; }
}
function saveModel(v: string) { localStorage.setItem(STORAGE_KEY, v); }

export default function AISettingsModal({ open, onClose }: Props) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { data, isLoading } = useSystemAIConfigsQuery();
  const configs: AIConfig[] = data?.items ?? [];

  const [selectedModel, setSelectedModel] = useState(loadModel);
  const [secretInput, setSecretInput] = useState<Record<string, string>>({});
  const [savingKey, setSavingKey] = useState<Record<string, boolean>>({});

  const modelOptions = configs
    .filter(c => c.enabled && c.has_secret)
    .flatMap(c => {
      const models = c.models?.length ? c.models : c.default_model ? [c.default_model] : [];
      return models.map(m => ({
        value: `${c.provider_id}|${m}`,
        label: `${c.name || c.provider_id} · ${m}`,
      }));
    });

  const handleSaveKey = useCallback(async (providerId: string) => {
    const key = secretInput[providerId]?.trim();
    if (!key) { message.warning('Enter an API key'); return; }
    setSavingKey(p => ({ ...p, [providerId]: true }));
    try {
      await updateSystemAISecret(providerId, key);
      queryClient.invalidateQueries({ queryKey: queryKeys.systemAI.configs });
      setSecretInput(p => { const n = { ...p }; delete n[providerId]; return n; });
      message.success(t('strategy.ai.keySaved', 'API key saved'));
    } catch (e: any) { message.error(e?.message || 'Failed to save key'); }
    finally { setSavingKey(p => ({ ...p, [providerId]: false })); }
  }, [secretInput, queryClient, t]);

  const handleClearKey = useCallback(async (providerId: string) => {
    setSavingKey(p => ({ ...p, [providerId]: true }));
    try {
      await clearSystemAISecret(providerId);
      queryClient.invalidateQueries({ queryKey: queryKeys.systemAI.configs });
      message.success(t('strategy.ai.keyCleared', 'API key cleared'));
    } catch (e: any) { message.error(e?.message || 'Failed to clear key'); }
    finally { setSavingKey(p => ({ ...p, [providerId]: false })); }
  }, [queryClient, t]);

  const handleModelChange = (v: string) => { setSelectedModel(v); saveModel(v); };

  return (
    <Modal title={<><SettingOutlined style={{ marginRight: 8 }} />{t('ai.settings.pageTitle', 'AI Settings')}</>}
      open={open} onCancel={onClose} footer={null} width={520}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        {/* Model selection */}
        <div>
          <div style={{ fontSize: 12, fontWeight: 600, color: '#262626', marginBottom: 6 }}>
            {t('strategy.ai.model', 'Model')}
          </div>
          <Select value={selectedModel || undefined} onChange={handleModelChange}
            loading={isLoading} placeholder="GPT-4o (default)" style={{ width: '100%' }}
            options={modelOptions} allowClear showSearch optionFilterProp="label" />
        </div>

        <Divider style={{ margin: 0 }} />

        {/* Provider API Keys */}
        <div>
          <div style={{ fontSize: 12, fontWeight: 600, color: '#262626', marginBottom: 8 }}>
            <KeyOutlined style={{ marginRight: 4 }} />API Keys
          </div>
          {configs.length === 0 && !isLoading && (
            <div style={{ color: '#8c8c8c', fontSize: 12 }}>
              {t('strategy.ai.noProviders', 'No AI providers configured.')}
            </div>
          )}
          {configs.map(c => (
            <div key={c.provider_id} style={{
              background: '#fafbfc', border: '1px solid #f0f0f0', borderRadius: 6,
              padding: '8px 10px', marginBottom: 8,
            }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
                <span style={{ fontSize: 12, fontWeight: 600 }}>{c.name || c.provider_id}</span>
                <Tag color={c.has_secret ? 'green' : 'orange'} style={{ fontSize: 10, margin: 0 }}
                  icon={c.has_secret ? <CheckCircleOutlined /> : <ExclamationCircleOutlined />}>
                  {c.has_secret ? t('strategy.ai.keySet', 'Key set') : t('strategy.ai.noKey', 'No key')}
                </Tag>
              </div>
              <Space.Compact style={{ width: '100%' }}>
                <Input.Password size="small"
                  value={secretInput[c.provider_id] || ''}
                  onChange={e => setSecretInput(p => ({ ...p, [c.provider_id]: e.target.value }))}
                  placeholder={c.has_secret ? 'Replace key...' : 'Paste API key...'} />
                <Button size="small" type="primary"
                  loading={savingKey[c.provider_id]}
                  onClick={() => handleSaveKey(c.provider_id)}>
                  {t('common.save', 'Save')}
                </Button>
                {c.has_secret && (
                  <Button size="small" danger
                    loading={savingKey[c.provider_id]}
                    onClick={() => handleClearKey(c.provider_id)}>
                    {t('strategy.ai.clear', 'Clear')}
                  </Button>
                )}
              </Space.Compact>
            </div>
          ))}
        </div>

        <Divider style={{ margin: 0 }} />
        <Button block icon={<ExportOutlined />} onClick={() => { onClose(); navigate('/ai/settings'); }}>
          {t('ai.requireConfig.actions.goSettings', 'Full AI Settings →')}
        </Button>
      </div>
    </Modal>
  );
}
