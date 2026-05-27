import { useAuthStore } from '@/stores/authStore';
import { apiBaseUrl } from '@/client/transport';

async function sreFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = useAuthStore.getState().accessToken;
  const url = `${apiBaseUrl}/api/admin/sre/${path}`;
  const res = await fetch(url, {
    ...init,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}`, ...init?.headers },
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `HTTP ${res.status}`);
  }
  return res.json();
}

export interface KillSwitchStatus {
  engaged: boolean; reason?: string; operator?: string; engaged_at?: string;
}

export interface BreakerStatus {
  strategy_id: string; state: 'closed' | 'open' | 'half_open';
  total_pnl: number; loss_percent: number; trade_count: number;
  tripped_at?: string; trip_reason?: string; allow_probe_trade?: boolean;
}

export interface CanaryConfig {
  strategy_id: string; version_tag: string; account_ids: string[];
  start_at: string; duration_days: number; promoted: boolean;
}

export const sreApi = {
  killSwitchStatus: () => sreFetch<KillSwitchStatus>('killswitch/status'),
  killSwitchEngage: (reason: string, operator: string) =>
    sreFetch<KillSwitchStatus>('killswitch/engage', { method: 'POST', body: JSON.stringify({ reason, operator }) }),
  killSwitchDisengage: () =>
    sreFetch<KillSwitchStatus>('killswitch/disengage', { method: 'POST' }),

  breakersList: () => sreFetch<BreakerStatus[]>('breakers'),
  breakerReset: (strategyId: string) =>
    sreFetch<{ status: string; strategy_id: string }>(`breakers/reset?strategy_id=${encodeURIComponent(strategyId)}`, { method: 'POST' }),

  canaryList: () => sreFetch<CanaryConfig[]>('canary'),
  canarySet: (cfg: Partial<CanaryConfig>) =>
    sreFetch<CanaryConfig>('canary/set', { method: 'POST', body: JSON.stringify(cfg) }),
  canaryDelete: (strategyId: string) =>
    sreFetch<{ status: string; strategy_id: string }>(`canary/delete?strategy_id=${encodeURIComponent(strategyId)}`, { method: 'POST' }),
};
