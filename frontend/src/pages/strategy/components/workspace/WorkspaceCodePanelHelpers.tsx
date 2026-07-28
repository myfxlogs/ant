import React, { useMemo } from 'react';
import { KeyOutlined, ThunderboltOutlined } from '@ant-design/icons';
import type { TFunction } from 'i18next';
import { GATEWAY_GROUP_MY_KEYS_KEY, GATEWAY_GROUP_GATEWAY_KEY, GATEWAY_GROUP_CURRENT_KEY } from '@/gen/ant/v1/i18n/ai_core_keys';

export function decodeModel(value: string): { providerId: string; model: string } {
  if (!value) return { providerId: '', model: '' };
  const idx = value.indexOf('|');
  if (idx < 0) return { providerId: value, model: '' };
  return { providerId: value.slice(0, idx), model: value.slice(idx + 1) };
}

interface ConfigItem {
  provider_id: string;
  name?: string;
  enabled: boolean;
  has_secret: boolean;
  models?: string[];
  default_model?: string;
}

interface GatewayModel {
  providerId: string;
  model: string;
  label: string;
}

export function useModelOptions(
  configs: ConfigItem[],
  gatewayModels: GatewayModel[],
  primaryValue: string,
  t: TFunction,
) {
  return useMemo(() => {
    const seen = new Set<string>();
    const groups: Array<{ label: React.ReactNode; title: string; options: Array<{ value: string; label: React.ReactNode; searchLabel: string }> }> = [];
    const ownItems: Array<{ value: string; label: React.ReactNode; searchLabel: string }> = [];
    configs
      .filter(c => c.enabled && c.has_secret)
      .flatMap(c => {
        const models = c.models?.length ? c.models : c.default_model ? [c.default_model] : [];
        return models.map(m => ({ value: `${c.provider_id}|${m}`, label: m }));
      }).forEach(o => {
        if (!seen.has(o.value)) {
          seen.add(o.value);
          ownItems.push({
            value: o.value,
            label: <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}><KeyOutlined style={{ color: '#1677ff', fontSize: 12 }} />{o.label}</span>,
            searchLabel: o.label,
          });
        }
      });
    if (ownItems.length > 0) {
      groups.push({ label: <span style={{ fontSize: 11, color: '#1677ff', fontWeight: 600 }}><KeyOutlined style={{ fontSize: 11 }} /> {t(GATEWAY_GROUP_MY_KEYS_KEY)}</span>, title: 'My Keys', options: ownItems });
    }
    const gwItems: Array<{ value: string; label: React.ReactNode; searchLabel: string }> = [];
    gatewayModels.forEach(g => {
      const v = `${g.providerId}|${g.model}`;
      if (!seen.has(v)) {
        seen.add(v);
        gwItems.push({
          value: v,
          label: <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}><ThunderboltOutlined style={{ color: '#722ed1', fontSize: 12 }} />{g.label}</span>,
          searchLabel: g.label,
        });
      }
    });
    if (gwItems.length > 0) {
      groups.push({ label: <span style={{ fontSize: 11, color: '#722ed1', fontWeight: 600 }}><ThunderboltOutlined style={{ fontSize: 11 }} /> {t(GATEWAY_GROUP_GATEWAY_KEY)}</span>, title: 'Gateway', options: gwItems });
    }
    if (primaryValue && !seen.has(primaryValue)) {
      const [pid, mdl] = primaryValue.split('|');
      if (pid && mdl) {
        groups.unshift({
          label: <span style={{ fontSize: 11, color: '#faad14', fontWeight: 600 }}>{t(GATEWAY_GROUP_CURRENT_KEY)}</span>, title: 'Current', options: [{
            value: primaryValue,
            label: <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>{mdl}</span>,
            searchLabel: mdl,
          }],
        });
      }
    }
    return groups;
  }, [configs, primaryValue, gatewayModels, t]);
}
