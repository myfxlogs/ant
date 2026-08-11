import { useState, useEffect, useCallback } from 'react';
import { message } from 'antd';
import { useTranslation } from 'react-i18next';
import { autoTradingApi } from '@/client/autoTrading';
import type { GlobalSettings, AutoTradingStatus } from '@/gen/ant/v1/auto_trading_settings_pb';
import type { TradingLog } from '@/gen/ant/v1/auto_trading_logs_pb';

export function useAutoTradingSettings() {
  const { t } = useTranslation();
  const [settings, setSettings] = useState<GlobalSettings | null>(null);
  const [status, setStatus] = useState<AutoTradingStatus | null>(null);
  const [logs, setLogs] = useState<TradingLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchAll = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [s, st, recentLogs] = await Promise.all([
        autoTradingApi.getGlobalSettings().catch(() => null),
        autoTradingApi.getAutoTradingStatus().catch(() => null),
        autoTradingApi.getRecentTradingLogs({ limit: 20 }).catch(() => ({ logs: [] })),
      ]);
      setSettings(s);
      setStatus(st);
      setLogs(recentLogs.logs || []);
    } catch {
      setError(t('autoTrading.messages.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => { fetchAll(); }, [fetchAll]);

  const handleToggle = useCallback(async (enabled: boolean) => {
    try {
      const result = await autoTradingApi.toggleAutoTrade(enabled);
      if (result.success !== false) {
        setSettings(prev => prev ? { ...prev, autoTradeEnabled: enabled } : prev);
        message.success(result.message);
      } else {
        message.error(result.message || t('autoTrading.messages.toggleFailed'));
      }
    } catch {
      message.error(t('autoTrading.messages.toggleFailed'));
    }
  }, [t]);

  const handleSave = useCallback(async (values: {
    maxRiskPercent?: number;
    maxPositions?: number;
    maxLotSize?: number;
    maxDailyLoss?: number;
    maxDrawdownPercent?: number;
  }) => {
    setSaving(true);
    try {
      const updated = await autoTradingApi.updateGlobalSettings(values);
      setSettings(updated);
      message.success(t('autoTrading.settings.saveSuccess'));
    } catch {
      message.error(t('autoTrading.settings.saveFailed'));
    } finally {
      setSaving(false);
    }
  }, [t]);

  const handleRefresh = useCallback(() => { fetchAll(); }, [fetchAll]);

  return {
    settings, status, logs,
    loading, saving, error,
    handleToggle, handleSave, handleRefresh,
  };
}
