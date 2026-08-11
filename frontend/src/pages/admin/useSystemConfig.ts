import { useEffect, useState } from 'react';
import { Form, message } from 'antd';
import { adminApi, type SystemConfig as AdminConfigType } from '@/client/admin';
import { getErrorMessage } from '@/utils/error';
import { useTranslation } from 'react-i18next';
import { saveConfigValue, parseEconAIConfigValue } from './useSystemConfigHelpers';

export function useSystemConfig() {
  const { t } = useTranslation();
  const [configs, setConfigs] = useState<AdminConfigType[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [currentConfig, setCurrentConfig] = useState<AdminConfigType | null>(null);
  const [form] = Form.useForm();

  const isAIProviderCatalog = currentConfig?.valueType === 'json' && currentConfig?.key === 'ai.provider_catalog';
  const isEconAIConfig = currentConfig?.key === 'econ.translation.ai_config';
  const isStrategyHealthConfig = currentConfig?.valueType === 'json' && currentConfig?.key === 'strategy.schedule.health_grading_config';
  const isJSONConfig = currentConfig?.valueType === 'json';

  const strategyHealthConfigTemplate = {
    green_success_rate: 90,
    green_max_failed_runs: 1,
    yellow_success_rate: 60,
    min_sample_size: 1,
  };

  const fetchConfigs = async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await adminApi.listConfigs();
      setConfigs(result || []);
    } catch (err) {
      const msg = getErrorMessage(err, t('admin.config.messages.loadFailed'));
      setError(msg);
      message.error(msg);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchConfigs();
  // eslint-disable-next-line react-hooks/exhaustive-deps -- fetchConfigs is stable enough (mount-only)  | REF: rd.md#part-0.2-hooks-deps
  }, []);

  const handleEdit = (config: AdminConfigType) => {
    setCurrentConfig(config);
    if (config.key === 'econ.translation.ai_config') {
      const initial = parseEconAIConfigValue(config);
      form.setFieldsValue({
        provider: initial.provider,
        api_key: initial.api_key,
        model: initial.model,
        base_url: initial.base_url,
        enabled: initial.enabled,
        description: config.description,
      });
    } else {
      form.setFieldsValue({ value: config.value, description: config.description });
    }
    setEditModalVisible(true);
  };

  const handleSave = async (values: Record<string, unknown>) => {
    if (!currentConfig) return;
    try {
      const ok = await saveConfigValue(currentConfig, values, { isStrategyHealthConfig, isEconAIConfig, isJSONConfig }, t);
      if (!ok) return;
      message.success(t('admin.config.messages.updated'));
      setEditModalVisible(false);
      fetchConfigs();
    } catch (error) {
      message.error(getErrorMessage(error, t('admin.config.messages.updateFailed')));
    }
  };

  const handleFormatJson = () => {
    if (!currentConfig || !isJSONConfig) return;
    const raw = (form.getFieldValue('value') || '').toString().trim();
    if (!raw) return;
    try {
      const obj = JSON.parse(raw);
      form.setFieldsValue({ value: JSON.stringify(obj, null, 2) });
    } catch {
      message.error(t('admin.config.validation.jsonInvalid'));
    }
  };

  const handleUseStrategyHealthTemplate = () => {
    if (!isStrategyHealthConfig) return;
    form.setFieldsValue({ value: JSON.stringify(strategyHealthConfigTemplate, null, 2) });
  };

  const handleToggleEnabled = async (key: string, enabled: boolean) => {
    try {
      await adminApi.toggleConfigEnabled(key, enabled);
      message.success(enabled ? t('admin.config.messages.enabled') : t('admin.config.messages.disabled'));
      fetchConfigs();
    } catch (error) {
      message.error(getErrorMessage(error, t('admin.config.messages.operationFailed')));
    }
  };

  const getKeyLabel = (key: string): string => {
    const labelMap: Record<string, string> = {
      'max_accounts_per_user': t('admin.config.maxAccountsPerUser'),
      'ai.provider_catalog': t('admin.config.aiProviderCatalog'),
      'econ.translation.ai_config': t('admin.config.econAIConfig'),
      'strategy.schedule.health_grading_config': t('admin.config.strategyHealthConfig'),
    };
    return labelMap[key] || key;
  };

  return {
    configs, loading, error, editModalVisible, currentConfig, form,
    isAIProviderCatalog, isEconAIConfig, isStrategyHealthConfig, isJSONConfig,
    fetchConfigs, handleEdit, handleSave, handleFormatJson,
    handleUseStrategyHealthTemplate, handleToggleEnabled, getKeyLabel,
    setEditModalVisible,
  };
}
