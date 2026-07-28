import { useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next'
import { DISCOVER_ERRORS_GENERIC_KEY, MESSAGES_VALIDATE_FAILED_KEY } from '@/gen/ant/v1/i18n/ai_settings_keys';
import { SYSTEM_A_I_CUSTOM_PROVIDER_NAME_REQUIRED_KEY, SYSTEM_A_I_MESSAGES_AUTO_DISCOVERED_MODELS_KEY, SYSTEM_A_I_MESSAGES_AUTO_VALIDATED_MODELS_KEY, SYSTEM_A_I_MESSAGES_SECRET_AUTO_SAVE_FAILED_KEY, SYSTEM_A_I_MESSAGES_SECRET_SAVED_AUTO_DISCOVER_KEY } from '@/gen/ant/v1/i18n/ai_core_keys';

;
import {
  discoverSystemAIModels,
  updateSystemAIConfig,
  updateSystemAISecret,
  validateSystemAI,
} from '../api';
import { OFFICIAL_PROVIDER_BASE_URLS, toFriendlyDiscoverMessage } from '../constants';
import type { AIConfig } from '../model';

interface UseProviderSyncParams {
  selectedConfig: AIConfig | null;
  draft: AIConfig | null;
  secretInput: string;
  selectedProviderId: string;
  lastAutoSavedSecretKey: string;
  lastAutoDiscoverKey: string;
  lastAutoValidateKey: string;
  setDraft: React.Dispatch<React.SetStateAction<AIConfig | null>>;
  setSecretInput: React.Dispatch<React.SetStateAction<string>>;
  setNotice: React.Dispatch<React.SetStateAction<string>>;
  setError: React.Dispatch<React.SetStateAction<string>>;
  setValidated: React.Dispatch<React.SetStateAction<boolean>>;
  setSavingSecret: React.Dispatch<React.SetStateAction<boolean>>;
  setDiscovering: React.Dispatch<React.SetStateAction<boolean>>;
  setValidating: React.Dispatch<React.SetStateAction<boolean>>;
  setLastAutoSavedSecretKey: React.Dispatch<React.SetStateAction<string>>;
  setLastAutoDiscoverKey: React.Dispatch<React.SetStateAction<string>>;
  setLastAutoValidateKey: React.Dispatch<React.SetStateAction<string>>;
  setDiscoveredModels: React.Dispatch<React.SetStateAction<string[]>>;
  isCustomProvider: (providerId: string) => boolean;
  validateBaseURL: (value: string) => string | null;
  persistDraftConfig: (cfg: AIConfig) => Promise<void>;
  silentReload: () => Promise<void>;
}

export function useProviderSync(params: UseProviderSyncParams) {
  const { t } = useTranslation();
  const {
    selectedConfig, draft, secretInput, selectedProviderId,
    lastAutoSavedSecretKey, lastAutoDiscoverKey, lastAutoValidateKey,
    setDraft, setSecretInput, setNotice, setError, setValidated,
    setSavingSecret, setDiscovering, setValidating,
    setLastAutoSavedSecretKey, setLastAutoDiscoverKey, setLastAutoValidateKey,
    setDiscoveredModels,
    isCustomProvider, validateBaseURL, persistDraftConfig, silentReload,
  } = params;

  const prevProviderIdRef = useRef<string>('');

  // Provider switch effect
  useEffect(() => {
    const nextId = selectedConfig?.provider_id || '';
    const providerChanged = nextId !== prevProviderIdRef.current;
    prevProviderIdRef.current = nextId;

    if (!selectedConfig) {
      setDraft((prev) => { if (!prev) return prev; return prev.provider_id === selectedProviderId ? prev : null; });
    } else if (providerChanged) {
      const fixedBase = OFFICIAL_PROVIDER_BASE_URLS[selectedConfig.provider_id];
      const enforcedBase = isCustomProvider(selectedConfig.provider_id)
        ? (selectedConfig.base_url || '')
        : (fixedBase || '');
      setDraft({
        ...selectedConfig,
        base_url: enforcedBase,
      });
    } else {
      setDraft((prev) => (prev ? {
        ...prev,
        has_secret: selectedConfig.has_secret,
        updated_at: selectedConfig.updated_at,
        models: prev.models && prev.models.length > 0 ? prev.models : selectedConfig.models,
      } : prev));
    }

    if (providerChanged) {
      setSecretInput('');
      setNotice('');
      setError('');
      setValidated(false);
      setLastAutoSavedSecretKey('');
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- intentional subset to prevent infinite loop
  }, [selectedConfig, selectedProviderId]);

  // Auto-save secret effect
  useEffect(() => {
    if (!draft) return;
    const secret = secretInput.trim();
    if (!secret) return;
    const key = `${draft.provider_id}|${secret}`;
    if (key === lastAutoSavedSecretKey) return;

    const timer = setTimeout(async () => {
      if (isCustomProvider(draft.provider_id) && !draft.name.trim()) {
        setError(t(SYSTEM_A_I_CUSTOM_PROVIDER_NAME_REQUIRED_KEY, { defaultValue: '请先填写自定义厂商名称' }));
        return;
      }
      setSavingSecret(true);
      try {
        await updateSystemAISecret(draft.provider_id, secret);
        setLastAutoSavedSecretKey(key);
        setError('');
        setValidated(false);
        setDraft((prev) => prev ? { ...prev, has_secret: true } : prev);
        setLastAutoDiscoverKey('');
        setNotice(t(SYSTEM_A_I_MESSAGES_SECRET_SAVED_AUTO_DISCOVER_KEY));
        void silentReload();
      } catch (e) {
        const msg = e instanceof Error ? e.message : t(SYSTEM_A_I_MESSAGES_SECRET_AUTO_SAVE_FAILED_KEY);
        setError(msg);
      } finally {
        setSavingSecret(false);
      }
    }, 700);
    return () => clearTimeout(timer);
  // eslint-disable-next-line react-hooks/exhaustive-deps -- intentional subset to prevent infinite loop
  }, [draft?.provider_id, secretInput, lastAutoSavedSecretKey, isCustomProvider, silentReload, t]);

  // Auto-discover effect
  useEffect(() => {
    if (!draft) return;
    const base = (draft.base_url || '').trim();
    if (!base) return;
    if (!draft.has_secret && !secretInput.trim()) return;
    const key = `${draft.provider_id}|${base}|${draft.has_secret ? 'saved' : 'pending'}`;
    if (key === lastAutoDiscoverKey) return;

    const timer = setTimeout(async () => {
      setDiscovering(true);
      try {
        const baseError = validateBaseURL(base);
        if (baseError) {
          setError(toFriendlyDiscoverMessage(baseError, t));
          return;
        }
        await persistDraftConfig(draft);
        const body = await discoverSystemAIModels(draft.provider_id);
        const models = (body?.models || []) as string[];
        if (models.length > 0) {
          setDiscoveredModels(models);
          setDraft((prev) => {
            if (!prev) return prev;
            const next = { ...prev, models };
            if (!(next.default_model || '').trim()) {
              next.default_model = body?.default_model || models[0];
            }
            return next;
          });
          setNotice(t(SYSTEM_A_I_MESSAGES_AUTO_DISCOVERED_MODELS_KEY, { count: models.length }));
          setError('');
          setValidated(false);
            void updateSystemAIConfig(draft.provider_id, {
                models: models || [],
                name: draft.name, base_url: draft.base_url, default_model: draft.default_model,
                enabled: draft.enabled, temperature: draft.temperature,
                timeout_seconds: draft.timeout_seconds, max_tokens: draft.max_tokens,
                purposes: draft.purposes, primary_for: draft.primary_for,
                organization: draft.organization,
              }).catch(() => {});
          setLastAutoDiscoverKey(key);
        }
      } catch (e) {
        const msg = e instanceof Error ? e.message : t(DISCOVER_ERRORS_GENERIC_KEY);
        setError(toFriendlyDiscoverMessage(msg, t));
      } finally {
        setDiscovering(false);
      }
    }, 700);
    return () => clearTimeout(timer);
  // eslint-disable-next-line react-hooks/exhaustive-deps -- intentional subset to prevent infinite loop
  }, [draft?.provider_id, draft?.base_url, draft?.has_secret, secretInput, lastAutoDiscoverKey, validateBaseURL, persistDraftConfig, t]);

  // Auto-validate effect
  useEffect(() => {
    if (!draft) return;
    const model = (draft.default_model || '').trim();
    if (!model) return;
    if (!(draft.has_secret || secretInput.trim())) return;
    const baseError = validateBaseURL(draft.base_url);
    if (baseError) return;
    const key = `${draft.provider_id}|${draft.base_url}|${model}`;
    if (key === lastAutoValidateKey) return;

    const timer = setTimeout(async () => {
      setLastAutoValidateKey(key);
      setValidating(true);
      try {
        await persistDraftConfig(draft);
        if (secretInput.trim()) {
          await updateSystemAISecret(draft.provider_id, secretInput.trim());
        }
        const body = await validateSystemAI(draft.provider_id);
        setValidated(true);
        setNotice(t(SYSTEM_A_I_MESSAGES_AUTO_VALIDATED_MODELS_KEY, { count: body.model_count ?? 0 }));
        setError('');
      } catch (e) {
        const msg = e instanceof Error ? e.message : t(MESSAGES_VALIDATE_FAILED_KEY);
        setValidated(false);
        setError(toFriendlyDiscoverMessage(msg, t));
      } finally {
        setValidating(false);
      }
    }, 500);
    return () => clearTimeout(timer);
  // eslint-disable-next-line react-hooks/exhaustive-deps -- intentional subset to prevent infinite loop
  }, [draft?.provider_id, draft?.base_url, draft?.default_model, draft?.has_secret, secretInput, lastAutoValidateKey, validateBaseURL, persistDraftConfig, t]);

  return { prevProviderIdRef };
}
