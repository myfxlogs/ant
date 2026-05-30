import { useCallback, useMemo } from 'react';
import {
  RobotOutlined,
  ReloadOutlined,
  StarOutlined,
  ThunderboltOutlined,
  SafetyCertificateOutlined,
  LinkOutlined,
  ControlOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { Button, Empty } from 'antd';
import { useSystemAIPage } from './systemai/hooks';
import DefaultPrimaryModelCard from './components/DefaultPrimaryModelCard';
import type { ProviderMeta } from './systemai/types';
import { StatusBanner } from './systemai/components/SharedComponents';
import { ProviderCardsSection } from './systemai/components/ProviderCards';
import { ConnectionFormSection } from './systemai/components/ConnectionForm';
import { AdvancedSection, SaveSection } from './systemai/components/AdvancedForm';

const PROVIDER_META: Record<string, ProviderMeta> = {
  openai:          { label: 'OpenAI',                    tagline: 'GPT series · Official',              icon: StarOutlined },
  anthropic:       { label: 'Anthropic',                 tagline: 'Claude family',                       icon: SafetyCertificateOutlined },
  deepseek:        { label: 'DeepSeek',                  tagline: 'DeepSeek · Cost-efficient',           icon: ThunderboltOutlined },
  moonshot:        { label: 'Moonshot',                  tagline: 'Kimi · Long context',                 icon: ControlOutlined },
  qwen:            { label: 'Qwen',                      tagline: 'Alibaba Cloud · CN-optimised',        icon: StarOutlined },
  zhipu:           { label: 'Zhipu AI',                  tagline: 'Tsinghua-affiliated · General',       icon: RobotOutlined },
  openai_compatible:{ label: 'Custom (OpenAI-compatible)', tagline: 'Any OpenAI-compatible endpoint',     icon: LinkOutlined },
};

function metaOf(providerId: string, fallbackName: string): ProviderMeta {
  return PROVIDER_META[providerId] || { label: fallbackName || providerId, tagline: '', icon: RobotOutlined };
}

function useProviderLabel() {
  const { t } = useTranslation();
  return useCallback((providerId: string, fallbackName?: string) => {
    const custom = providerId === 'openai_compatible' || providerId.startsWith('openai_compatible_');
    if (custom && fallbackName?.trim()) return fallbackName;
    const key = `ai.settings.providers.${custom ? 'openai_compatible' : providerId}`;
    const tr = t(key as Parameters<typeof t>[0]);
    if (tr && tr !== key) return tr as string;
    return PROVIDER_META[custom ? 'openai_compatible' : providerId]?.label || fallbackName || providerId;
  }, [t]);
}

function useProviderTagline() {
  const { t } = useTranslation();
  return useCallback((providerId: string) => {
    const key = `ai.systemAI.taglines.${providerId}`;
    const tr = t(key as Parameters<typeof t>[0]);
    if (tr && tr !== key) return tr as string;
    return PROVIDER_META[providerId]?.tagline || '';
  }, [t]);
}

export default function SystemAI() {
  const { t } = useTranslation();
  const providerLabel = useProviderLabel();
  const providerTagline = useProviderTagline();
  const {
    configs,
    loading,
    savingConfig,
    savingSecret,
    selectedProviderId,
    setSelectedProviderId,
    draft,
    setDraft,
    secretInput,
    setSecretInput,
    notice,
    error,
    validated,
    setValidated,
    validating,
    discovering,
    setLastAutoDiscoverKey,
    load,
    saveConfig,
    clearSecret,
    discoveredModels,
    startNewCustomProviderDraft,
  } = useSystemAIPage();

  const hasSecret = !!(secretInput.trim() || draft?.has_secret);
  const isCustomProvider = (providerId: string) => providerId === 'openai_compatible' || providerId.startsWith('openai_compatible_');

  const urlDiagnostics = useMemo(() => {
    const value = (draft?.base_url || '').trim();
    if (!value) return { ok: false, https: false };
    try {
      const u = new URL(value);
      return { ok: u.protocol === 'http:' || u.protocol === 'https:', https: u.protocol === 'https:' };
    } catch {
      return { ok: false, https: false };
    }
  }, [draft?.base_url]);

  const overallStatus: { tone: 'success' | 'warning' | 'error' | 'info'; title: string; desc: string } = useMemo(() => {
    if (!draft) return { tone: 'info', title: t('ai.systemAI.status.noProvider', { defaultValue: '尚未选择厂商' }), desc: t('ai.systemAI.status.noProviderDesc', { defaultValue: '请从下方卡片挑选一个模型厂商开始配置' }) };
    if (error) return { tone: 'error', title: t('ai.systemAI.status.error', { defaultValue: '存在异常' }), desc: error };
    const modelCount = (draft.models || []).length;
    if (validated && draft.enabled) {
      const summary = modelCount > 0
        ? `${providerLabel(draft.provider_id, draft.name)} · ${modelCount} ${t('ai.settings.providers.modelsUnit', { defaultValue: '个模型' })}`
        : `${providerLabel(draft.provider_id, draft.name)}`;
      return { tone: 'success', title: t('ai.systemAI.status.ready', { defaultValue: '运行就绪' }), desc: `${summary} ${t('ai.systemAI.status.readyDesc', { defaultValue: '已启用并连接正常' })}` };
    }
    if (validated) return { tone: 'warning', title: t('ai.systemAI.status.notEnabled', { defaultValue: '连接正常，尚未启用' }), desc: t('ai.systemAI.status.notEnabledDesc', { defaultValue: '打开「启用」开关即可投入使用' }) };
    if (hasSecret && urlDiagnostics.ok) return { tone: 'info', title: t('ai.systemAI.status.configReady', { defaultValue: '配置已就绪' }), desc: t('ai.systemAI.status.configReadyDesc', { defaultValue: '添加可用模型后系统将自动完成连通性检测' }) };
    if (hasSecret) return { tone: 'warning', title: t('ai.systemAI.status.checkUrl', { defaultValue: '请检查 Base URL' }), desc: t('ai.systemAI.status.checkUrlDesc', { defaultValue: 'API Key 已就绪，但地址似乎无效' }) };
    return { tone: 'info', title: t('ai.systemAI.status.needKey', { defaultValue: '请完成密钥配置' }), desc: t('ai.systemAI.status.needKeyDesc', { defaultValue: '填写 API Key 后将自动发现模型列表' }) };
  }, [draft, error, validated, hasSecret, urlDiagnostics.ok, t, providerLabel]);

  const selectedMeta = draft ? metaOf(draft.provider_id, draft.name) : null;
  const customCfg = configs.find((cfg) => cfg.provider_id === 'openai_compatible');
  const customConfigured = !!customCfg && (!!(customCfg.base_url || '').trim() || customCfg.has_secret || (customCfg.models || []).length > 0 || customCfg.enabled);
  const newCustomCard = { provider_id: '__new_openai_compatible__', name: '', base_url: '', organization: '', models: [] as string[], default_model: '', temperature: 0.2, timeout_seconds: 300, max_tokens: 4096, purposes: [] as string[], primary_for: [] as string[], enabled: false, has_secret: false, updated_at: '' };
  const providerCards = useMemo(() => {
    const cards = configs.filter((cfg) => customConfigured || cfg.provider_id !== 'openai_compatible');
    return [...cards, newCustomCard];
  }, [configs, customConfigured]);

  const handleSelectProvider = (id: string) => { setSelectedProviderId(id); setValidated(false); };

  const handleDraftChange = (patch: Record<string, unknown>) => {
    if (!draft) return;
    const next = { ...draft, ...patch } as typeof draft;
    setDraft(next);
    if ('base_url' in patch || 'models' in patch) setValidated(false);
    if ('base_url' in patch) setLastAutoDiscoverKey('');
  };

  return (
    <div className="space-y-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 flex items-center gap-2">
            <RobotOutlined className="w-6 h-6 text-slate-700" /> {t('ai.systemAI.pageTitle', { defaultValue: '系统 AI 配置' })}
          </h1>
          <p className="text-sm text-gray-500 mt-1">{t('ai.systemAI.pageSubtitle', { defaultValue: '统一管理大模型服务商、API 密钥与可用模型；支持 OpenAI 协议兼容端点。' })}</p>
        </div>
        <Button icon={<ReloadOutlined className="w-4 h-4" />} onClick={load} loading={loading}>
          {t('common.refresh', { defaultValue: '刷新' })}
        </Button>
      </div>

      {!loading && configs.length > 0 && (
        <StatusBanner
          tone={overallStatus.tone}
          title={overallStatus.title}
          description={overallStatus.desc}
          notice={draft && !error ? notice : ''}
        />
      )}

      {loading && (
        <div className="text-center py-16 bg-white rounded-xl shadow-sm border border-gray-100">
          <ReloadOutlined className="w-8 h-8 animate-spin mx-auto text-slate-600" />
          <p className="text-gray-500 mt-3">{t('common.loading', { defaultValue: '加载中...' })}</p>
        </div>
      )}

      {!loading && configs.length === 0 && (
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-12">
          <Empty description={t('ai.systemAI.emptyConfigs', { defaultValue: '暂无 AI Provider 配置（系统启动时会自动创建默认 Provider）' })} />
        </div>
      )}

      {!loading && configs.length > 0 && (
        <DefaultPrimaryModelCard systemConfigs={configs} labelOf={providerLabel} />
      )}

      {!loading && configs.length > 0 && (
        <ProviderCardsSection
          providerCards={providerCards}
          selectedProviderId={selectedProviderId}
          providerLabel={providerLabel}
          providerTagline={providerTagline}
          metaOf={metaOf}
          onSelectProvider={handleSelectProvider}
          onNewCustomProvider={startNewCustomProviderDraft}
        />
      )}

      {!loading && draft && selectedMeta && (
        <>
          <ConnectionFormSection
            draft={draft}
            providerLabel={providerLabel}
            isCustomProvider={isCustomProvider}
            urlHttps={urlDiagnostics.https}
            urlOk={urlDiagnostics.ok}
            secretInput={secretInput}
            onSecretInputChange={setSecretInput}
            onDraftChange={handleDraftChange}
            onClearSecret={clearSecret}
            savingSecret={savingSecret}
            discovering={discovering}
            discoveredModels={discoveredModels}
          />

          <AdvancedSection
            draft={draft}
            onDraftChange={handleDraftChange}
          />

          <SaveSection
            draft={draft}
            savingConfig={savingConfig}
            validating={validating}
            validated={validated}
            hasError={!!error}
            onSave={saveConfig}
          />
        </>
      )}
    </div>
  );
}
