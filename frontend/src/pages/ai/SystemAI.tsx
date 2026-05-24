import { useCallback, useMemo } from 'react'
import {
  Bot,
  RefreshCw,
  Check,
  AlertCircle,
  ExternalLink,
  Sparkles,
  Zap,
  ShieldCheck,
  Link2,
  Cpu,
  Eraser,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  Input,
  Button,
  Alert,
  Space,
  Switch,
  Slider,
  Checkbox,
  InputNumber,
  Tooltip,
  Empty,
  Select,
} from 'antd'

import { ALL_PURPOSES, PROVIDER_LINKS } from './systemai/constants'
import { useSystemAIPage } from './systemai/hooks'
import DefaultPrimaryModelCard from './components/DefaultPrimaryModelCard'

type ProviderMeta = {
  label: string
  tagline: string
  icon: React.ComponentType<{ className?: string }>
}

const PROVIDER_META: Record<string, ProviderMeta> = {
  // 注：label / tagline 都只是「i18n 缺失时的兜底字符串」，因此采用中性英文，
  //     避免英文 UI 仍漏出中文（之前 qwen 显示 "通义千问" 就是这个 bug）。
  openai: { label: 'OpenAI', tagline: 'GPT series · Official', icon: Sparkles },
  anthropic: { label: 'Anthropic', tagline: 'Claude family', icon: ShieldCheck },
  deepseek: { label: 'DeepSeek', tagline: 'DeepSeek · Cost-efficient', icon: Zap },
  moonshot: { label: 'Moonshot', tagline: 'Kimi · Long context', icon: Cpu },
  qwen: { label: 'Qwen', tagline: 'Alibaba Cloud · CN-optimised', icon: Sparkles },
  zhipu: { label: 'Zhipu AI', tagline: 'Tsinghua-affiliated · General', icon: Bot },
  openai_compatible: { label: 'Custom (OpenAI-compatible)', tagline: 'Any OpenAI-compatible endpoint', icon: Link2 },
}

function metaOf(providerId: string, fallbackName: string): ProviderMeta {
  return (
    PROVIDER_META[providerId] || {
      label: fallbackName || providerId,
      tagline: '',
      icon: Bot,
    }
  )
}

// 解析 provider 显示标签：优先走 i18n（ai.settings.providers.<id>），
// 找不到时退回 PROVIDER_META.label，最后退回到 db name / provider_id。
function useProviderLabel() {
  const { t } = useTranslation()
  return useCallback((providerId: string, fallbackName?: string) => {
    const custom = providerId === 'openai_compatible' || providerId.startsWith('openai_compatible_')
    if (custom && fallbackName?.trim()) return fallbackName
    const key = `ai.settings.providers.${custom ? 'openai_compatible' : providerId}`
    const tr = t(key as any)
    if (tr && tr !== key) return tr as string
    return PROVIDER_META[custom ? 'openai_compatible' : providerId]?.label || fallbackName || providerId
  }, [t])
}

// Provider tagline (i18n)：优先 ai.systemAI.taglines.<id>，缺失时回落到内置中文 tagline。
// 之所以不直接把 tagline 全删了走 i18n key — 是为了在 i18n 资源未加载或漏配时仍有可读文案。
function useProviderTagline() {
  const { t } = useTranslation()
  return useCallback((providerId: string) => {
    const key = `ai.systemAI.taglines.${providerId}`
    const tr = t(key as any)
    if (tr && tr !== key) return tr as string
    return PROVIDER_META[providerId]?.tagline || ''
  }, [t])
}

export default function SystemAI() {
  const { t } = useTranslation()
  const providerLabel = useProviderLabel()
  const providerTagline = useProviderTagline()
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
    validateConnection,
    discoveredModels,
    startNewCustomProviderDraft,
  } = useSystemAIPage()

  const hasSecret = !!(secretInput.trim() || draft?.has_secret)
  const isCustomProvider = (providerId: string) => providerId === 'openai_compatible' || providerId.startsWith('openai_compatible_')

  void validateConnection

  const urlDiagnostics = useMemo(() => {
    const value = (draft?.base_url || '').trim()
    if (!value) return { ok: false, https: false }
    try {
      const u = new URL(value)
      return { ok: u.protocol === 'http:' || u.protocol === 'https:', https: u.protocol === 'https:' }
    } catch {
      return { ok: false, https: false }
    }
  }, [draft?.base_url])

  const overallStatus: { tone: 'success' | 'warning' | 'error' | 'info'; title: string; desc: string } = useMemo(() => {
    if (!draft) return { tone: 'info', title: t('ai.systemAI.status.noProvider', { defaultValue: '尚未选择厂商' }), desc: t('ai.systemAI.status.noProviderDesc', { defaultValue: '请从下方卡片挑选一个模型厂商开始配置' }) }
    if (error) return { tone: 'error', title: t('ai.systemAI.status.error', { defaultValue: '存在异常' }), desc: error }
    const modelCount = (draft.models || []).length
    if (validated && draft.enabled) {
      const summary = modelCount > 0
        ? `${providerLabel(draft.provider_id, draft.name)} · ${modelCount} ${t('ai.settings.providers.modelsUnit', { defaultValue: '个模型' })}`
        : `${providerLabel(draft.provider_id, draft.name)}`
      return { tone: 'success', title: t('ai.systemAI.status.ready', { defaultValue: '运行就绪' }), desc: `${summary} ${t('ai.systemAI.status.readyDesc', { defaultValue: '已启用并连接正常' })}` }
    }
    if (validated) return { tone: 'warning', title: t('ai.systemAI.status.notEnabled', { defaultValue: '连接正常，尚未启用' }), desc: t('ai.systemAI.status.notEnabledDesc', { defaultValue: '打开「启用」开关即可投入使用' }) }
    if (hasSecret && urlDiagnostics.ok) return { tone: 'info', title: t('ai.systemAI.status.configReady', { defaultValue: '配置已就绪' }), desc: t('ai.systemAI.status.configReadyDesc', { defaultValue: '添加可用模型后系统将自动完成连通性检测' }) }
    if (hasSecret) return { tone: 'warning', title: t('ai.systemAI.status.checkUrl', { defaultValue: '请检查 Base URL' }), desc: t('ai.systemAI.status.checkUrlDesc', { defaultValue: 'API Key 已就绪，但地址似乎无效' }) }
    return { tone: 'info', title: t('ai.systemAI.status.needKey', { defaultValue: '请完成密钥配置' }), desc: t('ai.systemAI.status.needKeyDesc', { defaultValue: '填写 API Key 后将自动发现模型列表' }) }
  }, [draft, error, validated, hasSecret, urlDiagnostics.ok, t, providerLabel])

  const selectedMeta = draft ? metaOf(draft.provider_id, draft.name) : null
  const defaultCustomConfig = configs.find((cfg) => cfg.provider_id === 'openai_compatible') || null
  const defaultCustomConfigured = !!defaultCustomConfig && (
    !!defaultCustomConfig.base_url.trim() ||
    defaultCustomConfig.has_secret ||
    (defaultCustomConfig.models || []).length > 0 ||
    defaultCustomConfig.enabled
  )
  const providerCards = useMemo(() => {
    const cards = configs.filter((cfg) => defaultCustomConfigured || cfg.provider_id !== 'openai_compatible')
    return [
      ...cards,
      {
        provider_id: '__new_openai_compatible__',
        name: '',
        base_url: '',
        organization: '',
        models: [],
        default_model: '',
        temperature: 0.2,
        timeout_seconds: 300,
        max_tokens: 4096,
        purposes: [],
        primary_for: [],
        enabled: false,
        has_secret: false,
        updated_at: '',
      },
    ]
  }, [configs, defaultCustomConfigured])
  return (
    <div className="space-y-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 flex items-center gap-2">
            <Bot className="w-6 h-6 text-slate-700" /> {t('ai.systemAI.pageTitle', { defaultValue: '系统 AI 配置' })}
          </h1>
          <p className="text-sm text-gray-500 mt-1">{t('ai.systemAI.pageSubtitle', { defaultValue: '统一管理大模型服务商、API 密钥与可用模型；支持 OpenAI 协议兼容端点。' })}</p>
        </div>
        <Button icon={<RefreshCw className="w-4 h-4" />} onClick={load} loading={loading}>
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
          <RefreshCw className="w-8 h-8 animate-spin mx-auto text-slate-600" />
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
        <Section
          step={1}
          title={t('ai.systemAI.section1.title', { defaultValue: '选择模型厂商' })}
          subtitle={t('ai.systemAI.section1.subtitle', { defaultValue: '卡片直接展示每个厂商的配置与就绪状态，点击选择' })}
        >
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
              {providerCards.map((cfg) => {
                const isNewCustomCard = cfg.provider_id === '__new_openai_compatible__'
                const m = isNewCustomCard ? metaOf('openai_compatible', '') : metaOf(cfg.provider_id, cfg.name)
                const Icon = m.icon
                const cfgModelCount = (cfg.models || []).length
                const ready = cfg.has_secret && cfgModelCount > 0
                const isSelected = cfg.provider_id === selectedProviderId
                const stateLabel = !cfg.has_secret
                  ? t('ai.systemAI.cardState.noKey', { defaultValue: '未配置' })
                  : cfgModelCount === 0
                    ? t('ai.systemAI.cardState.noModel', { defaultValue: '待选模型' })
                    : cfg.enabled
                      ? t('ai.systemAI.cardState.enabled', { defaultValue: '已启用' })
                      : t('ai.systemAI.cardState.readyDisabled', { defaultValue: '已就绪 · 未启用' })
                return (
                  <button
                    key={cfg.provider_id}
                    type="button"
                    onClick={() => {
                      if (isNewCustomCard) {
                        startNewCustomProviderDraft()
                        return
                      }
                      if (cfg.provider_id === selectedProviderId) return
                      setSelectedProviderId(cfg.provider_id)
                      setValidated(false)
                    }}
                    className="text-left rounded-lg border p-3 transition-all hover:shadow-sm"
                    style={{
                      backgroundColor: isSelected ? 'rgba(212, 175, 55, 0.08)' : '#FFFFFF',
                      borderColor: isSelected ? '#D4AF37' : '#E5E7EB',
                      borderWidth: isSelected ? 2 : 1,
                    }}
                  >
                    <div className="flex items-center gap-3">
                      <div
                        className="w-9 h-9 rounded-md flex items-center justify-center border shrink-0"
                        style={{
                          backgroundColor: 'rgba(212, 175, 55, 0.08)',
                          borderColor: 'rgba(212, 175, 55, 0.35)',
                          color: '#B8960B',
                        }}
                      >
                        <Icon className="w-4 h-4 text-[#B8960B]" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-gray-900 truncate">{isNewCustomCard ? providerLabel('openai_compatible') : providerLabel(cfg.provider_id, cfg.name)}</span>
                          {isSelected && !isNewCustomCard && <SoftTag>{t('ai.systemAI.cardTags.current', { defaultValue: '当前' })}</SoftTag>}
                        </div>
                        <div className="text-xs text-gray-500 truncate">{providerTagline(isNewCustomCard ? 'openai_compatible' : cfg.provider_id)}</div>
                      </div>
                      <SoftTag>{isNewCustomCard ? t('ai.systemAI.cardState.noKey', { defaultValue: '未配置' }) : stateLabel}</SoftTag>
                    </div>
                    <div className="mt-2 flex items-center gap-1.5 flex-wrap text-xs">
                      <SoftTag>
                        {cfg.has_secret
                          ? t('ai.systemAI.cardTags.hasKey', { defaultValue: '已配密钥' })
                          : t('ai.systemAI.cardTags.noKey', { defaultValue: '未配密钥' })}
                      </SoftTag>
                      <SoftTag>
                        {cfgModelCount > 0
                          ? `${t('ai.settings.fields.availableModels', { defaultValue: '可用模型' })}: ${cfgModelCount}`
                          : t('ai.systemAI.cardTags.noModels', { defaultValue: '未配置可用模型' })}
                      </SoftTag>
                      {!ready && cfg.enabled && (
                        <SoftTag>{t('ai.systemAI.cardTags.enabledButUnavailable', { defaultValue: '启用但不可用' })}</SoftTag>
                      )}
                    </div>
                  </button>
                )
              })}
            </div>
          </Section>
      )}

      {!loading && draft && selectedMeta && (
        <>
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
                  <ExternalLink className="w-3 h-3" /> {t('ai.settings.sections.connectionApiKeyLink', { defaultValue: '前往申请 / 管理该厂商 API Key' })}
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
                    onChange={(e) => setDraft({ ...draft, name: e.target.value })}
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
                    if (!isCustomProvider(draft.provider_id)) return
                    setDraft({ ...draft, base_url: e.target.value })
                    setValidated(false)
                    setLastAutoDiscoverKey('')
                  }}
                  placeholder={
                    isCustomProvider(draft.provider_id)
                      ? t('ai.systemAI.fields.baseUrlCustomPlaceholder', { defaultValue: '例如: https://model.example.com/v1' })
                      : t('ai.systemAI.fields.baseUrlReadonlyPlaceholder', { defaultValue: '官方地址（只读）' })
                  }
                  disabled={!isCustomProvider(draft.provider_id)}
                />
                {draft.base_url && !urlDiagnostics.https && (
                  <p className="text-xs text-slate-600 flex items-center gap-1 mt-1.5">
                    <AlertCircle className="w-3.5 h-3.5" /> {t('ai.systemAI.fields.httpWarning', { defaultValue: '当前为 HTTP，生产环境建议使用 HTTPS' })}
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
                    onChange={(e) => setSecretInput(e.target.value)}
                    placeholder={draft.has_secret
                      ? t('ai.settings.fields.apiKeyReplaceHint', { defaultValue: '如需更换密钥，请重新输入' })
                      : t('ai.systemAI.fields.apiKeyPastePlaceholder', { defaultValue: '粘贴 API Key，将自动预保存' })}
                  />
                  <Button
                    size="large"
                    icon={<Eraser className="w-4 h-4" />}
                    onClick={clearSecret}
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
                          <RefreshCw className="w-3 h-3 animate-spin" /> {t('ai.systemAI.fields.autoFetching', { defaultValue: '自动拉取中' })}
                        </span>
                      ) : null}
                      {(draft.models || []).length > 0 ? (
                        <Button
                          size="small"
                          type="link"
                          onClick={() => {
                            setDraft({ ...draft, models: [], default_model: '' })
                            setValidated(false)
                          }}
                        >
                          {t('ai.settings.fields.clear', { defaultValue: '清空' })}
                        </Button>
                      ) : null}
                    </Space>
                  )}
                />
                {/* mode='tags' 允许下拉选择 + 手动键入。
                    value 严格等于 draft.models（用户策划清单），不会自动注入 discoveredModels。
                    保持向后兼容：保存时同步写回 default_model = models[0]（后端 system:<provider>
                    无 model 子串时仍能解析出一个可用 model）。 */}
                <Select
                  size="large"
                  mode="tags"
                  value={(draft.models || [])}
                  onChange={(vals) => {
                    const cleaned = Array.from(new Set((vals as string[]).map((v) => (v || '').trim()).filter(Boolean)))
                    // 保持 default_model 与 models[0] 一致（不再向用户暴露此字段）。
                    setDraft({ ...draft, models: cleaned, default_model: cleaned[0] || '' })
                    setValidated(false)
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

          <Section
            step={3}
            title={t('ai.settings.sections.advanced', { defaultValue: '高级参数' })}
            subtitle={t('ai.settings.sections.advancedHint', { defaultValue: '仅在了解含义时调整；默认值已适配大多数场景' })}
          >
            <div className="space-y-6">
                <div>
                  <Label text={t('ai.settings.fields.enabledStatus', { defaultValue: '启用状态' })} hint={t('ai.systemAI.fields.enabledHint', { defaultValue: '关闭后该厂商不参与系统路由' })} />
                  <label className="flex items-center gap-3 cursor-pointer select-none">
                    <Switch checked={draft.enabled} onChange={(v) => setDraft({ ...draft, enabled: v })} />
                    <span className="text-sm text-gray-700">
                      {draft.enabled ? (
                        <span className="text-slate-700 font-medium">{t('ai.settings.fields.enabledOn', { defaultValue: '已启用 → 点击关闭' })}</span>
                      ) : (
                        <span className="text-gray-500">{t('ai.settings.fields.enabledOff', { defaultValue: '未启用 → 点击开启' })}</span>
                      )}
                    </span>
                  </label>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div>
                    <Label text={`${t('ai.settings.fields.temperature', { defaultValue: 'Temperature' })}（${draft.temperature}）`} hint={t('ai.systemAI.fields.temperatureHint', { defaultValue: '越高越发散，越低越稳定' })} />
                    <Slider
                      min={0}
                      max={2}
                      step={0.1}
                      value={draft.temperature}
                      onChange={(v) => setDraft({ ...draft, temperature: Number(v) })}
                    />
                  </div>
                  <div>
                    <Label text={t('ai.settings.fields.timeoutSeconds', { defaultValue: 'Timeout（秒）' })} hint={t('ai.systemAI.fields.timeoutHint', { defaultValue: '单次请求最长等待时间' })} />
                    <InputNumber
                      value={draft.timeout_seconds}
                      min={1}
                      onChange={(v) => setDraft({ ...draft, timeout_seconds: Number(v || 0) })}
                      style={{ width: '100%' }}
                    />
                  </div>
                  <div>
                    <Label text={t('ai.settings.fields.maxTokens', { defaultValue: 'Max Tokens' })} hint={t('ai.systemAI.fields.maxTokensHint', { defaultValue: '单次响应最大 token 数' })} />
                    <InputNumber
                      value={draft.max_tokens}
                      min={1}
                      onChange={(v) => setDraft({ ...draft, max_tokens: Number(v || 0) })}
                      style={{ width: '100%' }}
                    />
                  </div>
                </div>

                <div>
                  <Label
                    text={t('ai.systemAI.fields.primaryFor', { defaultValue: '主要用途（Primary For）' })}
                    hint={t('ai.systemAI.fields.primaryForHint', { defaultValue: '仅用于服务内部路由：chat / embedding / summarizer / reasoning' })}
                  />
                  <Checkbox.Group
                    value={draft.primary_for}
                    onChange={(vals) => setDraft({ ...draft, primary_for: vals as string[] })}
                    options={ALL_PURPOSES.map((p) => ({ label: p, value: p }))}
                  />
                </div>
              </div>
          </Section>

          <Section
            step={4}
            title={t('ai.settings.actions.saveConfig', { defaultValue: '保存配置' })}
          >
            <div className="flex items-center justify-between">
              <div className="text-sm text-gray-600 flex items-center gap-2 flex-wrap">
                {draft.enabled
                  ? <SoftTag>{t('ai.systemAI.statusBar.enabled', { defaultValue: '已启用' })}</SoftTag>
                  : <SoftTag>{t('ai.systemAI.statusBar.disabled', { defaultValue: '未启用' })}</SoftTag>}
                {draft.has_secret && <SoftTag>{t('ai.systemAI.statusBar.keyReady', { defaultValue: '密钥就绪' })}</SoftTag>}
                {validating && <SoftTag>{t('ai.systemAI.statusBar.checking', { defaultValue: '连通性检测中…' })}</SoftTag>}
                {!validating && validated && <SoftTag>{t('ai.systemAI.statusBar.connected', { defaultValue: '连接正常' })}</SoftTag>}
                {!validating && !validated && (draft.models || []).length > 0 && error && (
                  <span className="text-xs text-slate-600">{t('ai.systemAI.status.connectionFailed', { defaultValue: '连接异常，请检查上方提示' })}</span>
                )}
              </div>
              <Button
                size="large"
                onClick={saveConfig}
                loading={savingConfig}
                icon={<Check className="w-4 h-4" />}
                type="primary"
              >
                {t('ai.settings.actions.saveConfig', { defaultValue: '保存配置' })}
              </Button>
            </div>
          </Section>
        </>
      )}
    </div>
  )
}

function Section({
  step,
  title,
  subtitle,
  children,
}: {
  step?: number
  title: string
  subtitle?: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <section className="bg-white rounded-xl shadow-sm border border-gray-100">
      <header className="flex items-start justify-between gap-4 px-6 py-4">
        <div className="flex items-start gap-3">
          {typeof step === 'number' && (
            <span
              className="w-7 h-7 rounded-full text-sm font-semibold flex items-center justify-center shrink-0"
              style={{ backgroundColor: '#D4AF37', border: '1px solid #D4AF37', color: '#FFFFFF' }}
            >
              {step}
            </span>
          )}
          <div>
            <h2 className="text-base font-semibold text-gray-900">{title}</h2>
            {subtitle && <div className="text-xs text-gray-500 mt-0.5">{subtitle}</div>}
          </div>
        </div>
      </header>
      <div className="px-6 pb-6 pt-2">{children}</div>
    </section>
  )
}

function Label({
  text,
  hint,
  badge,
}: {
  text: string
  hint?: string
  badge?: React.ReactNode
}) {
  return (
    <div className="flex items-center justify-between mb-1.5">
      <div className="flex items-center gap-1.5">
        <span className="text-sm font-medium text-gray-700">{text}</span>
        {hint && (
          <Tooltip title={hint}>
            <AlertCircle className="w-3.5 h-3.5 text-gray-400 cursor-help" />
          </Tooltip>
        )}
      </div>
      {badge}
    </div>
  )
}

function SoftTag({ children }: { children: React.ReactNode }) {
  return (
    <span
      className="inline-flex items-center rounded px-1.5 py-0.5 text-xs"
      style={{
        backgroundColor: 'rgba(212, 175, 55, 0.10)',
        border: '1px solid rgba(212, 175, 55, 0.32)',
        color: '#B8960B',
      }}
    >
      {children}
    </span>
  )
}

function StatusBanner({
  title,
  description,
  notice,
}: {
  tone: 'success' | 'warning' | 'error' | 'info'
  title: string
  description: string
  notice?: string
}) {
  return (
    <Alert
      type="info"
      showIcon
      className="ai-gold-alert"
      message={<span className="font-semibold">{title}</span>}
      description={
        <div className="space-y-1">
          <div>{description}</div>
          {notice && <div className="text-xs text-slate-600">{notice}</div>}
        </div>
      }
    />
  )
}
