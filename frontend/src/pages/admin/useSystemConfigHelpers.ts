import { message } from 'antd';
import type { TFunction } from 'i18next';
import { adminApi, type SystemConfig as AdminConfigType } from '@/client/admin';

interface ConfigFlags {
  isStrategyHealthConfig: boolean;
  isEconAIConfig: boolean;
  isJSONConfig: boolean;
}

export function validateStrategyHealthConfig(raw: string, t: TFunction): boolean {
  if (!raw) { message.error(t('admin.config.validation.jsonEmpty')); return false; }
  let parsed: Record<string, unknown>;
  try { parsed = JSON.parse(raw); } catch { message.error(t('admin.config.validation.jsonInvalid')); return false; }
  const checks: [number, string, (n: number) => boolean][] = [
    [Number(parsed?.green_success_rate), 'admin.config.validation.greenSuccessRateRange', (n) => !Number.isFinite(n) || n < 0 || n > 100],
    [Number(parsed?.yellow_success_rate), 'admin.config.validation.yellowSuccessRateRateRange', (n) => !Number.isFinite(n) || n < 0 || n > 100],
    [Number(parsed?.green_max_failed_runs), 'admin.config.validation.greenMaxFailedRunsNonNegative', (n) => !Number.isFinite(n) || n < 0],
    [Number(parsed?.min_sample_size), 'admin.config.validation.minSampleSizeNonNegative', (n) => !Number.isFinite(n) || n < 0],
  ];
  for (const [val, key, isInvalid] of checks) {
    if (isInvalid(val)) { message.error(t(key)); return false; }
  }
  if (Number(parsed?.yellow_success_rate) > Number(parsed?.green_success_rate)) {
    message.error(t('admin.config.validation.yellowNotGreaterThanGreen'));
    return false;
  }
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
    const raw = String(values.value || '').trim();
    if (!validateStrategyHealthConfig(raw, t)) return false;
  } else if (flags.isEconAIConfig) {
    const { ok, cfg } = validateEconAIConfig(values, t);
    if (!ok) return false;
    await adminApi.setConfig(currentConfig.key, {
      value: JSON.stringify(cfg),
      description: String(values.description || '') || currentConfig.description || '',
    });
    return true;
  } else if (flags.isJSONConfig) {
    const raw = (values.value || '').toString().trim();
    if (!validateJSONConfig(raw, t)) return false;
  }
    await adminApi.setConfig(currentConfig.key, values as { value: string; description?: string });
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
