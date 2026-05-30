import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  clearSystemAISecret,
  discoverSystemAIModels,
  updateSystemAIConfig,
  updateSystemAISecret,
  validateSystemAI,
} from '../api';
import { OFFICIAL_PROVIDER_BASE_URLS, toFriendlyDiscoverMessage } from '../constants';
import type { AIConfig } from '../model';

interface UseProviderActionsParams {
  draft: AIConfig | null;
  configs: AIConfig[];
  setConfigs: React.Dispatch<React.SetStateAction<AIConfig[]>>;
  setDraft: React.Dispatch<React.SetStateAction<AIConfig | null>>;
  setSavingConfig: React.Dispatch<React.SetStateAction<boolean>>;
  setSavingSecret: React.Dispatch<React.SetStateAction<boolean>>;
  setSecretInput: React.Dispatch<React.SetStateAction<string>>;
  setNotice: React.Dispatch<React.SetStateAction<string>>;
  setError: React.Dispatch<React.SetStateAction<string>>;
  setValidated: React.Dispatch<React.SetStateAction<boolean>>;
  setSelectedProviderId: React.Dispatch<React.SetStateAction<string>>;
  setLastAutoSavedSecretKey: React.Dispatch<React.SetStateAction<string>>;
  setLastAutoDiscoverKey: React.Dispatch<React.SetStateAction<string>>;
  setDiscoveredModels: React.Dispatch<React.SetStateAction<string[]>>;
  setValidating: React.Dispatch<React.SetStateAction<boolean>>;
  prevProviderIdRef: React.MutableRefObject<string>;
  secretInput: string;
  silentReload: () => Promise<void>;
  isCustomProvider: (providerId: string) => boolean;
  validateBaseURL: (value: string) => string | null;
  persistDraftConfig: (cfg: AIConfig) => Promise<void>;
}

export function useProviderActions(params: UseProviderActionsParams) {
  const { t } = useTranslation();
  const {
    draft, configs, setConfigs, setDraft, setSavingConfig, setSavingSecret,
    setSecretInput, setNotice, setError, setValidated, setSelectedProviderId,
    setLastAutoSavedSecretKey, setLastAutoDiscoverKey, setDiscoveredModels,
    setValidating,
    prevProviderIdRef, secretInput, silentReload, isCustomProvider, validateBaseURL, persistDraftConfig,
  } = params;

  const saveConfig = useCallback(async () => {
    if (!draft) return;
    setSavingConfig(true);
    try {
      await persistDraftConfig(draft);
      setConfigs((prev) => {
        const exists = prev.some((item) => item.provider_id === draft.provider_id);
        if (exists) {
          return prev.map((item) => item.provider_id === draft.provider_id ? draft : item);
        }
        return [...prev, draft];
      });
      setNotice(t('ai.systemAI.messages.configSaved'));
      setError('');
      void silentReload();
    } catch (e) {
      const msg = e instanceof Error ? e.message : t('ai.systemAI.messages.configSaveFailed');
      setError(msg);
      throw e;
    } finally {
      setSavingConfig(false);
    }
  }, [draft, t, persistDraftConfig, setSavingConfig, setConfigs, setNotice, setError, silentReload]);

  const startNewCustomProviderDraft = useCallback(() => {
    const providerId = `openai_compatible_${Date.now().toString(36)}`;
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
    };
    prevProviderIdRef.current = providerId;
    setConfigs((prev) => prev.some((item) => item.provider_id === providerId) ? prev : [...prev, cfg]);
    setSelectedProviderId(providerId);
    setDraft(cfg);
    setSecretInput('');
    setDiscoveredModels([]);
    setValidated(false);
    setNotice(t('ai.systemAI.customProvider.fillNameFirst', { defaultValue: '请先填写厂商名称，再保存这个自定义模型服务。' }));
    setError('');
  }, [t, prevProviderIdRef, setConfigs, setSelectedProviderId, setDraft, setSecretInput, setDiscoveredModels, setValidated, setNotice, setError]);

  const setEnabled = useCallback(async (next: boolean) => {
    if (!draft) return;
    const optimistic = { ...draft, enabled: next };
    setDraft(optimistic);
    setSavingConfig(true);
    try {
      await persistDraftConfig(optimistic);
      setNotice(next ? t('ai.settings.messages.enabled') : t('ai.settings.messages.disabled'));
      setError('');
      void silentReload();
    } catch (e) {
      setDraft((prev) => prev ? { ...prev, enabled: !next } : prev);
      const msg = e instanceof Error ? e.message : t('ai.systemAI.messages.toggleEnabledFailed');
      setError(msg);
    } finally {
      setSavingConfig(false);
    }
  }, [draft, t, persistDraftConfig, setDraft, setSavingConfig, setNotice, setError, silentReload]);

  const clearSecret = useCallback(async () => {
    if (!draft) return;
    setSavingSecret(true);
    const removedProviderId = draft.provider_id;
    const removeCustomProvider = removedProviderId.startsWith('openai_compatible_');
    const removeLocalCustomProvider = () => {
      const nextConfigs = configs.filter((cfg) => cfg.provider_id !== removedProviderId);
      const nextSelected = nextConfigs.find((cfg) => cfg.provider_id === 'openai_compatible') || nextConfigs[0] || null;
      setConfigs(nextConfigs);
      setSelectedProviderId(nextSelected?.provider_id || '');
      setDraft(nextSelected);
      setSecretInput('');
      setLastAutoSavedSecretKey('');
      setLastAutoDiscoverKey('');
      setDiscoveredModels([]);
      setNotice(t('ai.systemAI.customProvider.deleted', { defaultValue: '自定义厂商已删除' }));
      setError('');
      setValidated(false);
      void silentReload();
    };
    try {
      await clearSystemAISecret(removedProviderId);
      if (removeCustomProvider) {
        removeLocalCustomProvider();
        return;
      }
      const resetBaseURL = OFFICIAL_PROVIDER_BASE_URLS[draft.provider_id] || '';
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
      });
      setSecretInput('');
      setLastAutoSavedSecretKey('');
      setLastAutoDiscoverKey('');
      setDiscoveredModels([]);
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
      } : prev);
      setNotice(t('ai.systemAI.messages.secretDeletedConfigReset'));
      setError('');
      setValidated(false);
      void silentReload();
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      if (removeCustomProvider && (msg.includes('404') || msg.toLowerCase().includes('not found'))) {
        removeLocalCustomProvider();
        return;
      }
      setError(msg || t('ai.systemAI.messages.deleteSecretFailed'));
    } finally {
      setSavingSecret(false);
    }
  }, [draft, configs, t, setSavingSecret, setConfigs, setSelectedProviderId, setDraft, setSecretInput, setLastAutoSavedSecretKey, setLastAutoDiscoverKey, setDiscoveredModels, setNotice, setError, setValidated, silentReload]);

  const validateConnection = useCallback(async () => {
    if (!draft) return;
    setValidating(true);
    try {
      const baseError = validateBaseURL(draft.base_url);
      if (baseError) {
        setValidated(false);
        setError(toFriendlyDiscoverMessage(baseError, t));
        return;
      }
      await persistDraftConfig(draft);
      if (secretInput.trim()) {
        await updateSystemAISecret(draft.provider_id, secretInput.trim());
      }
      const body = await validateSystemAI(draft.provider_id);
      setValidated(true);
      setNotice(t('ai.systemAI.messages.validationPassedModels', { count: body.model_count ?? 0 }));
      setError('');
    } catch (e) {
      const msg = e instanceof Error ? e.message : t('ai.settings.messages.validateFailed');
      setValidated(false);
      if (msg.includes('401/403') && !draft.has_secret && !secretInput.trim()) {
        setError(t('ai.systemAI.messages.validationFailedNeedApiKey'));
      } else {
        setError(toFriendlyDiscoverMessage(msg, t));
      }
    } finally {
      setValidating(false);
    }
  }, [draft, secretInput, t, validateBaseURL, persistDraftConfig, setValidating, setValidated, setError, setNotice]);

  return {
    saveConfig,
    startNewCustomProviderDraft,
    setEnabled,
    clearSecret,
    validateConnection,
  };
}
