import { message } from 'antd';
import type { TFunction } from 'i18next';
import { adminApi, type SystemConfig as AdminConfigType } from '@/client/admin';
import { getErrorMessage } from '@/utils/error';

interface ConfigFlags {
  isStrategyHealthConfig: boolean;
  isEconAIConfig: boolean;
  isJSONConfig: boolean;
}

export function validateStrategyHealthConfig(raw: string, t: TFunction): boolean {
  if (!raw) { message.error(t('admin.config.validation.jsonEmpty')); return false; }
  let parsed: unknown;
  try { parsed = JSON.parse(raw); } catch { message.error(t('admin.config.validation.jsonInvalid')); return false; }
  const greenSuccessRate = Number(parsed?.green_success_rate);
  const yellowSuccessRate = Number(parsed?.yellow_success_rate);
  const greenMaxFailedRuns = Number(parsed?.green_max_failed_runs);
  const minSampleSize = Number(parsed?.min_sample_size);
  if (!Number.isFinite(greenSuccessRate) || greenSuccessRate < 0 || greenSuccessRate > 100) { message.error(t('admin.config.validation.greenSuccessRateRange')); return false; }
  if (!Number.isFinite(yellowSuccessRate) || yellowSuccessRate < 0 || yellowSuccessRate > 100) { message.error(t('admin.config.validation.yellowSuccessRateRange')); return false; }
  if (yellowSuccessRate > greenSuccessRate) { message.error(t('admin.config.validation.yellowNotGreaterThanGreen')); return false; }
  if (!Number.isFinite(greenMaxFailedRuns) || greenMaxFailedRuns < 0) { message.error(t('admin.config.validation.greenMaxFailedRunsNonNegative')); return false; }
  if (!Number.isFinite(minSampleSize) || minSampleSize < 0) { message.error(t('admin.config.validation.minSampleSizeNonNegative')); return false; }
  return true;
}

export function validateEconAIConfig(values: Record<string, unknown>, t: TFunction): { ok: boolean; cfg?: Record<string, unknown> } {
  const provider = (values.provider || 'zhipu').toString().trim();
  const apiKey = (values.api_key || '').toString().trim();
  const model = (values.model || '').toString().trim();
  const baseURL = (values.base_url || '').toString().trim();
  const enabled = values.enabled !== false;
  if (!apiKey) { message.error(t('admin.config.validation.apiKeyRequired')); return { ok: false }; }
  if (!model) { message.error(t('admin.config.validation.modelRequired')); return { ok: false }; }
  return { ok: true, cfg: { provider, api_key: apiKey, model, base_url: baseURL, enabled } };
}

export function validateJSONConfig(raw: string, t: TFunction): boolean {
  if (!raw) return true;
  try { JSON.parse(raw); return true; } catch { message.error(t('admin.config.validation.jsonInvalid')); return false; }
}

export async function saveConfigValue(
  currentConfig: AdminConfigType,
  values: Record<string, unknown>,
  flags: ConfigFlags,
  t: TFunction,
): Promise<boolean> {
  if (flags.isStrategyHealthConfig) {
    const raw = (values.value || '').trim();
    if (!validateStrategyHealthConfig(raw, t)) return false;
  } else if (flags.isEconAIConfig) {
    const { ok, cfg } = validateEconAIConfig(values, t);
    if (!ok) return false;
    await adminApi.setConfig(currentConfig.key, {
      value: JSON.stringify(cfg),
      description: values.description || currentConfig.description || '',
    });
    return true;
  } else if (flags.isJSONConfig) {
    const raw = (values.value || '').toString().trim();
    if (!validateJSONConfig(raw, t)) return false;
  }
  await adminApi.setConfig(currentConfig.key, values);
  return true;
}

export function parseEconAIConfigValue(config: AdminConfigType): Record<string, unknown> {
  const raw = (config.value || '').toString().trim();
  let initial: Record<string, unknown> = { provider: 'zhipu', api_key: '', model: 'glm-4-flash', base_url: '', enabled: true };
  if (raw) {
    try {
      const parsed = JSON.parse(raw);
      if (parsed && typeof parsed === 'object') initial = { ...initial, ...parsed as Record<string, unknown> };
    } catch { /* ignore parse error, use defaults */ }
  }
  return initial;
}

export { getErrorMessage };
