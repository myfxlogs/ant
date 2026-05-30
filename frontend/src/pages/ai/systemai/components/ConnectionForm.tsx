import { Input, Button, Space, Select } from 'antd';
import { ReloadOutlined, ExportOutlined, ExclamationCircleOutlined, ClearOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { PROVIDER_LINKS } from '../constants';
import { Section, Label, SoftTag } from './SharedComponents';
import type { AIConfig } from '../model';

export function ConnectionFormSection(props: {
  draft: AIConfig;
  providerLabel: (id: string, dbName?: string) => string;
  isCustomProvider: (id: string) => boolean;
  urlHttps: boolean;
  urlOk: boolean;
  secretInput: string;
  onSecretInputChange: (v: string) => void;
  onDraftChange: (patch: Partial<AIConfig>) => void;
  onClearSecret: () => void;
  savingSecret: boolean;
  discovering: boolean;
  discoveredModels: string[];
}) {
  const { t } = useTranslation();
  const {
    draft,
    providerLabel,
    isCustomProvider,
    urlHttps,
    urlOk,
    secretInput,
    onSecretInputChange,
    onDraftChange,
    onClearSecret,
    savingSecret,
    discovering,
    discoveredModels,
  } = props;

  return (
    <Section
      step={2}
      title={`${t('ai.settings.sections.connection', { defaultValue: '连接配置' })} · ${providerLabel(draft.provider_id, draft.name)}`}
      subtitle={
        PROVIDER_LINKS[draft.provider_id] ? (
          <a
            href={PROVIDER_LINKS[draft.provider_id]}
            target="_blank"
            rel="noreferrer"
            className="text-xs text-slate-600 hover:text-slate-800 hover:underline inline-flex items-center gap-1"
          >
            <ExportOutlined className="w-3 h-3" /> {t('ai.settings.sections.connectionApiKeyLink', { defaultValue: '前往申请 / 管理该厂商 API Key' })}
          </a>
        ) : null
      }
    >
      <div className="space-y-4">
        {isCustomProvider(draft.provider_id) ? (
          <div>
            <Label
              text={t('ai.systemAI.customProvider.nameLabel', { defaultValue: '厂商名称' })}
              hint={t('ai.systemAI.customProvider.nameHint', { defaultValue: '用于在厂商卡片、模型选择和路由配置中识别这个自定义模型服务。' })}
            />
            <Input
              size="large"
              value={draft.name}
              onChange={(e) => onDraftChange({ name: e.target.value })}
              placeholder={t('ai.systemAI.customProvider.namePlaceholder', { defaultValue: '例如：OpenRouter / SiliconFlow / 公司内网模型' })}
            />
          </div>
        ) : null}
        <div>
          <Label
            text={`${t('ai.settings.fields.baseUrl', { defaultValue: 'Base URL' })}${t('ai.settings.fields.baseUrlHint', { defaultValue: '（模型服务地址）' })}`}
            hint={
              isCustomProvider(draft.provider_id)
                ? t('ai.systemAI.fields.baseUrlCustomHint', { defaultValue: '输入 OpenAI 兼容端点，例如 https://model.example.com/v1' })
                : t('ai.systemAI.fields.baseUrlReadonlyHint', { defaultValue: '官方地址由系统维护，不可修改' })
            }
          />
          <Input
            size="large"
            value={draft.base_url}
            onChange={(e) => {
              if (!isCustomProvider(draft.provider_id)) return;
              onDraftChange({ base_url: e.target.value });
            }}
            placeholder={
              isCustomProvider(draft.provider_id)
                ? t('ai.systemAI.fields.baseUrlCustomPlaceholder', { defaultValue: '例如: https://model.example.com/v1' })
                : t('ai.systemAI.fields.baseUrlReadonlyPlaceholder', { defaultValue: '官方地址（只读）' })
            }
            disabled={!isCustomProvider(draft.provider_id)}
          />
          {draft.base_url && !urlHttps && urlOk && (
            <p className="text-xs text-slate-600 flex items-center gap-1 mt-1.5">
              <ExclamationCircleOutlined className="w-3.5 h-3.5" /> {t('ai.systemAI.fields.httpWarning', { defaultValue: '当前为 HTTP，生产环境建议使用 HTTPS' })}
            </p>
          )}
        </div>

        <div>
          <Label
            text={t('ai.settings.fields.apiKey', { defaultValue: 'API Key' })}
            hint={t('ai.systemAI.fields.apiKeyHint', { defaultValue: '输入后将自动加密保存，无需手动提交' })}
            badge={draft.has_secret ? <SoftTag>{t('ai.settings.fields.apiKeyConfigured', { defaultValue: '已配置' })}</SoftTag> : undefined}
          />
          <Space.Compact style={{ width: '100%' }}>
            <Input.Password
              size="large"
              value={secretInput}
              onChange={(e) => onSecretInputChange(e.target.value)}
              placeholder={draft.has_secret
                ? t('ai.settings.fields.apiKeyReplaceHint', { defaultValue: '如需更换密钥，请重新输入' })
                : t('ai.systemAI.fields.apiKeyPastePlaceholder', { defaultValue: '粘贴 API Key，将自动预保存' })}
            />
            <Button
              size="large"
              icon={<ClearOutlined className="w-4 h-4" />}
              onClick={onClearSecret}
              disabled={savingSecret || !draft.has_secret}
              loading={savingSecret}
            >
              {t('ai.settings.fields.deleteApiKey', { defaultValue: '删除密钥' })}
            </Button>
          </Space.Compact>
        </div>

        <div>
          <Label
            text={t('ai.settings.fields.availableModels', { defaultValue: '可用模型' })}
            hint={t('ai.settings.fields.availableModelsHint', { defaultValue: '同一 API Key 下可同时启用多个 model；这里的清单会出现在 /ai/agents 的下拉里。默认空白，从下拉选择或手动输入 model id 后回车添加；只加入显式选过的，不会自动并入全部已发现模型。' })}
            badge={(
              <Space size={4}>
                {discovering ? (
                  <span className="text-xs text-gray-500 flex items-center gap-1">
                    <ReloadOutlined className="w-3 h-3 animate-spin" /> {t('ai.systemAI.fields.autoFetching', { defaultValue: '自动拉取中' })}
                  </span>
                ) : null}
                {(draft.models || []).length > 0 ? (
                  <Button
                    size="small"
                    type="link"
                    onClick={() => onDraftChange({ models: [], default_model: '' })}
                  >
                    {t('ai.settings.fields.clear', { defaultValue: '清空' })}
                  </Button>
                ) : null}
              </Space>
            )}
          />
          <Select
            size="large"
            mode="tags"
            value={(draft.models || [])}
            onChange={(vals) => {
              const cleaned = Array.from(new Set((vals as string[]).map((v) => (v || '').trim()).filter(Boolean)));
              onDraftChange({ models: cleaned, default_model: cleaned[0] || '' });
            }}
            options={(discoveredModels || []).map((m) => ({ value: m, label: m }))}
            style={{ width: '100%' }}
            allowClear
            placeholder={t('ai.settings.fields.availableModelsPlaceholder', { defaultValue: '选择或手动输入 model id 后回车添加（默认空白）' })}
            tokenSeparators={[',', ' ', '\n']}
            notFoundContent={
              <span className="text-xs text-gray-500">{t('ai.settings.fields.availableModelsEmpty', { defaultValue: '直接输入 model id 后回车即可加入' })}</span>
            }
          />
          <p className="text-xs text-gray-500 mt-1.5">
            {t('ai.settings.fields.availableModelsTip', { defaultValue: '提示：删除某个模型不会立即清空 /ai/agents 中已绑定它的 Agent，但会将它从下拉建议中移除。' })}
          </p>
        </div>
      </div>
    </Section>
  );
}
