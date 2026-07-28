import { useEffect, useState } from 'react';
import { Form, message } from 'antd';
import { adminApi, type SystemConfig as AdminConfigType } from '@/client/admin';
import { getErrorMessage } from '@/utils/error';
import { useTranslation } from 'react-i18next';

export function useSystemConfig() {
  const { t } = useTranslation();
  const [configs, setConfigs] = useState<AdminConfigType[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [currentConfig, setCurrentConfig] = useState<AdminConfigType | null>(null);
  const [form] = Form.useForm();

  const isAIProviderCatalog = currentConfig?.value_type === 'json' && currentConfig?.key === 'ai.provider_catalog';
  const isEconAIConfig = currentConfig?.key === 'econ.translation.ai_config';
  const isStrategyHealthConfig = currentConfig?.value_type === 'json' && currentConfig?.key === 'strategy.schedule.health_grading_config';
  const isJSONConfig = currentConfig?.value_type === 'json';

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

  // eslint-disable-next-line react-hooks/exhaustive-deps -- fetchConfigs is stable enough (mount-only)
  useEffect(() => {
    fetchConfigs();
  }, []);

  const handleEdit = (config: AdminConfigType) => {
    setCurrentConfig(config);
    if (config.key === 'econ.translation.ai_config') {
      const raw = (config.value || '').toString().trim();
      let initial: unknown = {
        provider: 'zhipu',
        api_key: '',
        model: 'glm-4-flash',
        base_url: '',
        enabled: true,
      };
      if (raw) {
        try {
          const parsed = JSON.parse(raw);
          if (parsed && typeof parsed === 'object') {
            initial = { ...initial, ...parsed };
          }
        } catch {
          // ignore parse error, use defaults
        }
      }
      form.setFieldsValue({
        provider: initial.provider,
        api_key: initial.api_key,
        model: initial.model,
        base_url: initial.base_url,
        enabled: initial.enabled,
        description: config.description,
      });
    } else {
      form.setFieldsValue({
        value: config.value,
        description: config.description,
      });
    }
    setEditModalVisible(true);
  };

  const handleSave = async (values: Record<string, unknown>) => {
    if (!currentConfig) return;
    try {
      if (isStrategyHealthConfig) {
        const raw = (values.value || '').trim();
        if (!raw) {
          message.error(t('admin.config.validation.jsonEmpty'));
          return;
        }
        let parsed: unknown;
        try {
          parsed = JSON.parse(raw);
        } catch {
          message.error(t('admin.config.validation.jsonInvalid'));
          return;
        }
        const greenSuccessRate = Number(parsed?.green_success_rate);
        const yellowSuccessRate = Number(parsed?.yellow_success_rate);
        const greenMaxFailedRuns = Number(parsed?.green_max_failed_runs);
        const minSampleSize = Number(parsed?.min_sample_size);
        if (!Number.isFinite(greenSuccessRate) || greenSuccessRate < 0 || greenSuccessRate > 100) {
          message.error(t('admin.config.validation.greenSuccessRateRange'));
          return;
        }
        if (!Number.isFinite(yellowSuccessRate) || yellowSuccessRate < 0 || yellowSuccessRate > 100) {
          message.error(t('admin.config.validation.yellowSuccessRateRange'));
          return;
        }
        if (yellowSuccessRate > greenSuccessRate) {
          message.error(t('admin.config.validation.yellowNotGreaterThanGreen'));
          return;
        }
        if (!Number.isFinite(greenMaxFailedRuns) || greenMaxFailedRuns < 0) {
          message.error(t('admin.config.validation.greenMaxFailedRunsNonNegative'));
          return;
        }
        if (!Number.isFinite(minSampleSize) || minSampleSize < 0) {
          message.error(t('admin.config.validation.minSampleSizeNonNegative'));
          return;
        }
      } else if (isEconAIConfig) {
        const provider = (values.provider || 'zhipu').toString().trim();
        const apiKey = (values.api_key || '').toString().trim();
        const model = (values.model || '').toString().trim();
        const baseURL = (values.base_url || '').toString().trim();
        const enabled = values.enabled !== false;
        if (!apiKey) {
          message.error(t('admin.config.validation.apiKeyRequired'));
          return;
        }
        if (!model) {
          message.error(t('admin.config.validation.modelRequired'));
          return;
        }
        const cfg = {
          provider,
          api_key: apiKey,
          model,
          base_url: baseURL,
          enabled,
        };
        await adminApi.setConfig(currentConfig.key, {
          value: JSON.stringify(cfg),
          description: values.description || currentConfig.description || '',
        });
      } else if (isJSONConfig) {
        const raw = (values.value || '').toString().trim();
        if (raw) {
          try {
            JSON.parse(raw);
          } catch {
            message.error(t('admin.config.validation.jsonInvalid'));
            return;
          }
        }
        await adminApi.setConfig(currentConfig.key, values);
      } else {
        await adminApi.setConfig(currentConfig.key, values);
      }
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
    form.setFieldsValue({
      value: JSON.stringify(strategyHealthConfigTemplate, null, 2),
    });
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
