import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  clearSystemAISecret,
  discoverSystemAIModels,
  listSystemAIConfigs,
  updateSystemAIConfig,
  updateSystemAISecret,
  validateSystemAI,
} from './api'
import { OFFICIAL_PROVIDER_BASE_URLS, toFriendlyDiscoverMessage } from './constants'
import type { AIConfig } from './model'

export function useSystemAIPage() {
  const { t } = useTranslation()
  const mountedRef = useRef(true)
  const [configs, setConfigs] = useState<AIConfig[]>([])
  const [loading, setLoading] = useState(true)
  const [savingConfig, setSavingConfig] = useState(false)
  const [savingSecret, setSavingSecret] = useState(false)
  const [selectedProviderId, setSelectedProviderId] = useState('')
  const [draft, setDraft] = useState<AIConfig | null>(null)
  const [secretInput, setSecretInput] = useState('')
  const [notice, setNotice] = useState('')
  const [error, setError] = useState('')
  const [validated, setValidated] = useState(false)
  const [validating, setValidating] = useState(false)
  const [discovering, setDiscovering] = useState(false)
  const [lastAutoDiscoverKey, setLastAutoDiscoverKey] = useState('')
  const [lastAutoSavedSecretKey, setLastAutoSavedSecretKey] = useState('')
  // 与 draft.models 严格分离：discoveredModels 仅作下拉「建议项」，
  // 不会写回 system_ai_configs.models（那是用户「已启用模型」的策划清单）。
  const [discoveredModels, setDiscoveredModels] = useState<string[]>([])

  const isCustomProvider = (providerId: string) =>
    providerId === 'openai_compatible' || providerId.startsWith('openai_compatible_')

  const validateBaseURL = (value: string): string | null => {
    const input = value.trim()
    if (!input) return '__DISCOVER_BASE_URL_EMPTY__'
    let parsed: URL
    try {
      parsed = new URL(input)
    } catch {
      return 'base url format invalid'
    }
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
      return 'base url format invalid'
    }
    return null
  }

  const persistDraftConfig = async (cfg: AIConfig) => {
    if (isCustomProvider(cfg.provider_id) && !cfg.name.trim()) {
      throw new Error(t('ai.systemAI.customProvider.nameRequired', { defaultValue: '请先填写自定义厂商名称' }))
    }
    await updateSystemAIConfig(cfg.provider_id, {
      name: cfg.name,
      base_url: cfg.base_url,
      organization: cfg.organization,
      models: cfg.models,
      default_model: cfg.default_model,
      temperature: cfg.temperature,
      timeout_seconds: cfg.timeout_seconds,
      max_tokens: cfg.max_tokens,
      purposes: cfg.purposes,
      primary_for: cfg.primary_for,
      enabled: cfg.enabled,
    })
  }

  const fetchConfigs = useCallback(async (): Promise<AIConfig[]> => {
    const json = await listSystemAIConfigs()
    const items = json.items || []
    setConfigs(items)
    return items
  }, [])

  const load = useCallback(async () => {
    setLoading(true)
    try {
      await fetchConfigs()
      if (!mountedRef.current) return
    } catch (err) {
      if (!mountedRef.current) return
      setError(t('ai.systemAI.messages.loadConfigFailed'))
    } finally {
      if (mountedRef.current) setLoading(false)
    }
  }, [fetchConfigs])

  // 静默刷新：不触发全局 loading，仅同步最新 configs
  const silentReload = async () => {
    try {
      await fetchConfigs()
    } catch (err) {
      // silently reloaded configs failed; state is unchanged
    }
  }

  useEffect(() => { load() }, [load])
  useEffect(() => { return () => { mountedRef.current = false } }, [])

  const selectedConfig = useMemo(
    () => configs.find((c) => c.provider_id === selectedProviderId) || null,
    [configs, selectedProviderId],
  )

  const prevProviderIdRef = useRef<string>('')
  useEffect(() => {
    const nextId = selectedConfig?.provider_id || ''
    const providerChanged = nextId !== prevProviderIdRef.current
    prevProviderIdRef.current = nextId

    if (!selectedConfig) {
      setDraft((prev) => prev?.provider_id === selectedProviderId ? prev : null)
    } else if (providerChanged) {
      const fixedBase = OFFICIAL_PROVIDER_BASE_URLS[selectedConfig.provider_id]
      const enforcedBase = isCustomProvider(selectedConfig.provider_id)
        ? (selectedConfig.base_url || '')
        : (fixedBase || '')
      setDraft({
        ...selectedConfig,
        base_url: enforcedBase,
      })
    } else {
      setDraft((prev) => (prev ? {
        ...prev,
        has_secret: selectedConfig.has_secret,
        updated_at: selectedConfig.updated_at,
        models: prev.models && prev.models.length > 0 ? prev.models : selectedConfig.models,
      } : prev))
    }

    if (providerChanged) {
      setSecretInput('')
      setNotice('')
      setError('')
      setValidated(false)
      setLastAutoSavedSecretKey('')
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedConfig])

  useEffect(() => {
    if (!draft) return
    const secret = secretInput.trim()
    if (!secret) return
    const key = `${draft.provider_id}|${secret}`
    if (key === lastAutoSavedSecretKey) return

    const timer = setTimeout(async () => {
      if (isCustomProvider(draft.provider_id) && !draft.name.trim()) {
        setError(t('ai.systemAI.customProvider.nameRequired', { defaultValue: '请先填写自定义厂商名称' }))
        return
      }
      setSavingSecret(true)
      try {
        await updateSystemAISecret(draft.provider_id, secret)
        setLastAutoSavedSecretKey(key)
        setError('')
        setValidated(false)
        setDraft((prev) => prev ? { ...prev, has_secret: true } : prev)
        setLastAutoDiscoverKey('')
        setNotice(t('ai.systemAI.messages.secretSavedAutoDiscover'))
        void silentReload()
      } catch (e) {
        const msg = e instanceof Error ? e.message : t('ai.systemAI.messages.secretAutoSaveFailed')
        setError(msg)
      } finally {
        setSavingSecret(false)
      }
    }, 700)
    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draft?.provider_id, secretInput, lastAutoSavedSecretKey])

  useEffect(() => {
    if (!draft) return
    const base = (draft.base_url || '').trim()
    if (!base) return
    if (!draft.has_secret && !secretInput.trim()) return
    const key = `${draft.provider_id}|${base}|${draft.has_secret ? 'saved' : 'pending'}`
    if (key === lastAutoDiscoverKey) return

    const timer = setTimeout(async () => {
      setDiscovering(true)
      try {
        const baseError = validateBaseURL(base)
        if (baseError) {
          setError(toFriendlyDiscoverMessage(baseError, t))
          return
        }
        await persistDraftConfig(draft)
        const body = await discoverSystemAIModels(draft.provider_id)
        const models = (body?.models || []) as string[]
        if (models.length > 0) {
          setDiscoveredModels(models)
          // 仅在用户尚未选定 default_model 时，用建议清单的首项填一个，
          // 避免覆盖手动键入的免费模型 id。
          setDraft((prev) => {
            if (!prev) return prev
            const next = { ...prev }
            if (!(next.default_model || '').trim()) {
              next.default_model = body?.default_model || models[0]
            }
            return next
          })
          setNotice(t('ai.systemAI.messages.autoDiscoveredModels', { count: models.length }))
          setError('')
          setValidated(false)
          setLastAutoDiscoverKey(key)
        }
      } catch (e) {
        const msg = e instanceof Error ? e.message : t('ai.settings.discoverErrors.generic')
        setError(toFriendlyDiscoverMessage(msg, t))
      } finally {
        setDiscovering(false)
      }
    }, 700)

    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draft?.provider_id, draft?.base_url, draft?.has_secret, secretInput, lastAutoDiscoverKey])

  const [lastAutoValidateKey, setLastAutoValidateKey] = useState('')
  useEffect(() => {
    if (!draft) return
    const model = (draft.default_model || '').trim()
    if (!model) return
    if (!(draft.has_secret || secretInput.trim())) return
    const baseError = validateBaseURL(draft.base_url)
    if (baseError) return
    const key = `${draft.provider_id}|${draft.base_url}|${model}`
    if (key === lastAutoValidateKey) return

    const timer = setTimeout(async () => {
      setLastAutoValidateKey(key)
      setValidating(true)
      try {
        await persistDraftConfig(draft)
        if (secretInput.trim()) {
          await updateSystemAISecret(draft.provider_id, secretInput.trim())
        }
        const body = await validateSystemAI(draft.provider_id)
        setValidated(true)
        setNotice(t('ai.systemAI.messages.autoValidatedModels', { count: body.model_count ?? 0 }))
        setError('')
      } catch (e) {
        const msg = e instanceof Error ? e.message : t('ai.settings.messages.validateFailed')
        setValidated(false)
        setError(toFriendlyDiscoverMessage(msg, t))
      } finally {
        setValidating(false)
      }
    }, 500)
    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draft?.provider_id, draft?.base_url, draft?.default_model, draft?.has_secret, secretInput, lastAutoValidateKey])

  const saveConfig = async () => {
    if (!draft) return
    setSavingConfig(true)
    try {
      await persistDraftConfig(draft)
      setConfigs((prev) => prev.some((item) => item.provider_id === draft.provider_id) ? prev.map((item) => item.provider_id === draft.provider_id ? draft : item) : [...prev, draft])
      setNotice(t('ai.systemAI.messages.configSaved'))
      setError('')
      void silentReload()
    } catch (e) {
      const msg = e instanceof Error ? e.message : t('ai.systemAI.messages.configSaveFailed')
      setError(msg)
      throw e
    } finally {
      setSavingConfig(false)
    }
  }

  const startNewCustomProviderDraft = () => {
    const providerId = `openai_compatible_${Date.now().toString(36)}`
    const cfg: AIConfig = {
      provider_id: providerId,
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
    }
    prevProviderIdRef.current = providerId
    setConfigs((prev) => prev.some((item) => item.provider_id === providerId) ? prev : [...prev, cfg])
    setSelectedProviderId(providerId)
    setDraft(cfg)
    setSecretInput('')
    setDiscoveredModels([])
    setValidated(false)
    setNotice(t('ai.systemAI.customProvider.fillNameFirst', { defaultValue: '请先填写厂商名称，再保存这个自定义模型服务。' }))
    setError('')
  }

  // setEnabled toggles the current draft's `enabled` and persists immediately.
  // Each provider has its own row in system_ai_configs, so flipping one provider
  // never touches the others — this lets users enable several providers in
  // sequence without losing the toggle on tab switches. Mirrors the secret /
  // model auto-save flows above.
  const setEnabled = async (next: boolean) => {
    if (!draft) return
    const optimistic = { ...draft, enabled: next }
    setDraft(optimistic)
    setSavingConfig(true)
    try {
      await persistDraftConfig(optimistic)
      setNotice(next ? t('ai.settings.messages.enabled') : t('ai.settings.messages.disabled'))
      setError('')
      void silentReload()
    } catch (e) {
      setDraft((prev) => prev ? { ...prev, enabled: !next } : prev)
      const msg = e instanceof Error ? e.message : t('ai.systemAI.messages.toggleEnabledFailed')
      setError(msg)
    } finally {
      setSavingConfig(false)
    }
  }

  const clearSecret = async () => {
    if (!draft) return
    setSavingSecret(true)
    const removedProviderId = draft.provider_id
    const removeCustomProvider = removedProviderId.startsWith('openai_compatible_')
    const removeLocalCustomProvider = () => {
      const nextConfigs = configs.filter((cfg) => cfg.provider_id !== removedProviderId)
      const nextSelected = nextConfigs.find((cfg) => cfg.provider_id === 'openai_compatible') || nextConfigs[0] || null
      setConfigs(nextConfigs)
      setSelectedProviderId(nextSelected?.provider_id || '')
      setDraft(nextSelected)
      setSecretInput('')
      setLastAutoSavedSecretKey('')
      setLastAutoDiscoverKey('')
      setDiscoveredModels([])
      setNotice(t('ai.systemAI.customProvider.deleted', { defaultValue: '自定义厂商已删除' }))
      setError('')
      setValidated(false)
      void silentReload()
    }
    try {
      await clearSystemAISecret(removedProviderId)
      if (removeCustomProvider) {
        removeLocalCustomProvider()
        return
      }
      const resetBaseURL = OFFICIAL_PROVIDER_BASE_URLS[draft.provider_id] || ''
      await updateSystemAIConfig(draft.provider_id, {
        name: draft.name,
        base_url: resetBaseURL,
        organization: '',
        models: [],
        default_model: '',
        temperature: 0.2,
        timeout_seconds: 300,
        max_tokens: 4096,
        purposes: draft.purposes || [],
        primary_for: [],
        enabled: false,
      })
      setSecretInput('')
      setLastAutoSavedSecretKey('')
      setLastAutoDiscoverKey('')
      setDiscoveredModels([])
      setDraft((prev) => prev ? {
        ...prev,
        base_url: resetBaseURL,
        organization: '',
        models: [],
        default_model: '',
        temperature: 0.2,
        timeout_seconds: 300,
        max_tokens: 4096,
        primary_for: [],
        enabled: false,
        has_secret: false,
      } : prev)
      setNotice(t('ai.systemAI.messages.secretDeletedConfigReset'))
      setError('')
      setValidated(false)
      void silentReload()
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      if (removeCustomProvider && (msg.includes('404') || msg.toLowerCase().includes('not found'))) {
        removeLocalCustomProvider()
        return
      }
      setError(msg || t('ai.systemAI.messages.deleteSecretFailed'))
    } finally {
      setSavingSecret(false)
    }
  }

  const validateConnection = async () => {
    if (!draft) return
    setValidating(true)
    try {
      const baseError = validateBaseURL(draft.base_url)
      if (baseError) {
        setValidated(false)
        setError(toFriendlyDiscoverMessage(baseError, t))
        return
      }
      await persistDraftConfig(draft)
      if (secretInput.trim()) {
        await updateSystemAISecret(draft.provider_id, secretInput.trim())
      }
      const body = await validateSystemAI(draft.provider_id)
      setValidated(true)
      setNotice(t('ai.systemAI.messages.validationPassedModels', { count: body.model_count ?? 0 }))
      setError('')
    } catch (e) {
      const msg = e instanceof Error ? e.message : t('ai.settings.messages.validateFailed')
      setValidated(false)
      if (msg.includes('401/403') && !draft.has_secret && !secretInput.trim()) {
        setError(t('ai.systemAI.messages.validationFailedNeedApiKey'))
      } else {
        setError(toFriendlyDiscoverMessage(msg, t))
      }
    } finally {
      setValidating(false)
    }
  }

  return {
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
    startNewCustomProviderDraft,
    setEnabled,
    clearSecret,
    validateConnection,
    discoveredModels,
  }
}
