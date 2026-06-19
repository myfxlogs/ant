import { message } from 'antd';
import { useCallback } from 'react';
import { useTranslation } from 'react-i18next'
import { MESSAGES_DISABLED_KEY, MESSAGES_ENABLED_KEY, MESSAGES_VALIDATE_FAILED_KEY } from '@/gen/ant/v1/i18n/ai_settings_keys';
import { SYSTEM_A_I_CUSTOM_PROVIDER_DELETED_KEY, SYSTEM_A_I_CUSTOM_PROVIDER_FILL_NAME_FIRST_KEY, SYSTEM_A_I_MESSAGES_CONFIG_SAVED_KEY, SYSTEM_A_I_MESSAGES_CONFIG_SAVE_FAILED_KEY, SYSTEM_A_I_MESSAGES_DELETE_SECRET_FAILED_KEY, SYSTEM_A_I_MESSAGES_SECRET_DELETED_CONFIG_RESET_KEY, SYSTEM_A_I_MESSAGES_TOGGLE_ENABLED_FAILED_KEY, SYSTEM_A_I_MESSAGES_VALIDATION_FAILED_NEED_API_KEY_KEY, SYSTEM_A_I_MESSAGES_VALIDATION_PASSED_MODELS_KEY } from '@/gen/ant/v1/i18n/ai_core_keys';

;
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
      setNotice(t(SYSTEM_A_I_MESSAGES_CONFIG_SAVED_KEY));
      setError('');
      void silentReload();
    } catch (e) {
      const msg = e instanceof Error ? e.message : t(SYSTEM_A_I_MESSAGES_CONFIG_SAVE_FAILED_KEY);
      message.error(msg, 3);
      setError(msg);
      console.error('saveConfig failed', e);
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
    setNotice(t(SYSTEM_A_I_CUSTOM_PROVIDER_FILL_NAME_FIRST_KEY, { defaultValue: '请先填写厂商名称，再保存这个自定义模型服务。' }));
    setError('');
  }, [t, prevProviderIdRef, setConfigs, setSelectedProviderId, setDraft, setSecretInput, setDiscoveredModels, setValidated, setNotice, setError]);

  const setEnabled = useCallback(async (next: boolean) => {
    if (!draft) return;
    const optimistic = { ...draft, enabled: next };
    setDraft(optimistic);
    setSavingConfig(true);
    try {
      await persistDraftConfig(optimistic);
      setNotice(next ? t(MESSAGES_ENABLED_KEY) : t(MESSAGES_DISABLED_KEY));
      setError('');
      void silentReload();
    } catch (e) {
      setDraft((prev) => prev ? { ...prev, enabled: !next } : prev);
      const msg = e instanceof Error ? e.message : t(SYSTEM_A_I_MESSAGES_TOGGLE_ENABLED_FAILED_KEY);
      message.error(msg, 3);
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
      setNotice(t(SYSTEM_A_I_CUSTOM_PROVIDER_DELETED_KEY, { defaultValue: '自定义厂商已删除' }));
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
      setNotice(t(SYSTEM_A_I_MESSAGES_SECRET_DELETED_CONFIG_RESET_KEY));
      setError('');
      setValidated(false);
      void silentReload();
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      if (removeCustomProvider && (msg.toLowerCase().includes('status 404') || msg.toLowerCase().includes('not found'))) {
        removeLocalCustomProvider();
        return;
      }
      setError(msg || t(SYSTEM_A_I_MESSAGES_DELETE_SECRET_FAILED_KEY));
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
      setNotice(t(SYSTEM_A_I_MESSAGES_VALIDATION_PASSED_MODELS_KEY, { count: body.model_count ?? 0 }));
      setError('');
    } catch (e) {
      const msg = e instanceof Error ? e.message : t(MESSAGES_VALIDATE_FAILED_KEY);
      setValidated(false);
      if (msg.includes('401/403') && !draft.has_secret && !secretInput.trim()) {
        setError(t(SYSTEM_A_I_MESSAGES_VALIDATION_FAILED_NEED_API_KEY_KEY));
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
