import { adminSREClient } from '@/client/connect';

export interface KillSwitchStatus {
  engaged: boolean; reason?: string; operator?: string; engaged_at?: string;
}

export interface BreakerStatus {
  strategy_id: string; state: string;
  total_pnl: number; loss_percent: number; trade_count: number;
  tripped_at?: string; trip_reason?: string; allow_probe_trade?: boolean;
}

export const sreApi = {
  killSwitchStatus: async (): Promise<KillSwitchStatus> => {
    const r = await adminSREClient.getKillSwitch({});
    return { engaged: r.enabled, reason: r.reason, operator: r.setBy, engaged_at: r.setAtUnixMs ? new Date(r.setAtUnixMs).toISOString() : undefined };
  },
  killSwitchEngage: async (reason: string, _operator: string): Promise<KillSwitchStatus> => {
    const r = await adminSREClient.setKillSwitch({ enabled: true, reason });
    return { engaged: r.enabled, reason: r.reason, operator: r.setBy, engaged_at: r.setAtUnixMs ? new Date(r.setAtUnixMs).toISOString() : undefined };
  },
  killSwitchDisengage: async (): Promise<KillSwitchStatus> => {
    const r = await adminSREClient.setKillSwitch({ enabled: false, reason: '' });
    return { engaged: r.enabled, reason: r.reason, operator: r.setBy, engaged_at: r.setAtUnixMs ? new Date(r.setAtUnixMs).toISOString() : undefined };
  },
  breakersList: async (): Promise<BreakerStatus[]> => {
    const r = await adminSREClient.listBreakers({});
    return r.breakers.map(b => ({
      strategy_id: b.name, state: b.open ? 'open' : 'closed',
      total_pnl: 0, loss_percent: 0, trade_count: b.failureCount,
    }));
  },
  breakerReset: async (name: string): Promise<BreakerStatus> => {
    const r = await adminSREClient.resetBreaker({ name });
    return { strategy_id: r.name, state: r.open ? 'open' : 'closed', total_pnl: 0, loss_percent: 0, trade_count: 0 };
  },
  canaryList: async (): Promise<{ strategy_id: string; version_tag: string }[]> => {
    const r = await adminSREClient.getCanary({});
    return r.targetVersion ? [{ strategy_id: '', version_tag: r.targetVersion }] : [];
  },
  canarySet: async (strategyId: string, versionTag: string, durationDays: number) => {
    await adminSREClient.setCanary({ enabled: true, targetVersion: versionTag, trafficPercent: durationDays });
  },
};
