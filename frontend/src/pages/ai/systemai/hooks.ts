import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  listSystemAIConfigs,
  updateSystemAIConfig,
} from './api';
import type { AIConfig } from './model';
import { useProviderSync } from './hooks/useProviderSync';
import { useProviderActions } from './hooks/useProviderActions';

export function useSystemAIPage() {
  const { t } = useTranslation();
  const mountedRef = useRef(true);
  const [configs, setConfigs] = useState<AIConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [savingConfig, setSavingConfig] = useState(false);
  const [savingSecret, setSavingSecret] = useState(false);
  const [selectedProviderId, setSelectedProviderId] = useState('');
  const [draft, setDraft] = useState<AIConfig | null>(null);
  const [secretInput, setSecretInput] = useState('');
  const [notice, setNotice] = useState('');
  const [error, setError] = useState('');
  const [validated, setValidated] = useState(false);
  const [validating, setValidating] = useState(false);
  const [discovering, setDiscovering] = useState(false);
  const [lastAutoDiscoverKey, setLastAutoDiscoverKey] = useState('');
  const [lastAutoSavedSecretKey, setLastAutoSavedSecretKey] = useState('');
  const [discoveredModels, setDiscoveredModels] = useState<string[]>([]);
  const [lastAutoValidateKey, setLastAutoValidateKey] = useState('');

  const isCustomProvider = (providerId: string) =>
    providerId === 'openai_compatible' || providerId.startsWith('openai_compatible_');

  const validateBaseURL = (value: string): string | null => {
    const input = value.trim();
    if (!input) return '__DISCOVER_BASE_URL_EMPTY__';
    let parsed: URL;
    try {
      parsed = new URL(input);
    } catch {
      return 'base url format invalid';
    }
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
      return 'base url format invalid';
    }
    return null;
  };

  const persistDraftConfig = async (cfg: AIConfig) => {
    if (isCustomProvider(cfg.provider_id) && !cfg.name.trim()) {
      throw new Error(t('ai.systemAI.customProvider.nameRequired', { defaultValue: '请先填写自定义厂商名称' }));
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
    });
  };

  const fetchConfigs = useCallback(async (): Promise<AIConfig[]> => {
    const json = await listSystemAIConfigs();
    const items = json.items || [];
    setConfigs(items);
    return items;
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      await fetchConfigs();
      if (!mountedRef.current) return;
    } catch {
      if (!mountedRef.current) return;
      setError(t('ai.systemAI.messages.loadConfigFailed'));
    } finally {
      if (mountedRef.current) setLoading(false);
    }
  }, [fetchConfigs, t]);

  const silentReload = useCallback(async () => {
    try {
      await fetchConfigs();
      if (!mountedRef.current) return;
    } catch {
      if (!mountedRef.current) return;
    }
  }, [fetchConfigs]);

  useEffect(() => { load(); }, [load]);
  useEffect(() => { return () => { mountedRef.current = false; }; }, []);

  const selectedConfig = useMemo(
    () => configs.find((c) => c.provider_id === selectedProviderId) || null,
    [configs, selectedProviderId],
  );

  // Provider synchronization effects (switch, auto-save, discover, validate)
  const { prevProviderIdRef } = useProviderSync({
    selectedConfig,
    draft,
    secretInput,
    selectedProviderId,
    lastAutoSavedSecretKey,
    lastAutoDiscoverKey,
    lastAutoValidateKey,
    setDraft,
    setSecretInput,
    setNotice,
    setError,
    setValidated,
    setSavingSecret,
    setDiscovering,
    setValidating,
    setLastAutoSavedSecretKey,
    setLastAutoDiscoverKey,
    setLastAutoValidateKey,
    setDiscoveredModels,
    isCustomProvider,
    validateBaseURL,
    persistDraftConfig,
    silentReload,
  });

  // Provider action handlers (save, clear, validate, etc.)
  const { saveConfig, startNewCustomProviderDraft, setEnabled, clearSecret, validateConnection } = useProviderActions({
    draft,
    configs,
    setConfigs,
    setDraft,
    setSavingConfig,
    setSavingSecret,
    setSecretInput,
    setNotice,
    setError,
    setValidated,
    setValidating,
    setSelectedProviderId,
    setLastAutoSavedSecretKey,
    setLastAutoDiscoverKey,
    setDiscoveredModels,
    prevProviderIdRef,
    secretInput,
    silentReload,
    isCustomProvider,
    validateBaseURL,
    persistDraftConfig,
  });

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
  };
}
