import { Input, Button, Space, Select } from 'antd';
import { ReloadOutlined, ExportOutlined, ExclamationCircleOutlined, ClearOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { FIELDS_API_KEY_CONFIGURED_KEY, FIELDS_API_KEY_KEY, FIELDS_API_KEY_REPLACE_HINT_KEY, FIELDS_AVAILABLE_MODELS_EMPTY_KEY, FIELDS_AVAILABLE_MODELS_HINT_KEY, FIELDS_AVAILABLE_MODELS_KEY, FIELDS_AVAILABLE_MODELS_PLACEHOLDER_KEY, FIELDS_AVAILABLE_MODELS_TIP_KEY, FIELDS_BASE_URL_HINT_KEY, FIELDS_BASE_URL_KEY, FIELDS_CLEAR_KEY, FIELDS_DELETE_API_KEY_KEY, SECTIONS_CONNECTION_API_KEY_LINK_KEY, SECTIONS_CONNECTION_KEY } from '@/gen/ant/v1/i18n/ai_settings_keys';

;
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
      title={`${t(SECTIONS_CONNECTION_KEY, { defaultValue: '连接配置' })} · ${providerLabel(draft.provider_id, draft.name)}`}
      subtitle={
        PROVIDER_LINKS[draft.provider_id] ? (
          <a
            href={PROVIDER_LINKS[draft.provider_id]}
            target="_blank"
            rel="noreferrer"
            className="text-xs text-slate-600 hover:text-slate-800 hover:underline inline-flex items-center gap-1"
          >
            <ExportOutlined className="w-3 h-3" /> {t(SECTIONS_CONNECTION_API_KEY_LINK_KEY, { defaultValue: '前往申请 / 管理该厂商 API Key' })}
          </a>
        ) : null
      }
    >
      <div className="space-y-4">
        {isCustomProvider(draft.provider_id) ? (
          <div>
            <Label
              text={t(SYSTEM_A_I_CUSTOM_PROVIDER_NAME_LABEL_KEY, { defaultValue: '厂商名称' })}
              hint={t(SYSTEM_A_I_CUSTOM_PROVIDER_NAME_HINT_KEY, { defaultValue: '用于在厂商卡片、模型选择和路由配置中识别这个自定义模型服务。' })}
            />
            <Input
              size="large"
              value={draft.name}
              onChange={(e) => onDraftChange({ name: e.target.value })}
              placeholder={t(SYSTEM_A_I_CUSTOM_PROVIDER_NAME_PLACEHOLDER_KEY, { defaultValue: '例如：OpenRouter / SiliconFlow / 公司内网模型' })}
            />
          </div>
        ) : null}
        <div>
          <Label
            text={`${t(FIELDS_BASE_URL_KEY, { defaultValue: 'Base URL' })}${t(FIELDS_BASE_URL_HINT_KEY, { defaultValue: '（模型服务地址）' })}`}
            hint={
              isCustomProvider(draft.provider_id)
                ? t(SYSTEM_A_I_FIELDS_BASE_URL_CUSTOM_HINT_KEY, { defaultValue: '输入 OpenAI 兼容端点，例如 https://model.example.com/v1' })
                : t(SYSTEM_A_I_FIELDS_BASE_URL_READONLY_HINT_KEY, { defaultValue: '官方地址由系统维护，不可修改' })
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
                ? t(SYSTEM_A_I_FIELDS_BASE_URL_CUSTOM_PLACEHOLDER_KEY, { defaultValue: '例如: https://model.example.com/v1' })
                : t(SYSTEM_A_I_FIELDS_BASE_URL_READONLY_PLACEHOLDER_KEY, { defaultValue: '官方地址（只读）' })
            }
            disabled={!isCustomProvider(draft.provider_id)}
          />
          {draft.base_url && !urlHttps && urlOk && (
            <p className="text-xs text-slate-600 flex items-center gap-1 mt-1.5">
              <ExclamationCircleOutlined className="w-3.5 h-3.5" /> {t(SYSTEM_A_I_FIELDS_HTTP_WARNING_KEY, { defaultValue: '当前为 HTTP，生产环境建议使用 HTTPS' })}
            </p>
          )}
        </div>

        <div>
          <Label
            text={t(FIELDS_API_KEY_KEY, { defaultValue: 'API Key' })}
            hint={t(SYSTEM_A_I_FIELDS_API_KEY_HINT_KEY, { defaultValue: '输入后将自动加密保存，无需手动提交' })}
            badge={draft.has_secret ? <SoftTag>{t(FIELDS_API_KEY_CONFIGURED_KEY, { defaultValue: '已配置' })}</SoftTag> : undefined}
          />
          <Space.Compact style={{ width: '100%' }}>
            <Input.Password
              size="large"
              value={secretInput}
              onChange={(e) => onSecretInputChange(e.target.value)}
              placeholder={draft.has_secret
                ? t(FIELDS_API_KEY_REPLACE_HINT_KEY, { defaultValue: '如需更换密钥，请重新输入' })
                : t(SYSTEM_A_I_FIELDS_API_KEY_PASTE_PLACEHOLDER_KEY, { defaultValue: '粘贴 API Key，将自动预保存' })}
            />
            <Button
              size="large"
              icon={<ClearOutlined className="w-4 h-4" />}
              onClick={onClearSecret}
              disabled={savingSecret || !draft.has_secret}
              loading={savingSecret}
            >
              {t(FIELDS_DELETE_API_KEY_KEY, { defaultValue: '删除密钥' })}
            </Button>
          </Space.Compact>
        </div>

        <div>
          <Label
            text={t(FIELDS_AVAILABLE_MODELS_KEY, { defaultValue: '可用模型' })}
            hint={t(FIELDS_AVAILABLE_MODELS_HINT_KEY, { defaultValue: '同一 API Key 下可同时启用多个 model；这里的清单会出现在 /ai/agents 的下拉里。默认空白，从下拉选择或手动输入 model id 后回车添加；只加入显式选过的，不会自动并入全部已发现模型。' })}
            badge={(
              <Space size={4}>
                {discovering ? (
                  <span className="text-xs text-gray-500 flex items-center gap-1">
                    <ReloadOutlined className="w-3 h-3 animate-spin" /> {t(SYSTEM_A_I_FIELDS_AUTO_FETCHING_KEY, { defaultValue: '自动拉取中' })}
                  </span>
                ) : null}
                {(draft.models || []).length > 0 ? (
                  <Button
                    size="small"
                    type="link"
                    onClick={() => onDraftChange({ models: [], default_model: '' })}
                  >
                    {t(FIELDS_CLEAR_KEY, { defaultValue: '清空' })}
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
            placeholder={t(FIELDS_AVAILABLE_MODELS_PLACEHOLDER_KEY, { defaultValue: '选择或手动输入 model id 后回车添加（默认空白）' })}
            tokenSeparators={[',', ' ', '\n']}
            notFoundContent={
              <span className="text-xs text-gray-500">{t(FIELDS_AVAILABLE_MODELS_EMPTY_KEY, { defaultValue: '直接输入 model id 后回车即可加入' })}</span>
            }
          />
          <p className="text-xs text-gray-500 mt-1.5">
            {t(FIELDS_AVAILABLE_MODELS_TIP_KEY, { defaultValue: '提示：删除某个模型不会立即清空 /ai/agents 中已绑定它的 Agent，但会将它从下拉建议中移除。' })}
          </p>
        </div>
      </div>
    </Section>
  );
}
